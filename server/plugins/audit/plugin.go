package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	auditcore "graft/server/internal/audit"
	"graft/server/internal/container"
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
	recorder      *auditcore.Service
	monitorBinder incidentMonitorEvidenceBinder
}

const eventMetadataExtraFields = 3

// NewPlugin 创建最小审计插件。
func NewPlugin(repo auditstore.AuditRepository) (*Plugin, error) {
	recorder, err := auditcore.NewService(repo)
	if err != nil {
		return nil, err
	}

	pluginInstance := &Plugin{recorder: recorder}
	if binder, ok := repo.(incidentMonitorEvidenceBinder); ok {
		pluginInstance.monitorBinder = binder
	}

	return pluginInstance, nil
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

		if err := recordEvent(eventCtx, logger, p.recorder, payload); err != nil {
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

// Boot resolves optional cross-plugin capabilities after all plugins have completed Register.
func (p *Plugin) Boot(ctx *plugin.Context) error {
	if p == nil || p.monitorBinder == nil || ctx == nil || ctx.Services == nil {
		return nil
	}

	resolved, err := ctx.Services.Resolve((*pluginapi.MonitorIncidentEvidenceService)(nil))
	if err != nil {
		if errors.Is(err, container.ErrServiceNotRegistered) {
			return nil
		}
		return fmt.Errorf("resolve monitor incident evidence service: %w", err)
	}

	service, ok := resolved.(pluginapi.MonitorIncidentEvidenceService)
	if !ok {
		return fmt.Errorf("resolve monitor incident evidence service: unexpected type %T", resolved)
	}

	p.monitorBinder.BindMonitorEvidence(service)
	return nil
}

// Shutdown 当前没有额外资源需要释放。
func (p *Plugin) Shutdown(_ *plugin.Context) error {
	return nil
}

type incidentMonitorEvidenceBinder interface {
	BindMonitorEvidence(pluginapi.MonitorIncidentEvidenceService)
}

func requestAuditMiddleware(logger *zap.Logger, recorder *auditcore.Service) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(ctx *gin.Context) {
		ctx.Next()

		candidate := requestAuditCandidate(ctx)
		if _, recorded, err := recorder.RecordCandidate(ctx.Request.Context(), candidate); err != nil {
			logger.Error("write request audit log failed",
				zap.String("plugin", "audit"),
				zap.String("action", candidate.Action),
				zap.Error(err),
			)
		} else if !recorded {
			logger.Debug("skip request audit candidate by policy",
				zap.String("plugin", pluginID),
				zap.String("method", candidate.RequestMethod),
				zap.String("path", candidate.RequestPath),
			)
		}
	}
}

func recordEvent(ctx context.Context, logger *zap.Logger, recorder *auditcore.Service, payload pluginapi.AuditEvent) error {
	candidate := eventAuditCandidate(ctx, payload)
	_, recorded, err := recorder.RecordCandidate(ctx, candidate)
	if err != nil {
		return err
	}
	if recorded || candidate.Source != auditstore.AuditSourceSecurityEvent {
		return nil
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	logger.Warn("skip security audit candidate by policy",
		zap.String("plugin", pluginID),
		zap.String("action", candidate.Action),
		zap.String("eventType", candidate.EventType),
		zap.String("path", candidate.RequestPath),
	)
	return nil
}

func requestAuditCandidate(ctx *gin.Context) auditstore.AuditCandidate {
	candidate := auditstore.AuditCandidate{
		Source:        auditstore.AuditSourceRequest,
		Action:        buildAction(ctx),
		ResourceType:  currentResourceType(ctx),
		ResourceID:    currentResourceID(ctx),
		ResourceName:  currentResourceName(ctx),
		RequestMethod: strings.TrimSpace(ctx.Request.Method),
		RequestPath:   currentRoutePath(ctx),
		StatusCode:    ctx.Writer.Status(),
		RequestID:     httpx.EnsureRequestID(ctx),
		TraceID:       httpx.EnsureRequestID(ctx),
		IP:            strings.TrimSpace(ctx.ClientIP()),
		UserAgent:     strings.TrimSpace(ctx.Request.UserAgent()),
		Success:       ctx.Writer.Status() < http.StatusBadRequest,
		Message:       currentAuditMessage(ctx),
	}
	if requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx.Request.Context()); ok {
		if requestAuth.User != nil {
			candidate.ActorUserID = &requestAuth.User.ID
			candidate.ActorUsername = strings.TrimSpace(requestAuth.User.Username)
			candidate.ActorDisplayName = strings.TrimSpace(requestAuth.User.DisplayName)
		}
		if requestAuth.Claims != nil {
			candidate.SessionID = strings.TrimSpace(requestAuth.Claims.SessionID)
		}
	}

	return candidate
}

func eventAuditCandidate(ctx context.Context, payload pluginapi.AuditEvent) auditstore.AuditCandidate {
	requestAudit := resolveRequestAuditContext(ctx)
	operator := resolveEventOperator(ctx, payload)

	candidate := auditstore.AuditCandidate{
		Source:        auditSourceFromEvent(payload),
		Action:        strings.TrimSpace(payload.Action),
		EventType:     strings.TrimSpace(payload.Action),
		ResourceType:  strings.TrimSpace(payload.ResourceType),
		ResourceID:    strings.TrimSpace(payload.ResourceID),
		ResourceName:  strings.TrimSpace(payload.ResourceName),
		RequestMethod: firstNonEmptyTrimmed(payload.RequestMethod, requestAudit.Method),
		RequestPath:   firstNonEmptyTrimmed(payload.RequestPath, requestAudit.Route),
		StatusCode:    payload.StatusCode,
		RequestID:     firstNonEmptyTrimmed(payload.RequestID, requestAudit.RequestID),
		TraceID:       firstNonEmptyTrimmed(payload.RequestID, requestAudit.TraceID, requestAudit.RequestID),
		IP:            firstNonEmptyTrimmed(payload.IP, requestAudit.ClientIP),
		UserAgent:     firstNonEmptyTrimmed(payload.UserAgent, requestAudit.UserAgent),
		Success:       payload.Success,
		Message:       strings.TrimSpace(payload.Message),
		Metadata:      mustMarshalAuditEventMetadata(eventMetadata(payload)),
		CreatedAt:     payload.CreatedAt,
	}
	if operator != nil {
		candidate.ActorUserID = &operator.ID
		candidate.ActorUsername = strings.TrimSpace(operator.Username)
		candidate.ActorDisplayName = strings.TrimSpace(operator.DisplayName)
	}

	return candidate
}

func resolveRequestAuditContext(ctx context.Context) httpx.RequestAuditContext {
	requestAudit, _ := httpx.RequestAuditContextFromContext(ctx)
	return requestAudit
}

func resolveEventOperator(ctx context.Context, payload pluginapi.AuditEvent) *pluginapi.CurrentUser {
	if payload.Operator != nil {
		return payload.Operator
	}
	if requestAuth, ok := pluginapi.RequestAuthContextFromContext(ctx); ok && requestAuth.User != nil {
		return requestAuth.User
	}

	return nil
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func auditSourceFromEvent(payload pluginapi.AuditEvent) auditstore.AuditSource {
	switch payload.Kind {
	case pluginapi.AuditEventKindSecurity:
		return auditstore.AuditSourceSecurityEvent
	default:
		return auditstore.AuditSourceDomainEvent
	}
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
	if payload.StatusCode > 0 {
		metadata["status_code"] = payload.StatusCode
	}
	return metadata
}

func mustMarshalAuditEventMetadata(metadata map[string]any) json.RawMessage {
	if len(metadata) == 0 {
		return json.RawMessage([]byte("{}"))
	}

	payload, err := json.Marshal(metadata)
	if err != nil {
		return json.RawMessage([]byte("{}"))
	}

	return json.RawMessage(payload)
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
