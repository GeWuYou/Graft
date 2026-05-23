package user

import (
	"graft/server/internal/httpx"
	"graft/server/internal/plugin"
	usercontract "graft/server/plugins/user/contract"
)

type authRouteRegistrar struct {
	ctx          *plugin.Context
	pluginName   string
	authSvc      *authService
	bootstrapSvc bootstrapReader
	guards       routeGuards
}

type userRouteRegistrar struct {
	ctx        *plugin.Context
	pluginName string
	userSvc    userService
	authSvc    *authService
	guards     routeGuards
}

func registerAuthRoutes(
	ctx *plugin.Context,
	pluginName string,
	authSvc *authService,
	bootstrapSvc bootstrapReader,
	guards *routeGuards,
) error {
	authGroup := ctx.Router.Group(usercontract.AuthGroup)
	guards.restrictedSession = newRestrictedSessionGuard(ctx.I18n, authSvc, authGroup.BasePath())

	registrar := authRouteRegistrar{
		ctx:          ctx,
		pluginName:   pluginName,
		authSvc:      authSvc,
		bootstrapSvc: bootstrapSvc,
		guards:       *guards,
	}
	authGroup.Use(httpx.RequestIDMiddleware())
	registrar.registerLoginRoutes(authGroup)
	registrar.registerCurrentUserSessionRoutes(authGroup)
	registrar.registerBootstrapAndPasswordRoutes(authGroup)

	return nil
}

func registerUserRoutes(
	ctx *plugin.Context,
	pluginName string,
	userSvc userService,
	authSvc *authService,
	guards routeGuards,
) error {
	registrar := userRouteRegistrar{
		ctx:        ctx,
		pluginName: pluginName,
		userSvc:    userSvc,
		authSvc:    authSvc,
		guards:     guards,
	}

	group := registrar.ctx.Router.Group(usercontract.UsersGroup)
	group.Use(httpx.RequestIDMiddleware())
	registrar.registerUserReadRoutes(group)
	registrar.registerUserWriteRoutes(group)
	registrar.registerAdminSessionRoutes(group)

	return nil
}
