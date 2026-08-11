package registry

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	registrystore "graft/server/modules/registry/store"
)

const (
	registryProviderGenericOCI = "generic_oci"
	verificationUnknown        = "unknown"
	verificationSucceeded      = "succeeded"
	verificationFailed         = "failed"
	registryVerifyTimeout      = 12 * time.Second
	registryHTTPTimeout        = 10 * time.Second
	registryDialTimeout        = 5 * time.Second
)

// ConnectionVerifier probes a saved non-secret endpoint and returns only stable outcome codes.
type ConnectionVerifier interface {
	Verify(context.Context, registrystore.Connection) error
}

// HTTPConnectionVerifier 只探测已保存的公共 Registry endpoint，并在每次拨号前复验 DNS 结果。
// 这使管理 API 不会把任意保存地址变成访问内网的 SSRF 入口。
type HTTPConnectionVerifier struct{ resolver registryEndpointResolver }

type registryEndpointResolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

// NewHTTPConnectionVerifier 创建只允许公共网络地址的 V2 探测器。
func NewHTTPConnectionVerifier() *HTTPConnectionVerifier {
	return &HTTPConnectionVerifier{resolver: net.DefaultResolver}
}

// Verify 探测保存的 endpoint；401 仅证明 V2 challenge 可达，不声称凭据已验证。
//
//nolint:cyclop // 协议状态与脱敏错误分类共同构成调用方可见的探测边界。
func (v *HTTPConnectionVerifier) Verify(ctx context.Context, connection registrystore.Connection) error {
	if v == nil || v.resolver == nil {
		return errors.New("registry verifier is unavailable")
	}
	endpoint, err := registryVerificationEndpoint(connection.Endpoint, connection.Insecure)
	if err != nil {
		return err
	}
	if _, err := resolvePublicRegistryEndpoint(ctx, v.resolver, endpoint.Hostname()); err != nil {
		return err
	}
	client := newRegistryVerificationClient(endpoint, v.resolver)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(endpoint.String(), "/")+"/v2/", nil)
	if err != nil {
		return errors.New("registry_v2_unsupported")
	}
	response, err := client.Do(request)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return errors.New("timeout")
		}
		var urlError *url.Error
		if errors.As(err, &urlError) && strings.Contains(strings.ToLower(urlError.Err.Error()), "certificate") {
			return errors.New("tls_untrusted")
		}
		return errors.New("network_failed")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusUnauthorized {
		return nil
	}
	if response.StatusCode == http.StatusForbidden {
		return errors.New("authorization_denied")
	}
	return errors.New("registry_v2_unsupported")
}

//nolint:cyclop // URL 拒绝条件是同一 SSRF 信任边界，拆开会弱化审计性。
func registryVerificationEndpoint(raw string, insecure bool) (*url.URL, error) {
	endpoint, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || endpoint == nil || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || endpoint.Hostname() == "" {
		return nil, errors.New("registry_v2_unsupported")
	}
	if (insecure && endpoint.Scheme != "http") || (!insecure && endpoint.Scheme != "https") {
		return nil, errors.New("http_disallowed")
	}
	return endpoint, nil
}

func resolvePublicRegistryEndpoint(ctx context.Context, resolver registryEndpointResolver, host string) ([]netip.Addr, error) {
	if address, err := netip.ParseAddr(host); err == nil {
		if !isPublicRegistryAddress(address) {
			return nil, errors.New("network_denied")
		}
		return []netip.Addr{address}, nil
	}
	addresses, err := resolver.LookupNetIP(ctx, "ip", host)
	if err != nil || len(addresses) == 0 {
		return nil, errors.New("dns_failed")
	}
	for _, address := range addresses {
		if !isPublicRegistryAddress(address) {
			return nil, errors.New("network_denied")
		}
	}
	return addresses, nil
}

func newRegistryVerificationClient(endpoint *url.URL, resolver registryEndpointResolver) *http.Client {
	expectedHost := endpoint.Hostname()
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil || !strings.EqualFold(strings.TrimSuffix(host, "."), strings.TrimSuffix(expectedHost, ".")) {
			return nil, errors.New("registry verification dial target is not allowed")
		}
		addresses, err := resolvePublicRegistryEndpoint(ctx, resolver, expectedHost)
		if err != nil {
			return nil, err
		}
		dialer := &net.Dialer{Timeout: registryDialTimeout}
		var lastErr error
		for _, candidate := range addresses {
			connection, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(candidate.String(), port))
			if dialErr == nil {
				return connection, nil
			}
			lastErr = dialErr
		}
		return nil, lastErr
	}
	return &http.Client{Transport: transport, Timeout: registryHTTPTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

func isPublicRegistryAddress(address netip.Addr) bool {
	address = address.Unmap()
	return address.IsValid() && address.IsGlobalUnicast() && !address.IsPrivate() && !address.IsLoopback() && !address.IsLinkLocalUnicast() && !address.IsMulticast() && !address.IsUnspecified()
}

// CreateConnection validates management input before Registry persistence receives it.
func (s *Service) CreateConnection(ctx context.Context, input registrystore.ConnectionInput, actorID uint64) (registrystore.Connection, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return registrystore.Connection{}, err
	}
	if input, err = normalizeConnectionInput(input); err != nil {
		return registrystore.Connection{}, err
	}
	return repository.CreateConnection(ctx, input, actorID)
}

// UpdateConnection 更新连接配置并清除此前验证状态。
func (s *Service) UpdateConnection(ctx context.Context, connectionRef string, input registrystore.ConnectionInput, actorID uint64) (registrystore.Connection, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return registrystore.Connection{}, err
	}
	if input, err = normalizeConnectionInput(input); err != nil {
		return registrystore.Connection{}, err
	}
	return repository.UpdateConnection(ctx, connectionRef, input, actorID)
}

// ListConnections 分页返回 Registry 管理列表。
func (s *Service) ListConnections(ctx context.Context, search string, limit, offset int) ([]registrystore.Connection, int, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return nil, 0, err
	}
	return repository.ListConnections(ctx, search, limit, offset)
}

// GetConnection 读取一个 Registry Connection。
func (s *Service) GetConnection(ctx context.Context, connectionRef string) (registrystore.Connection, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return registrystore.Connection{}, err
	}
	return repository.GetConnection(ctx, connectionRef)
}

// DeleteConnection 软删除未被活跃 Repository 使用的连接。
func (s *Service) DeleteConnection(ctx context.Context, connectionRef string, actorID uint64) error {
	repository, err := s.managementRepository()
	if err != nil {
		return err
	}
	return repository.DeleteConnection(ctx, connectionRef, actorID)
}

// VerifyConnection 持久化保存连接的脱敏 V2 探测结果。
func (s *Service) VerifyConnection(ctx context.Context, connectionRef string, verifier ConnectionVerifier) (registrystore.Connection, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return registrystore.Connection{}, err
	}
	connection, err := repository.GetConnection(ctx, connectionRef)
	if err != nil {
		return registrystore.Connection{}, err
	}
	if !connection.Enabled {
		return repository.SetVerification(ctx, connectionRef, false, verificationFailed, "connection_disabled")
	}
	if verifier == nil {
		return registrystore.Connection{}, errors.New("registry verifier is unavailable")
	}
	verifyCtx, cancel := context.WithTimeout(ctx, registryVerifyTimeout)
	defer cancel()
	if err := verifier.Verify(verifyCtx, connection); err != nil {
		return repository.SetVerification(ctx, connectionRef, false, verificationFailed, verificationErrorCode(err))
	}
	return repository.SetVerification(ctx, connectionRef, true, verificationSucceeded, "")
}

// ListRepositories 返回连接下的受管 Repository。
func (s *Service) ListRepositories(ctx context.Context, connectionRef string) ([]registrystore.Repository, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return nil, err
	}
	return repository.ListRepositories(ctx, connectionRef)
}

// CreateRepository 创建带 push/pull 策略的 Repository。
func (s *Service) CreateRepository(ctx context.Context, connectionRef string, input registrystore.RepositoryInput, actorID uint64) (registrystore.Repository, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return registrystore.Repository{}, err
	}
	if input, err = normalizeRepositoryInput(input); err != nil {
		return registrystore.Repository{}, err
	}
	return repository.CreateRepository(ctx, connectionRef, input, actorID)
}

// UpdateRepository 更新已存在 Repository 的展示和授权策略。
func (s *Service) UpdateRepository(ctx context.Context, connectionRef, repositoryRef string, input registrystore.RepositoryInput, actorID uint64) (registrystore.Repository, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return registrystore.Repository{}, err
	}
	if input, err = normalizeRepositoryInput(input); err != nil {
		return registrystore.Repository{}, err
	}
	return repository.UpdateRepository(ctx, connectionRef, repositoryRef, input, actorID)
}

// DeleteRepository 软删除 Repository。
func (s *Service) DeleteRepository(ctx context.Context, connectionRef, repositoryRef string, actorID uint64) error {
	repository, err := s.managementRepository()
	if err != nil {
		return err
	}
	return repository.DeleteRepository(ctx, connectionRef, repositoryRef, actorID)
}

// ListAssignments 返回 Repository 的有效用户授权。
func (s *Service) ListAssignments(ctx context.Context, connectionRef, repositoryRef string) ([]registrystore.UserAssignment, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return nil, err
	}
	return repository.ListAssignments(ctx, connectionRef, repositoryRef)
}

// ReplaceAssignments 用指定用户集替换 Repository 授权，供批量管理场景使用。
func (s *Service) ReplaceAssignments(ctx context.Context, connectionRef, repositoryRef string, userIDs []uint64, actorID uint64) ([]registrystore.UserAssignment, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return nil, err
	}
	return repository.ReplaceAssignments(ctx, connectionRef, repositoryRef, userIDs, actorID)
}

// GrantAssignment 原子授予单个用户使用 Repository 的权限。
func (s *Service) GrantAssignment(ctx context.Context, connectionRef, repositoryRef string, userID, actorID uint64) (registrystore.UserAssignment, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return registrystore.UserAssignment{}, err
	}
	return repository.GrantAssignment(ctx, connectionRef, repositoryRef, userID, actorID)
}

// RevokeAssignment 原子撤销单个用户授权。
func (s *Service) RevokeAssignment(ctx context.Context, connectionRef, repositoryRef string, userID, actorID uint64) error {
	repository, err := s.managementRepository()
	if err != nil {
		return err
	}
	return repository.RevokeAssignment(ctx, connectionRef, repositoryRef, userID, actorID)
}

// ListAvailableDestinations 返回调用方可提交给 Build v2 的非秘密目的地。
func (s *Service) ListAvailableDestinations(ctx context.Context, actorID uint64) ([]registrystore.Destination, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return nil, err
	}
	return repository.ListAvailableDestinations(ctx, actorID)
}

func (s *Service) managementRepository() (registrystore.ManagementRepository, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("registry management service is unavailable")
	}
	repository, ok := s.repository.(registrystore.ManagementRepository)
	if !ok {
		return nil, errors.New("registry management repository is unavailable")
	}
	return repository, nil
}

//nolint:cyclop // 连接字段和认证模式在一个写入边界内一致校验。
func normalizeConnectionInput(input registrystore.ConnectionInput) (registrystore.ConnectionInput, error) {
	input.ConnectionRef = strings.TrimSpace(input.ConnectionRef)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Provider = strings.TrimSpace(input.Provider)
	input.CredentialRef = strings.TrimSpace(input.CredentialRef)
	input.Description = strings.TrimSpace(input.Description)
	input.AuthMode = strings.TrimSpace(input.AuthMode)
	if input.ConnectionRef == "" || len(input.ConnectionRef) > 128 || input.DisplayName == "" || len(input.DisplayName) > 128 || input.Provider != registryProviderGenericOCI || len(input.Description) > 500 {
		return registrystore.ConnectionInput{}, errors.New("invalid registry connection")
	}
	if input.AuthMode == "" {
		if input.CredentialRef == "" {
			input.AuthMode = registrystore.AuthModeAnonymous
		} else {
			input.AuthMode = registrystore.AuthModeCredentialRef
		}
	}
	if input.AuthMode != registrystore.AuthModeAnonymous && input.AuthMode != registrystore.AuthModeCredentialRef {
		return registrystore.ConnectionInput{}, errors.New("invalid registry authentication mode")
	}
	if input.AuthMode == registrystore.AuthModeAnonymous {
		input.CredentialRef = ""
	}
	if input.AuthMode == registrystore.AuthModeCredentialRef && input.CredentialRef == "" {
		return registrystore.ConnectionInput{}, errors.New("registry credential reference is required")
	}
	endpoint, err := normalizeEndpoint(input.Endpoint, input.Insecure)
	if err != nil {
		return registrystore.ConnectionInput{}, err
	}
	input.Endpoint = endpoint
	return input, nil
}

//nolint:cyclop // endpoint 的 URL 信任约束不能拆散到调用方。
func normalizeEndpoint(raw string, insecure bool) (string, error) {
	endpoint, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || endpoint == nil || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || endpoint.Hostname() == "" {
		return "", errors.New("invalid registry endpoint")
	}
	if (insecure && endpoint.Scheme != "http") || (!insecure && endpoint.Scheme != "https") {
		return "", errors.New("registry endpoint scheme is invalid")
	}
	endpoint.Host = strings.ToLower(endpoint.Host)
	endpoint.Path = strings.TrimRight(endpoint.Path, "/")
	return endpoint.String(), nil
}

func normalizeRepositoryInput(input registrystore.RepositoryInput) (registrystore.RepositoryInput, error) {
	input.RepositoryRef = strings.Trim(strings.TrimSpace(input.RepositoryRef), "/")
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.RepositoryRef == "" || len(input.RepositoryRef) > 255 || input.DisplayName == "" || len(input.DisplayName) > 128 || strings.Contains(input.RepositoryRef, "//") || strings.ContainsAny(input.RepositoryRef, "\\\t\r\n") {
		return registrystore.RepositoryInput{}, errors.New("invalid artifact repository")
	}
	return input, nil
}

func verificationErrorCode(err error) string {
	code := strings.TrimSpace(err.Error())
	if code == "timeout" || code == "authorization_denied" || code == "registry_v2_unsupported" || code == "network_failed" || code == "dns_failed" || code == "network_denied" || code == "tls_untrusted" || code == "http_disallowed" {
		return code
	}
	return "verification_failed"
}
