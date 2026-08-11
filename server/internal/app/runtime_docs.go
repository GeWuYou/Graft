package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"graft/server/internal/container"
	capcontract "graft/server/internal/contract/capability"
	healthopenapi "graft/server/internal/contract/openapi/health"
	"graft/server/internal/logger"
	"graft/server/internal/permission"
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
	mcpDocs, err := buildMCPDocsCatalog(OpenAPIDocsBundle(), r.config.MCP.Enabled)
	if err != nil {
		return fmt.Errorf("build MCP docs catalog: %w", err)
	}
	r.mcpDocs = mcpDocs
	return nil
}

func (r *Runtime) registerCoreRoutes(engine *gin.Engine) error {
	if engine == nil {
		return nil
	}

	if err := r.registerRealtimeGatewayRoute(engine); err != nil {
		return err
	}
	r.permissionRegistry.Register(permission.Item{
		Code:           capcontract.ReadPermission,
		DisplayKey:     "rbac.permissionCatalog.platformCapabilitiesRead.display",
		DescriptionKey: "rbac.permissionCatalog.platformCapabilitiesRead.description",
		Module:         "core",
		Resource:       "platform-capabilities",
		Action:         "read",
		RiskLevel:      permission.RiskLevelLow,
		RiskCategory:   permission.RiskCategoryRead,
	})
	r.registerHealthRoute(engine)
	r.registerOpenAPIRoutes(engine)
	r.registerMCPDocsRoutes(engine)
	return nil
}

func (r *Runtime) registerMCPDocsRoutes(engine *gin.Engine) {
	if r.config == nil || !r.config.Docs.Enabled || len(r.mcpDocs) == 0 {
		return
	}

	engine.GET(mcpDocsJSONPath, r.handleMCPDocsJSON)
	engine.GET(mcpDocsPath, r.handleMCPDocs)
}

func (r *Runtime) registerRealtimeGatewayRoute(engine *gin.Engine) error {
	ticketService, err := r.injectedRealtimeTicketService()
	if errors.Is(err, container.ErrServiceNotRegistered) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve realtime gateway ticket service: %w", err)
	}
	if ticketService == nil {
		return nil
	}

	registration := realtime.GatewayRegistration{
		Hub:                   r.realtimeHub,
		I18n:                  r.i18n,
		Tickets:               ticketService,
		WebSocketAllowOrigins: append([]string(nil), r.config.HTTPX.WebSocketAllowedOrigins...),
		Logger:                r.logger,
	}
	r.logRealtimeGatewayConfiguration()
	if err := realtime.RegisterWebSocketGateway(engine, registration); err != nil {
		return fmt.Errorf("register realtime websocket gateway: %w", err)
	}
	if err := realtime.RegisterSSEGateway(engine, registration); err != nil {
		return fmt.Errorf("register realtime SSE gateway: %w", err)
	}
	return nil
}

// logRealtimeGatewayConfiguration 暴露运行时实际选择的 dotenv 路径及白名单数量，避免将来源漂移误判为票据故障。
func (r *Runtime) logRealtimeGatewayConfiguration() {
	if r == nil || r.config == nil {
		return
	}
	dotenvPath := r.config.DotenvPath
	r.appLogger().Info(context.Background(), "realtime gateway configuration resolved",
		logger.StringField(logger.FieldOperation, "realtime_gateway_config"),
		logger.StringField("dotenvPath", dotenvPath),
		logger.IntField("websocketAllowedOriginCount", len(r.config.HTTPX.WebSocketAllowedOrigins)),
	)
	normalizedDotenvPath := strings.TrimPrefix(strings.ReplaceAll(dotenvPath, "\\", "/"), "./")
	if strings.HasPrefix(normalizedDotenvPath, ".data/docker-builder-agent-dev/") ||
		strings.Contains(normalizedDotenvPath, "/.data/docker-builder-agent-dev/") {
		r.appLogger().Warn(context.Background(), "realtime gateway is using Docker Builder Agent environment",
			logger.StringField(logger.FieldOperation, "realtime_gateway_config"),
			logger.StringField("dotenvPath", dotenvPath),
		)
	}
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
	html, err := renderScalarDocsHTML(openapiJSONPath, r.openapiDocs.summary)
	if err != nil {
		if r.logger != nil {
			r.appLogger().Error(ctx.Request.Context(), "render docs page", logger.ErrorField(err))
		}
		ctx.String(http.StatusInternalServerError, "failed to render docs page")
		return
	}
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", html)
}

func (r *Runtime) handleMCPDocsJSON(ctx *gin.Context) {
	ctx.Data(http.StatusOK, "application/json; charset=utf-8", r.mcpDocs)
}

func (r *Runtime) handleMCPDocs(ctx *gin.Context) {
	ctx.Data(http.StatusOK, "text/html; charset=utf-8", renderMCPDocsHTML())
}

var _ healthopenapi.ServerInterface = coreHealthGeneratedHandler{}

type coreHealthGeneratedHandler struct{}

func (h coreHealthGeneratedHandler) GetHealthz() {
	_ = h
}

// buildLegacyOpenAPIYAML 将 OpenAPI JSON 规范转换为 YAML。
// 输入为空时返回错误；当 JSON 解析或 YAML 编码失败时返回带阶段信息的错误。
// 参数：spec 是 OpenAPI 规范的 JSON 内容。
// 返回值：转换后的 YAML 字节和错误。
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
