// Package user 提供接入 MVP 运行时的首个示例业务模块。
package user

import (
	"context"
	"database/sql"
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
	usercontract "graft/server/modules/user/contract"
	userstore "graft/server/modules/user/store"
)

// Module 是用于验证扩展路径的示例用户能力模块。
//
// 该模块展示业务能力如何在 Register 阶段声明边界，在 Boot/Shutdown 阶段保持显式生命周期。
type Module struct {
	routeAuthorizer  *deferredAuthorizer
	bootstrapAccess  *deferredRBACAccessService
	userRepo         userstore.UserRepository
	userCredentials  *deferredCredentialManagementService
	authTransactions *deferredAuthTransactionFactory
	authCapabilities *deferredAuthCapabilities
}

var (
	errCannotDisableOwnUser           = errors.New("cannot disable own user")
	errCannotDeleteOwnUser            = errors.New("cannot delete own user")
	errInvalidUserStatus              = errors.New("invalid user status")
	errInvalidUserPayload             = errors.New("invalid user payload")
	errProtectedDefaultAdminImmutable = errors.New("protected default admin is immutable for this operation")
)

// NewModule 创建示例用户模块。
func NewModule(userRepo userstore.UserRepository) *Module {
	return &Module{userRepo: userRepo}
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
		services,
		p.routeAuthorizer,
		httpx.NewSecurityAuditPublisher(ctx.EventBus, ctx.Logger, moduleID),
	)
	if err := registerUserRoutes(ctx, moduleID, services.user, services.authSessions, guards); err != nil {
		return err
	}

	return nil
}

// Boot 在注册完成后启动用户模块的运行时行为。
//
// 当前阶段只绑定在 Register 阶段尚不可用的 auth 与 RBAC capability。
func (p *Module) Boot(ctx *module.Context) error {
	if err := p.bindRouteAuthorizer(ctx); err != nil {
		return err
	}
	if err := p.bindBootstrapAccess(ctx); err != nil {
		return err
	}
	if err := p.bindCredentialManagement(ctx); err != nil {
		return err
	}
	if err := p.bindAuthTransactions(ctx); err != nil {
		return err
	}
	if err := p.bindAuthCapabilities(ctx); err != nil {
		return err
	}
	return nil
}

func (p *Module) bindAuthTransactions(ctx *module.Context) error {
	if p.authTransactions == nil {
		return errors.New("auth transaction adapter proxy is unavailable")
	}
	factory, err := resolveService[moduleapi.AuthTransactionAdapterFactory](ctx, (*moduleapi.AuthTransactionAdapterFactory)(nil), "auth transaction adapter factory")
	if err != nil {
		return err
	}
	return p.authTransactions.SetTarget(factory)
}

func (p *Module) bindAuthCapabilities(ctx *module.Context) error {
	if p.authCapabilities == nil {
		return errors.New("auth capability proxy is unavailable")
	}
	authService, err := resolveService[moduleapi.AuthService](ctx, (*moduleapi.AuthService)(nil), "auth service")
	if err != nil {
		return err
	}
	sessions, err := resolveService[moduleapi.AuthSessionService](ctx, (*moduleapi.AuthSessionService)(nil), "auth session service")
	if err != nil {
		return err
	}
	flow, err := resolveService[moduleapi.AuthFlowService](ctx, (*moduleapi.AuthFlowService)(nil), "auth flow service")
	if err != nil {
		return err
	}
	return p.authCapabilities.SetTargets(authService, sessions, flow)
}

func (p *Module) bindCredentialManagement(ctx *module.Context) error {
	if p.userServiceCredentials() == nil {
		return errors.New("auth credential management proxy is unavailable")
	}
	credentials, err := resolveService[moduleapi.AuthCredentialManagementService](ctx, (*moduleapi.AuthCredentialManagementService)(nil), "auth credential management service")
	if err != nil {
		return err
	}
	return p.userServiceCredentials().SetTarget(credentials)
}

func (p *Module) userServiceCredentials() *deferredCredentialManagementService {
	if p.userCredentials == nil {
		return nil
	}
	return p.userCredentials
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
	users        userstore.UserRepository
	rbac         moduleapi.RBACAccessService
	auditBus     eventbus.Bus
	logger       *zap.Logger
	credentials  moduleapi.AuthCredentialManagementService
	transactions userstore.TransactionRunner
	composites   userstore.CompositeTransactionRunner
	authTx       moduleapi.AuthTransactionAdapterFactory
}

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
		ID:                    record.ID,
		Username:              record.Username,
		Display:               record.Display,
		ProtectedDefaultAdmin: record.ProtectedDefaultAdmin,
	}, nil
}

func (s userService) CountUsers(ctx context.Context) (int, error) {
	if s.users == nil {
		return 0, errors.New("user repository is unavailable")
	}

	return s.users.Count(ctx)
}

// ListSecuritySummaries 返回供授权聚合读取方使用的有界用户状态页，并按用户 ID 递增读取。
func (s userService) ListSecuritySummaries(ctx context.Context, afterID uint64, limit int) ([]moduleapi.UserSecuritySummary, error) {
	if s.users == nil {
		return nil, errors.New("user repository is unavailable")
	}

	users, err := s.users.ListSecuritySummaries(ctx, afterID, limit)
	if err != nil {
		return nil, err
	}
	summaries := make([]moduleapi.UserSecuritySummary, 0, len(users))
	for _, user := range users {
		summaries = append(summaries, moduleapi.UserSecuritySummary{ID: user.ID, Status: user.Status})
	}
	return summaries, nil
}

// GetUser 让路由处理器停留在公开服务边界，同时保留 HTTP 响应映射所需的完整受管用户记录。
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
	input := userstore.CreateUserInput{
		Username: username,
		Display:  display,
		Status:   normalizeManagedUserStatus(""),
		ActorID:  command.ActorID,
	}

	var created userstore.User
	if err := s.runCompositeTransaction(ctx, func(txCtx context.Context, profiles userstore.UserRepository, auth moduleapi.AuthTransactionAdapter) error {
		var err error
		created, err = profiles.Create(txCtx, input)
		if err != nil {
			return fmt.Errorf("create user profile: %w", err)
		}
		if err := auth.ProvisionPasswordCredential(txCtx, moduleapi.AuthCredentialProvisionInput{UserID: created.ID, Password: command.Password, MustChangePassword: true}); err != nil {
			return fmt.Errorf("provision user password credential: %w", err)
		}
		return nil
	}); err != nil {
		return userstore.User{}, err
	}

	s.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "user.create",
		ResourceType: "user",
		ResourceID:   formatUserAuditID(created.ID),
		ResourceName: created.Username,
		Success:      true,
		MessageKey:   "user.audit.userCreated",
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
	current, err := s.users.GetByID(ctx, command.ID)
	if err != nil {
		return userstore.User{}, err
	}
	username := strings.TrimSpace(command.Username)
	display := strings.TrimSpace(command.Display)
	if username == "" || display == "" {
		return userstore.User{}, errInvalidUserPayload
	}
	if current.ProtectedDefaultAdmin && username != current.Username {
		return userstore.User{}, errProtectedDefaultAdminImmutable
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
		MessageKey:   "user.audit.userUpdated",
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
	command UpdateUserStatusCommand,
) (userstore.User, error) {
	if s.users == nil {
		return userstore.User{}, errors.New("user repository is unavailable")
	}
	if s.authTx == nil {
		return userstore.User{}, errors.New("auth repository is unavailable")
	}

	status, err := s.validateSetUserStatusPreconditions(ctx, command)
	if err != nil {
		return userstore.User{}, err
	}

	input := userstore.SetUserStatusInput{
		ID:      command.ID,
		Status:  status,
		ActorID: command.ActorID,
	}

	var updated userstore.User
	if err := s.runCompositeTransaction(ctx, func(txCtx context.Context, profiles userstore.UserRepository, auth moduleapi.AuthTransactionAdapter) error {
		var err error
		updated, err = profiles.SetStatus(txCtx, input)
		if err != nil {
			return fmt.Errorf("set user profile status: %w", err)
		}
		if status == usercontract.UserStatusDisabled {
			if err := auth.RevokeSessions(txCtx, input.ID); err != nil {
				return fmt.Errorf("revoke user sessions after disabling profile: %w", err)
			}
		}
		return nil
	}); err != nil {
		return userstore.User{}, err
	}

	s.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "user.status.update",
		ResourceType: "user",
		ResourceID:   formatUserAuditID(updated.ID),
		ResourceName: updated.Username,
		Success:      true,
		MessageKey:   "user.audit.userStatusUpdated",
		Message:      "user status updated",
		Metadata: map[string]any{
			"username": updated.Username,
			"status":   updated.Status,
		},
	})

	return updated, nil
}

func (s userService) validateSetUserStatusPreconditions(
	ctx context.Context,
	command UpdateUserStatusCommand,
) (string, error) {
	status := normalizeExplicitManagedUserStatus(command.Status)
	if status == "" {
		return "", errInvalidUserStatus
	}
	if status == usercontract.UserStatusDisabled && requestActorOwnsUser(ctx, command.ID) {
		return "", errCannotDisableOwnUser
	}

	current, err := s.users.GetByID(ctx, command.ID)
	if err != nil {
		return "", err
	}
	if current.ProtectedDefaultAdmin && status != current.Status {
		return "", errProtectedDefaultAdminImmutable
	}

	return status, nil
}

func (s userService) DeleteUser(ctx context.Context, userID uint64) error {
	if s.users == nil {
		return errors.New("user repository is unavailable")
	}
	if s.authTx == nil {
		return errors.New("auth repository is unavailable")
	}
	if requestActorOwnsUser(ctx, userID) {
		return errCannotDeleteOwnUser
	}
	current, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if current.ProtectedDefaultAdmin {
		return errProtectedDefaultAdminImmutable
	}

	if err := s.runCompositeTransaction(ctx, func(txCtx context.Context, profiles userstore.UserRepository, auth moduleapi.AuthTransactionAdapter) error {
		if err := profiles.Delete(txCtx, userstore.DeleteUserInput{
			ID:        userID,
			DeletedAt: time.Now().UTC(),
			ActorID:   requestActorID(ctx),
		}); err != nil {
			return fmt.Errorf("delete user profile: %w", err)
		}
		if err := auth.RevokeSessions(txCtx, userID); err != nil {
			return fmt.Errorf("revoke user sessions after deleting profile: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	s.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "user.delete",
		ResourceType: "user",
		ResourceID:   formatUserAuditID(userID),
		Success:      true,
		MessageKey:   "user.audit.userDeleted",
		Message:      "user deleted",
	})

	return nil
}

func (s userService) runCompositeTransaction(ctx context.Context, callback func(context.Context, userstore.UserRepository, moduleapi.AuthTransactionAdapter) error) error {
	if s.composites == nil || s.authTx == nil {
		return errors.New("user/auth composite transaction is unavailable")
	}
	err := s.composites.RunInCompositeTransaction(ctx, func(txCtx context.Context, profiles userstore.UserRepository, tx *sql.Tx) error {
		auth, err := s.authTx.BindAuthTransaction(tx)
		if err != nil {
			return fmt.Errorf("bind auth transaction adapter: %w", err)
		}
		if err := callback(txCtx, profiles, auth); err != nil {
			return fmt.Errorf("run user/auth composite operation: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("run user/auth composite transaction: %w", err)
	}
	return nil
}

func (s userService) ResetUserPassword(
	ctx context.Context,
	userID uint64,
	newPassword string,
) error {
	if s.credentials == nil {
		return errors.New("auth repository is unavailable")
	}
	if err := s.credentials.ResetPassword(ctx, userID, newPassword); err != nil {
		return err
	}

	s.publishAudit(ctx, moduleapi.AuditEvent{
		Action:       "user.password.reset",
		ResourceType: "user",
		ResourceID:   formatUserAuditID(userID),
		Success:      true,
		MessageKey:   "user.audit.userPasswordReset",
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
		Name:    string(moduleapi.AuditRecordEventName),
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

// formatUserAuditID 将用户标识格式化为审计事件使用的字符串；标识为零时返回空字符串。
func formatUserAuditID(id uint64) string {
	if id == 0 {
		return ""
	}
	return strconv.FormatUint(id, 10)
}

// parseUserID 将路由参数解析为大于零的用户 ID；输入格式无效或 ID 为零时返回错误。
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
// 当前只允许显式 limit，并把约束留在模块层，避免为了轻量分页提前扩展仓储
// 空值返回默认选项；非空值必须为 1 至 maxUserSessionListLimit 范围内的整数。
func parseSessionListOptions(rawLimit string) (userSessionListOptions, error) {
	rawLimit = strings.TrimSpace(rawLimit)
	if rawLimit == "" {
		return userSessionListOptions{}, nil
	}

	limit, err := strconv.Atoi(rawLimit)
	if err != nil {
		return userSessionListOptions{}, fmt.Errorf("parse session limit %q: %w", rawLimit, err)
	}
	if limit <= 0 || limit > maxUserSessionListLimit {
		return userSessionListOptions{}, fmt.Errorf("session limit %d is out of range", limit)
	}

	return userSessionListOptions{Limit: limit}, nil
}

// mapAuthError 将鉴权错误映射为稳定的 HTTP 状态码和消息键。
// 未认证错误映射为 401 和缺少鉴权令牌的消息键，其余错误映射为 500 和通用内部错误消息键。
func mapAuthError(err error) (int, messagecontract.Key) {
	for _, mapping := range []struct {
		match  error
		status int
		key    messagecontract.Key
	}{
		{match: moduleapi.ErrUnauthenticated, status: http.StatusUnauthorized, key: messagecontract.AuthTokenMissing},
	} {
		if errors.Is(err, mapping.match) {
			return mapping.status, mapping.key
		}
	}

	return http.StatusInternalServerError, messagecontract.CommonInternalError
}

func authErrorDetails(err error) map[string]any {
	_ = err
	return nil
}
