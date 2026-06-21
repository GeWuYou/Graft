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
	userstore "graft/server/modules/user/store"
)

const userMenuOrderList = 2

func registerUserPermissions(registry *permission.Registry, moduleName string) {
	for _, item := range userPermissionItems(moduleName) {
		registry.Register(item)
	}
}

func registerUserMenu(registry *menu.Registry, moduleName string) {
	registry.Register(menu.Item{
		Code:       "user.list",
		Title:      "",
		TitleKey:   usercontract.UserListMenuTitle.String(),
		Path:       "/access-control/users",
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
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           usercontract.UserCreatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userCreate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userCreate.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           usercontract.UserUpdatePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userUpdate.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userUpdate.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           usercontract.UserDisablePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userDisable.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userDisable.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           usercontract.UserSessionRevokePermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userSessionRevoke.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userSessionRevoke.description",
			Category:       "api",
			Module:         moduleName,
		},
		{
			Code:           usercontract.UserSessionReadPermission.String(),
			Name:           "",
			DisplayKey:     "rbac.permissionCatalog.userSessionRead.display",
			Description:    "",
			DescriptionKey: "rbac.permissionCatalog.userSessionRead.description",
			Category:       "api",
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
	authRepo     userstore.AuthRepository
	passwords    passwordHasher
	policy       passwordPolicy
}

func (p *Module) registerServices(ctx *module.Context) (registeredServices, error) {
	userRepo := p.userRepo
	authRepo := p.authRepo
	if userRepo == nil {
		return registeredServices{}, errors.New("user repository is unavailable")
	}
	if authRepo == nil {
		return registeredServices{}, errors.New("auth repository is unavailable")
	}
	logger := ctx.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	p.bootstrapAccess = newDeferredRBACAccessService()
	userSvc := userService{
		users:    userRepo,
		rbac:     p.bootstrapAccess,
		auditBus: ctx.EventBus,
		logger:   logger,
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.UserService)(nil), func(_ container.Resolver) (any, error) {
		return userSvc, nil
	}); err != nil {
		return registeredServices{}, err
	}

	authSvc, err := newAuthService(ctx.Config.Auth, authRepo, userRepo)
	if err != nil {
		return registeredServices{}, err
	}
	bootstrapSvc := newBootstrapReader(ctx.Config.I18n, ctx.I18n, ctx.MenuRegistry, ctx.Services, authRepo, p.bootstrapAccess)
	p.defaultAdminAuth = authSvc

	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthService)(nil), func(_ container.Resolver) (any, error) {
		return authSvc, nil
	}); err != nil {
		return registeredServices{}, err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthSessionService)(nil), func(_ container.Resolver) (any, error) {
		return authSvc, nil
	}); err != nil {
		return registeredServices{}, err
	}
	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthFlowService)(nil), func(_ container.Resolver) (any, error) {
		return authFlowBridge{
			auth:      authSvc,
			bootstrap: bootstrapSvc,
		}, nil
	}); err != nil {
		return registeredServices{}, err
	}

	return registeredServices{
		user:         userSvc,
		auth:         authSvc,
		authSessions: authSvc,
		authFlow:     authFlowBridge{auth: authSvc, bootstrap: bootstrapSvc},
		bootstrap:    bootstrapSvc,
		authRepo:     authSvc.auth,
		passwords:    authSvc.passwords,
		policy:       authSvc.policy,
	}, nil
}

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
	authRepo               userstore.AuthRepository
	passwords              passwordHasher
	policy                 passwordPolicy
}
