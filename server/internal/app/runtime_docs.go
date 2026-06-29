package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"graft/server/internal/container"
	healthopenapi "graft/server/internal/contract/openapi/health"
	"graft/server/internal/logger"
	"graft/server/internal/realtime"
)

func (r *Runtime) loadOptionalDocsAssets() error {
	if r.config == nil || !r.config.Docs.Enabled {
		return nil
	}

	docsAssets, err := loadOpenAPIDocsAssets()
	if err != nil {
		return fmt.Errorf("load openapi docs assets: %w", err)
	}

	r.openapiDocs = docsAssets
	return nil
}

func (r *Runtime) registerCoreRoutes(engine *gin.Engine) error {
	if engine == nil {
		return nil
	}

	if err := r.registerRealtimeGatewayRoute(engine); err != nil {
		return err
	}
	r.registerHealthRoute(engine)
	r.registerOpenAPIRoutes(engine)
	return nil
}

func (r *Runtime) registerRealtimeGatewayRoute(engine *gin.Engine) error {
	ticketService, err := r.injectedRealtimeTicketService()
	if errors.Is(err, container.ErrServiceNotRegistered) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve realtime websocket gateway ticket service: %w", err)
	}
	if ticketService == nil {
		return nil
	}

	if err := realtime.RegisterWebSocketGateway(engine, realtime.GatewayRegistration{
		Hub:                   r.realtimeHub,
		I18n:                  r.i18n,
		Tickets:               ticketService,
		WebSocketAllowOrigins: append([]string(nil), r.config.HTTPX.WebSocketAllowedOrigins...),
	}); err != nil {
		return fmt.Errorf("register realtime websocket gateway: %w", err)
	}
	return nil
}

func (r *Runtime) registerHealthRoute(engine *gin.Engine) {
	engine.GET("/healthz", func(ctx *gin.Context) {
		coreHealthGeneratedHandler{}.GetHealthz()
		ctx.JSON(http.StatusOK, gin.H{
			"status":         "ok",
			"defaultLocale":  r.i18n.DefaultLocale(),
			"fallbackLocale": r.i18n.FallbackLocale(),
			"menus":          len(r.menuRegistry.Items()),
			"permissions":    len(r.permissionRegistry.Items()),
			"jobs":           len(r.cronRegistry.Items()),
		})
	})
}

func (r *Runtime) registerOpenAPIRoutes(engine *gin.Engine) {
	if r.config == nil || !r.config.Docs.Enabled || r.openapiDocs == nil {
		return
	}

	engine.GET(openapiJSONPath, r.handleOpenAPIJSON)
	engine.GET(openapiYAMLPath, r.handleOpenAPIYAML)
	engine.GET(openapiDocsPath, r.handleOpenAPIDocs)
}

func (r *Runtime) handleOpenAPIJSON(ctx *gin.Context) {
	ctx.Data(http.StatusOK, "application/json; charset=utf-8", r.openapiDocs.json)
}

func (r *Runtime) handleOpenAPIYAML(ctx *gin.Context) {
	yamlSpec, err := buildLegacyOpenAPIYAML(r.openapiDocs.json)
	if err != nil {
		if r.logger != nil {
			r.appLogger().Error(ctx.Request.Context(), "build legacy openapi yaml", logger.ErrorField(err))
		}
		ctx.String(http.StatusInternalServerError, "failed to render openapi yaml")
		return
	}
	ctx.Data(http.StatusOK, "application/yaml; charset=utf-8", yamlSpec)
}

func (r *Runtime) handleOpenAPIDocs(ctx *gin.Context) {
	html, err := renderScalarDocsHTML(openapiJSONPath)
	if err != nil {
		if r.logger != nil {
			r.appLogger().Error(ctx.Request.Context(), "render docs page", logger.ErrorField(err))
		}
		ctx.String(http.StatusInternalServerError, "failed to render docs page")
		return
	}
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", html)
}

var _ healthopenapi.ServerInterface = coreHealthGeneratedHandler{}

type coreHealthGeneratedHandler struct{}

func (h coreHealthGeneratedHandler) GetHealthz() {
	_ = h
}

func buildLegacyOpenAPIYAML(spec []byte) ([]byte, error) {
	if len(spec) == 0 {
		return nil, fmt.Errorf("generated bundled openapi spec is empty")
	}

	var document any
	if err := json.Unmarshal(spec, &document); err != nil {
		return nil, fmt.Errorf("decode generated bundled openapi json: %w", err)
	}

	yamlSpec, err := yaml.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode generated bundled openapi yaml: %w", err)
	}
	return yamlSpec, nil
}
