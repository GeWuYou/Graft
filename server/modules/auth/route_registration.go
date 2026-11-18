package auth

import (
	"context"
	"github.com/gin-gonic/gin"

	messagecontract "graft/server/internal/contract/message"
	authopenapi "graft/server/internal/contract/openapi/auth"
	"graft/server/internal/contract/openapi/strictsupport"
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
	// wrapper 承载生成的 strict server 绑定入口；路由仍按模块 contract 片段挂载。
	wrapper authopenapi.ServerInterfaceWrapper
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
	// 生成 wrapper 承担路径/请求头与请求体绑定；绑定失败沿用统一 envelope 的 body 字段标注。
	registrar.wrapper = authopenapi.ServerInterfaceWrapper{
		Handler: authopenapi.NewStrictHandlerWithOptions(
			authStrictServer{registrar: registrar},
			nil,
			authopenapi.StrictGinServerOptions{
				RequestErrorHandlerFunc: strictsupport.RequestErrorHandler(ctx.I18n, messagecontract.CommonInvalidArgument.String(), gin.H{"field": "body"}),
				HandlerErrorFunc:        strictsupport.HandlerErrorHandler(ctx.I18n, ctx.Logger),
			},
		),
		ErrorHandler: strictsupport.BindingErrorHandler(ctx.I18n, messagecontract.CommonInvalidArgument.String()),
	}
	authGroup.Use(httpx.RequestIDMiddleware())
	registrar.registerLoginRoutes(authGroup)
	registrar.registerCurrentUserSessionRoutes(authGroup)
	registrar.registerPersonalAccessTokenRoutes(authGroup)
	registrar.registerBootstrapAndPasswordRoutes(authGroup)

	return nil
}

// authStrictServer 将认证路由主体适配到生成的 strict server 契约；
// 编译期断言保证 OpenAPI 认证操作变化时必须在适配层补齐。
// 公共登录/刷新路由不挂鉴权 guard，其余路由的 guard 链保持迁移前顺序。
type authStrictServer struct {
	registrar authRouteRegistrar
}

var _ authopenapi.StrictServerInterface = authStrictServer{}

func (s authStrictServer) PostAuthLogin(ctx context.Context, request authopenapi.PostAuthLoginRequestObject) (authopenapi.PostAuthLoginResponseObject, error) {
	if request.Body == nil {
		return nil, strictsupport.ErrMissingRequestBody
	}
	return nil, strictsupport.Dispatch(ctx, func(ginCtx *gin.Context) { s.registrar.handleLoginRoute(ginCtx, *request.Body) })
}

func (s authStrictServer) PostAuthRefresh(ctx context.Context, _ authopenapi.PostAuthRefreshRequestObject) (authopenapi.PostAuthRefreshResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handleRefreshRoute)
}

func (s authStrictServer) PostAuthLogout(ctx context.Context, _ authopenapi.PostAuthLogoutRequestObject) (authopenapi.PostAuthLogoutResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handleLogoutRoute)
}

func (s authStrictServer) PostAuthSessionsRevokeAll(ctx context.Context, _ authopenapi.PostAuthSessionsRevokeAllRequestObject) (authopenapi.PostAuthSessionsRevokeAllResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handleSessionsRevokeAllRoute)
}

func (s authStrictServer) PostAuthSessionsRevokeOthers(ctx context.Context, _ authopenapi.PostAuthSessionsRevokeOthersRequestObject) (authopenapi.PostAuthSessionsRevokeOthersResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handleSessionsRevokeOthersRoute)
}

func (s authStrictServer) GetAuthSessions(ctx context.Context, _ authopenapi.GetAuthSessionsRequestObject) (authopenapi.GetAuthSessionsResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handleSessionsListRoute)
}

func (s authStrictServer) PostAuthSessionRevoke(ctx context.Context, _ authopenapi.PostAuthSessionRevokeRequestObject) (authopenapi.PostAuthSessionRevokeResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handleSessionRevokeRoute)
}

func (s authStrictServer) GetAuthPersonalAccessTokens(ctx context.Context, _ authopenapi.GetAuthPersonalAccessTokensRequestObject) (authopenapi.GetAuthPersonalAccessTokensResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handlePersonalAccessTokenListRoute)
}

func (s authStrictServer) PostAuthPersonalAccessTokens(ctx context.Context, request authopenapi.PostAuthPersonalAccessTokensRequestObject) (authopenapi.PostAuthPersonalAccessTokensResponseObject, error) {
	if request.Body == nil {
		return nil, strictsupport.ErrMissingRequestBody
	}
	return nil, strictsupport.Dispatch(ctx, func(ginCtx *gin.Context) { s.registrar.handlePersonalAccessTokenCreateRoute(ginCtx, *request.Body) })
}

func (s authStrictServer) PostAuthPersonalAccessTokenRevoke(ctx context.Context, _ authopenapi.PostAuthPersonalAccessTokenRevokeRequestObject) (authopenapi.PostAuthPersonalAccessTokenRevokeResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handlePersonalAccessTokenRevokeRoute)
}

func (s authStrictServer) GetAuthBootstrap(ctx context.Context, _ authopenapi.GetAuthBootstrapRequestObject) (authopenapi.GetAuthBootstrapResponseObject, error) {
	return nil, strictsupport.Dispatch(ctx, s.registrar.handleBootstrapRoute)
}

func (s authStrictServer) PostAuthChangePassword(ctx context.Context, request authopenapi.PostAuthChangePasswordRequestObject) (authopenapi.PostAuthChangePasswordResponseObject, error) {
	if request.Body == nil {
		return nil, strictsupport.ErrMissingRequestBody
	}
	return nil, strictsupport.Dispatch(ctx, func(ginCtx *gin.Context) { s.registrar.handleChangePasswordRoute(ginCtx, *request.Body) })
}

func (s authStrictServer) PostAuthCompleteRequiredPasswordChange(ctx context.Context, request authopenapi.PostAuthCompleteRequiredPasswordChangeRequestObject) (authopenapi.PostAuthCompleteRequiredPasswordChangeResponseObject, error) {
	if request.Body == nil {
		return nil, strictsupport.ErrMissingRequestBody
	}
	return nil, strictsupport.Dispatch(ctx, func(ginCtx *gin.Context) { s.registrar.handleCompleteRequiredPasswordChangeRoute(ginCtx, *request.Body) })
}
