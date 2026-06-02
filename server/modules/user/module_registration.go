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

func registerUserPermissions(registry *permission.Registry, pluginName string) {
	for _, item := range userPermissionItems(pluginName) {
		registry.Register(item)
	}
}

func registerUserMenu(registry *menu.Registry, pluginName string) {
	registry.Register(menu.Item{
		Code:       "user.list",
		Title:      "用户管理",
		TitleKey:   usercontract.UserListMenuTitle.String(),
		Path:       "/access-control/users",
		Icon:       "usergroup",
		Order:      userMenuOrderList,
		Permission: usercontract.UserReadPermission.String(),
		Module:     pluginName,
	})
}

func userPermissionItems(pluginName string) []permission.Item {
	return []permission.Item{
		{
			Code:        usercontract.UserReadPermission.String(),
			Name:        "Read Users",
			Description: "Allows reading user management data.",
			Category:    "api",
			Module:      pluginName,
		},
		{
			Code:        usercontract.UserCreatePermission.String(),
			Name:        "Create Users",
			Description: "Allows creating user management data.",
			Category:    "api",
			Module:      pluginName,
		},
		{
			Code:        usercontract.UserUpdatePermission.String(),
			Name:        "Update Users",
			Description: "Allows updating user management data.",
			Category:    "api",
			Module:      pluginName,
		},
		{
			Code:        usercontract.UserDisablePermission.String(),
			Name:        "Disable Users",
			Description: "Allows disabling or deleting managed users.",
			Category:    "api",
			Module:      pluginName,
		},
		{
			Code:        usercontract.UserSessionRevokePermission.String(),
			Name:        "Revoke User Sessions",
			Description: "Allows revoking refresh sessions for a specified user.",
			Category:    "api",
			Module:      pluginName,
		},
		{
			Code:        usercontract.UserSessionReadPermission.String(),
			Name:        "Read User Sessions",
			Description: "Allows reading active refresh sessions for a specified user.",
			Category:    "api",
			Module:      pluginName,
		},
	}
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
			return fmt.Errorf("register user module messages: %w", err)
		}
	}

	return nil
}

type registeredServices struct {
	user      userService
	auth      *authService
	bootstrap bootstrapReader
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
	bootstrapSvc := newBootstrapReader(ctx.Config.I18n, ctx.I18n, ctx.MenuRegistry, authRepo, p.bootstrapAccess)
	p.defaultAdminAuth = authSvc

	if err := ctx.Services.RegisterSingleton((*moduleapi.AuthService)(nil), func(_ container.Resolver) (any, error) {
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
		user:      userSvc,
		auth:      authSvc,
		bootstrap: bootstrapSvc,
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
