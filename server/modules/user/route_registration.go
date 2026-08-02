package user

import (
	"github.com/gin-gonic/gin"

	useropenapi "graft/server/internal/contract/openapi/user"
	"graft/server/internal/httpx"
	applog "graft/server/internal/logger"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	authruntime "graft/server/modules/auth"
	usercontract "graft/server/modules/user/contract"
)

type userRouteRegistrar struct {
	ctx          *module.Context
	moduleName   string
	userSvc      userService
	authSessions moduleapi.AuthSessionService
	cookies      authruntime.CookieManager
	guards       routeGuards
	appLog       applog.AppLogger
}

// registerUserRoutes 注册用户相关的 HTTP 路由，并为路由组启用请求 ID 中间件。
// 缺少用户列表保存视图服务时返回装配错误。
func registerUserRoutes(
	ctx *module.Context,
	moduleName string,
	userSvc userService,
	authSessions moduleapi.AuthSessionService,
	guards routeGuards,
) error {
	registrar := userRouteRegistrar{
		ctx:          ctx,
		moduleName:   moduleName,
		userSvc:      userSvc,
		authSessions: authSessions,
		cookies:      authruntime.NewCookieManager(ctx.Config.Auth),
		guards:       guards,
		appLog:       resolveUserRouteAppLogger(ctx),
	}

	group := registrar.ctx.Router.Group(usercontract.UsersGroup)
	group.Use(httpx.RequestIDMiddleware())
	registrar.registerUserReadRoutes(group)
	if err := registrar.registerUserSavedViewRoutes(group); err != nil {
		return err
	}
	registrar.registerUserWriteRoutes(group)
	registrar.registerAdminSessionRoutes(group)

	return nil
}

func (r userRouteRegistrar) registerUserSavedViewRoutes(group *gin.RouterGroup) error {
	savedViews, err := module.ResolveService[moduleapi.SavedViewService](r.ctx.Services, (*moduleapi.SavedViewService)(nil))
	if err != nil {
		return err
	}
	registerUserSavedViews(group, r.ctx, savedViews, r.guards.userRead)
	return nil
}

var _ useropenapi.WriteServerInterface = userWriteGeneratedHandler{}
var _ useropenapi.ReadServerInterface = userReadGeneratedHandler{}
