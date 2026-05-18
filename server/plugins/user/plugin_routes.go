package user

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/container"
	messagecontract "graft/server/internal/contract/message"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
	usercontract "graft/server/plugins/user/contract"
)

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
		Path:       usercontract.UsersGroup,
		Icon:       "usergroup",
		Permission: usercontract.UserReadPermission.String(),
		Plugin:     pluginName,
	})
}

func userPermissionItems(pluginName string) []permission.Item {
	return []permission.Item{
		{
			Code:        usercontract.UserReadPermission.String(),
			Name:        "Read Users",
			Description: "Allows reading user management data.",
			Category:    "api",
			Plugin:      pluginName,
		},
		{
			Code:        usercontract.UserSessionRevokePermission.String(),
			Name:        "Revoke User Sessions",
			Description: "Allows revoking refresh sessions for a specified user.",
			Category:    "api",
			Plugin:      pluginName,
		},
		{
			Code:        usercontract.UserSessionReadPermission.String(),
			Name:        "Read User Sessions",
			Description: "Allows reading active refresh sessions for a specified user.",
			Category:    "api",
			Plugin:      pluginName,
		},
	}
}

type registeredServices struct {
	user      userService
	auth      *authService
	bootstrap bootstrapReader
}

func (p *Plugin) registerServices(ctx *plugin.Context) (registeredServices, error) {
	userSvc := userService{users: ctx.Stores.Users()}
	if err := ctx.Services.RegisterSingleton((*pluginapi.UserService)(nil), func(_ container.Resolver) (any, error) {
		return userSvc, nil
	}); err != nil {
		return registeredServices{}, err
	}

	authSvc, err := newAuthService(ctx.Config.Auth, ctx.Stores.Auth(), ctx.Stores.Users())
	if err != nil {
		return registeredServices{}, err
	}
	bootstrapSvc := newBootstrapReader(ctx.Config.I18n, ctx.I18n, ctx.MenuRegistry, ctx.Stores.Auth(), ctx.Stores.RBAC())
	p.defaultAdminAuth = authSvc

	if err := ctx.Services.RegisterSingleton((*pluginapi.AuthService)(nil), func(_ container.Resolver) (any, error) {
		return authSvc, nil
	}); err != nil {
		return registeredServices{}, err
	}

	return registeredServices{
		user:      userSvc,
		auth:      authSvc,
		bootstrap: bootstrapSvc,
	}, nil
}

type routeGuards struct {
	authenticated          gin.HandlerFunc
	requiredPasswordChange gin.HandlerFunc
	restrictedSession      gin.HandlerFunc
	userRead               gin.HandlerFunc
	userSessionRead        gin.HandlerFunc
	userSessionRevoke      gin.HandlerFunc
}

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

type routeRuntime struct {
	localizer  *i18n.Service
	logger     *zap.Logger
	pluginName string
}

// deferredAuthorizer 让用户路由在 Register 阶段先完成装配，再在 Boot 阶段绑定
// 已注册的共享 Authorizer，避免复制 RBAC 授权语义或把 Resolve 扩散到请求热路径。
type deferredAuthorizer struct {
	mu     sync.RWMutex
	target pluginapi.Authorizer
}

func newDeferredAuthorizer() *deferredAuthorizer {
	return &deferredAuthorizer{}
}

func (a *deferredAuthorizer) SetTarget(target pluginapi.Authorizer) error {
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
	request pluginapi.RequestAuthContext,
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

func newRouteGuards(localizer *i18n.Service, authSvc *authService, authorizer pluginapi.Authorizer) routeGuards {
	return routeGuards{
		authenticated:          httpx.RequirePermission(localizer, authSvc, nil, ""),
		requiredPasswordChange: newRequiredPasswordChangeGuard(localizer, authSvc),
		userRead:               httpx.RequirePermission(localizer, authSvc, authorizer, usercontract.UserReadPermission.String()),
		userSessionRead:        httpx.RequirePermission(localizer, authSvc, authorizer, usercontract.UserSessionReadPermission.String()),
		userSessionRevoke:      httpx.RequirePermission(localizer, authSvc, authorizer, usercontract.UserSessionRevokePermission.String()),
	}
}

var _ pluginapi.Authorizer = (*deferredAuthorizer)(nil)

func newRequiredPasswordChangeGuard(localizer *i18n.Service, authSvc *authService) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		restricted, ok := loadRestrictedPasswordChangeState(ginCtx, localizer, authSvc)
		if !ok {
			return
		}
		if !restricted {
			abortLocalizedContractError(ginCtx, localizer, http.StatusForbidden, messagecontract.AuthForbidden, nil)
			return
		}

		ginCtx.Next()
	}
}

func newRestrictedSessionGuard(localizer *i18n.Service, authSvc *authService, apiBasePath string) gin.HandlerFunc {
	allowedPaths := []string{
		usercontract.JoinRoute(apiBasePath, usercontract.AuthBootstrap),
		usercontract.JoinRoute(apiBasePath, usercontract.AuthCompleteRequiredPasswordChange),
	}

	return func(ginCtx *gin.Context) {
		restricted, ok := loadRestrictedPasswordChangeState(ginCtx, localizer, authSvc)
		if !ok {
			return
		}
		if !restricted {
			ginCtx.Next()
			return
		}

		for _, allowedPath := range allowedPaths {
			if ginCtx.FullPath() == allowedPath {
				ginCtx.Next()
				return
			}
		}

		abortLocalizedContractError(ginCtx, localizer, http.StatusForbidden, messagecontract.AuthForbidden, nil)
	}
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

func (r authRouteRegistrar) registerLoginRoutes(authGroup *gin.RouterGroup) {
	authGroup.POST(usercontract.AuthLogin, func(ginCtx *gin.Context) {
		var request loginRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}
		normalizedUsername := strings.TrimSpace(request.Username)
		if normalizedUsername == "" {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "username")
			return
		}
		if request.Password == "" {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "password")
			return
		}

		result, err := r.authSvc.LoginWithRefresh(ginCtx.Request.Context(), normalizedUsername, request.Password)
		if err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "login failed", err)
			return
		}

		r.authSvc.cookies.writeRefreshCookie(ginCtx, result.RefreshToken, result.RefreshExpiry)
		httpx.WriteSuccess(ginCtx, http.StatusOK, loginResponse{
			AccessToken:        result.AccessToken,
			ExpiresAt:          result.AccessExpiry,
			MustChangePassword: result.MustChangePassword,
			User:               result.User,
		})
	})
	authGroup.POST(usercontract.AuthRefresh, func(ginCtx *gin.Context) {
		refreshToken, err := r.authSvc.cookies.readRefreshCookie(ginCtx)
		if err != nil {
			writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
			return
		}

		result, err := r.authSvc.RefreshWithRotation(ginCtx.Request.Context(), refreshToken)
		if err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "refresh session failed", err)
			return
		}

		r.authSvc.cookies.writeRefreshCookie(ginCtx, result.RefreshToken, result.RefreshExpiry)
		httpx.WriteSuccess(ginCtx, http.StatusOK, loginResponse{
			AccessToken:        result.AccessToken,
			ExpiresAt:          result.AccessExpiry,
			MustChangePassword: result.MustChangePassword,
			User:               result.User,
		})
	})
	authGroup.POST(usercontract.AuthLogout, func(ginCtx *gin.Context) {
		refreshToken, err := r.authSvc.cookies.readRefreshCookie(ginCtx)
		if err != nil {
			writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
			return
		}

		if err := r.authSvc.LogoutCurrentSession(ginCtx.Request.Context(), refreshToken); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "logout session failed", err)
			return
		}

		r.authSvc.cookies.clearRefreshCookie(ginCtx)
		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func (r authRouteRegistrar) registerCurrentUserSessionRoutes(authGroup *gin.RouterGroup) {
	authGroup.POST(usercontract.AuthSessionsRevokeAll, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		if err := r.authSvc.RevokeAllCurrentUserSessions(ginCtx.Request.Context()); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "revoke all refresh sessions failed", err)
			return
		}

		r.authSvc.cookies.clearRefreshCookie(ginCtx)
		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
	authGroup.POST(usercontract.AuthSessionsRevokeOthers, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		if err := r.authSvc.RevokeOtherCurrentUserSessions(ginCtx.Request.Context()); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "revoke other user refresh sessions failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
	authGroup.GET(usercontract.AuthSessions, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		listOptions, err := parseSessionListOptions(ginCtx.Query("limit"))
		if err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "limit")
			return
		}

		sessions, err := r.authSvc.ListCurrentUserSessions(ginCtx.Request.Context(), listOptions)
		if err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "list current user refresh sessions failed", err)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, sessions)
	})
	authGroup.POST(usercontract.AuthSessionRevoke, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		sessionID, ok := readSessionIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		handleSessionRevocation(
			ginCtx,
			func(ctx context.Context) error {
				return r.authSvc.RevokeCurrentUserSession(ctx, sessionID)
			},
			func(err error) {
				r.runtime().writeAuthRouteError(ginCtx, "revoke current user refresh session failed", err, zap.String("sessionID", sessionID))
			},
			r.authSvc,
			func(claims *pluginapi.AccessTokenClaims) bool {
				return claims.SessionID == sessionID
			},
		)
	})
}

func (r authRouteRegistrar) registerBootstrapAndPasswordRoutes(authGroup *gin.RouterGroup) {
	r.registerBootstrapRoute(authGroup)
	r.registerChangePasswordRoute(authGroup)
	r.registerCompleteRequiredPasswordChangeRoute(authGroup)
}

func (r authRouteRegistrar) registerBootstrapRoute(authGroup *gin.RouterGroup) {
	authGroup.GET(usercontract.AuthBootstrap, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		payload, err := r.bootstrapSvc.Read(ginCtx.Request.Context(), ginCtx.Request)
		if err != nil {
			if errors.Is(err, pluginapi.ErrUnauthenticated) {
				writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
				return
			}

			r.runtime().logger.Error("read bootstrap payload failed",
				zap.String("plugin", r.pluginName),
				zap.Error(err),
			)
			writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	})
}

func (r authRouteRegistrar) registerChangePasswordRoute(authGroup *gin.RouterGroup) {
	authGroup.POST(usercontract.AuthChangePassword, r.guards.authenticated, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		var request changePasswordRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}
		if strings.TrimSpace(request.CurrentPassword) == "" {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "current_password")
			return
		}
		if strings.TrimSpace(request.NewPassword) == "" {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "new_password")
			return
		}

		if err := r.authSvc.ChangeCurrentUserPassword(ginCtx.Request.Context(), request.CurrentPassword, request.NewPassword); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "change current user password failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func (r authRouteRegistrar) registerCompleteRequiredPasswordChangeRoute(authGroup *gin.RouterGroup) {
	authGroup.POST(usercontract.AuthCompleteRequiredPasswordChange, r.guards.authenticated, r.guards.requiredPasswordChange, func(ginCtx *gin.Context) {
		var request completeRequiredPasswordChangeRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "body")
			return
		}
		if strings.TrimSpace(request.NewPassword) == "" {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "new_password")
			return
		}

		if err := r.authSvc.CompleteRequiredPasswordChange(ginCtx.Request.Context(), request.NewPassword); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "complete required password change failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func handleSessionRevocation(
	ginCtx *gin.Context,
	revoke func(context.Context) error,
	writeRouteError func(error),
	authSvc *authService,
	shouldClearCookie func(*pluginapi.AccessTokenClaims) bool,
) {
	if err := revoke(ginCtx.Request.Context()); err != nil {
		writeRouteError(err)
		return
	}

	clearRefreshCookieWhen(ginCtx, authSvc, shouldClearCookie)
	httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
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
	registrar.registerAdminSessionRoutes(group)

	return nil
}

func (r userRouteRegistrar) registerUserReadRoutes(group *gin.RouterGroup) {
	group.GET(usercontract.UserCollection, r.guards.userRead, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		users, err := r.userSvc.ListUsers(ginCtx.Request.Context())
		if err != nil {
			r.runtime().logger.Error("list users failed",
				zap.String("plugin", r.pluginName),
				zap.Error(err),
			)
			writeLocalizedContractError(ginCtx, r.ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
			return
		}

		items := make([]userListItem, 0, len(users))
		for _, user := range users {
			items = append(items, userListItem{
				ID:        user.ID,
				Username:  user.Username,
				Display:   user.Display,
				CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339),
				UpdatedAt: user.UpdatedAt.UTC().Format(time.RFC3339),
			})
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, userListResponse{Items: items})
	})
	group.GET(usercontract.UserByID, r.guards.userRead, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		rawID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		summary, err := r.userSvc.GetUserByID(ginCtx.Request.Context(), rawID)
		if err != nil {
			r.runtime().writeUserLookupError(ginCtx, rawID, "get user by id failed", err)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, summary)
	})
}

func (r userRouteRegistrar) registerAdminSessionRoutes(group *gin.RouterGroup) {
	r.registerAdminSessionReadRoute(group)
	r.registerAdminSessionRevokeRoutes(group)
}

func (r userRouteRegistrar) registerAdminSessionReadRoute(group *gin.RouterGroup) {
	group.GET(usercontract.UserSessions, r.guards.userSessionRead, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		rawID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		summary, err := r.userSvc.GetUserByID(ginCtx.Request.Context(), rawID)
		if err != nil {
			r.runtime().writeUserLookupError(ginCtx, rawID, "get user by id before listing sessions failed", err)
			return
		}

		listOptions, err := parseSessionListOptions(ginCtx.Query("limit"))
		if err != nil {
			writeInvalidArgumentField(ginCtx, r.ctx.I18n, "limit")
			return
		}

		sessions, err := r.authSvc.ListUserSessions(ginCtx.Request.Context(), summary.ID, listOptions)
		if err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "list user refresh sessions failed", err, zap.Uint64("userID", rawID))
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, sessions)
	})
}

func (r userRouteRegistrar) registerAdminSessionRevokeRoutes(group *gin.RouterGroup) {
	r.registerAdminRevokeSingleSessionRoute(group)
	r.registerAdminRevokeAllSessionsRoute(group)
}

func (r userRouteRegistrar) registerAdminRevokeSingleSessionRoute(group *gin.RouterGroup) {
	group.POST(usercontract.UserSessionByIDRevoke, r.guards.userSessionRevoke, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		rawID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		sessionID, ok := readSessionIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		summary, err := r.userSvc.GetUserByID(ginCtx.Request.Context(), rawID)
		if err != nil {
			r.runtime().writeUserLookupError(ginCtx, rawID, "get user by id before revoking session failed", err)
			return
		}

		if err := r.authSvc.RevokeUserSession(ginCtx.Request.Context(), summary.ID, sessionID); err != nil {
			r.runtime().writeAuthRouteError(
				ginCtx,
				"admin revoke user refresh session failed",
				err,
				zap.Uint64("userID", rawID),
				zap.String("sessionID", sessionID),
			)
			return
		}

		clearRefreshCookieWhen(ginCtx, r.authSvc, func(claims *pluginapi.AccessTokenClaims) bool {
			return claims.UserID == rawID && claims.SessionID == sessionID
		})
		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func (r userRouteRegistrar) registerAdminRevokeAllSessionsRoute(group *gin.RouterGroup) {
	group.POST(usercontract.UserSessionsRevokeAll, r.guards.userSessionRevoke, r.guards.restrictedSession, func(ginCtx *gin.Context) {
		rawID, ok := readUserIDParam(ginCtx, r.ctx.I18n)
		if !ok {
			return
		}

		if err := r.authSvc.RevokeAllUserSessions(ginCtx.Request.Context(), rawID); err != nil {
			r.runtime().writeAuthRouteError(ginCtx, "admin revoke user refresh sessions failed", err, zap.Uint64("userID", rawID))
			return
		}

		clearRefreshCookieWhen(ginCtx, r.authSvc, func(claims *pluginapi.AccessTokenClaims) bool {
			return claims.UserID == rawID
		})
		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func (r authRouteRegistrar) runtime() routeRuntime {
	return routeRuntime{
		localizer:  r.ctx.I18n,
		logger:     r.ctx.Logger,
		pluginName: r.pluginName,
	}
}

func (r userRouteRegistrar) runtime() routeRuntime {
	return routeRuntime{
		localizer:  r.ctx.I18n,
		logger:     r.ctx.Logger,
		pluginName: r.pluginName,
	}
}

func loadRestrictedPasswordChangeState(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	authSvc *authService,
) (bool, bool) {
	if authSvc == nil {
		abortLocalizedContractError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
		return false, false
	}

	restricted, err := authSvc.isRestrictedPasswordChangeSession(ginCtx.Request.Context())
	if err != nil {
		if errors.Is(err, pluginapi.ErrUnauthenticated) {
			abortLocalizedContractError(ginCtx, localizer, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
			return false, false
		}

		abortLocalizedContractError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
		return false, false
	}

	return restricted, true
}

func readUserIDParam(ginCtx *gin.Context, localizer *i18n.Service) (uint64, bool) {
	rawID, err := parseUserID(ginCtx.Param("id"))
	if err != nil {
		writeInvalidArgumentField(ginCtx, localizer, "id")
		return 0, false
	}

	return rawID, true
}

func readSessionIDParam(ginCtx *gin.Context, localizer *i18n.Service) (string, bool) {
	sessionID := strings.TrimSpace(ginCtx.Param("sessionID"))
	if sessionID == "" {
		writeInvalidArgumentField(ginCtx, localizer, "sessionID")
		return "", false
	}

	return sessionID, true
}

func clearRefreshCookieWhen(
	ginCtx *gin.Context,
	authSvc *authService,
	matches func(*pluginapi.AccessTokenClaims) bool,
) {
	if authSvc == nil || matches == nil {
		return
	}

	requestAuth, ok := pluginapi.RequestAuthContextFromContext(ginCtx.Request.Context())
	if !ok || requestAuth.Claims == nil || !matches(requestAuth.Claims) {
		return
	}

	authSvc.cookies.clearRefreshCookie(ginCtx)
}

func (r routeRuntime) writeAuthRouteError(ginCtx *gin.Context, message string, err error, fields ...zap.Field) {
	status, messageKey := mapAuthError(err)
	if status == http.StatusInternalServerError {
		logFields := append([]zap.Field{zap.String("plugin", r.pluginName), zap.Error(err)}, fields...)
		r.logger.Error(message, logFields...)
	}

	writeLocalizedContractError(ginCtx, r.localizer, status, messageKey, authErrorDetails(err))
}

func (r routeRuntime) writeUserLookupError(ginCtx *gin.Context, userID uint64, message string, err error) {
	status := http.StatusInternalServerError
	messageKey := messagecontract.CommonInternalError
	if errors.Is(err, store.ErrUserNotFound) {
		status = http.StatusNotFound
		messageKey = messagecontract.UserNotFound
	} else {
		r.logger.Error(message,
			zap.String("plugin", r.pluginName),
			zap.Uint64("userID", userID),
			zap.Error(err),
		)
	}

	writeLocalizedContractError(ginCtx, r.localizer, status, messageKey, nil)
}

func writeLocalizedContractError(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	status int,
	key messagecontract.Key,
	data map[string]any,
) {
	httpx.WriteLocalizedError(ginCtx, localizer, status, key.String(), data)
}

func writeInvalidArgumentField(ginCtx *gin.Context, localizer *i18n.Service, field string) {
	writeLocalizedContractError(ginCtx, localizer, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
		"field": field,
	})
}

func abortLocalizedContractError(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	status int,
	key messagecontract.Key,
	data map[string]any,
) {
	writeLocalizedContractError(ginCtx, localizer, status, key, data)
	ginCtx.Abort()
}
