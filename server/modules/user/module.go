// Package user 提供接入 MVP 运行时的首个示例业务模块。
package user

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	authcontract "graft/server/modules/auth/contract"
	usercontract "graft/server/modules/user/contract"
	userstore "graft/server/modules/user/store"
)

// Module 是用于验证扩展路径的示例用户能力模块。
//
// 该模块展示业务能力如何在 Register 阶段声明边界，在 Boot/Shutdown 阶段保持显式生命周期。
type Module struct {
	defaultAdminAuth *authService
	routeAuthorizer  *deferredAuthorizer
	bootstrapAccess  *deferredRBACAccessService
	userRepo         userstore.UserRepository
	authRepo         userstore.AuthRepository
}

var (
	errCannotDisableOwnUser = errors.New("cannot disable own user")
	errCannotDeleteOwnUser  = errors.New("cannot delete own user")
	errInvalidUserStatus    = errors.New("invalid user status")
	errInvalidUserPayload   = errors.New("invalid user payload")
)

// NewModule 创建示例用户模块。
func NewModule(userRepo userstore.UserRepository, authRepo userstore.AuthRepository) *Module {
	return &Module{
		userRepo: userRepo,
		authRepo: authRepo,
	}
}

// Register 声明用户模块需要的权限、菜单、路由和公开服务。
func (p *Module) Register(ctx *module.Context) error {
	if err := registerMessages(ctx.I18n); err != nil {
		return err
	}
	registerUserPermissions(ctx.PermissionRegistry, moduleID)
	registerUserMenu(ctx.MenuRegistry, moduleID)

	services, err := p.registerServices(ctx)
	if err != nil {
		return err
	}
	p.routeAuthorizer = newDeferredAuthorizer()
	guards := newRouteGuards(
		ctx.I18n,
		services.auth,
		p.routeAuthorizer,
		httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID),
	)
	authGroup := ctx.Router.Group(authcontract.AuthGroup)
	guards.restrictedSession = newRestrictedSessionGuard(
		ctx.I18n,
		services.auth,
		authGroup.BasePath(),
	)
	if err := registerUserRoutes(ctx, moduleID, services.user, services.auth, guards); err != nil {
		return err
	}

	return nil
}

// Boot 在注册完成后启动用户模块的运行时行为。
//
// 当前阶段只在这里执行默认管理员引导初始化，确保 Register 保持纯声明式装配。
func (p *Module) Boot(ctx *module.Context) error {
	if err := p.bindRouteAuthorizer(ctx); err != nil {
		return err
	}
	if err := p.bindBootstrapAccess(ctx); err != nil {
		return err
	}
	if p.defaultAdminAuth == nil {
		return errors.New("default admin bootstrap service is unavailable")
	}

	rbacBootstrap, err := resolveService[moduleapi.RBACBootstrapService](ctx, (*moduleapi.RBACBootstrapService)(nil), "rbac bootstrap service")
	if err != nil {
		return err
	}

	if err := p.defaultAdminAuth.ensureDefaultAdmin(ctx.LifecycleContext, rbacBootstrap, ctx.PermissionRegistry.Items()); err != nil {
		return err
	}

	return nil
}

// Shutdown 在应用停止时释放用户模块资源。
//
// 当前实现没有自主管理的外部资源，因此关闭阶段保持幂等空操作。
func (p *Module) Shutdown(_ *module.Context) error {
	return nil
}

func (p *Module) bindRouteAuthorizer(ctx *module.Context) error {
	if p.routeAuthorizer == nil {
		return errors.New("route authorizer is unavailable")
	}

	authorizer, err := resolveService[moduleapi.Authorizer](ctx, (*moduleapi.Authorizer)(nil), "route authorizer")
	if err != nil {
		return err
	}

	if err := p.routeAuthorizer.SetTarget(authorizer); err != nil {
		return fmt.Errorf("bind route authorizer: %w", err)
	}

	return nil
}

func (p *Module) bindBootstrapAccess(ctx *module.Context) error {
	if p.bootstrapAccess == nil {
		return errors.New("bootstrap access service is unavailable")
	}

	accessService, err := resolveService[moduleapi.RBACAccessService](ctx, (*moduleapi.RBACAccessService)(nil), "rbac access service")
	if err != nil {
		return err
	}

	if err := p.bootstrapAccess.SetTarget(accessService); err != nil {
		return fmt.Errorf("bind bootstrap access service: %w", err)
	}

	return nil
}

func resolveService[T any](ctx *module.Context, key any, label string) (T, error) {
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

// userService 把用户模块内部仓储读取收敛为跨模块稳定用户摘要服务。
type userService struct {
	users    userstore.UserRepository
	rbac     moduleapi.RBACAccessService
	auditBus eventbus.Bus
	logger   *zap.Logger
}

// authService 是 `moduleapi.AuthService` 在用户模块内的最小实现。
//
// 它把 access token 解析、refresh session 状态校验、当前用户读取和会话治理
// 保持在同一模块边界内，避免把生命周期敏感的鉴权协作拆散到 core 或其他模块。
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

// GetUserByID 通过稳定仓储契约读取用户，并收敛为跨模块 DTO。
func (s userService) GetUserByID(ctx context.Context, id uint64) (moduleapi.UserSummary, error) {
	record, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			return moduleapi.UserSummary{}, moduleapi.ErrUserNotFound
		}
		return moduleapi.UserSummary{}, err
	}

	return moduleapi.UserSummary{
		ID:       record.ID,
		Username: record.Username,
		Display:  record.Display,
	}, nil
}

// GetUser keeps route handlers on the public service boundary while preserving
// the full managed-user record needed for HTTP response mapping.
func (s userService) GetUser(ctx context.Context, id uint64) (userstore.User, error) {
	if s.users == nil {
		return userstore.User{}, errors.New("user repository is unavailable")
	}

	return s.users.GetByID(ctx, id)
}

// ListUsers 读取用户列表，供当前模块路由在不暴露 store factory 的前提下复用。
func (s userService) ListUsers(ctx context.Context) ([]userstore.User, error) {
	if s.users == nil {
		return nil, errors.New("user repository is unavailable")
	}

	return s.users.List(ctx)
}

func (s userService) ListUserRoleSummaries(ctx context.Context, userIDs []uint64) (map[uint64][]moduleapi.RoleSummary, error) {
	if s.rbac == nil {
		return nil, errors.New("rbac access service is unavailable")
	}

	summaries, err := s.rbac.ListRoleSummariesByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("list user role summaries: %w", err)
	}
	return summaries, nil
}

func (s userService) CreateUser(
	ctx context.Context,
	passwords passwordHasher,
	policy passwordPolicy,
	command CreateUserCommand,
) (userstore.User, error) {
	if s.users == nil {
		return userstore.User{}, errors.New("user repository is unavailable")
	}
	username := strings.TrimSpace(command.Username)
	display := strings.TrimSpace(command.Display)
	if username == "" {
		return userstore.User{}, errInvalidUserPayload
	}
	if display == "" {
		return userstore.User{}, errInvalidUserPayload
	}
	if err := policy.ValidateNewPassword(command.Password); err != nil {
		return userstore.User{}, err
	}

	hash, err := passwords.Hash(command.Password)
	if err != nil {
		return userstore.User{}, err
	}
	input := userstore.CreateUserInput{
		Username:           username,
		Display:            display,
		Status:             normalizeManagedUserStatus(""),
		PasswordHash:       hash,
		MustChangePassword: true,
		ActorID:            command.ActorID,
	}

	created, err := s.users.Create(ctx, input)
	if err != nil {
		return userstore.User{}, err
	}

	s.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "user.create",
		ResourceType: "user",
		ResourceID:   formatUserAuditID(created.ID),
		ResourceName: created.Username,
		Success:      true,
		Message:      "user created",
		Metadata: map[string]any{
			"username":             created.Username,
			"display_name":         created.Display,
			"status":               created.Status,
			"must_change_password": true,
		},
	})

	return created, nil
}

func (s userService) UpdateUser(ctx context.Context, command UpdateUserCommand) (userstore.User, error) {
	if s.users == nil {
		return userstore.User{}, errors.New("user repository is unavailable")
	}
	username := strings.TrimSpace(command.Username)
	display := strings.TrimSpace(command.Display)
	if username == "" || display == "" {
		return userstore.User{}, errInvalidUserPayload
	}

	updated, err := s.users.Update(ctx, userstore.UpdateUserInput{
		ID:       command.ID,
		Username: username,
		Display:  display,
		ActorID:  command.ActorID,
	})
	if err != nil {
		return userstore.User{}, err
	}

	s.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "user.update",
		ResourceType: "user",
		ResourceID:   formatUserAuditID(updated.ID),
		ResourceName: updated.Username,
		Success:      true,
		Message:      "user updated",
		Metadata: map[string]any{
			"username":     updated.Username,
			"display_name": updated.Display,
			"status":       updated.Status,
		},
	})

	return updated, nil
}

func (s userService) SetUserStatus(
	ctx context.Context,
	authRepo userstore.AuthRepository,
	command UpdateUserStatusCommand,
) (userstore.User, error) {
	if s.users == nil {
		return userstore.User{}, errors.New("user repository is unavailable")
	}
	if authRepo == nil {
		return userstore.User{}, errors.New("auth repository is unavailable")
	}

	status := normalizeExplicitManagedUserStatus(command.Status)
	if status == "" {
		return userstore.User{}, errInvalidUserStatus
	}
	if status == usercontract.UserStatusDisabled && requestActorOwnsUser(ctx, command.ID) {
		return userstore.User{}, errCannotDisableOwnUser
	}

	input := userstore.SetUserStatusInput{
		ID:      command.ID,
		Status:  status,
		ActorID: command.ActorID,
	}

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

	s.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "user.status.update",
		ResourceType: "user",
		ResourceID:   formatUserAuditID(updated.ID),
		ResourceName: updated.Username,
		Success:      true,
		Message:      "user status updated",
		Metadata: map[string]any{
			"username": updated.Username,
			"status":   updated.Status,
		},
	})

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

	if err := authRepo.RevokeRefreshSessionsByUserID(ctx, userstore.RevokeRefreshSessionsByUserIDInput{
		UserID:    userID,
		RevokedAt: time.Now().UTC(),
	}); err != nil {
		return err
	}

	s.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "user.delete",
		ResourceType: "user",
		ResourceID:   formatUserAuditID(userID),
		Success:      true,
		Message:      "user deleted",
	})

	return nil
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

	if err := authRepo.ResetPasswordAndRevokeRefreshSessions(ctx, userstore.ResetPasswordAndRevokeSessionsInput{
		UserID:             userID,
		PasswordHash:       hash,
		MustChangePassword: true,
		ChangedAt:          time.Now().UTC(),
	}); err != nil {
		return err
	}

	s.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "user.password.reset",
		ResourceType: "user",
		ResourceID:   formatUserAuditID(userID),
		Success:      true,
		Message:      "user password reset",
		Metadata: map[string]any{
			"must_change_password": true,
		},
	})

	return nil
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
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	return ok && requestAuth.User != nil && requestAuth.User.ID == userID
}

func requestActorID(ctx context.Context) uint64 {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil {
		return 0
	}

	return requestAuth.User.ID
}

func (s userService) publishAudit(ctx context.Context, event moduleapi.AuditEvent) {
	if s.auditBus == nil {
		return
	}

	event.Operator = currentAuditOperator(ctx)
	if err := s.auditBus.Publish(ctx, eventbus.Event{
		Name:    moduleapi.AuditRecordEventName,
		Source:  moduleID,
		Payload: event,
	}); err != nil && s.logger != nil {
		s.logger.Warn("publish user audit event failed",
			zap.String("module", moduleID),
			zap.String("action", strings.TrimSpace(event.Action)),
			zap.Error(err),
		)
	}
}

func currentAuditOperator(ctx context.Context) *moduleapi.CurrentUser {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil {
		return nil
	}

	user := *requestAuth.User
	return &user
}

func formatUserAuditID(id uint64) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatUint(id, 10)
}

// CurrentUser 根据请求上下文中已解析的访问令牌声明返回当前主体摘要。
//
// 该实现要求调用链先通过鉴权中间件写入稳定 claims，再按用户仓储读取跨
// 模块可见的最小用户资料，不把 token 解析细节泄漏给业务调用方。
func (s authService) CurrentUser(ctx context.Context) (*moduleapi.CurrentUser, error) {
	if s.users == nil {
		return nil, errors.New("user repository is unavailable")
	}

	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.Claims == nil {
		return nil, moduleapi.ErrUnauthenticated
	}

	record, err := s.users.GetByID(ctx, requestAuth.Claims.UserID)
	if err != nil {
		if errors.Is(err, userstore.ErrUserNotFound) {
			return nil, moduleapi.ErrUnauthenticated
		}
		return nil, err
	}

	return &moduleapi.CurrentUser{
		ID:          record.ID,
		Username:    record.Username,
		DisplayName: record.Display,
	}, nil
}

// ParseAccessToken 校验 access token 并返回跨模块稳定 claims。
func (s authService) ParseAccessToken(ctx context.Context, token string) (*moduleapi.AccessTokenClaims, error) {
	if s.tokens == nil {
		return nil, errors.New("access token manager is unavailable")
	}

	claims, err := s.tokens.Parse(strings.TrimSpace(token))
	if err != nil {
		switch {
		case errors.Is(err, errExpiredAccessToken):
			return nil, moduleapi.ErrExpiredAccessToken
		case errors.Is(err, errInvalidAccessToken):
			return nil, moduleapi.ErrInvalidAccessToken
		default:
			return nil, err
		}
	}

	if err := s.validateAccessSession(ctx, claims); err != nil {
		if errors.Is(err, errAccessSessionFailed) {
			return nil, moduleapi.ErrInvalidAccessToken
		}
		return nil, err
	}

	return claims, nil
}

var _ moduleapi.AuthService = authService{}

// parseUserID 将路由参数转换为模块内部统一使用的正整数 ID。
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

// parseSessionListOptions 将列表查询参数收敛为模块内最小会话列表约束。
//
// 当前只允许显式 limit，并把约束留在插件层，避免为了轻量分页提前扩展仓储
// 或跨模块契约。
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

// mapAuthError 把模块内部鉴权/会话错误收敛为稳定 HTTP 状态与消息键。
func mapAuthError(err error) (int, messagecontract.Key) {
	for _, mapping := range []struct {
		match  error
		status int
		key    messagecontract.Key
	}{
		{match: moduleapi.ErrUnauthenticated, status: http.StatusUnauthorized, key: messagecontract.AuthTokenMissing},
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
