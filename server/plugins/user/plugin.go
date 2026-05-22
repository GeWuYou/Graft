// Package user 提供接入 MVP 运行时的首个示例业务插件。
package user

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/i18n"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	usercontract "graft/server/plugins/user/contract"
	userstore "graft/server/plugins/user/store"
)

// Plugin 是用于验证扩展路径的示例用户能力插件。
//
// 该插件展示业务能力如何在 Register 阶段声明边界，在 Boot/Shutdown 阶段保持显式生命周期。
type Plugin struct {
	defaultAdminAuth *authService
	routeAuthorizer  *deferredAuthorizer
	bootstrapAccess  *deferredRBACAccessService
	userRepo         userstore.UserRepository
	authRepo         userstore.AuthRepository
}

type userListResponse struct {
	Items []userListItem `json:"items"`
}

type userListItem struct {
	ID        uint64 `json:"id"`
	Username  string `json:"username"`
	Display   string `json:"display"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

var (
	errCannotDisableOwnUser = errors.New("cannot disable own user")
	errCannotDeleteOwnUser  = errors.New("cannot delete own user")
	errInvalidUserStatus    = errors.New("invalid user status")
	errInvalidUserPayload   = errors.New("invalid user payload")
)

// NewPlugin 创建示例用户插件。
func NewPlugin(userRepo userstore.UserRepository, authRepo userstore.AuthRepository) *Plugin {
	return &Plugin{
		userRepo: userRepo,
		authRepo: authRepo,
	}
}

// Name 返回插件的稳定标识。
func (p *Plugin) Name() string {
	return pluginID
}

// Version 返回当前示例插件版本。
func (p *Plugin) Version() string {
	return pluginVersion
}

// DependsOn 返回当前插件的依赖列表。
func (p *Plugin) DependsOn() []string {
	return nil
}

// Register 声明用户插件需要的权限、菜单、路由和公开服务。
//
// 约束：
//   - 只注册跨插件可见的稳定接口，不暴露具体仓储或 ORM 实现。
//   - 只做声明式装配，不启动后台 goroutine 或持久占用额外资源。
//
// 失败语义：
//   - 任一注册步骤失败都会中止插件装配，并由上层运行时负责整体回滚。
func (p *Plugin) Register(ctx *plugin.Context) error {
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	registerUserPermissions(ctx.PermissionRegistry, p.Name())
	registerUserMenu(ctx.MenuRegistry, p.Name())

	services, err := p.registerServices(ctx)
	if err != nil {
		return err
	}
	p.routeAuthorizer = newDeferredAuthorizer()
	guards := newRouteGuards(ctx.I18n, services.auth, p.routeAuthorizer)
	if err := registerAuthRoutes(ctx, p.Name(), services.auth, services.bootstrap, &guards); err != nil {
		return err
	}
	if err := registerUserRoutes(ctx, p.Name(), services.user, services.auth, guards); err != nil {
		return err
	}

	return nil
}

func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, registration := range []i18n.Registration{
		{
			Namespace: "user",
			Locale:    i18n.LocaleZHCN,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(usercontract.UserListMenuTitle.String()), Text: "用户管理"},
			},
		},
		{
			Namespace: "user",
			Locale:    i18n.LocaleENUS,
			Messages: []i18n.MessageResource{
				{Key: i18n.MessageKey(usercontract.UserListMenuTitle.String()), Text: "User Management"},
			},
		},
	} {
		if err := localizer.RegisterMessages(registration); err != nil {
			return fmt.Errorf("register user plugin messages: %w", err)
		}
	}

	return nil
}

// Boot 在注册完成后启动用户插件的运行时行为。
//
// 当前阶段只在这里执行默认管理员引导初始化，确保 Register 保持纯声明式装配。
func (p *Plugin) Boot(ctx *plugin.Context) error {
	if err := p.bindRouteAuthorizer(ctx); err != nil {
		return err
	}
	if err := p.bindBootstrapAccess(ctx); err != nil {
		return err
	}
	if p.defaultAdminAuth == nil {
		return errors.New("default admin bootstrap service is unavailable")
	}

	rbacBootstrap, err := resolveService[pluginapi.RBACBootstrapService](ctx, (*pluginapi.RBACBootstrapService)(nil), "rbac bootstrap service")
	if err != nil {
		return err
	}

	if err := p.defaultAdminAuth.ensureDefaultAdmin(ctx.LifecycleContext, rbacBootstrap, ctx.PermissionRegistry.Items()); err != nil {
		return err
	}

	return nil
}

// Shutdown 在应用停止时释放用户插件资源。
//
// 当前实现没有自主管理的外部资源，因此关闭阶段保持幂等空操作。
func (p *Plugin) Shutdown(_ *plugin.Context) error {
	return nil
}

func (p *Plugin) bindRouteAuthorizer(ctx *plugin.Context) error {
	if p.routeAuthorizer == nil {
		return errors.New("route authorizer is unavailable")
	}

	authorizer, err := resolveService[pluginapi.Authorizer](ctx, (*pluginapi.Authorizer)(nil), "route authorizer")
	if err != nil {
		return err
	}

	if err := p.routeAuthorizer.SetTarget(authorizer); err != nil {
		return fmt.Errorf("bind route authorizer: %w", err)
	}

	return nil
}

func (p *Plugin) bindBootstrapAccess(ctx *plugin.Context) error {
	if p.bootstrapAccess == nil {
		return errors.New("bootstrap access service is unavailable")
	}

	accessService, err := resolveService[pluginapi.RBACAccessService](ctx, (*pluginapi.RBACAccessService)(nil), "rbac access service")
	if err != nil {
		return err
	}

	if err := p.bootstrapAccess.SetTarget(accessService); err != nil {
		return fmt.Errorf("bind bootstrap access service: %w", err)
	}

	return nil
}

func resolveService[T any](ctx *plugin.Context, key any, label string) (T, error) {
	var zero T

	resolved, err := ctx.Services.Resolve(key)
	if err != nil {
		return zero, fmt.Errorf("resolve %s: %w", label, err)
	}

	service, ok := resolved.(T)
	if !ok {
		return zero, fmt.Errorf("resolve %s: unexpected type %T", label, resolved)
	}

	return service, nil
}

// userService 把用户插件内部仓储读取收敛为跨插件稳定用户摘要服务。
type userService struct {
	users userstore.UserRepository
}

// authService 是 `pluginapi.AuthService` 在用户插件内的最小实现。
//
// 它把 access token 解析、refresh session 状态校验、当前用户读取和会话治理
// 保持在同一插件边界内，避免把生命周期敏感的鉴权协作拆散到 core 或其他插件。
type authService struct {
	auth            userstore.AuthRepository           // auth 负责 refresh session 持久化与轮换状态读取。
	passwordChanges userstore.PasswordChangeRepository // passwordChanges 负责原子改密与会话撤销写路径。
	users           userstore.UserRepository           // users 提供当前主体与登录路径所需的稳定用户读取能力。
	passwords       passwordHasher                     // passwords 统一封装口令散列与校验策略。
	policy          passwordPolicy                     // policy 固定收敛当前 MVP 的默认管理员与改密规则。
	tokens          *accessTokenManager                // tokens 负责 access token 的签发与解析。
	refreshTokens   *refreshTokenManager               // refreshTokens 负责 refresh token 的签发与解析。
	cookies         authCookieManager                  // cookies 收敛 refresh cookie 的读写与清理约束。
}

const maxSessionListLimit = 100

// GetUserByID 通过稳定仓储契约读取用户，并收敛为跨插件 DTO。
func (s userService) GetUserByID(ctx context.Context, id uint64) (pluginapi.UserSummary, error) {
	record, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			return pluginapi.UserSummary{}, pluginapi.ErrUserNotFound
		}
		return pluginapi.UserSummary{}, err
	}

	return pluginapi.UserSummary{
		ID:       record.ID,
		Username: record.Username,
		Display:  record.Display,
	}, nil
}

// ListUsers 读取用户列表，供当前插件路由在不暴露 store factory 的前提下复用。
func (s userService) ListUsers(ctx context.Context) ([]userstore.User, error) {
	if s.users == nil {
		return nil, errors.New("user repository is unavailable")
	}

	return s.users.List(ctx)
}

func (s userService) CreateUser(
	ctx context.Context,
	passwords passwordHasher,
	policy passwordPolicy,
	input userstore.CreateUserInput,
) (userstore.User, error) {
	if s.users == nil {
		return userstore.User{}, errors.New("user repository is unavailable")
	}
	if strings.TrimSpace(input.Username) == "" {
		return userstore.User{}, errInvalidUserPayload
	}
	if strings.TrimSpace(input.Display) == "" {
		return userstore.User{}, errInvalidUserPayload
	}
	if err := policy.ValidateNewPassword(input.PasswordHash); err != nil {
		return userstore.User{}, err
	}

	hash, err := passwords.Hash(input.PasswordHash)
	if err != nil {
		return userstore.User{}, err
	}
	input.PasswordHash = hash
	input.Status = normalizeManagedUserStatus(input.Status)
	input.MustChangePassword = true

	return s.users.Create(ctx, input)
}

func (s userService) UpdateUser(ctx context.Context, input userstore.UpdateUserInput) (userstore.User, error) {
	if s.users == nil {
		return userstore.User{}, errors.New("user repository is unavailable")
	}
	if strings.TrimSpace(input.Username) == "" || strings.TrimSpace(input.Display) == "" {
		return userstore.User{}, errInvalidUserPayload
	}

	return s.users.Update(ctx, input)
}

func (s userService) SetUserStatus(
	ctx context.Context,
	authRepo userstore.AuthRepository,
	input userstore.SetUserStatusInput,
) (userstore.User, error) {
	if s.users == nil {
		return userstore.User{}, errors.New("user repository is unavailable")
	}
	if authRepo == nil {
		return userstore.User{}, errors.New("auth repository is unavailable")
	}

	status := normalizeExplicitManagedUserStatus(input.Status)
	if status == "" {
		return userstore.User{}, errInvalidUserStatus
	}
	if status == usercontract.UserStatusDisabled && requestActorOwnsUser(ctx, input.ID) {
		return userstore.User{}, errCannotDisableOwnUser
	}
	input.Status = status

	updated, err := s.users.SetStatus(ctx, input)
	if err != nil {
		return userstore.User{}, err
	}
	if status == usercontract.UserStatusDisabled {
		if err := authRepo.RevokeRefreshSessionsByUserID(ctx, userstore.RevokeRefreshSessionsByUserIDInput{
			UserID:    input.ID,
			RevokedAt: time.Now().UTC(),
		}); err != nil {
			return userstore.User{}, err
		}
	}

	return updated, nil
}

func (s userService) DeleteUser(ctx context.Context, authRepo userstore.AuthRepository, userID uint64) error {
	if s.users == nil {
		return errors.New("user repository is unavailable")
	}
	if authRepo == nil {
		return errors.New("auth repository is unavailable")
	}
	if requestActorOwnsUser(ctx, userID) {
		return errCannotDeleteOwnUser
	}

	if err := s.users.Delete(ctx, userstore.DeleteUserInput{
		ID:        userID,
		DeletedAt: time.Now().UTC(),
		ActorID:   requestActorID(ctx),
	}); err != nil {
		return err
	}

	return authRepo.RevokeRefreshSessionsByUserID(ctx, userstore.RevokeRefreshSessionsByUserIDInput{
		UserID:    userID,
		RevokedAt: time.Now().UTC(),
	})
}

func (s userService) ResetUserPassword(
	ctx context.Context,
	authRepo userstore.AuthRepository,
	passwords passwordHasher,
	policy passwordPolicy,
	userID uint64,
	newPassword string,
) error {
	if authRepo == nil {
		return errors.New("auth repository is unavailable")
	}
	if err := policy.ValidateNewPassword(newPassword); err != nil {
		return err
	}

	hash, err := passwords.Hash(newPassword)
	if err != nil {
		return err
	}

	return authRepo.ResetPasswordAndRevokeRefreshSessions(ctx, userstore.ResetPasswordAndRevokeSessionsInput{
		UserID:             userID,
		PasswordHash:       hash,
		MustChangePassword: true,
		ChangedAt:          time.Now().UTC(),
	})
}

func normalizeManagedUserStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "", usercontract.UserStatusEnabled:
		return usercontract.UserStatusEnabled
	case usercontract.UserStatusDisabled:
		return usercontract.UserStatusDisabled
	default:
		return ""
	}
}

func normalizeExplicitManagedUserStatus(status string) string {
	switch strings.TrimSpace(status) {
	case usercontract.UserStatusEnabled:
		return usercontract.UserStatusEnabled
	case usercontract.UserStatusDisabled:
		return usercontract.UserStatusDisabled
	default:
		return ""
	}
}

func requestActorOwnsUser(ctx context.Context, userID uint64) bool {
	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	return ok && requestAuth.User != nil && requestAuth.User.ID == userID
}

func requestActorID(ctx context.Context) uint64 {
	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil {
		return 0
	}

	return requestAuth.User.ID
}

// CurrentUser 根据请求上下文中已解析的访问令牌声明返回当前主体摘要。
//
// 该实现要求调用链先通过鉴权中间件写入稳定 claims，再按用户仓储读取跨
// 插件可见的最小用户资料，不把 token 解析细节泄漏给业务调用方。
func (s authService) CurrentUser(ctx context.Context) (*pluginapi.CurrentUser, error) {
	if s.users == nil {
		return nil, errors.New("user repository is unavailable")
	}

	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.Claims == nil {
		return nil, pluginapi.ErrUnauthenticated
	}

	record, err := s.users.GetByID(ctx, requestAuth.Claims.UserID)
	if err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			return nil, pluginapi.ErrUnauthenticated
		}
		return nil, err
	}

	return &pluginapi.CurrentUser{
		ID:          record.ID,
		Username:    record.Username,
		DisplayName: record.Display,
	}, nil
}

// ParseAccessToken 校验 access token 并返回跨插件稳定 claims。
func (s authService) ParseAccessToken(ctx context.Context, token string) (*pluginapi.AccessTokenClaims, error) {
	if s.tokens == nil {
		return nil, errors.New("access token manager is unavailable")
	}

	claims, err := s.tokens.Parse(strings.TrimSpace(token))
	if err != nil {
		switch {
		case errors.Is(err, errExpiredAccessToken):
			return nil, pluginapi.ErrExpiredAccessToken
		case errors.Is(err, errInvalidAccessToken):
			return nil, pluginapi.ErrInvalidAccessToken
		default:
			return nil, err
		}
	}

	if err := s.validateAccessSession(ctx, claims); err != nil {
		if errors.Is(err, errAccessSessionFailed) {
			return nil, pluginapi.ErrInvalidAccessToken
		}
		return nil, err
	}

	return claims, nil
}

var _ pluginapi.AuthService = authService{}

// parseUserID 将路由参数转换为插件内部统一使用的正整数 ID。
func parseUserID(input string) (uint64, error) {
	id, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse user id %q: %w", input, err)
	}
	if id == 0 {
		return 0, errors.New("id must be greater than zero")
	}
	return id, nil
}

// parseSessionListOptions 将列表查询参数收敛为插件内最小会话列表约束。
//
// 当前只允许显式 limit，并把约束留在插件层，避免为了轻量分页提前扩展仓储
// 或跨插件契约。
func parseSessionListOptions(rawLimit string) (sessionListOptions, error) {
	rawLimit = strings.TrimSpace(rawLimit)
	if rawLimit == "" {
		return sessionListOptions{}, nil
	}

	limit, err := strconv.Atoi(rawLimit)
	if err != nil {
		return sessionListOptions{}, fmt.Errorf("parse session limit %q: %w", rawLimit, err)
	}
	if limit <= 0 || limit > maxSessionListLimit {
		return sessionListOptions{}, fmt.Errorf("session limit %d is out of range", limit)
	}

	return sessionListOptions{Limit: limit}, nil
}

// mapAuthError 把插件内部鉴权/会话错误收敛为稳定 HTTP 状态与消息键。
func mapAuthError(err error) (int, messagecontract.Key) {
	for _, mapping := range []struct {
		match  error
		status int
		key    messagecontract.Key
	}{
		{match: pluginapi.ErrUnauthenticated, status: http.StatusUnauthorized, key: messagecontract.AuthTokenMissing},
		{match: errInvalidLoginCredentials, status: http.StatusBadRequest, key: messagecontract.AuthInvalidCredentials},
		{match: errRefreshTokenRequired, status: http.StatusUnauthorized, key: messagecontract.AuthTokenMissing},
		{match: errExpiredRefreshToken, status: http.StatusUnauthorized, key: messagecontract.AuthTokenExpired},
		{match: errInvalidRefreshToken, status: http.StatusUnauthorized, key: messagecontract.AuthTokenInvalid},
		{match: errSessionNotFound, status: http.StatusNotFound, key: messagecontract.AuthSessionNotFound},
		{match: errRequiredPasswordChangeOnly, status: http.StatusForbidden, key: messagecontract.AuthForbidden},
		{match: errCurrentPasswordRequired, status: http.StatusBadRequest, key: messagecontract.CommonInvalidArgument},
		{match: errPasswordPolicyViolation, status: http.StatusBadRequest, key: messagecontract.AuthPasswordPolicyViolation},
		{match: errPasswordReuseForbidden, status: http.StatusBadRequest, key: messagecontract.AuthPasswordReuseForbidden},
		{match: errCurrentPasswordInvalid, status: http.StatusBadRequest, key: messagecontract.AuthCurrentPasswordInvalid},
		{match: errRefreshSessionFailed, status: http.StatusUnauthorized, key: messagecontract.AuthTokenInvalid},
	} {
		if errors.Is(err, mapping.match) {
			return mapping.status, mapping.key
		}
	}

	return http.StatusInternalServerError, messagecontract.CommonInternalError
}

func authErrorDetails(err error) map[string]any {
	if errors.Is(err, errCurrentPasswordRequired) {
		return map[string]any{"field": "current_password"}
	}

	return nil
}
