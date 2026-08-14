// Package credential 提供 core 所有的 Registry 运行期凭据解析。
package credential

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"graft/server/internal/moduleapi"
)

const (
	credentialFileVersion       = 1
	maximumCredentialSessionTTL = 10 * time.Minute
	credentialConfigFileMode    = 0o600
	credentialConfigDirMode     = 0o700
)

// FileProvider 从显式部署 Secret 文件解析会过期的 Registry 凭据。
// 明文只会保留在活跃内存会话中，直到 Runtime Target 撤销该会话。
type FileProvider struct {
	path     string
	now      func() time.Time
	mu       sync.Mutex
	sessions map[string]sessionRecord
}

type credentialFile struct {
	Version     int              `json:"version"`
	Credentials []fileCredential `json:"credentials"`
}

type fileCredential struct {
	CredentialRef string    `json:"credential_ref"`
	Endpoint      string    `json:"endpoint"`
	Repositories  []string  `json:"repositories"`
	Operations    []string  `json:"operations"`
	Username      string    `json:"username"`
	Password      string    `json:"password"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type sessionRecord struct {
	endpoint   string
	repository string
	username   string
	password   string
	expiresAt  time.Time
}

// NewFileProvider 为一个绝对的部署挂载 Secret 文件创建提供方。
// 它在创建时校验文件，避免已配置但无效的来源通过运行时准入。
func NewFileProvider(path string) (*FileProvider, error) {
	path = strings.TrimSpace(path)
	if path == "" || !filepath.IsAbs(path) {
		return nil, errors.New("registry credential source is unavailable")
	}
	provider := &FileProvider{path: filepath.Clean(path), now: func() time.Time { return time.Now().UTC() }, sessions: make(map[string]sessionRecord)}
	if _, err := provider.loadCredentials(); err != nil {
		return nil, err
	}
	return provider, nil
}

// Assess 返回已知 Registry scope 的非秘密签发资格。它不创建会话，且不会列出其他凭据。
func (p *FileProvider) Assess(ctx context.Context, request moduleapi.CredentialEligibilityRequest) (moduleapi.CredentialEligibility, error) {
	if p == nil || p.now == nil {
		return moduleapi.CredentialEligibility{}, errors.New("registry credential provider is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return moduleapi.CredentialEligibility{}, err
	}
	request, ok := normalizeEligibilityRequest(request)
	if !ok {
		return moduleapi.CredentialEligibility{Status: moduleapi.CredentialEligibilityIneligible}, nil
	}
	credentials, err := p.loadCredentials()
	if err != nil {
		return moduleapi.CredentialEligibility{}, err
	}
	for _, candidate := range credentials {
		if !credentialMatchesScope(candidate, request.CredentialRef, request.Endpoint, request.RepositoryRef, request.Operation) {
			continue
		}
		if candidate.ExpiresAt.After(p.now().UTC()) {
			return moduleapi.CredentialEligibility{Status: moduleapi.CredentialEligibilityEligible}, nil
		}
		break
	}
	return moduleapi.CredentialEligibility{Status: moduleapi.CredentialEligibilityIneligible}, nil
}

// Prepare 依据最新部署 Secret 校验请求，并且只返回不透明会话句柄。
//
//nolint:cyclop // 凭据作用域、时效和来源校验必须在同一签发边界内完成。
func (p *FileProvider) Prepare(ctx context.Context, request moduleapi.CredentialRequest) (moduleapi.EphemeralCredentialSession, error) {
	if p == nil || p.now == nil {
		return moduleapi.EphemeralCredentialSession{}, errors.New("registry credential provider is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return moduleapi.EphemeralCredentialSession{}, err
	}
	now := p.now().UTC()
	request, err := normalizeRequest(request, now)
	if err != nil {
		return moduleapi.EphemeralCredentialSession{}, err
	}
	credentials, err := p.loadCredentials()
	if err != nil {
		return moduleapi.EphemeralCredentialSession{}, err
	}
	for _, candidate := range credentials {
		if !credentialMatchesScope(candidate, request.CredentialRef, request.Endpoint, request.RepositoryRef, request.Operation) {
			continue
		}
		if !candidate.ExpiresAt.After(now) {
			return moduleapi.EphemeralCredentialSession{}, errors.New("registry credential is expired")
		}
		expiresAt := request.ExpiresAt
		if candidate.ExpiresAt.Before(expiresAt) {
			expiresAt = candidate.ExpiresAt
		}
		id := uuid.NewString()
		p.mu.Lock()
		for existingID, existing := range p.sessions {
			if !existing.expiresAt.After(now) {
				delete(p.sessions, existingID)
			}
		}
		p.sessions[id] = sessionRecord{endpoint: candidate.Endpoint, repository: request.RepositoryRef, username: candidate.Username, password: candidate.Password, expiresAt: expiresAt}
		p.mu.Unlock()
		return moduleapi.EphemeralCredentialSession{ID: id, ExpiresAt: expiresAt}, nil
	}
	return moduleapi.EphemeralCredentialSession{}, errors.New("registry credential scope is unavailable")
}

// Inject 只会向 adapter 创建的隔离 Docker 配置目录写入活跃会话。
//
//nolint:cyclop // 注入前必须同时核验会话、受众和隔离目录，不能拆成可绕过的调用步骤。
func (p *FileProvider) Inject(ctx context.Context, session moduleapi.EphemeralCredentialSession, target moduleapi.CredentialInjectionTarget) error {
	if p == nil || p.now == nil {
		return errors.New("registry credential provider is unavailable")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	endpoint, err := normalizeEndpoint(target.Endpoint)
	if err != nil {
		return err
	}
	repository := normalizeRepository(target.RepositoryRef)
	if repository == "" {
		return errors.New("registry credential injection target is invalid")
	}
	p.mu.Lock()
	record, ok := p.sessions[session.ID]
	if !ok || !record.expiresAt.After(p.now().UTC()) || !session.ExpiresAt.Equal(record.expiresAt) || record.endpoint != endpoint || record.repository != repository {
		p.mu.Unlock()
		return errors.New("registry credential session is invalid")
	}
	p.mu.Unlock()
	return writeDockerConfig(target.ConfigDir, dockerAuthKey(endpoint), record.username, record.password)
}

// Revoke 删除活跃会话；重复清理保持无害。
func (p *FileProvider) Revoke(_ context.Context, session moduleapi.EphemeralCredentialSession) error {
	if p == nil {
		return errors.New("registry credential provider is unavailable")
	}
	p.mu.Lock()
	delete(p.sessions, session.ID)
	p.mu.Unlock()
	return nil
}

func (p *FileProvider) loadCredentials() ([]fileCredential, error) {
	info, err := os.Stat(p.path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o022 != 0 {
		return nil, errors.New("registry credential source is unavailable")
	}
	contents, err := os.ReadFile(p.path)
	if err != nil {
		return nil, errors.New("registry credential source is unavailable")
	}
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	decoder.DisallowUnknownFields()
	var source credentialFile
	if err := decoder.Decode(&source); err != nil || source.Version != credentialFileVersion || len(source.Credentials) == 0 {
		return nil, errors.New("registry credential source is invalid")
	}
	for index := range source.Credentials {
		if err := normalizeCredential(&source.Credentials[index]); err != nil {
			return nil, err
		}
	}
	return source.Credentials, nil
}

//nolint:cyclop // Secret source 的全部结构和作用域约束必须在读取边界统一拒绝。
func normalizeCredential(credential *fileCredential) error {
	if credential == nil {
		return errors.New("registry credential source is invalid")
	}
	credential.CredentialRef = strings.TrimSpace(credential.CredentialRef)
	endpoint, err := normalizeEndpoint(credential.Endpoint)
	if credential.CredentialRef == "" || err != nil || strings.TrimSpace(credential.Username) == "" || credential.Password == "" || credential.ExpiresAt.IsZero() {
		return errors.New("registry credential source is invalid")
	}
	credential.Endpoint = endpoint
	for index, repository := range credential.Repositories {
		credential.Repositories[index] = normalizeRepositoryPattern(repository)
		if credential.Repositories[index] == "" {
			return errors.New("registry credential source is invalid")
		}
	}
	if len(credential.Repositories) == 0 || len(credential.Operations) == 0 {
		return errors.New("registry credential source is invalid")
	}
	for index, operation := range credential.Operations {
		credential.Operations[index] = strings.TrimSpace(operation)
		if credential.Operations[index] == "" {
			return errors.New("registry credential source is invalid")
		}
	}
	return nil
}

func normalizeRequest(request moduleapi.CredentialRequest, now time.Time) (moduleapi.CredentialRequest, error) {
	normalized, ok := normalizeEligibilityRequest(moduleapi.CredentialEligibilityRequest{CredentialRef: request.CredentialRef, Endpoint: request.Endpoint, RepositoryRef: request.RepositoryRef, Operation: request.Operation})
	request.CredentialRef = normalized.CredentialRef
	request.Endpoint = normalized.Endpoint
	request.RepositoryRef = normalized.RepositoryRef
	request.Operation = normalized.Operation
	if !ok || !request.ExpiresAt.After(now) || request.ExpiresAt.After(now.Add(maximumCredentialSessionTTL)) {
		return moduleapi.CredentialRequest{}, errors.New("registry credential request is invalid")
	}
	return request, nil
}

func normalizeEligibilityRequest(request moduleapi.CredentialEligibilityRequest) (moduleapi.CredentialEligibilityRequest, bool) {
	request.CredentialRef = strings.TrimSpace(request.CredentialRef)
	endpoint, err := normalizeEndpoint(request.Endpoint)
	request.Endpoint = endpoint
	request.RepositoryRef = normalizeRepository(request.RepositoryRef)
	request.Operation = strings.TrimSpace(request.Operation)
	return request, request.CredentialRef != "" && err == nil && request.RepositoryRef != "" && request.Operation != ""
}

func credentialMatchesScope(candidate fileCredential, credentialRef, endpoint, repository, operation string) bool {
	return candidate.CredentialRef == credentialRef && candidate.Endpoint == endpoint && repositoryAllowed(candidate.Repositories, repository) && operationAllowed(candidate.Operations, operation)
}

func normalizeEndpoint(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("registry credential scope is invalid")
	}
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String(), nil
}

func dockerAuthKey(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	if parsed.Host == "docker.io" || parsed.Host == "index.docker.io" {
		return "https://index.docker.io/v1/"
	}
	return parsed.Host
}

func normalizeRepository(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "/")
	if value == "" || strings.Contains(value, "//") || strings.ContainsAny(value, "\\\t\r\n") {
		return ""
	}
	return value
}

func normalizeRepositoryPattern(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasSuffix(value, "/*") {
		prefix := normalizeRepository(strings.TrimSuffix(value, "/*"))
		if prefix == "" {
			return ""
		}
		return prefix + "/*"
	}
	return normalizeRepository(value)
}

func repositoryAllowed(patterns []string, repository string) bool {
	for _, pattern := range patterns {
		if pattern == repository || strings.HasSuffix(pattern, "/*") && strings.HasPrefix(repository, strings.TrimSuffix(pattern, "*")) {
			return true
		}
	}
	return false
}

func operationAllowed(operations []string, operation string) bool {
	_, ok := operationSet(operations)[operation]
	return ok
}

func operationSet(operations []string) map[string]struct{} {
	set := make(map[string]struct{}, len(operations))
	for _, operation := range operations {
		set[operation] = struct{}{}
	}
	return set
}

func writeDockerConfig(directory, endpoint, username, password string) error {
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0o077 != 0 {
		return errors.New("registry credential injection target is invalid")
	}
	config := struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}{Auths: make(map[string]struct {
		Auth string `json:"auth"`
	})}
	path := filepath.Join(directory, "config.json")
	// #nosec G304 -- directory is the adapter-created isolated credential directory.
	if contents, readErr := os.ReadFile(path); readErr == nil {
		if err := json.Unmarshal(contents, &config); err != nil || config.Auths == nil {
			return errors.New("read registry credential config")
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return errors.New("read registry credential config")
	}
	config.Auths[endpoint] = struct {
		Auth string `json:"auth"`
	}{Auth: base64.StdEncoding.EncodeToString([]byte(username + ":" + password))}
	contents, err := json.Marshal(config)
	if err != nil {
		return errors.New("create registry credential config")
	}
	// #nosec G304 -- directory is the adapter-created isolated credential directory.
	if err := os.WriteFile(path, contents, credentialConfigFileMode); err != nil {
		return fmt.Errorf("write registry credential config: %w", err)
	}
	return nil
}
