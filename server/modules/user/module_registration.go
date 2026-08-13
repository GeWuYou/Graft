package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/container"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	"graft/server/internal/permission"
	usercontract "graft/server/modules/user/contract"
	userstore "graft/server/modules/user/store"
)

const userMenuOrderList = 3

// registerUserPermissions 将用户模块的权限项注册到指定的权限注册表中。
func registerUserPermissions(registry *permission.Registry, moduleName string) {
	for _, item := range userPermissionItems(moduleName) {
		registry.Register(item)
	}
}

// registerUserMenu 在安全域下注册用户列表菜单；菜单可见性由权限注册结果决定，接口仍执行独立鉴权。
func registerUserMenu(registry *menu.Registry, moduleName string) {
	registry.Register(menu.Item{
		Code:            "user.list",
		ParentCode:      "domain.security",
		Kind:            menu.NodeKindEntry,
		Title:           "",
		TitleKey:        usercontract.UserListMenuTitle.String(),
		SectionKey:      menu.AccessControlSectionKey,
		SectionTitleKey: menu.AccessControlSectionTitleKey,
		Path:            usercontract.UserListMenuPath,
		Icon:            "user-identity",
		Order:           userMenuOrderList,
		Permission:      usercontract.UserReadPermission.String(),
		Module:          moduleName,
	})
}

func userPermissionItems(moduleName string) []permission.Item {
	return []permission.Item{
		{
			Code:           usercontract.UserReadPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userRead.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userRead.description",
			Module:         moduleName,
			Resource:       "user",
			Action:         "read",
			RiskLevel:      permission.RiskLevelLow,
			RiskCategory:   permission.RiskCategorySecurity,
		},
		{
			Code:           usercontract.UserCreatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userCreate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userCreate.description",
			Module:         moduleName,
			Resource:       "user",
			Action:         "create",
			RiskLevel:      permission.RiskLevelMedium,
			RiskCategory:   permission.RiskCategorySecurity,
		},
		{
			Code:           usercontract.UserUpdatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userUpdate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userUpdate.description",
			Module:         moduleName,
			Resource:       "user",
			Action:         "update",
			RiskLevel:      permission.RiskLevelMedium,
			RiskCategory:   permission.RiskCategorySecurity,
		},
		{
			Code:           usercontract.UserDisablePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userDisable.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userDisable.description",
			Module:         moduleName,
			Resource:       "user",
			Action:         "disable",
			RiskLevel:      permission.RiskLevelHigh,
			RiskCategory:   permission.RiskCategorySecurity,
		},
		{
			Code:           usercontract.UserSessionRevokePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userSessionRevoke.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userSessionRevoke.description",
			Module:         moduleName,
			Resource:       "user.session",
			Action:         "revoke",
			RiskLevel:      permission.RiskLevelHigh,
			RiskCategory:   permission.RiskCategorySecurity,
		},
		{
			Code:           usercontract.UserSessionReadPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userSessionRead.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userSessionRead.description",
			Module:         moduleName,
			Resource:       "user.session",
			Action:         "read",
			RiskLevel:      permission.RiskLevelLow,
			RiskCategory:   permission.RiskCategorySecurity,
		},
	}
}

func registerMessages(localizer *i18n.Service) error {
	if localizer == nil {
		return errors.New("i18n service is unavailable")
	}

	for _, locale := range []i18n.LocaleTag{i18n.LocaleZHCN, i18n.LocaleENUS} {
		matches := localizer.RegisteredMessageResources(locale, i18n.MessageKey(usercontract.UserListMenuTitle.String()))
		if len(matches) == 0 {
			return fmt.Errorf("register user module messages: locale resource %s missing key %s", locale, usercontract.UserListMenuTitle.String())
		}
	}

	return nil
}

type registeredServices struct {
	user         userService
	auth         moduleapi.AuthService
	authSessions moduleapi.AuthSessionService
	authFlow     moduleapi.AuthFlowService
	bootstrap    bootstrapReader
}

func (p *Module) registerServices(ctx *module.Context) (registeredServices, error) {
	userRepo := p.userRepo
	if userRepo == nil {
		return registeredServices{}, errors.New("user repository is unavailable")
	}
	transactions, ok := userRepo.(userstore.TransactionRunner)
	if !ok {
		return registeredServices{}, errors.New("user repository does not support profile transactions")
	}
	composites, ok := userRepo.(userstore.CompositeTransactionRunner)
	if !ok {
		return registeredServices{}, errors.New("user repository does not support composite transactions")
	}
	logger := ctx.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	p.bootstrapAccess = newDeferredRBACAccessService()
	p.userCredentials = newDeferredCredentialManagementService()
	p.authTransactions = newDeferredAuthTransactionFactory()
	userSvc := userService{
		users:        userRepo,
		transactions: transactions,
		composites:   composites,
		authTx:       p.authTransactions,
		rbac:         p.bootstrapAccess,
		credentials:  p.userCredentials,
		auditBus:     ctx.EventBus,
		logger:       logger,
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.UserService)(nil), func(_ container.Resolver) (any, error) {
		return userSvc, nil
	}); err != nil {
		return registeredServices{}, err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.UserCandidateReader)(nil), func(_ container.Resolver) (any, error) {
		return userSvc, nil
	}); err != nil {
		return registeredServices{}, err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.UserSecurityReader)(nil), func(_ container.Resolver) (any, error) {
		return userSvc, nil
	}); err != nil {
		return registeredServices{}, err
	}

	identity := userIdentityProvider{users: userRepo}
	if err := ctx.Services.RegisterSingleton((*moduleapi.UserIdentityProvider)(nil), func(_ container.Resolver) (any, error) {
		return identity, nil
	}); err != nil {
		return registeredServices{}, err
	}
	bootstrapSvc := newBootstrapReader(ctx.Config.I18n, ctx.I18n, ctx.MenuRegistry, ctx.Services, p.bootstrapAccess)
	if err := ctx.Services.RegisterSingleton((*moduleapi.UserBootstrapProvider)(nil), func(_ container.Resolver) (any, error) {
		return bootstrapSvc, nil
	}); err != nil {
		return registeredServices{}, err
	}
	deferredAuth := newDeferredAuthCapabilities()
	p.authCapabilities = deferredAuth

	return registeredServices{
		user:         userSvc,
		auth:         deferredAuth,
		authSessions: deferredAuth,
		authFlow:     deferredAuth,
		bootstrap:    bootstrapSvc,
	}, nil
}

type deferredAuthTransactionFactory struct {
	mu     sync.RWMutex
	target moduleapi.AuthTransactionAdapterFactory
}

func newDeferredAuthTransactionFactory() *deferredAuthTransactionFactory {
	return &deferredAuthTransactionFactory{}
}
func (f *deferredAuthTransactionFactory) SetTarget(target moduleapi.AuthTransactionAdapterFactory) error {
	if target == nil {
		return errors.New("auth transaction adapter factory is required")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.target = target
	return nil
}
func (f *deferredAuthTransactionFactory) BindAuthTransaction(tx *sql.Tx) (moduleapi.AuthTransactionAdapter, error) {
	f.mu.RLock()
	target := f.target
	f.mu.RUnlock()
	if target == nil {
		return nil, errors.New("auth transaction adapter factory is unavailable")
	}
	return target.BindAuthTransaction(tx)
}

var _ moduleapi.AuthTransactionAdapterFactory = (*deferredAuthTransactionFactory)(nil)

type deferredCredentialManagementService struct {
	mu     sync.RWMutex
	target moduleapi.AuthCredentialManagementService
}

// newDeferredCredentialManagementService 创建未绑定目标的凭据管理服务；绑定前调用会返回服务不可用错误。
func newDeferredCredentialManagementService() *deferredCredentialManagementService {
	return &deferredCredentialManagementService{}
}

func (s *deferredCredentialManagementService) SetTarget(target moduleapi.AuthCredentialManagementService) error {
	if target == nil {
		return errors.New("auth credential management service is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.target = target
	return nil
}

func (s *deferredCredentialManagementService) targetService() (moduleapi.AuthCredentialManagementService, error) {
	s.mu.RLock()
	target := s.target
	s.mu.RUnlock()
	if target == nil {
		return nil, errors.New("auth credential management service is unavailable")
	}
	return target, nil
}

func (s *deferredCredentialManagementService) ProvisionPasswordCredential(ctx context.Context, userID uint64, password string, mustChangePassword bool) error {
	target, err := s.targetService()
	if err != nil {
		return err
	}
	return target.ProvisionPasswordCredential(ctx, userID, password, mustChangePassword)
}
func (s *deferredCredentialManagementService) ResetPassword(ctx context.Context, userID uint64, password string) error {
	target, err := s.targetService()
	if err != nil {
		return err
	}
	return target.ResetPassword(ctx, userID, password)
}
func (s *deferredCredentialManagementService) RevokeSessions(ctx context.Context, userID uint64) error {
	target, err := s.targetService()
	if err != nil {
		return err
	}
	return target.RevokeSessions(ctx, userID)
}

var _ moduleapi.AuthCredentialManagementService = (*deferredCredentialManagementService)(nil)

// deferredAuthorizer 让用户路由在 Register 阶段先完成装配，再在 Boot 阶段绑定
// 已注册的共享 Authorizer，避免复制 RBAC 授权语义或把 Resolve 扩散到请求热路径。
type deferredAuthorizer struct {
	mu     sync.RWMutex
	target moduleapi.Authorizer
}

func newDeferredAuthorizer() *deferredAuthorizer {
	return &deferredAuthorizer{}
}

func (a *deferredAuthorizer) SetTarget(target moduleapi.Authorizer) error {
	if target == nil {
		return errors.New("authorizer is required")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.target = target
	return nil
}

func (a *deferredAuthorizer) Authorize(
	ctx context.Context,
	request moduleapi.RequestAuthContext,
	permission string,
) error {
	a.mu.RLock()
	target := a.target
	a.mu.RUnlock()

	if target == nil {
		return errors.New("authorizer is unavailable")
	}

	return target.Authorize(ctx, request, permission)
}

var _ moduleapi.Authorizer = (*deferredAuthorizer)(nil)

type deferredRBACAccessService struct {
	mu     sync.RWMutex
	target moduleapi.RBACAccessService
}

func newDeferredRBACAccessService() *deferredRBACAccessService {
	return &deferredRBACAccessService{}
}

func (s *deferredRBACAccessService) SetTarget(target moduleapi.RBACAccessService) error {
	if target == nil {
		return errors.New("rbac access service is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.target = target
	return nil
}

func (s *deferredRBACAccessService) ListRoleNamesByUserID(ctx context.Context, userID uint64) ([]string, error) {
	s.mu.RLock()
	target := s.target
	s.mu.RUnlock()

	if target == nil {
		return nil, errors.New("rbac access service is unavailable")
	}

	return target.ListRoleNamesByUserID(ctx, userID)
}

func (s *deferredRBACAccessService) ListPermissionCodesByUserID(ctx context.Context, userID uint64) ([]string, error) {
	s.mu.RLock()
	target := s.target
	s.mu.RUnlock()

	if target == nil {
		return nil, errors.New("rbac access service is unavailable")
	}

	return target.ListPermissionCodesByUserID(ctx, userID)
}

func (s *deferredRBACAccessService) ListUserIDsByPermissionCode(ctx context.Context, permissionCode string) ([]uint64, error) {
	s.mu.RLock()
	target := s.target
	s.mu.RUnlock()

	if target == nil {
		return nil, errors.New("rbac access service is unavailable")
	}

	return target.ListUserIDsByPermissionCode(ctx, permissionCode)
}

func (s *deferredRBACAccessService) ListUserIDsByRoleID(ctx context.Context, roleID uint64) ([]uint64, error) {
	s.mu.RLock()
	target := s.target
	s.mu.RUnlock()
	if target == nil {
		return nil, errors.New("rbac access service is unavailable")
	}

	return target.ListUserIDsByRoleID(ctx, roleID)
}

func (s *deferredRBACAccessService) ListRoleSummariesByUserIDs(
	ctx context.Context,
	userIDs []uint64,
) (map[uint64][]moduleapi.RoleSummary, error) {
	s.mu.RLock()
	target := s.target
	s.mu.RUnlock()

	if target == nil {
		return nil, errors.New("rbac access service is unavailable")
	}

	return target.ListRoleSummariesByUserIDs(ctx, userIDs)
}

var _ moduleapi.RBACAccessService = (*deferredRBACAccessService)(nil)

type routeGuards struct {
	authenticated          gin.HandlerFunc
	requiredPasswordChange gin.HandlerFunc
	restrictedSession      gin.HandlerFunc
	userRead               gin.HandlerFunc
	userCreate             gin.HandlerFunc
	userUpdate             gin.HandlerFunc
	userDisable            gin.HandlerFunc
	userSessionRead        gin.HandlerFunc
	userSessionRevoke      gin.HandlerFunc
}
