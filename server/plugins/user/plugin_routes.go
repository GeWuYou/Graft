package user

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"graft/server/internal/container"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/menu"
	"graft/server/internal/permission"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	"graft/server/internal/store"
)

func registerUserPermissions(registry *permission.Registry, pluginName string) {
	registry.Register(permission.Item{
		Code:        "user.read",
		Name:        "Read Users",
		Description: "Allows reading user management data.",
		Plugin:      pluginName,
	})
	registry.Register(permission.Item{
		Code:        "user.session.revoke",
		Name:        "Revoke User Sessions",
		Description: "Allows revoking refresh sessions for a specified user.",
		Plugin:      pluginName,
	})
	registry.Register(permission.Item{
		Code:        "user.session.read",
		Name:        "Read User Sessions",
		Description: "Allows reading active refresh sessions for a specified user.",
		Plugin:      pluginName,
	})
}

func registerUserMenu(registry *menu.Registry, pluginName string) {
	registry.Register(menu.Item{
		Code:       "user.list",
		Title:      "用户管理",
		Path:       "/users",
		Icon:       "usergroup",
		Permission: "user.read",
		Plugin:     pluginName,
	})
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

func registerAuthRoutes(ctx *plugin.Context, pluginName string, authSvc *authService, bootstrapSvc bootstrapReader) error {
	authGroup := ctx.Router.Group("/auth")
	authGroup.Use(httpx.RequestIDMiddleware())
	registerLoginRoutes(authGroup, ctx, pluginName, authSvc)
	registerCurrentUserSessionRoutes(authGroup, ctx, pluginName, authSvc)
	registerBootstrapAndPasswordRoutes(authGroup, ctx, pluginName, authSvc, bootstrapSvc)

	return nil
}

func registerLoginRoutes(authGroup *gin.RouterGroup, ctx *plugin.Context, pluginName string, authSvc *authService) {
	authGroup.POST("/login", func(ginCtx *gin.Context) {
		var request loginRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "body",
			})
			return
		}
		normalizedUsername := strings.TrimSpace(request.Username)
		if normalizedUsername == "" {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "username",
			})
			return
		}
		if request.Password == "" {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
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
	authGroup.POST("/refresh", func(ginCtx *gin.Context) {
		refreshToken, err := authSvc.cookies.readRefreshCookie(ginCtx)
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusUnauthorized, "auth.token_missing", nil)
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
	authGroup.POST("/logout", func(ginCtx *gin.Context) {
		refreshToken, err := authSvc.cookies.readRefreshCookie(ginCtx)
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusUnauthorized, "auth.token_missing", nil)
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

func registerCurrentUserSessionRoutes(authGroup *gin.RouterGroup, ctx *plugin.Context, pluginName string, authSvc *authService) {
	authGroup.POST("/sessions/revoke-all", httpx.RequirePermission(ctx.I18n, ctx.Services, ""), func(ginCtx *gin.Context) {
		if err := authSvc.RevokeAllCurrentUserSessions(ginCtx.Request.Context()); err != nil {
			writeAuthRouteError(ginCtx, ctx.I18n, ctx.Logger, pluginName, "revoke all refresh sessions failed", err)
			return
		}

		authSvc.cookies.clearRefreshCookie(ginCtx)
		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
	authGroup.POST("/sessions/revoke-others", httpx.RequirePermission(ctx.I18n, ctx.Services, ""), func(ginCtx *gin.Context) {
		if err := authSvc.RevokeOtherCurrentUserSessions(ginCtx.Request.Context()); err != nil {
			writeAuthRouteError(ginCtx, ctx.I18n, ctx.Logger, pluginName, "revoke other user refresh sessions failed", err)
			return
		}

		httpx.WriteSuccess[any](ginCtx, http.StatusOK, nil)
	})
	authGroup.GET("/sessions", httpx.RequirePermission(ctx.I18n, ctx.Services, ""), func(ginCtx *gin.Context) {
		listOptions, err := parseSessionListOptions(ginCtx.Query("limit"))
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
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
	authGroup.POST("/sessions/:sessionID/revoke", httpx.RequirePermission(ctx.I18n, ctx.Services, ""), func(ginCtx *gin.Context) {
		sessionID := strings.TrimSpace(ginCtx.Param("sessionID"))
		if sessionID == "" {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
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
) {
	authGroup.GET("/bootstrap", httpx.RequirePermission(ctx.I18n, ctx.Services, ""), func(ginCtx *gin.Context) {
		payload, err := bootstrapSvc.Read(ginCtx.Request.Context(), ginCtx.Request)
		if err != nil {
			if errors.Is(err, pluginapi.ErrUnauthenticated) {
				httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusUnauthorized, "auth.token_missing", nil)
				return
			}

			ctx.Logger.Error("read bootstrap payload failed",
				zap.String("plugin", pluginName),
				zap.Error(err),
			)
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, "common.internal_error", nil)
			return
		}

		httpx.WriteSuccess(ginCtx, http.StatusOK, payload)
	})
	authGroup.POST("/change-password", httpx.RequirePermission(ctx.I18n, ctx.Services, ""), func(ginCtx *gin.Context) {
		var request changePasswordRequest
		if err := ginCtx.ShouldBindJSON(&request); err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "body",
			})
			return
		}
		if strings.TrimSpace(request.CurrentPassword) == "" {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "current_password",
			})
			return
		}
		if strings.TrimSpace(request.NewPassword) == "" {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
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
}

func registerUserRoutes(ctx *plugin.Context, pluginName string, userSvc userService, authSvc *authService) error {
	group := ctx.Router.Group("/users")
	group.Use(httpx.RequestIDMiddleware())
	registerUserReadRoutes(group, ctx, pluginName, userSvc)
	registerAdminSessionRoutes(group, ctx, pluginName, userSvc, authSvc)

	return nil
}

func registerUserReadRoutes(group *gin.RouterGroup, ctx *plugin.Context, pluginName string, userSvc userService) {
	group.GET("", httpx.RequirePermission(ctx.I18n, ctx.Services, "user.read"), func(ginCtx *gin.Context) {
		users, err := ctx.Stores.Users().List(ginCtx.Request.Context())
		if err != nil {
			ctx.Logger.Error("list users failed",
				zap.String("plugin", pluginName),
				zap.Error(err),
			)
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusInternalServerError, "common.internal_error", nil)
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
	group.GET("/:id", httpx.RequirePermission(ctx.I18n, ctx.Services, "user.read"), func(ginCtx *gin.Context) {
		rawID, err := parseUserID(ginCtx.Param("id"))
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
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
) {
	registerAdminSessionReadRoute(group, ctx, pluginName, userSvc, authSvc)
	registerAdminSessionRevokeRoutes(group, ctx, pluginName, userSvc, authSvc)
}

func registerAdminSessionReadRoute(
	group *gin.RouterGroup,
	ctx *plugin.Context,
	pluginName string,
	userSvc userService,
	authSvc *authService,
) {
	group.GET("/:id/sessions", httpx.RequirePermission(ctx.I18n, ctx.Services, "user.session.read"), func(ginCtx *gin.Context) {
		rawID, err := parseUserID(ginCtx.Param("id"))
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
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
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
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
) {
	registerAdminRevokeSingleSessionRoute(group, ctx, pluginName, userSvc, authSvc)
	registerAdminRevokeAllSessionsRoute(group, ctx, pluginName, authSvc)
}

func registerAdminRevokeSingleSessionRoute(
	group *gin.RouterGroup,
	ctx *plugin.Context,
	pluginName string,
	userSvc userService,
	authSvc *authService,
) {
	group.POST("/:id/sessions/:sessionID/revoke", httpx.RequirePermission(ctx.I18n, ctx.Services, "user.session.revoke"), func(ginCtx *gin.Context) {
		rawID, err := parseUserID(ginCtx.Param("id"))
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
				"field": "id",
			})
			return
		}

		sessionID := strings.TrimSpace(ginCtx.Param("sessionID"))
		if sessionID == "" {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
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
) {
	group.POST("/:id/sessions/revoke-all", httpx.RequirePermission(ctx.I18n, ctx.Services, "user.session.revoke"), func(ginCtx *gin.Context) {
		rawID, err := parseUserID(ginCtx.Param("id"))
		if err != nil {
			httpx.WriteLocalizedError(ginCtx, ctx.I18n, http.StatusBadRequest, "common.invalid_argument", map[string]any{
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

	httpx.WriteLocalizedError(ginCtx, localizer, status, messageKey, nil)
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
	messageKey := "common.internal_error"
	if errors.Is(err, store.ErrUserNotFound) {
		status = http.StatusNotFound
		messageKey = "user.not_found"
	} else {
		logger.Error(message,
			zap.String("plugin", pluginName),
			zap.Uint64("userID", userID),
			zap.Error(err),
		)
	}

	httpx.WriteLocalizedError(ginCtx, localizer, status, messageKey, nil)
}
