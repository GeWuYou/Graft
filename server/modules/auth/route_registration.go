package auth

import (
	"github.com/gin-gonic/gin"

	authopenapi "graft/server/internal/contract/openapi/auth"
	"graft/server/internal/httpx"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	authcontract "graft/server/modules/auth/contract"
)

type routeGuards struct {
	authenticated          gin.HandlerFunc
	requiredPasswordChange gin.HandlerFunc
	restrictedSession      gin.HandlerFunc
}

type authRouteRegistrar struct {
	ctx            *module.Context
	moduleName     string
	authFlow       moduleapi.AuthFlowService
	personalTokens moduleapi.PersonalAccessTokenService
	cookies        CookieManager
	guards         routeGuards
}

func registerAuthRoutes(
	ctx *module.Context,
	moduleName string,
	authService moduleapi.AuthService,
	personalTokens moduleapi.PersonalAccessTokenService,
	authFlow moduleapi.AuthFlowService,
) error {
	authGroup := ctx.Router.Group(authcontract.AuthGroup)
	guards := newRouteGuards(ctx, authService, authFlow, authGroup.BasePath())

	registrar := authRouteRegistrar{
		ctx:            ctx,
		moduleName:     moduleName,
		authFlow:       authFlow,
		personalTokens: personalTokens,
		cookies:        NewCookieManager(ctx.Config.Auth),
		guards:         guards,
	}
	authGroup.Use(httpx.RequestIDMiddleware())
	registrar.registerLoginRoutes(authGroup)
	registrar.registerCurrentUserSessionRoutes(authGroup)
	registrar.registerPersonalAccessTokenRoutes(authGroup)
	registrar.registerBootstrapAndPasswordRoutes(authGroup)

	return nil
}

var _ authopenapi.ServerInterface = authGeneratedHandler{}
