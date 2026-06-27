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

	"graft/server/internal/container"
	"graft/server/internal/drilldown"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/logger/logsafe"
	"graft/server/internal/module"
	"graft/server/internal/moduleapi"
	auditstore "graft/server/modules/audit/store"
)

// Module 是当前 MVP 阶段的最小审计模块。
//
// 该模块在 Register 阶段挂载请求级自动审计中间件、受权只读查询路由、
// 菜单/权限声明，并订阅主动审计事件；当前不承载归档和分析逻辑。
type Module struct {
	recorder      *Service
	monitorBinder incidentMonitorEvidenceBinder
	notifier      moduleapi.NotificationPublisher
}

const eventMetadataExtraFields = 3

// NewModule 创建最小审计模块。
func NewModule(repo auditstore.AuditRepository) (*Module, error) {
	recorder, err := NewService(repo)
	if err != nil {
		return nil, err
	}

	moduleInstance := &Module{recorder: recorder}
	if binder, ok := repo.(incidentMonitorEvidenceBinder); ok {
		moduleInstance.monitorBinder = binder
	}

	return moduleInstance, nil
}

// NewModuleWithDrilldown creates the audit module with a drilldown-enabled read service.
func NewModuleWithDrilldown(
	repo auditstore.AuditRepository,
	drilldownService *drilldown.Service[ListQuery, ListQuery],
) (*Module, error) {
	recorder, err := NewServiceWithDrilldown(repo, drilldownService)
	if err != nil {
		return nil, err
	}

	moduleInstance := &Module{recorder: recorder}
	if binder, ok := repo.(incidentMonitorEvidenceBinder); ok {
		moduleInstance.monitorBinder = binder
	}

	return moduleInstance, nil
}

// Register 挂载 HTTP 自动审计、受权查询路由与 event bus 主动审计接线。
func (p *Module) Register(ctx *module.Context) error {
	if p.recorder == nil {
		return errors.New("audit recorder is unavailable")
	}
	if err := registerAuditMessages(ctx.I18n); err != nil {
		return err
	}
	if err := registerAuditLogRetentionConfigMessages(ctx.I18n); err != nil {
		return err
	}
	registerAuditPermissions(ctx.PermissionRegistry, moduleID)
	registerAuditMenu(ctx.MenuRegistry, moduleID)
	if err := registerAuditService(ctx, p.recorder); err != nil {
		return err
	}
	if err := registerAuditDashboardWidget(ctx, p.recorder); err != nil {
		return err
	}
	logger := ctx.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	if err := p.registerRetention(ctx, logger); err != nil {
		return err
	}
	if err := p.registerHTTP(ctx, logger); err != nil {
		return err
	}
	if ctx.EventBus == nil {
		return errors.New("event bus is unavailable")
	}

	return subscribeAuditRecordEvents(ctx.EventBus, logger, p.recorder, func() moduleapi.NotificationPublisher {
		return p.notifier
	})
}

func (p *Module) registerRetention(ctx *module.Context, logger *zap.Logger) error {
	if ctx.Config == nil {
		return errors.New("audit module config is unavailable")
	}
	if err := registerAuditLogRetentionConfigDefinition(ctx.ConfigRegistry); err != nil {
		return fmt.Errorf("register audit log retention config definition: %w", err)
	}
	if err := registerAuditLogRetentionCleanupJob(ctx.CronRegistry, logger, p.recorder, ctx.Config.Audit); err != nil {
		return fmt.Errorf("register audit log retention cleanup job: %w", err)
	}

	return nil
}

func (p *Module) registerHTTP(ctx *module.Context, logger *zap.Logger) error {
	if ctx.Router == nil {
		return nil
	}

	ctx.Router.Use(requestAuditMiddleware(logger, p.recorder, func() moduleapi.NotificationPublisher {
		return p.notifier
	}))
	guard, err := p.resolveRouteGuard(ctx)
	if err != nil {
		return err
	}
	registerAuditRoutes(ctx, moduleID, p.recorder, guard)

	return nil
}

func subscribeAuditRecordEvents(
	bus eventbus.Bus,
	logger *zap.Logger,
	recorder *Service,
	notifier func() moduleapi.NotificationPublisher,
) error {
	return bus.Subscribe(string(moduleapi.AuditRecordEventName), func(eventCtx context.Context, event eventbus.Event) error {
		return consumeAuditRecordEvent(eventCtx, logger, recorder, notifier, event)
	})
}

func consumeAuditRecordEvent(
	eventCtx context.Context,
	logger *zap.Logger,
	recorder *Service,
	notifier func() moduleapi.NotificationPublisher,
	event eventbus.Event,
) error {
	payload, err := resolveAuditEventPayload(event.Payload)
	if err != nil {
		logger.Error("drop malformed audit event payload",
			zap.String("module", moduleID),
			zap.String("event", string(moduleapi.AuditRecordEventName)),
			zap.Error(err),
		)
		return nil
	}

	if err := recordEvent(eventCtx, logger, recorder, notifier, payload); err != nil {
		logger.Error("write active audit log failed",
			zap.String("module", moduleID),
			zap.String("event", string(moduleapi.AuditRecordEventName)),
			zap.String("action", strings.TrimSpace(payload.Action)),
			zap.Error(err),
		)
	}

	return nil
}

// Boot resolves optional cross-module capabilities after all modules have completed Register.
func (p *Module) Boot(ctx *module.Context) error {
	if p == nil || ctx == nil || ctx.Services == nil {
		return nil
	}

	if err := p.bindMonitorEvidence(ctx); err != nil {
		return err
	}
	return p.bindNotificationPublisher(ctx)
}

// Shutdown 当前没有额外资源需要释放。
func (p *Module) Shutdown(_ *module.Context) error {
	return nil
}

type incidentMonitorEvidenceBinder interface {
	BindMonitorEvidence(moduleapi.MonitorIncidentEvidenceService)
}

func (p *Module) bindMonitorEvidence(ctx *module.Context) error {
	if p.monitorBinder == nil {
		return nil
	}
	resolved, err := ctx.Services.Resolve((*moduleapi.MonitorIncidentEvidenceService)(nil))
	if err != nil {
		if errors.Is(err, container.ErrServiceNotRegistered) {
			return nil
		}
		return fmt.Errorf("resolve monitor incident evidence service: %w", err)
	}
	service, ok := resolved.(moduleapi.MonitorIncidentEvidenceService)
	if !ok {
		return fmt.Errorf("resolve monitor incident evidence service: unexpected type %T", resolved)
	}
	p.monitorBinder.BindMonitorEvidence(service)
	return nil
}

func (p *Module) bindNotificationPublisher(ctx *module.Context) error {
	resolvedNotifier, err := ctx.Services.Resolve((*moduleapi.NotificationPublisher)(nil))
	if err != nil {
		if errors.Is(err, container.ErrServiceNotRegistered) {
			return nil
		}
		return fmt.Errorf("resolve notification publisher: %w", err)
	}
	notifier, ok := resolvedNotifier.(moduleapi.NotificationPublisher)
	if !ok {
		return fmt.Errorf("resolve notification publisher: unexpected type %T", resolvedNotifier)
	}
	p.notifier = notifier
	return nil
}

// 当审计记录可见时，会发布审计通知。
func requestAuditMiddleware(
	logger *zap.Logger,
	recorder *Service,
	notifier func() moduleapi.NotificationPublisher,
) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}

	return func(ctx *gin.Context) {
		ctx.Next()

		candidate := requestAuditCandidate(ctx)
		if record, recorded, err := recorder.RecordCandidate(ctx.Request.Context(), candidate); err != nil {
			logsafe.Error(logger, "write request audit log failed",
				zap.String("module", moduleID),
				zap.String("action", candidate.Action),
				zap.Error(err),
			)
		} else if !recorded {
			logsafe.Debug(logger, "skip request audit candidate by policy",
				zap.String("module", moduleID),
				zap.String("method", candidate.RequestMethod),
				zap.String("path", candidate.RequestPath),
			)
		} else if record.Visibility == auditstore.AuditVisibilityStrategyVisible {
			publishAuditNotification(ctx.Request.Context(), logger, resolveNotificationPublisher(notifier), record)
		}
	}
}

// 当安全事件因策略未被记录时，它会记录一条告警日志。
func recordEvent(
	ctx context.Context,
	logger *zap.Logger,
	recorder *Service,
	notifier func() moduleapi.NotificationPublisher,
	payload moduleapi.AuditEvent,
) error {
	candidate := eventAuditCandidate(ctx, payload)
	record, recorded, err := recorder.RecordCandidate(ctx, candidate)
	if err != nil {
		return err
	}
	if recorded && record.Visibility == auditstore.AuditVisibilityStrategyVisible {
		publishAuditNotification(ctx, logger, resolveNotificationPublisher(notifier), record)
	}
	if recorded || candidate.Source != auditstore.AuditSourceSecurityEvent {
		return nil
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	logsafe.Warn(logger, "skip security audit candidate by policy",
		zap.String("module", moduleID),
		zap.String("action", candidate.Action),
		zap.String("eventType", candidate.EventType),
		zap.String("path", candidate.RequestPath),
	)
	return nil
}

func resolveNotificationPublisher(resolve func() moduleapi.NotificationPublisher) moduleapi.NotificationPublisher {
	if resolve == nil {
		return nil
	}
	return resolve()
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
	if requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx.Request.Context()); ok {
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

func eventAuditCandidate(ctx context.Context, payload moduleapi.AuditEvent) auditstore.AuditCandidate {
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

func resolveEventOperator(ctx context.Context, payload moduleapi.AuditEvent) *moduleapi.CurrentUser {
	if payload.Operator != nil {
		return payload.Operator
	}
	if requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx); ok && requestAuth.User != nil {
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

func auditSourceFromEvent(payload moduleapi.AuditEvent) auditstore.AuditSource {
	switch payload.Kind {
	case moduleapi.AuditEventKindSecurity:
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

func eventMetadata(payload moduleapi.AuditEvent) map[string]any {
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
	if messageKey := strings.TrimSpace(payload.MessageKey); messageKey != "" {
		metadata["message_key"] = messageKey
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

func resolveAuditEventPayload(payload any) (moduleapi.AuditEvent, error) {
	switch typed := payload.(type) {
	case moduleapi.AuditEvent:
		return typed, nil
	case *moduleapi.AuditEvent:
		if typed == nil {
			return moduleapi.AuditEvent{}, errors.New("nil audit event payload")
		}
		return *typed, nil
	default:
		return moduleapi.AuditEvent{}, fmt.Errorf("unexpected audit event payload type %T", payload)
	}
}
