package registry

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"graft/server/internal/moduleapi"
	registrystore "graft/server/modules/registry/store"
)

const (
	registryProviderGenericOCI = "generic_oci"
	verificationUnknown        = "unknown"
	verificationSucceeded      = "verified"
	verificationFailed         = "failed"
	registryVerifyTimeout      = 12 * time.Second
)

var repositoryReferencePattern = regexp.MustCompile(`^[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*(?:/[a-z0-9]+(?:(?:[._]|__|-+)[a-z0-9]+)*)*$`)

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
	existing, err := repository.GetConnection(ctx, connectionRef)
	if err != nil {
		return registrystore.Connection{}, err
	}
	if existing.SystemManaged {
		return registrystore.Connection{}, registrystore.ErrSystemManaged
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
	connection, err := repository.GetConnection(ctx, connectionRef)
	if err != nil {
		return err
	}
	if connection.SystemManaged {
		return registrystore.ErrSystemManaged
	}
	return repository.DeleteConnection(ctx, connectionRef, actorID)
}

// VerificationInput 是一次认证验证的瞬时非秘密选择；它不成为连接或 Build 目的地事实。
type VerificationInput struct {
	RuntimeTargetID int64
	RepositoryRef   string
}

// VerifyConnection 通过 Runtime Target 的隔离凭据执行边界验证保存的连接，并持久化 Registry-owned 的脱敏结果。
func (s *Service) VerifyConnection(ctx context.Context, connectionRef string, actorID uint64, input VerificationInput) (registrystore.Connection, error) {
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
	binding, errorCode, err := s.prepareConnectionVerification(ctx, connectionRef, actorID, input)
	if err != nil {
		return registrystore.Connection{}, err
	}
	if errorCode != "" {
		return repository.SetVerification(ctx, connectionRef, false, verificationFailed, errorCode)
	}
	return s.executeConnectionVerification(ctx, repository, connectionRef, input.RuntimeTargetID, binding)
}

func (s *Service) prepareConnectionVerification(ctx context.Context, connectionRef string, actorID uint64, input VerificationInput) (registrystore.AuthorizedRepository, string, error) {
	input.RepositoryRef = strings.TrimSpace(input.RepositoryRef)
	if err := s.authorizeVerificationTarget(ctx, actorID, input.RuntimeTargetID); err != nil {
		return registrystore.AuthorizedRepository{}, "", err
	}
	binding, err := s.repository.ResolveRepositoryBinding(ctx, connectionRef, input.RepositoryRef)
	if err != nil {
		return registrystore.AuthorizedRepository{}, "", err
	}
	if binding.ConnectionRef != connectionRef || binding.RepositoryRef != input.RepositoryRef || strings.TrimSpace(binding.Endpoint) == "" || strings.TrimSpace(binding.CredentialRef) == "" {
		return registrystore.AuthorizedRepository{}, "credential_not_configured", nil
	}
	if s.execution == nil {
		return registrystore.AuthorizedRepository{}, "verification_unavailable", nil
	}
	return binding, "", nil
}

func (s *Service) authorizeVerificationTarget(ctx context.Context, actorID uint64, targetID int64) error {
	if targetID < 1 {
		return errors.New("invalid registry verification input")
	}
	if actorID == 0 || s.targets == nil {
		return errors.New("registry verification target authorization is unavailable")
	}
	allowed, err := s.targets.CanUseBuildTarget(ctx, actorID, targetID)
	if err != nil {
		return fmt.Errorf("authorize registry verification target: %w", err)
	}
	if !allowed {
		return errors.New("registry verification target is not authorized")
	}
	return nil
}

func (s *Service) executeConnectionVerification(ctx context.Context, repository registrystore.ManagementRepository, connectionRef string, targetID int64, binding registrystore.AuthorizedRepository) (registrystore.Connection, error) {
	verifyCtx, cancel := context.WithTimeout(ctx, registryVerifyTimeout)
	defer cancel()
	result, err := s.execution.VerifyOCIRegistry(verifyCtx, moduleapi.OCIRegistryVerificationRequest{
		RuntimeTargetID: targetID,
		CredentialRef:   binding.CredentialRef,
		Endpoint:        binding.Endpoint,
		RepositoryRef:   binding.RepositoryRef,
		Operation:       "push",
	})
	if err != nil || !verifiedOCIRegistryAuthentication(result) {
		return repository.SetVerification(ctx, connectionRef, false, verificationFailed, "verification_failed")
	}
	return repository.SetVerification(ctx, connectionRef, true, verificationSucceeded, "")
}

func verifiedOCIRegistryAuthentication(result moduleapi.OCIRegistryVerificationResult) bool {
	return result.Reachable && result.ProtocolCompatible && result.AuthenticationChallenged && result.AuthenticationSucceeded && result.ProviderScopeConforms
}

// ListRepositories 返回连接下的受管 Repository。
func (s *Service) ListRepositories(ctx context.Context, connectionRef string, limit, offset int) ([]registrystore.Repository, int, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return nil, 0, fmt.Errorf("list registry artifact repositories: %w", err)
	}
	return repository.ListRepositories(ctx, connectionRef, limit, offset)
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
func (s *Service) ListAssignments(ctx context.Context, connectionRef, repositoryRef string, limit, offset int) ([]registrystore.UserAssignment, int, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return nil, 0, fmt.Errorf("list registry artifact repository assignments: %w", err)
	}
	return repository.ListAssignments(ctx, connectionRef, repositoryRef, limit, offset)
}

// AssignmentCandidate 是 Registry 对候选用户授权状态的管理面投影。
type AssignmentCandidate struct {
	UserID                  uint64
	Username                string
	Display                 string
	Status                  string
	AssignedRepositoryCount int
	SelectedRepositoryCount int
	AuthorizationState      registrystore.AssignmentCandidateState
}

// ListAssignmentCandidates 返回搜索和分页后的用户候选，并聚合其在所选 Repository 的授权状态。
func (s *Service) ListAssignmentCandidates(ctx context.Context, connectionRef string, repositoryRefs []string, search string, limit, offset int) ([]AssignmentCandidate, int, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return nil, 0, fmt.Errorf("list registry repository assignment candidates: %w", err)
	}
	repositoryRefs, err = validateAssignmentCandidateQuery(s, repositoryRefs, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	users, total, err := s.users.ListUserCandidates(ctx, moduleapi.UserCandidateQuery{Search: search, Limit: limit, Offset: offset})
	if err != nil {
		return nil, 0, fmt.Errorf("list candidate users: %w", err)
	}
	userIDs := make([]uint64, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}
	states, err := repository.ListAssignmentCandidates(ctx, connectionRef, repositoryRefs, userIDs)
	if err != nil {
		return nil, 0, err
	}
	items := make([]AssignmentCandidate, 0, len(users))
	for _, user := range users {
		state, exists := states[user.ID]
		if !exists {
			state = registrystore.AssignmentCandidate{UserID: user.ID, SelectedRepositoryCount: len(repositoryRefs), AuthorizationState: registrystore.AssignmentCandidateStateNone}
		}
		items = append(items, AssignmentCandidate{UserID: user.ID, Username: user.Username, Display: user.Display, Status: user.Status, AssignedRepositoryCount: state.AssignedRepositoryCount, SelectedRepositoryCount: state.SelectedRepositoryCount, AuthorizationState: state.AuthorizationState})
	}
	return items, total, nil
}

func validateAssignmentCandidateQuery(service *Service, repositoryRefs []string, limit, offset int) ([]string, error) {
	if service == nil || service.users == nil {
		return nil, errors.New("registry user candidate reader is unavailable")
	}
	repositoryRefs = normalizeRepositoryRefs(repositoryRefs)
	if len(repositoryRefs) == 0 {
		return nil, errors.New("invalid repository references")
	}
	if limit < 1 || limit > 100 || offset < 0 {
		return nil, errors.New("invalid candidate page bounds")
	}
	return repositoryRefs, nil
}

func normalizeRepositoryRefs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	refs := make([]string, 0, len(values))
	for _, value := range values {
		ref := strings.TrimSpace(value)
		if ref == "" {
			continue
		}
		if _, exists := seen[ref]; exists {
			continue
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}
	return refs
}

// ReplaceAssignments 用指定用户集替换 Repository 授权，供批量管理场景使用。
func (s *Service) ReplaceAssignments(ctx context.Context, connectionRef, repositoryRef string, userIDs []uint64, actorID uint64) ([]registrystore.UserAssignment, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return nil, err
	}
	return repository.ReplaceAssignments(ctx, connectionRef, repositoryRef, userIDs, actorID)
}

// AddAssignments 原子追加一组用户到一组 Repository，并保留既有有效授权。
func (s *Service) AddAssignments(ctx context.Context, connectionRef string, input registrystore.AssignmentBatchAddInput, actorID uint64) (registrystore.AssignmentBatchAddResult, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return registrystore.AssignmentBatchAddResult{}, err
	}
	return repository.AddAssignments(ctx, connectionRef, input, actorID)
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
func (s *Service) ListAvailableDestinations(ctx context.Context, actorID uint64, limit, offset int) ([]registrystore.Destination, int, error) {
	repository, err := s.managementRepository()
	if err != nil {
		return nil, 0, fmt.Errorf("list available registry destinations: %w", err)
	}
	return repository.ListAvailableDestinations(ctx, actorID, limit, offset)
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
	if input.ConnectionRef == "" || utf8.RuneCountInString(input.ConnectionRef) > 128 || input.DisplayName == "" || utf8.RuneCountInString(input.DisplayName) > 128 || input.Provider != registryProviderGenericOCI || utf8.RuneCountInString(input.Description) > 500 {
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
	if input.RepositoryRef == "" || utf8.RuneCountInString(input.RepositoryRef) > 255 || input.DisplayName == "" || utf8.RuneCountInString(input.DisplayName) > 128 || !repositoryReferencePattern.MatchString(input.RepositoryRef) {
		return registrystore.RepositoryInput{}, errors.New("invalid artifact repository")
	}
	return input, nil
}
