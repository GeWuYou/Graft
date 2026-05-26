package user

import (
	useropenapi "graft/server/internal/contract/openapi/user"
	"graft/server/internal/httpx"
	"graft/server/internal/plugin"
	usercontract "graft/server/plugins/user/contract"
)

type userRouteRegistrar struct {
	ctx        *plugin.Context
	pluginName string
	userSvc    userService
	authSvc    *authService
	guards     routeGuards
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

var _ useropenapi.WriteServerInterface = userWriteGeneratedHandler{}
var _ useropenapi.ReadServerInterface = userReadGeneratedHandler{}
