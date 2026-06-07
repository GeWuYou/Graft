package dashboard

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"graft/server/internal/config"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/httpx"
	"graft/server/internal/i18n"
	"graft/server/internal/logger"
	"graft/server/internal/moduleapi"
)

const (
	routeGroup         = "/dashboard"
	routeWidgetIDParam = "widget_id"
)

// Registration contains core dependencies needed to expose dashboard aggregate routes.
type Registration struct {
	I18n                 *i18n.Service
	Config               *config.Config
	Registry             *Registry
	Logger               logger.AppLogger
	ModuleRuntimeSummary ModuleRuntimeSummaryProvider
}

// Register exposes authenticated dashboard aggregate routes.
func Register(
	registration Registration,
	router gin.IRouter,
	authService moduleapi.AuthService,
	authorizer moduleapi.Authorizer,
) error {
	if router == nil {
		return errors.New("dashboard router is unavailable")
	}
	if authService == nil {
		return errors.New("dashboard auth service is unavailable")
	}
	if registration.Registry == nil {
		return errors.New("dashboard registry is unavailable")
	}

	service := NewService(ServiceOptions{
		Config:               registration.Config,
		Registry:             registration.Registry,
		Authorizer:           authorizer,
		Logger:               registration.Logger,
		ModuleRuntimeSummary: registration.ModuleRuntimeSummary,
	})

	group := router.Group(routeGroup)
	group.Use(httpx.RequestIDMiddleware())
	group.GET("/summary", httpx.RequirePermission(registration.I18n, authService, authorizer, ""), func(ctx *gin.Context) {
		requestAuth := RequestAuthFromContext(ctx.Request.Context())
		httpx.WriteSuccess(ctx, http.StatusOK, service.Summary(ctx.Request.Context(), requestAuth))
	})
	group.GET("/widgets/:"+routeWidgetIDParam, httpx.RequirePermission(registration.I18n, authService, authorizer, ""), func(ctx *gin.Context) {
		requestAuth := RequestAuthFromContext(ctx.Request.Context())
		widgetID := strings.TrimSpace(ctx.Param(routeWidgetIDParam))
		widget, ok := service.Widget(ctx.Request.Context(), requestAuth, widgetID)
		if !ok {
			httpx.AbortLocalizedError(ctx, registration.I18n, http.StatusNotFound, "common.not_found", map[string]any{
				"field": routeWidgetIDParam,
			})
			return
		}
		httpx.WriteSuccess(ctx, http.StatusOK, widget)
	})

	return nil
}

// Compile-time guard for OpenAPI response DTO usage.
var (
	_ = generated.DashboardSummaryResponse{}
	_ = generated.DashboardWidget{}
)
