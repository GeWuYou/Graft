package credentialvault

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"graft/server/internal/config"
	"graft/server/internal/moduleapi"
)

// ErrAgentCertificateIssuerUnavailable 表示当前进程没有已配置的 Vault PKI adapter。
// 调用方必须将其视为 fail-closed 的服务不可用状态，不能以本地凭据替代。
var ErrAgentCertificateIssuerUnavailable = errors.New("agent certificate issuer is unavailable")

const (
	vaultRequestTimeout   = 10 * time.Second
	vaultResponseMaxBytes = 1 << 20
)

// VaultPKIAdapter 是 Credential Vault 拥有的真实 Vault PKI 集成 adapter seam。
// adapter 只能返回非秘密的 moduleapi DTO；私钥、PEM 和 enrollment secret 必须留在 Vault 或部署交付
// 通道中，不能由本模块 materialize。
type VaultPKIAdapter interface {
	moduleapi.AgentCertificateIssuer
}

// unavailableAgentCertificateIssuer 使已启用但尚未实现的 Vault 集成显式 fail-closed。
// 它刻意不含持久化或 fallback 行为，避免部分配置的部署在本地签发或伪造 Agent 证书。
type unavailableAgentCertificateIssuer struct{}

// IssueCSR 在真实 Vault PKI adapter 注册前拒绝签发。
func (unavailableAgentCertificateIssuer) IssueCSR(context.Context, moduleapi.AgentCertificateIssuanceRequest) (moduleapi.IssuedAgentCertificate, error) {
	return moduleapi.IssuedAgentCertificate{}, ErrAgentCertificateIssuerUnavailable
}

// ReconcileCSR 在真实 Vault PKI adapter 注册前拒绝读取签发协调结果。
func (unavailableAgentCertificateIssuer) ReconcileCSR(context.Context, string) (moduleapi.IssuedAgentCertificate, error) {
	return moduleapi.IssuedAgentCertificate{}, ErrAgentCertificateIssuerUnavailable
}

// ReadTrustBundle 在真实 Vault PKI adapter 注册前拒绝信任束读取。
func (unavailableAgentCertificateIssuer) ReadTrustBundle(context.Context, moduleapi.TrustBundleRequest) (moduleapi.TrustBundleReference, error) {
	return moduleapi.TrustBundleReference{}, ErrAgentCertificateIssuerUnavailable
}

// RevokeCertificate 在真实 Vault PKI adapter 注册前拒绝证书撤销。
func (unavailableAgentCertificateIssuer) RevokeCertificate(context.Context, moduleapi.AgentCertificateRevocation) error {
	return ErrAgentCertificateIssuerUnavailable
}

var _ VaultPKIAdapter = unavailableAgentCertificateIssuer{}

// IssuanceState 保存可重启恢复的非秘密 Vault 签发证据。实现必须将其持久化到模块拥有的 durable store。
type IssuanceState struct {
	IssuanceKey string
	Serial      string
}

// IssuanceStateStore 是 Vault adapter 的窄持久化边界，不保存证书 PEM、私钥或令牌。
type IssuanceStateStore interface {
	Load(ctx context.Context, issuanceKey string) (IssuanceState, error)
	Save(ctx context.Context, state IssuanceState) error
}

// SQLIssuanceStateStore 将 Vault 签发恢复所需的非秘密序列号保存到 Credential Vault 自有表。
type SQLIssuanceStateStore struct {
	db *sql.DB
}

// NewSQLIssuanceStateStore 创建使用运行时共享连接池的签发状态仓储。
func NewSQLIssuanceStateStore(db *sql.DB) (*SQLIssuanceStateStore, error) {
	if db == nil {
		return nil, errors.New("credential vault issuance state database is required")
	}
	return &SQLIssuanceStateStore{db: db}, nil
}

// Load 读取稳定签发键对应的序列号；不存在时返回统一的恢复哨兵错误。
func (s *SQLIssuanceStateStore) Load(ctx context.Context, issuanceKey string) (IssuanceState, error) {
	if s == nil || s.db == nil || strings.TrimSpace(issuanceKey) == "" {
		return IssuanceState{}, moduleapi.ErrAgentCertificateIssuanceNotFound
	}
	var state IssuanceState
	err := s.db.QueryRowContext(ctx, `SELECT issuance_key, certificate_serial
FROM credential_vault_issuance_states
WHERE issuance_key = $1 AND deleted_at = 0`, strings.TrimSpace(issuanceKey)).Scan(&state.IssuanceKey, &state.Serial)
	if errors.Is(err, sql.ErrNoRows) {
		return IssuanceState{}, moduleapi.ErrAgentCertificateIssuanceNotFound
	}
	if err != nil {
		return IssuanceState{}, fmt.Errorf("load credential vault issuance state: %w", err)
	}
	return state, nil
}

// Save 持久化签发序列号，并拒绝同一签发键绑定到不同序列号。
func (s *SQLIssuanceStateStore) Save(ctx context.Context, state IssuanceState) error {
	if s == nil || s.db == nil || strings.TrimSpace(state.IssuanceKey) == "" || strings.TrimSpace(state.Serial) == "" {
		return errors.New("credential vault issuance state is invalid")
	}
	key, serial := strings.TrimSpace(state.IssuanceKey), strings.TrimSpace(state.Serial)
	_, err := s.db.ExecContext(ctx, `INSERT INTO credential_vault_issuance_states (issuance_key, certificate_serial)
VALUES ($1, $2) ON CONFLICT (issuance_key) DO NOTHING`, key, serial)
	if err != nil {
		return fmt.Errorf("save credential vault issuance state: %w", err)
	}
	var recorded string
	if err := s.db.QueryRowContext(ctx, `SELECT certificate_serial FROM credential_vault_issuance_states
WHERE issuance_key = $1 AND deleted_at = 0`, key).Scan(&recorded); err != nil {
		return fmt.Errorf("verify credential vault issuance state: %w", err)
	}
	if recorded != serial {
		return errors.New("credential vault issuance key is already bound to another serial")
	}
	return nil
}

// VaultPKIClient 使用 Vault AppRole 与 PKI HTTP API；秘密仅在请求生命周期内驻留内存。
type VaultPKIClient struct {
	config config.CredentialVaultConfig
	store  IssuanceStateStore
	http   *http.Client
}

// NewVaultPKIClient 创建生产 Vault adapter。store 为空时拒绝启动，避免内存-only 恢复语义。
func NewVaultPKIClient(configuration config.CredentialVaultConfig, store IssuanceStateStore) (*VaultPKIClient, error) {
	if store == nil {
		return nil, errors.New("vault PKI issuance state store is required")
	}
	if err := validateVaultPKIConfiguration(configuration); err != nil {
		return nil, err
	}
	client, err := newVaultTLSClient(configuration.CAFile)
	if err != nil {
		return nil, err
	}
	return &VaultPKIClient{config: configuration, store: store, http: client}, nil
}

func validateVaultPKIConfiguration(configuration config.CredentialVaultConfig) error {
	if strings.TrimSpace(configuration.Address) == "" || strings.TrimSpace(configuration.CAFile) == "" || strings.TrimSpace(configuration.AuthMount) == "" || strings.TrimSpace(configuration.AuthRole) == "" || strings.TrimSpace(configuration.AuthRoleIDFile) == "" || strings.TrimSpace(configuration.AuthSecretIDFile) == "" || strings.TrimSpace(configuration.PKIMount) == "" || strings.TrimSpace(configuration.PKIRole) == "" {
		return errors.New("vault PKI configuration is incomplete")
	}
	return nil
}

func newVaultTLSClient(caFile string) (*http.Client, error) {
	// #nosec G304 -- caFile 是已校验的 Vault 信任锚生产配置。
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read Vault CA file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("vault CA file contains no certificates")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool}
	return &http.Client{Timeout: vaultRequestTimeout, Transport: transport}, nil
}

// IssueCSR 使用稳定签发键协调 Vault PKI 外部副作用，并只返回非秘密证书材料。
//
//nolint:cyclop // 签发流程必须在同一边界内完成 durable 状态恢复、Vault 调用和非秘密结果校验。
func (v *VaultPKIClient) IssueCSR(ctx context.Context, request moduleapi.AgentCertificateIssuanceRequest) (moduleapi.IssuedAgentCertificate, error) {
	if v == nil || strings.TrimSpace(request.IssuanceKey) == "" || strings.TrimSpace(request.SPIFFEURI) == "" || len(request.CSRDER) == 0 {
		return moduleapi.IssuedAgentCertificate{}, errors.New("invalid certificate issuance request")
	}
	csr, err := x509.ParseCertificateRequest(request.CSRDER)
	if err != nil || csr.CheckSignature() != nil {
		return moduleapi.IssuedAgentCertificate{}, errors.New("invalid certificate signing request")
	}
	if state, err := v.store.Load(ctx, request.IssuanceKey); err == nil && state.Serial != "" {
		return v.readCertificate(ctx, request.IssuanceKey, state.Serial)
	} else if err != nil && !errors.Is(err, moduleapi.ErrAgentCertificateIssuanceNotFound) {
		return moduleapi.IssuedAgentCertificate{}, fmt.Errorf("load issuance state: %w", err)
	}
	token, err := v.login(ctx)
	if err != nil {
		return moduleapi.IssuedAgentCertificate{}, err
	}
	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr.Raw})
	var response vaultIssueResponse
	if err := v.call(ctx, token, http.MethodPost, "/v1/"+pathEscape(v.config.PKIMount)+"/sign/"+pathEscape(v.config.PKIRole), map[string]any{"csr": string(csrPEM), "uri_sans": strings.TrimSpace(request.SPIFFEURI), "format": "pem"}, &response); err != nil {
		return moduleapi.IssuedAgentCertificate{}, err
	}
	serial := strings.TrimSpace(response.Data.SerialNumber)
	if serial == "" {
		return moduleapi.IssuedAgentCertificate{}, errors.New("vault PKI response missing serial number")
	}
	if err := v.store.Save(ctx, IssuanceState{IssuanceKey: request.IssuanceKey, Serial: serial}); err != nil {
		return moduleapi.IssuedAgentCertificate{}, fmt.Errorf("save issuance state: %w", err)
	}
	return v.readCertificateResponse(request.IssuanceKey, response.Data)
}

// ReconcileCSR 从 durable 状态恢复同一签发键对应的 Vault 证书结果。
func (v *VaultPKIClient) ReconcileCSR(ctx context.Context, issuanceKey string) (moduleapi.IssuedAgentCertificate, error) {
	if v == nil || strings.TrimSpace(issuanceKey) == "" {
		return moduleapi.IssuedAgentCertificate{}, moduleapi.ErrAgentCertificateIssuanceNotFound
	}
	state, err := v.store.Load(ctx, issuanceKey)
	if err != nil {
		return moduleapi.IssuedAgentCertificate{}, err
	}
	if state.Serial == "" {
		return moduleapi.IssuedAgentCertificate{}, moduleapi.ErrAgentCertificateIssuanceNotFound
	}
	return v.readCertificate(ctx, issuanceKey, state.Serial)
}

// ReadTrustBundle 返回不透明信任束引用及其 Vault PKI CA 到期时间，不将 PEM 带出模块边界。
func (v *VaultPKIClient) ReadTrustBundle(ctx context.Context, _ moduleapi.TrustBundleRequest) (moduleapi.TrustBundleReference, error) {
	if v == nil || strings.TrimSpace(v.config.TrustBundleRef) == "" {
		return moduleapi.TrustBundleReference{}, errors.New("vault trust bundle is not configured")
	}
	token, err := v.login(ctx)
	if err != nil {
		return moduleapi.TrustBundleReference{}, err
	}
	var response vaultTrustBundleResponse
	if err := v.call(ctx, token, http.MethodGet, "/v1/"+pathEscape(v.config.PKIMount)+"/cert/ca", nil, &response); err != nil {
		return moduleapi.TrustBundleReference{}, err
	}
	certificate, err := decodeCertificate(response.Data.Certificate)
	if err != nil {
		return moduleapi.TrustBundleReference{}, err
	}
	if !certificate.NotAfter.After(time.Now().UTC()) {
		return moduleapi.TrustBundleReference{}, errors.New("vault trust bundle is expired")
	}
	return moduleapi.TrustBundleReference{Reference: v.config.TrustBundleRef, Version: "vault-pki", ExpiresAt: certificate.NotAfter.UTC()}, nil
}

// RevokeCertificate 向 Vault 提交幂等证书撤销请求。
func (v *VaultPKIClient) RevokeCertificate(ctx context.Context, revocation moduleapi.AgentCertificateRevocation) error {
	if v == nil || strings.TrimSpace(revocation.CertificateSerial) == "" {
		return errors.New("certificate serial is required")
	}
	token, err := v.login(ctx)
	if err != nil {
		return err
	}
	return v.call(ctx, token, http.MethodPost, "/v1/"+pathEscape(v.config.PKIMount)+"/revoke", map[string]any{"serial_number": revocation.CertificateSerial}, nil)
}

type vaultIssueResponse struct {
	Data vaultCertificateData `json:"data"`
}
type vaultCertificateData struct {
	Certificate  string   `json:"certificate"`
	CAChain      []string `json:"ca_chain"`
	SerialNumber string   `json:"serial_number"`
	Expiration   int64    `json:"expiration"`
	IssuingCA    string   `json:"issuing_ca"`
}
type vaultLoginResponse struct {
	Auth struct {
		ClientToken string `json:"client_token"`
	} `json:"auth"`
}
type vaultTrustBundleResponse struct {
	Data struct {
		Certificate string `json:"certificate"`
	} `json:"data"`
}

func (v *VaultPKIClient) login(ctx context.Context) (string, error) {
	roleID, err := os.ReadFile(v.config.AuthRoleIDFile)
	if err != nil {
		return "", fmt.Errorf("read vault role id file: %w", err)
	}
	secretID, err := os.ReadFile(v.config.AuthSecretIDFile)
	if err != nil {
		return "", fmt.Errorf("read vault secret id file: %w", err)
	}
	var response vaultLoginResponse
	if err := v.call(ctx, "", http.MethodPost, "/v1/auth/"+pathEscape(v.config.AuthMount)+"/login", map[string]any{"role_id": strings.TrimSpace(string(roleID)), "secret_id": strings.TrimSpace(string(secretID))}, &response); err != nil {
		return "", err
	}
	if strings.TrimSpace(response.Auth.ClientToken) == "" {
		return "", errors.New("vault AppRole login returned no token")
	}
	return response.Auth.ClientToken, nil
}

func (v *VaultPKIClient) readCertificate(ctx context.Context, issuanceKey, serial string) (moduleapi.IssuedAgentCertificate, error) {
	token, err := v.login(ctx)
	if err != nil {
		return moduleapi.IssuedAgentCertificate{}, err
	}
	var response vaultIssueResponse
	if err := v.call(ctx, token, http.MethodGet, "/v1/"+pathEscape(v.config.PKIMount)+"/cert/"+url.PathEscape(serial), nil, &response); err != nil {
		return moduleapi.IssuedAgentCertificate{}, err
	}
	// 持久化序列号是重启协调的 authority，不能被 Vault 读取响应中的字段覆盖。
	response.Data.SerialNumber = serial
	return v.readCertificateResponse(issuanceKey, response.Data)
}

func (v *VaultPKIClient) readCertificateResponse(issuanceKey string, data vaultCertificateData) (moduleapi.IssuedAgentCertificate, error) {
	leaf, err := decodeCertificate(data.Certificate)
	if err != nil {
		return moduleapi.IssuedAgentCertificate{}, err
	}
	chain := [][]byte{leaf.Raw}
	for _, item := range data.CAChain {
		cert, err := decodeCertificate(item)
		if err != nil {
			return moduleapi.IssuedAgentCertificate{}, err
		}
		chain = append(chain, cert.Raw)
	}
	if strings.TrimSpace(data.IssuingCA) != "" && len(chain) == 1 {
		cert, err := decodeCertificate(data.IssuingCA)
		if err == nil {
			chain = append(chain, cert.Raw)
		}
	}
	fingerprint := sha256.Sum256(leaf.RawSubjectPublicKeyInfo)
	expires := leaf.NotAfter
	if data.Expiration > 0 {
		expires = time.Unix(data.Expiration, 0).UTC()
	}
	// X.509 的十进制 serial 是 mTLS listener 导出的唯一 canonical 身份证据；Vault 的原始
	// serial 仅保存在 issuance state 中，用于后续 Vault 证书读取。
	serial := leaf.SerialNumber.String()
	return moduleapi.IssuedAgentCertificate{IssuanceKey: issuanceKey, CertificateSerial: serial, CertificateChainDER: chain, PublicKeyFingerprint: "sha256:" + hex.EncodeToString(fingerprint[:]), ExpiresAt: expires, TrustBundle: moduleapi.TrustBundleReference{Reference: v.config.TrustBundleRef, Version: "vault-pki", ExpiresAt: expires}}, nil
}

func decodeCertificate(value string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, errors.New("vault response missing certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse vault certificate: %w", err)
	}
	return cert, nil
}

//nolint:cyclop // HTTP 请求构造、响应分类和受限解码必须在同一外部系统边界内完成。
func (v *VaultPKIClient) call(ctx context.Context, token, method, path string, body map[string]any, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = strings.NewReader(string(encoded))
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(v.config.Address, "/")+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Vault-Token", token)
	}
	if strings.TrimSpace(v.config.Namespace) != "" {
		req.Header.Set("X-Vault-Namespace", v.config.Namespace)
	}
	client := v.http
	if client == nil {
		client = &http.Client{Timeout: vaultRequestTimeout}
	}
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("vault request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("vault request returned status %d", response.StatusCode)
	}
	if out == nil {
		return nil
	}
	limitedBody := &io.LimitedReader{R: response.Body, N: vaultResponseMaxBytes + 1}
	if err := json.NewDecoder(limitedBody).Decode(out); err != nil {
		if limitedBody.N == 0 {
			return errors.New("vault response exceeds maximum size")
		}
		return fmt.Errorf("decode vault response: %w", err)
	}
	if limitedBody.N == 0 {
		return errors.New("vault response exceeds maximum size")
	}
	return nil
}
func pathEscape(value string) string { return url.PathEscape(strings.Trim(value, "/")) }

var _ VaultPKIAdapter = (*VaultPKIClient)(nil)
