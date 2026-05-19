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
// 该插件在 Register 阶段挂载请求级自动审计中间件，并订阅主动审计事件；
// 当前不暴露查询路由，也不承载归档和分析逻辑。
type Plugin struct {
	recorder *auditcore.Service
}

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
	return nil
}

// Register 挂载 HTTP 自动审计与 event bus 主动审计接线。
func (p *Plugin) Register(ctx *plugin.Context) error {
	if p.recorder == nil {
		return errors.New("audit recorder is unavailable")
	}
	if ctx.Router != nil {
		ctx.Router.Use(requestAuditMiddleware(ctx.Logger, p.recorder))
	}
	if ctx.EventBus == nil {
		return errors.New("event bus is unavailable")
	}

	return ctx.EventBus.Subscribe(pluginapi.AuditRecordEventName, func(eventCtx context.Context, event eventbus.Event) error {
		payload, err := resolveAuditEventPayload(event.Payload)
		if err != nil {
			return fmt.Errorf("unexpected audit event payload type %T", event.Payload)
		}

		return recordEvent(eventCtx, p.recorder, payload)
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
			Action:        buildAction(ctx),
			ResourceType:  currentResourceType(ctx),
			ResourceID:    currentResourceID(ctx),
			RequestMethod: ctx.Request.Method,
			RequestPath:   currentRoutePath(ctx),
			IP:            ctx.ClientIP(),
			UserAgent:     ctx.Request.UserAgent(),
			Success:       ctx.Writer.Status() < http.StatusBadRequest,
			ErrorMessage:  currentAuditErrorMessage(ctx),
		}

		if requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx.Request.Context()); ok && requestAuth.User != nil {
			input.OperatorID = &requestAuth.User.ID
			input.OperatorName = actorName(*requestAuth.User)
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
		Action:        strings.TrimSpace(payload.Action),
		ResourceType:  strings.TrimSpace(payload.ResourceType),
		ResourceID:    strings.TrimSpace(payload.ResourceID),
		RequestMethod: strings.TrimSpace(payload.RequestMethod),
		RequestPath:   strings.TrimSpace(payload.RequestPath),
		IP:            strings.TrimSpace(payload.IP),
		UserAgent:     strings.TrimSpace(payload.UserAgent),
		Success:       payload.Success,
		ErrorMessage:  strings.TrimSpace(payload.ErrorMessage),
		CreatedAt:     payload.CreatedAt,
	}
	if payload.Operator != nil {
		input.OperatorID = &payload.Operator.ID
		input.OperatorName = actorName(*payload.Operator)
	}

	_, err := recorder.Record(ctx, input)
	return err
}

func actorName(user pluginapi.CurrentUser) string {
	if strings.TrimSpace(user.DisplayName) != "" {
		return strings.TrimSpace(user.DisplayName)
	}

	return strings.TrimSpace(user.Username)
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

func currentAuditErrorMessage(ctx *gin.Context) string {
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
