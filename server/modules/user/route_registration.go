package user

import (
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/contract/openapi/strictsupport"

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
	wrapper      useropenapi.ServerInterfaceWrapper
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

	savedService, err := module.ResolveService[moduleapi.SavedViewService](ctx.Services, (*moduleapi.SavedViewService)(nil))
	if err != nil {
		return err
	}
	saved := newUserSavedViewHandlers(ctx.I18n, savedService)
	registrar.wrapper = registrar.newStrictWrapper(ctx, saved)
	group := registrar.ctx.Router.Group(usercontract.UsersGroup)
	group.Use(httpx.RequestIDMiddleware())
	registrar.registerUserReadRoutes(group)
	registerUserSavedViews(group, saved, registrar.wrapper, guards.userRead)
	registrar.registerUserWriteRoutes(group)
	registrar.registerAdminSessionRoutes(group)

	return nil
}

func (r userRouteRegistrar) newStrictWrapper(ctx *module.Context, saved userSavedViewHandlers) useropenapi.ServerInterfaceWrapper {
	return useropenapi.ServerInterfaceWrapper{
		Handler: useropenapi.NewStrictHandlerWithOptions(userStrictServer{registrar: r, saved: saved}, nil, useropenapi.StrictGinServerOptions{
			RequestErrorHandlerFunc: strictsupport.RequestErrorHandler(ctx.I18n, messagecontract.CommonInvalidArgument.String(), nil),
			HandlerErrorFunc:        strictsupport.HandlerErrorHandler(ctx.I18n, ctx.Logger),
		}),
		ErrorHandler: strictsupport.BindingErrorHandler(ctx.I18n, messagecontract.CommonInvalidArgument.String()),
	}
}
