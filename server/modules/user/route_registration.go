package user

import (
	useropenapi "graft/server/internal/contract/openapi/user"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	usercontract "graft/server/modules/user/contract"
)

type userRouteRegistrar struct {
	ctx        *module.Context
	moduleName string
	userSvc    userService
	authSvc    *authService
	guards     routeGuards
}

func registerUserRoutes(
	ctx *module.Context,
	moduleName string,
	userSvc userService,
	authSvc *authService,
	guards routeGuards,
) error {
	registrar := userRouteRegistrar{
		ctx:        ctx,
		moduleName: moduleName,
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
