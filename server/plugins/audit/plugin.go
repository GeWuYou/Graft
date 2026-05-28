package audit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	auditcore "graft/server/internal/audit"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/plugin"
	"graft/server/internal/pluginapi"
	auditstore "graft/server/plugins/audit/store"
)

// Plugin 是当前 MVP 阶段的最小审计插件。
//
// 该插件在 Register 阶段挂载请求级自动审计中间件、受权只读查询路由、
// 菜单/权限声明，并订阅主动审计事件；当前不承载归档和分析逻辑。
type Plugin struct {
	recorder *auditcore.Service
}

const eventMetadataExtraFields = 2

// NewPlugin 创建最小审计插件。
func NewPlugin(repo auditstore.AuditRepository) (*Plugin, error) {
	recorder, err := auditcore.NewService(repo)
	if err != nil {
		return nil, err
	}

	return &Plugin{recorder: recorder}, nil
}

// Name 返回插件稳定标识。
func (p *Plugin) Name() string {
	return pluginID
}

// Version 返回当前插件版本。
func (p *Plugin) Version() string {
	return pluginVersion
}

// DependsOn 返回当前插件依赖列表。
func (p *Plugin) DependsOn() []string {
	return append([]string(nil), pluginDependencies...)
}

// Register 挂载 HTTP 自动审计、受权查询路由与 event bus 主动审计接线。
func (p *Plugin) Register(ctx *plugin.Context) error {
	if p.recorder == nil {
		return errors.New("audit recorder is unavailable")
	}
	if err := registerAuditMessages(ctx.I18n); err != nil {
		return err
	}
	registerAuditPermissions(ctx.PermissionRegistry, p.Name())
	registerAuditMenu(ctx.MenuRegistry, p.Name())
	if err := registerAuditService(ctx, p.recorder); err != nil {
		return err
	}

	logger := ctx.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	if ctx.Router != nil {
		ctx.Router.Use(requestAuditMiddleware(logger, p.recorder))
		guard, err := p.resolveRouteGuard(ctx)
		if err != nil {
			return err
		}
		registerAuditRoutes(ctx, p.Name(), p.recorder, guard)
	}
	if ctx.EventBus == nil {
		return errors.New("event bus is unavailable")
	}

	return ctx.EventBus.Subscribe(pluginapi.AuditRecordEventName, func(eventCtx context.Context, event eventbus.Event) error {
		payload, err := resolveAuditEventPayload(event.Payload)
		if err != nil {
			logger.Error("drop malformed audit event payload",
				zap.String("plugin", pluginID),
				zap.String("event", pluginapi.AuditRecordEventName),
				zap.Error(fmt.Errorf("unexpected audit event payload type %T", event.Payload)),
			)
			return nil
		}

		if err := recordEvent(eventCtx, p.recorder, payload); err != nil {
			logger.Error("write active audit log failed",
				zap.String("plugin", pluginID),
				zap.String("event", pluginapi.AuditRecordEventName),
				zap.String("action", strings.TrimSpace(payload.Action)),
				zap.Error(err),
			)
		}

		return nil
	})
}

// Boot 当前没有额外运行时行为需要启动。
func (p *Plugin) Boot(_ *plugin.Context) error {
	return nil
}

// Shutdown 当前没有额外资源需要释放。
func (p *Plugin) Shutdown(_ *plugin.Context) error {
	return nil
}

func requestAuditMiddleware(logger *zap.Logger, recorder *auditcore.Service) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(ctx *gin.Context) {
		ctx.Next()

		input := auditcore.RecordInput{
			Action:       buildAction(ctx),
			ResourceType: currentResourceType(ctx),
			ResourceID:   currentResourceID(ctx),
			ResourceName: currentResourceName(ctx),
			RequestID:    httpx.EnsureRequestID(ctx),
			IP:           ctx.ClientIP(),
			UserAgent:    ctx.Request.UserAgent(),
			Success:      ctx.Writer.Status() < http.StatusBadRequest,
			Message:      currentAuditMessage(ctx),
			Metadata: map[string]any{
				"request_method": ctx.Request.Method,
				"request_path":   currentRoutePath(ctx),
				"status_code":    ctx.Writer.Status(),
			},
		}

		if requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx.Request.Context()); ok && requestAuth.User != nil {
			input.ActorUserID = &requestAuth.User.ID
			input.ActorUsername = strings.TrimSpace(requestAuth.User.Username)
			input.ActorDisplayName = strings.TrimSpace(requestAuth.User.DisplayName)
		}

		if _, err := recorder.Record(ctx.Request.Context(), input); err != nil {
			logger.Error("write request audit log failed",
				zap.String("plugin", "audit"),
				zap.String("action", input.Action),
				zap.Error(err),
			)
		}
	}
}

func recordEvent(ctx context.Context, recorder *auditcore.Service, payload pluginapi.AuditEvent) error {
	input := auditcore.RecordInput{
		Action:       strings.TrimSpace(payload.Action),
		ResourceType: strings.TrimSpace(payload.ResourceType),
		ResourceID:   strings.TrimSpace(payload.ResourceID),
		ResourceName: strings.TrimSpace(payload.ResourceName),
		RequestID:    strings.TrimSpace(payload.RequestID),
		IP:           strings.TrimSpace(payload.IP),
		UserAgent:    strings.TrimSpace(payload.UserAgent),
		Success:      payload.Success,
		Message:      strings.TrimSpace(payload.Message),
		Metadata:     eventMetadata(payload),
		CreatedAt:    payload.CreatedAt,
	}
	if payload.Operator != nil {
		input.ActorUserID = &payload.Operator.ID
		input.ActorUsername = strings.TrimSpace(payload.Operator.Username)
		input.ActorDisplayName = strings.TrimSpace(payload.Operator.DisplayName)
	}

	_, err := recorder.Record(ctx, input)
	return err
}

func buildAction(ctx *gin.Context) string {
	return strings.TrimSpace(ctx.Request.Method + " " + currentRoutePath(ctx))
}

func currentRoutePath(ctx *gin.Context) string {
	if ctx == nil || ctx.Request == nil {
		return ""
	}
	if route := strings.TrimSpace(ctx.FullPath()); route != "" {
		return route
	}

	return strings.TrimSpace(ctx.Request.URL.Path)
}

func currentResourceType(ctx *gin.Context) string {
	route := strings.TrimSpace(currentRoutePath(ctx))
	if route == "" {
		return ""
	}

	segments := strings.Split(strings.Trim(route, "/"), "/")
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" || strings.HasPrefix(segment, ":") || segment == "api" {
			continue
		}

		return segment
	}

	return ""
}

func currentResourceID(ctx *gin.Context) string {
	for _, key := range []string{"id", "sessionID"} {
		if value := strings.TrimSpace(ctx.Param(key)); value != "" {
			return value
		}
	}

	return ""
}

func currentResourceName(ctx *gin.Context) string {
	for _, key := range []string{"name", "username"} {
		if value := strings.TrimSpace(ctx.Param(key)); value != "" {
			return value
		}
	}

	return ""
}

func currentAuditMessage(ctx *gin.Context) string {
	if ctx == nil {
		return ""
	}
	if key, ok := httpx.LastErrorMessageKey(ctx); ok {
		return key
	}
	if ctx.Writer.Status() >= http.StatusBadRequest {
		return strings.TrimSpace(http.StatusText(ctx.Writer.Status()))
	}

	return ""
}

func eventMetadata(payload pluginapi.AuditEvent) map[string]any {
	metadata := make(map[string]any, len(payload.Metadata)+eventMetadataExtraFields)
	for key, value := range payload.Metadata {
		metadata[key] = value
	}
	if method := strings.TrimSpace(payload.RequestMethod); method != "" {
		metadata["request_method"] = method
	}
	if path := strings.TrimSpace(payload.RequestPath); path != "" {
		metadata["request_path"] = path
	}
	return metadata
}

func resolveAuditEventPayload(payload any) (pluginapi.AuditEvent, error) {
	switch typed := payload.(type) {
	case pluginapi.AuditEvent:
		return typed, nil
	case *pluginapi.AuditEvent:
		if typed == nil {
			return pluginapi.AuditEvent{}, errors.New("nil audit event payload")
		}
		return *typed, nil
	default:
		return pluginapi.AuditEvent{}, fmt.Errorf("unexpected audit event payload type %T", payload)
	}
}
