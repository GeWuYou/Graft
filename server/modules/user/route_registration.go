package user

import (
	useropenapi "graft/server/internal/contract/openapi/user"
	"graft/server/internal/httpx"
	applog "graft/server/internal/logger"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	authruntime "graft/server/modules/auth"
	usercontract "graft/server/modules/user/contract"
	userstore "graft/server/modules/user/store"
)

type userRouteRegistrar struct {
	ctx          *module.Context
	moduleName   string
	userSvc      userService
	authSessions moduleapi.AuthSessionService
	cookies      authruntime.CookieManager
	authRepo     userstore.AuthRepository
	passwords    passwordHasher
	policy       passwordPolicy
	guards       routeGuards
	appLog       applog.AppLogger
}

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
		authRepo:     guards.authRepo,
		passwords:    guards.passwords,
		policy:       guards.policy,
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
