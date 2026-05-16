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
			Plugin:      pluginName,
		},
		{
			Code:        usercontract.UserSessionRevokePermission.String(),
			Name:        "Revoke User Sessions",
			Description: "Allows revoking refresh sessions for a specified user.",
			Plugin:      pluginName,
		},
		{
			Code:        usercontract.UserSessionReadPermission.String(),
			Name:        "Read User Sessions",
			Description: "Allows reading active refresh sessions for a specified user.",
			Plugin:      pluginName,
		},
	}
}

func (p *Plugin) registerServices(ctx *plugin.Context) (userService, *authService, bootstrapReader, error) {
	userSvc := userService{users: ctx.Stores.Users()}
	if err := ctx.Services.RegisterSingleton((*pluginapi.UserService)(nil), func(_ container.Resolver) (any, error) {
		return userSvc, nil
	}); err != nil {
		return userService{}, nil, bootstrapReader{}, err
	}

	authSvc, err := newAuthService(ctx.Config.Auth, ctx.Stores.Auth(), ctx.Stores.Users())
	if err != nil {
		return userService{}, nil, bootstrapReader{}, err
	}
	bootstrapSvc := newBootstrapReader(ctx.Config.I18n, ctx.I18n, ctx.MenuRegistry, ctx.Stores.Auth(), ctx.Stores.RBAC())
	p.defaultAdminAuth = authSvc

	if err := ctx.Services.RegisterSingleton((*pluginapi.AuthService)(nil), func(_ container.Resolver) (any, error) {
		return authSvc, nil
	}); err != nil {
		return userService{}, nil, bootstrapReader{}, err
	}

	return userSvc, authSvc, bootstrapSvc, nil
}

type routeGuards struct {
	authenticated          gin.HandlerFunc
	requiredPasswordChange gin.HandlerFunc
	restrictedSession      gin.HandlerFunc
	userRead               gin.HandlerFunc
	userSessionRead        gin.HandlerFunc
	userSessionRevoke      gin.HandlerFunc
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
		restrictedSession:      newRestrictedSessionGuard(localizer, authSvc),
		userRead:               httpx.RequirePermission(localizer, authSvc, authorizer, usercontract.UserReadPermission.String()),
		userSessionRead:        httpx.RequirePermission(localizer, authSvc, authorizer, usercontract.UserSessionReadPermission.String()),
		userSessionRevoke:      httpx.RequirePermission(localizer, authSvc, authorizer, usercontract.UserSessionRevokePermission.String()),
	}
}

var _ pluginapi.Authorizer = (*deferredAuthorizer)(nil)

func newRequiredPasswordChangeGuard(localizer *i18n.Service, authSvc *authService) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		if authSvc == nil {
			writeLocalizedContractError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
			ginCtx.Abort()
			return
		}

		restricted, err := authSvc.isRestrictedPasswordChangeSession(ginCtx.Request.Context())
		if err != nil {
			if errors.Is(err, pluginapi.ErrUnauthenticated) {
				writeLocalizedContractError(ginCtx, localizer, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
				ginCtx.Abort()
				return
			}

			writeLocalizedContractError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
			ginCtx.Abort()
			return
		}
		if !restricted {
			writeLocalizedContractError(ginCtx, localizer, http.StatusForbidden, messagecontract.AuthForbidden, nil)
			ginCtx.Abort()
			return
		}

		ginCtx.Next()
	}
}

func newRestrictedSessionGuard(localizer *i18n.Service, authSvc *authService) gin.HandlerFunc {
	allowedPaths := []string{
		usercontract.JoinRoute(usercontract.AuthGroup, usercontract.AuthBootstrap),
		usercontract.JoinRoute(usercontract.AuthGroup, usercontract.AuthCompleteRequiredPasswordChange),
	}

	return func(ginCtx *gin.Context) {
		if authSvc == nil {
			writeLocalizedContractError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
			ginCtx.Abort()
			return
		}

		restricted, err := authSvc.isRestrictedPasswordChangeSession(ginCtx.Request.Context())
		if err != nil {
			if errors.Is(err, pluginapi.ErrUnauthenticated) {
				writeLocalizedContractError(ginCtx, localizer, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
				ginCtx.Abort()
				return
			}

			writeLocalizedContractError(ginCtx, localizer, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
			ginCtx.Abort()
			return
		}
		if !restricted {
			ginCtx.Next()
			return
		}

		for _, allowedPath := range allowedPaths {
			if strings.HasSuffix(ginCtx.FullPath(), allowedPath) {
				ginCtx.Next()
				return
			}
		}

		writeLocalizedContractError(ginCtx, localizer, http.StatusForbidden, messagecontract.AuthForbidden, nil)
		ginCtx.Abort()
	}
}

func registerAuthRoutes(
	ctx *plugin.Context,
	pluginName string,
	authSvc *authService,
	bootstrapSvc bootstrapReader,
	guards routeGuards,
) error {
	authGroup := ctx.Router.Group(usercontract.AuthGroup)
	authGroup.Use(httpx.RequestIDMiddleware())
	registerLoginRoutes(authGroup, ctx, pluginName, authSvc)
	registerCurrentUserSessionRoutes(authGroup, ctx, pluginName, authSvc, guards)
	registerBootstrapAndPasswordRoutes(authGroup, ctx, pluginName, authSvc, bootstrapSvc, guards)

	return nil
}

func registerLoginRoutes(authGroup *gin.RouterGroup, ctx *plugin.Context, pluginName string, authSvc *authService) {
	authGroup.POST(usercontract.AuthLogin, func(ginCtx *gin.Context) {
		var request loginRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "body",
			})
			return
		}
		normalizedUsername := strings.TrimSpace(request.Username)
		if normalizedUsername == "" {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "username",
			})
			return
		}
		if request.Password == "" {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "password",
			})
			return
		}

		result, err := authSvc.LoginWithRefresh(ginCtx.Request.Context(), normalizedUsername, request.Password)
		if err != nil {
			writeAuthRouteError(ginCtx, ctx.I18n, ctx.Logger, pluginName, "login failed", err)
			return
		}

		authSvc.cookies.writeRefreshCookie(ginCtx, result.RefreshToken, result.RefreshExpiry)
		httpx.WriteSuccess(ginCtx, http.StatusOK, loginResponse{
			AccessToken:        result.AccessToken,
			ExpiresAt:          result.AccessExpiry,
			MustChangePassword: result.MustChangePassword,
			User:               result.User,
		})
	})
	authGroup.POST(usercontract.AuthRefresh, func(ginCtx *gin.Context) {
		refreshToken, err := authSvc.cookies.readRefreshCookie(ginCtx)
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
			return
		}

		result, err := authSvc.RefreshWithRotation(ginCtx.Request.Context(), refreshToken)
		if err != nil {
			writeAuthRouteError(ginCtx, ctx.I18n, ctx.Logger, pluginName, "refresh session failed", err)
			return
		}

		authSvc.cookies.writeRefreshCookie(ginCtx, result.RefreshToken, result.RefreshExpiry)
		httpx.WriteSuccess(ginCtx, http.StatusOK, loginResponse{
			AccessToken:        result.AccessToken,
			ExpiresAt:          result.AccessExpiry,
			MustChangePassword: result.MustChangePassword,
			User:               result.User,
		})
	})
	authGroup.POST(usercontract.AuthLogout, func(ginCtx *gin.Context) {
		refreshToken, err := authSvc.cookies.readRefreshCookie(ginCtx)
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
			return
		}

		if err := authSvc.LogoutCurrentSession(ginCtx.Request.Context(), refreshToken); err != nil {
			writeAuthRouteError(ginCtx, ctx.I18n, ctx.Logger, pluginName, "logout session failed", err)
			return
		}

		authSvc.cookies.clearRefreshCookie(ginCtx)
		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func registerCurrentUserSessionRoutes(
	authGroup *gin.RouterGroup,
	ctx *plugin.Context,
	pluginName string,
	authSvc *authService,
	guards routeGuards,
) {
	authGroup.POST(usercontract.AuthSessionsRevokeAll, guards.authenticated, guards.restrictedSession, func(ginCtx *gin.Context) {
		if err := authSvc.RevokeAllCurrentUserSessions(ginCtx.Request.Context()); err != nil {
			writeAuthRouteError(ginCtx, ctx.I18n, ctx.Logger, pluginName, "revoke all refresh sessions failed", err)
			return
		}

		authSvc.cookies.clearRefreshCookie(ginCtx)
		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
	authGroup.POST(usercontract.AuthSessionsRevokeOthers, guards.authenticated, guards.restrictedSession, func(ginCtx *gin.Context) {
		if err := authSvc.RevokeOtherCurrentUserSessions(ginCtx.Request.Context()); err != nil {
			writeAuthRouteError(ginCtx, ctx.I18n, ctx.Logger, pluginName, "revoke other user refresh sessions failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
	authGroup.GET(usercontract.AuthSessions, guards.authenticated, guards.restrictedSession, func(ginCtx *gin.Context) {
		listOptions, err := parseSessionListOptions(ginCtx.Query("limit"))
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "limit",
			})
			return
		}

		sessions, err := authSvc.ListCurrentUserSessions(ginCtx.Request.Context(), listOptions)
		if err != nil {
			writeAuthRouteError(ginCtx, ctx.I18n, ctx.Logger, pluginName, "list current user refresh sessions failed", err)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, sessions)
	})
	authGroup.POST(usercontract.AuthSessionRevoke, guards.authenticated, guards.restrictedSession, func(ginCtx *gin.Context) {
		sessionID := strings.TrimSpace(ginCtx.Param("sessionID"))
		if sessionID == "" {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "sessionID",
			})
			return
		}

		if err := authSvc.RevokeCurrentUserSession(ginCtx.Request.Context(), sessionID); err != nil {
			writeAuthRouteErrorWithFields(
				ginCtx,
				ctx.I18n,
				ctx.Logger,
				pluginName,
				"revoke current user refresh session failed",
				err,
				zap.String("sessionID", sessionID),
			)
			return
		}

		if requestAuth, ok := pluginapi.RequestAuthContextFromContext(ginCtx.Request.Context()); ok &&
			requestAuth.Claims != nil &&
			requestAuth.Claims.SessionID == sessionID {
			authSvc.cookies.clearRefreshCookie(ginCtx)
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func registerBootstrapAndPasswordRoutes(
	authGroup *gin.RouterGroup,
	ctx *plugin.Context,
	pluginName string,
	authSvc *authService,
	bootstrapSvc bootstrapReader,
	guards routeGuards,
) {
	authGroup.GET(usercontract.AuthBootstrap, guards.authenticated, guards.restrictedSession, func(ginCtx *gin.Context) {
		payload, err := bootstrapSvc.Read(ginCtx.Request.Context(), ginCtx.Request)
		if err != nil {
			if errors.Is(err, pluginapi.ErrUnauthenticated) {
				writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusUnauthorized, messagecontract.AuthTokenMissing, nil)
				return
			}

			ctx.Logger.Error("read bootstrap payload failed",
				zap.String("plugin", pluginName),
				zap.Error(err),
			)
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	})
	authGroup.POST(usercontract.AuthChangePassword, guards.authenticated, guards.restrictedSession, func(ginCtx *gin.Context) {
		var request changePasswordRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "body",
			})
			return
		}
		if strings.TrimSpace(request.CurrentPassword) == "" {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "current_password",
			})
			return
		}
		if strings.TrimSpace(request.NewPassword) == "" {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "new_password",
			})
			return
		}

		if err := authSvc.ChangeCurrentUserPassword(ginCtx.Request.Context(), request.CurrentPassword, request.NewPassword); err != nil {
			writeAuthRouteError(ginCtx, ctx.I18n, ctx.Logger, pluginName, "change current user password failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
	authGroup.POST(usercontract.AuthCompleteRequiredPasswordChange, guards.authenticated, guards.requiredPasswordChange, func(ginCtx *gin.Context) {
		var request completeRequiredPasswordChangeRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "body",
			})
			return
		}
		if strings.TrimSpace(request.NewPassword) == "" {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "new_password",
			})
			return
		}

		if err := authSvc.CompleteRequiredPasswordChange(ginCtx.Request.Context(), request.NewPassword); err != nil {
			writeAuthRouteError(ginCtx, ctx.I18n, ctx.Logger, pluginName, "complete required password change failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func registerUserRoutes(
	ctx *plugin.Context,
	pluginName string,
	userSvc userService,
	authSvc *authService,
	guards routeGuards,
) error {
	group := ctx.Router.Group(usercontract.UsersGroup)
	group.Use(httpx.RequestIDMiddleware())
	registerUserReadRoutes(group, ctx, pluginName, userSvc, guards)
	registerAdminSessionRoutes(group, ctx, pluginName, userSvc, authSvc, guards)

	return nil
}

func registerUserReadRoutes(
	group *gin.RouterGroup,
	ctx *plugin.Context,
	pluginName string,
	userSvc userService,
	guards routeGuards,
) {
	group.GET(usercontract.UserCollection, guards.userRead, guards.restrictedSession, func(ginCtx *gin.Context) {
		users, err := userSvc.ListUsers(ginCtx.Request.Context())
		if err != nil {
			ctx.Logger.Error("list users failed",
				zap.String("plugin", pluginName),
				zap.Error(err),
			)
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusInternalServerError, messagecontract.CommonInternalError, nil)
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
	group.GET(usercontract.UserByID, guards.userRead, guards.restrictedSession, func(ginCtx *gin.Context) {
		rawID, err := parseUserID(ginCtx.Param("id"))
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "id",
			})
			return
		}

		summary, err := userSvc.GetUserByID(ginCtx.Request.Context(), rawID)
		if err != nil {
			writeUserLookupError(ginCtx, ctx.I18n, ctx.Logger, pluginName, rawID, "get user by id failed", err)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, summary)
	})
}

func registerAdminSessionRoutes(
	group *gin.RouterGroup,
	ctx *plugin.Context,
	pluginName string,
	userSvc userService,
	authSvc *authService,
	guards routeGuards,
) {
	registerAdminSessionReadRoute(group, ctx, pluginName, userSvc, authSvc, guards)
	registerAdminSessionRevokeRoutes(group, ctx, pluginName, userSvc, authSvc, guards)
}

func registerAdminSessionReadRoute(
	group *gin.RouterGroup,
	ctx *plugin.Context,
	pluginName string,
	userSvc userService,
	authSvc *authService,
	guards routeGuards,
) {
	group.GET(usercontract.UserSessions, guards.userSessionRead, guards.restrictedSession, func(ginCtx *gin.Context) {
		rawID, err := parseUserID(ginCtx.Param("id"))
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "id",
			})
			return
		}

		summary, err := userSvc.GetUserByID(ginCtx.Request.Context(), rawID)
		if err != nil {
			writeUserLookupError(ginCtx, ctx.I18n, ctx.Logger, pluginName, rawID, "get user by id before listing sessions failed", err)
			return
		}

		listOptions, err := parseSessionListOptions(ginCtx.Query("limit"))
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "limit",
			})
			return
		}

		sessions, err := authSvc.ListUserSessions(ginCtx.Request.Context(), summary.ID, listOptions)
		if err != nil {
			writeAuthRouteErrorWithFields(
				ginCtx,
				ctx.I18n,
				ctx.Logger,
				pluginName,
				"list user refresh sessions failed",
				err,
				zap.Uint64("userID", rawID),
			)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, sessions)
	})
}

func registerAdminSessionRevokeRoutes(
	group *gin.RouterGroup,
	ctx *plugin.Context,
	pluginName string,
	userSvc userService,
	authSvc *authService,
	guards routeGuards,
) {
	registerAdminRevokeSingleSessionRoute(group, ctx, pluginName, userSvc, authSvc, guards)
	registerAdminRevokeAllSessionsRoute(group, ctx, pluginName, authSvc, guards)
}

func registerAdminRevokeSingleSessionRoute(
	group *gin.RouterGroup,
	ctx *plugin.Context,
	pluginName string,
	userSvc userService,
	authSvc *authService,
	guards routeGuards,
) {
	group.POST(usercontract.UserSessionByIDRevoke, guards.userSessionRevoke, guards.restrictedSession, func(ginCtx *gin.Context) {
		rawID, err := parseUserID(ginCtx.Param("id"))
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "id",
			})
			return
		}

		sessionID := strings.TrimSpace(ginCtx.Param("sessionID"))
		if sessionID == "" {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "sessionID",
			})
			return
		}

		summary, err := userSvc.GetUserByID(ginCtx.Request.Context(), rawID)
		if err != nil {
			writeUserLookupError(ginCtx, ctx.I18n, ctx.Logger, pluginName, rawID, "get user by id before revoking session failed", err)
			return
		}

		if err := authSvc.RevokeUserSession(ginCtx.Request.Context(), summary.ID, sessionID); err != nil {
			writeAuthRouteErrorWithFields(
				ginCtx,
				ctx.I18n,
				ctx.Logger,
				pluginName,
				"admin revoke user refresh session failed",
				err,
				zap.Uint64("userID", rawID),
				zap.String("sessionID", sessionID),
			)
			return
		}

		if requestAuth, ok := pluginapi.RequestAuthContextFromContext(ginCtx.Request.Context()); ok &&
			requestAuth.Claims != nil &&
			requestAuth.Claims.UserID == rawID &&
			requestAuth.Claims.SessionID == sessionID {
			authSvc.cookies.clearRefreshCookie(ginCtx)
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func registerAdminRevokeAllSessionsRoute(
	group *gin.RouterGroup,
	ctx *plugin.Context,
	pluginName string,
	authSvc *authService,
	guards routeGuards,
) {
	group.POST(usercontract.UserSessionsRevokeAll, guards.userSessionRevoke, guards.restrictedSession, func(ginCtx *gin.Context) {
		rawID, err := parseUserID(ginCtx.Param("id"))
		if err != nil {
			writeLocalizedContractError(ginCtx, ctx.I18n, http.StatusBadRequest, messagecontract.CommonInvalidArgument, map[string]any{
				"field": "id",
			})
			return
		}

		if err := authSvc.RevokeAllUserSessions(ginCtx.Request.Context(), rawID); err != nil {
			writeAuthRouteErrorWithFields(
				ginCtx,
				ctx.I18n,
				ctx.Logger,
				pluginName,
				"admin revoke user refresh sessions failed",
				err,
				zap.Uint64("userID", rawID),
			)
			return
		}

		if requestAuth, ok := pluginapi.RequestAuthContextFromContext(ginCtx.Request.Context()); ok &&
			requestAuth.Claims != nil &&
			requestAuth.Claims.UserID == rawID {
			authSvc.cookies.clearRefreshCookie(ginCtx)
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
}

func writeAuthRouteError(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	logger *zap.Logger,
	pluginName string,
	message string,
	err error,
) {
	writeAuthRouteErrorWithFields(ginCtx, localizer, logger, pluginName, message, err)
}

func writeAuthRouteErrorWithFields(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	logger *zap.Logger,
	pluginName string,
	message string,
	err error,
	fields ...zap.Field,
) {
	status, messageKey := mapAuthError(err)
	if status == http.StatusInternalServerError {
		logFields := append([]zap.Field{zap.String("plugin", pluginName), zap.Error(err)}, fields...)
		logger.Error(message, logFields...)
	}

	writeLocalizedContractError(ginCtx, localizer, status, messageKey, authErrorDetails(err))
}

func writeUserLookupError(
	ginCtx *gin.Context,
	localizer *i18n.Service,
	logger *zap.Logger,
	pluginName string,
	userID uint64,
	message string,
	err error,
) {
	status := http.StatusInternalServerError
	messageKey := messagecontract.CommonInternalError
	if errors.Is(err, store.ErrUserNotFound) {
		status = http.StatusNotFound
		messageKey = messagecontract.UserNotFound
	} else {
		logger.Error(message,
			zap.String("plugin", pluginName),
			zap.Uint64("userID", userID),
			zap.Error(err),
		)
	}

	writeLocalizedContractError(ginCtx, localizer, status, messageKey, nil)
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
