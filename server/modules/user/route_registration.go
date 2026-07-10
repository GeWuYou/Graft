package user

import (
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
// 返回注册结果；当前始终为 nil。
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
	registrar.registerUserWriteRoutes(group)
	registrar.registerAdminSessionRoutes(group)

	return nil
}

var _ useropenapi.WriteServerInterface = userWriteGeneratedHandler{}
var _ useropenapi.ReadServerInterface = userReadGeneratedHandler{}
