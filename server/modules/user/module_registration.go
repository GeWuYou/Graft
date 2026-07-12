package user

import (
	"context"
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
)

const userMenuOrderList = 2

func registerUserPermissions(registry *permission.Registry, moduleName string) {
	for _, item := range userPermissionItems(moduleName) {
		registry.Register(item)
	}
}

// registerUserMenu registers the user list menu entry under the security domain.
func registerUserMenu(registry *menu.Registry, moduleName string) {
	registry.Register(menu.Item{
		Code:       "user.list",
		ParentCode: "domain.security",
		Kind:       menu.NodeKindEntry,
		Title:      "",
		TitleKey:   usercontract.UserListMenuTitle.String(),
		Path:       "/users",
		Icon:       "usergroup",
		Order:      userMenuOrderList,
		Permission: usercontract.UserReadPermission.String(),
		Module:     moduleName,
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
		},
		{
			Code:           usercontract.UserCreatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userCreate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userCreate.description",
			Module:         moduleName,
		},
		{
			Code:           usercontract.UserUpdatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userUpdate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userUpdate.description",
			Module:         moduleName,
		},
		{
			Code:           usercontract.UserDisablePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userDisable.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userDisable.description",
			Module:         moduleName,
		},
		{
			Code:           usercontract.UserSessionRevokePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userSessionRevoke.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userSessionRevoke.description",
			Module:         moduleName,
		},
		{
			Code:           usercontract.UserSessionReadPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userSessionRead.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userSessionRead.description",
			Module:         moduleName,
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
	logger := ctx.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	p.bootstrapAccess = newDeferredRBACAccessService()
	p.userCredentials = newDeferredCredentialManagementService()
	userSvc := userService{
		users:       userRepo,
		rbac:        p.bootstrapAccess,
		credentials: p.userCredentials,
		auditBus:    ctx.EventBus,
		logger:      logger,
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.UserService)(nil), func(_ container.Resolver) (any, error) {
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

type deferredCredentialManagementService struct {
	mu     sync.RWMutex
	target moduleapi.AuthCredentialManagementService
}

// newDeferredCredentialManagementService creates a credential management service without a configured target.
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
