package container

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	"graft/server/internal/realtimeauth"
	containercontract "graft/server/modules/container/contract"
	"graft/server/modules/container/terminal"
)

// ShellSessionRequest describes one requested interactive container shell session.
type ShellSessionRequest struct {
	Command string
	Cols    int
	Rows    int
}

// ShellSession contains the issued shell ticket and websocket bootstrap data.
type ShellSession struct {
	SessionID    string
	Command      string
	Cols         int
	Rows         int
	ExpiresAt    time.Time
	WebSocketURL string
}

// ShellHandshake contains the validated ticket payload used to open a terminal session.
type ShellHandshake struct {
	SessionID    string
	Command      string
	Cols         int
	Rows         int
	ResourceID   string
	ResourceName string
	UserID       uint64
}

// ShellSessionCloseSummary carries audit-safe shell session close details.
type ShellSessionCloseSummary struct {
	SessionID    string
	ResourceID   string
	ResourceName string
	Command      string
	UserID       uint64
}

type shellAuditPayload struct {
	action  string
	detail  Detail
	issued  *realtimeauth.IssuedTicket
	command string
	reason  string
	err     error
}

const (
	containerShellScope        = "container.shell"
	containerShellResourceType = "container"
)

func (s *service) IssueShellSession(ctx context.Context, ref Ref, request ShellSessionRequest) (ShellSession, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return ShellSession{}, err
	}
	if !s.shellAllowed(ctx) {
		return ShellSession{}, errShellDisabled
	}
	normalized, err := normalizeShellSessionRequest(request)
	if err != nil {
		return ShellSession{}, err
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		return ShellSession{}, err
	}
	detail, err := runtime.Detail(ctx, ref)
	if err != nil {
		return ShellSession{}, err
	}
	if strings.TrimSpace(strings.ToLower(detail.State)) != "running" {
		return ShellSession{}, errContainerNotRunning
	}
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil {
		return ShellSession{}, errShellForbidden
	}
	s.publishShellAudit(ctx, shellAuditPayload{
		action:  containercontract.ContainerAuditActionShellSessionRequested.String(),
		detail:  detail,
		command: normalized.Command,
	})
	issued, err := s.realtimeTickets.Issue(ctx, realtimeauth.IssueRequest{
		UserID:       requestAuth.User.ID,
		ResourceType: containerShellResourceType,
		ResourceID:   ref.Value,
		Scope:        containerShellScope,
		ClientIP:     currentRequestClientIP(ctx),
		UserAgent:    currentRequestUserAgent(ctx),
		Command:      normalized.Command,
		Cols:         normalized.Cols,
		Rows:         normalized.Rows,
		TTL:          containerOperationTTL,
	})
	if err != nil {
		s.publishShellAudit(ctx, shellAuditPayload{
			action:  containercontract.ContainerAuditActionShellTicketRejected.String(),
			detail:  detail,
			command: normalized.Command,
			reason:  "ticket_issue_failed",
			err:     errShellSessionFailed,
		})
		return ShellSession{}, errShellSessionFailed
	}
	s.publishShellAudit(ctx, shellAuditPayload{
		action:  containercontract.ContainerAuditActionShellTicketIssued.String(),
		detail:  detail,
		issued:  &issued,
		command: normalized.Command,
	})
	return ShellSession{
		SessionID:    issued.SessionID,
		Command:      issued.Command,
		Cols:         issued.Cols,
		Rows:         issued.Rows,
		ExpiresAt:    issued.ExpiresAt,
		WebSocketURL: buildShellWebSocketURL(ref, issued.Ticket),
	}, nil
}

func (s *service) ConsumeShellSessionTicket(ctx context.Context, ref Ref, ticket string, origin string) (ShellHandshake, error) {
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return ShellHandshake{}, err
	}
	if !s.shellAllowed(ctx) {
		return ShellHandshake{}, errShellDisabled
	}
	if err := realtimeauth.ValidateOrigin(origin, s.websocketAllowedOrigins); err != nil {
		return ShellHandshake{}, errShellOriginDenied
	}
	consumed, err := s.realtimeTickets.Consume(ctx, realtimeauth.ConsumeRequest{
		Ticket:       ticket,
		ResourceType: containerShellResourceType,
		ResourceID:   ref.Value,
		Scope:        containerShellScope,
	})
	if err != nil {
		return ShellHandshake{}, mapRealtimeTicketError(err)
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		return ShellHandshake{}, err
	}
	detail, err := runtime.Detail(ctx, ref)
	if err != nil {
		return ShellHandshake{}, err
	}
	if strings.TrimSpace(strings.ToLower(detail.State)) != "running" {
		s.publishShellAudit(ctx, shellAuditPayload{
			action:  containercontract.ContainerAuditActionShellTicketRejected.String(),
			detail:  detail,
			command: consumed.Command,
			reason:  "container_not_running",
			err:     errContainerNotRunning,
		})
		return ShellHandshake{}, errContainerNotRunning
	}
	s.publishShellAudit(ctx, shellAuditPayload{
		action:  containercontract.ContainerAuditActionShellSessionStarted.String(),
		detail:  detail,
		command: consumed.Command,
	})
	return ShellHandshake{
		SessionID:    consumed.SessionID,
		Command:      consumed.Command,
		Cols:         consumed.Cols,
		Rows:         consumed.Rows,
		ResourceID:   detail.ID,
		ResourceName: detail.Name,
		UserID:       consumed.UserID,
	}, nil
}

func (s *service) OpenShellTerminalSession(ctx context.Context, ref Ref, handshake ShellHandshake) (terminal.Session, error) {
	if s == nil {
		return nil, errShellSessionFailed
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		s.publishShellSessionFailed(ctx, handshake, "runtime_unavailable", err)
		return nil, err
	}
	session, err := runtime.Shell(ctx, ref, handshake.Command)
	if err != nil {
		s.publishShellSessionFailed(ctx, handshake, "session_open_failed", err)
		return nil, err
	}
	return session, nil
}

// normalizeShellSessionRequest 验证并规范化 Shell 会话请求。
// 确保命令为 sh、bash 或 ash 之一，且终端行列尺寸均为正数。
// 返回规范化后的请求或错误。
func normalizeShellSessionRequest(request ShellSessionRequest) (ShellSessionRequest, error) {
	command := strings.TrimSpace(strings.ToLower(request.Command))
	switch command {
	case "sh", "bash", "ash":
	default:
		return ShellSessionRequest{}, errShellCommandNotFound
	}
	if request.Cols <= 0 || request.Rows <= 0 {
		return ShellSessionRequest{}, errShellInvalidSize
	}
	return ShellSessionRequest{
		Command: command,
		Cols:    request.Cols,
		Rows:    request.Rows,
	}, nil
}

// buildShellWebSocketURL constructs a WebSocket URL for accessing the shell of a specified container.
func buildShellWebSocketURL(ref Ref, ticket string) string {
	values := url.Values{}
	values.Set("ticket", ticket)
	return "/api" + containercontract.ContainerAPIGroup + "/" + url.PathEscape(ref.Value) + "/shell/ws?" + values.Encode()
}

// mapRealtimeTicketError 将实时票证错误映射为对应的 Shell 特定错误。
// 若错误为未知类型，则返回会话失败错误。
func mapRealtimeTicketError(err error) error {
	switch {
	case errors.Is(err, realtimeauth.ErrExpiredTicket):
		return errShellTicketExpired
	case errors.Is(err, realtimeauth.ErrUsedTicket):
		return errShellTicketUsed
	case errors.Is(err, realtimeauth.ErrResourceMismatch), errors.Is(err, realtimeauth.ErrScopeMismatch), errors.Is(err, realtimeauth.ErrInvalidTicket), errors.Is(err, realtimeauth.ErrTicketRequired):
		return errShellTicketInvalid
	default:
		return errShellSessionFailed
	}
}

// CurrentRequestClientIP 从请求审计上下文中提取客户端 IP 地址。如果审计上下文不存在，返回空字符串。
func currentRequestClientIP(ctx context.Context) string {
	requestAudit, ok := httpx.RequestAuditContextFromContext(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(requestAudit.ClientIP)
}

// currentRequestUserAgent returns the User-Agent from the current request's audit context, or an empty string if unavailable.
func currentRequestUserAgent(ctx context.Context) string {
	requestAudit, ok := httpx.RequestAuditContextFromContext(ctx)
	if !ok {
		return ""
	}
	return strings.TrimSpace(requestAudit.UserAgent)
}

func (s *service) publishShellSessionClosed(
	ctx context.Context,
	handshake ShellHandshake,
	startedAt time.Time,
	reason string,
	err error,
) {
	if s == nil || s.auditBus == nil {
		return
	}
	auditCtx, cancel := detachedAuditContext(ctx)
	defer cancel()
	duration := time.Since(startedAt)
	metadata := map[string]any{
		"container_id":   handshake.ResourceID,
		"container_name": handshake.ResourceName,
		"command":        handshake.Command,
		"result":         auditResult(err),
		"session_id":     handshake.SessionID,
		"duration_ms":    duration.Milliseconds(),
		"close_reason":   strings.TrimSpace(reason),
	}
	if requestAudit, ok := httpx.RequestAuditContextFromContext(auditCtx); ok {
		metadata["requestId"] = requestAudit.RequestID
		metadata["traceId"] = requestAudit.TraceID
		metadata["route"] = requestAudit.Route
		metadata["client_ip"] = requestAudit.ClientIP
		metadata["user_agent"] = requestAudit.UserAgent
	}
	user := currentAuditOperator(auditCtx)
	if user == nil && handshake.UserID != 0 {
		user = &moduleapi.CurrentUser{ID: handshake.UserID}
	}
	event := moduleapi.AuditEvent{
		Kind:         moduleapi.AuditEventKindDomain,
		Operator:     user,
		Action:       containercontract.ContainerAuditActionShellSessionClosed.String(),
		ResourceType: containerResourceType,
		ResourceID:   firstNonEmpty(handshake.ResourceID, handshake.ResourceName),
		ResourceName: handshake.ResourceName,
		StatusCode:   auditStatusCode(err),
		Success:      err == nil,
		Metadata:     metadata,
	}
	if err != nil {
		event.MessageKey = messageKeyForError(err).String()
		event.Message = fallbackMessageForError(err)
	}
	if publishErr := s.auditBus.Publish(auditCtx, eventbus.Event{
		Name:    string(moduleapi.AuditRecordEventName),
		Source:  s.moduleName,
		Payload: event,
	}); publishErr != nil && s.logger != nil {
		s.logger.Warn("publish container shell close audit event failed",
			zap.String("module", s.moduleName),
			zap.String("action", containercontract.ContainerAuditActionShellSessionClosed.String()),
			zap.Error(publishErr),
		)
	}
}

func (s *service) publishShellSessionFailed(ctx context.Context, handshake ShellHandshake, reason string, err error) {
	if s == nil || s.auditBus == nil {
		return
	}
	auditCtx, cancel := detachedAuditContext(ctx)
	defer cancel()
	metadata := map[string]any{
		"container_id":   handshake.ResourceID,
		"container_name": handshake.ResourceName,
		"command":        handshake.Command,
		"result":         auditResult(err),
		"session_id":     handshake.SessionID,
		"reason":         strings.TrimSpace(reason),
	}
	if requestAudit, ok := httpx.RequestAuditContextFromContext(auditCtx); ok {
		metadata["requestId"] = requestAudit.RequestID
		metadata["traceId"] = requestAudit.TraceID
		metadata["route"] = requestAudit.Route
		metadata["client_ip"] = requestAudit.ClientIP
		metadata["user_agent"] = requestAudit.UserAgent
	}
	user := currentAuditOperator(auditCtx)
	if user == nil && handshake.UserID != 0 {
		user = &moduleapi.CurrentUser{ID: handshake.UserID}
	}
	event := moduleapi.AuditEvent{
		Kind:         moduleapi.AuditEventKindDomain,
		Operator:     user,
		Action:       containercontract.ContainerAuditActionShellSessionFailed.String(),
		ResourceType: containerResourceType,
		ResourceID:   firstNonEmpty(handshake.ResourceID, handshake.ResourceName),
		ResourceName: handshake.ResourceName,
		StatusCode:   auditStatusCode(err),
		Success:      false,
		Metadata:     metadata,
	}
	if err != nil {
		event.MessageKey = messageKeyForError(err).String()
		event.Message = fallbackMessageForError(err)
	}
	if publishErr := s.auditBus.Publish(auditCtx, eventbus.Event{
		Name:    string(moduleapi.AuditRecordEventName),
		Source:  s.moduleName,
		Payload: event,
	}); publishErr != nil && s.logger != nil {
		s.logger.Warn("publish container shell failure audit event failed",
			zap.String("module", s.moduleName),
			zap.String("action", containercontract.ContainerAuditActionShellSessionFailed.String()),
			zap.Error(publishErr),
		)
	}
}

func (s *service) publishShellAudit(ctx context.Context, payload shellAuditPayload) {
	if s == nil || s.auditBus == nil {
		return
	}
	metadata := map[string]any{
		"container_id":   payload.detail.ID,
		"container_name": payload.detail.Name,
		"command":        strings.TrimSpace(payload.command),
		"result":         auditResult(payload.err),
	}
	if payload.reason != "" {
		metadata["reason"] = payload.reason
	}
	if payload.issued != nil {
		metadata["session_id"] = payload.issued.SessionID
		metadata["ticket_id"] = payload.issued.TicketID
		metadata["expires_at"] = payload.issued.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if requestAudit, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		metadata["requestId"] = requestAudit.RequestID
		metadata["traceId"] = requestAudit.TraceID
		metadata["route"] = requestAudit.Route
		metadata["client_ip"] = requestAudit.ClientIP
		metadata["user_agent"] = requestAudit.UserAgent
	}
	event := moduleapi.AuditEvent{
		Kind:         moduleapi.AuditEventKindDomain,
		Operator:     currentAuditOperator(ctx),
		Action:       payload.action,
		ResourceType: containerResourceType,
		ResourceID:   firstNonEmpty(payload.detail.ID, payload.detail.Name),
		ResourceName: payload.detail.Name,
		StatusCode:   auditStatusCode(payload.err),
		Success:      payload.err == nil,
		Metadata:     metadata,
	}
	if payload.err != nil {
		event.MessageKey = messageKeyForError(payload.err).String()
		event.Message = fallbackMessageForError(payload.err)
	}
	if publishErr := s.auditBus.Publish(ctx, eventbus.Event{
		Name:    string(moduleapi.AuditRecordEventName),
		Source:  s.moduleName,
		Payload: event,
	}); publishErr != nil && s.logger != nil {
		s.logger.Warn("publish container shell audit event failed",
			zap.String("module", s.moduleName),
			zap.String("action", payload.action),
			zap.Error(publishErr),
		)
	}
}

type disabledRuntime struct{}

func (disabledRuntime) Info(context.Context) (RuntimeInfo, error) {
	return RuntimeInfo{Runtime: runtimeNameDocker, Status: "disabled"}, errRuntimeDisabled
}
func (disabledRuntime) List(context.Context, ListQuery) ([]Summary, error) {
	return nil, errRuntimeDisabled
}
func (disabledRuntime) Detail(context.Context, Ref) (Detail, error) {
	return Detail{}, errRuntimeDisabled
}
func (disabledRuntime) Mounts(context.Context, Ref) ([]Mount, error) {
	return nil, errRuntimeDisabled
}
func (disabledRuntime) MountUsage(context.Context, Ref, string) (MountUsage, error) {
	return MountUsage{}, errRuntimeDisabled
}
func (disabledRuntime) Logs(context.Context, Ref, LogQuery) (Logs, error) {
	return Logs{}, errRuntimeDisabled
}
func (disabledRuntime) StreamLogs(context.Context, Ref, LogQuery, func(LogChunk) error) error {
	return errRuntimeDisabled
}
func (disabledRuntime) StreamRuntimeEvents(context.Context, func(RuntimeEventCandidate) error) error {
	return errRuntimeDisabled
}
func (disabledRuntime) Shell(context.Context, Ref, string) (terminal.Session, error) {
	return nil, errRuntimeDisabled
}
func (disabledRuntime) Start(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, errRuntimeDisabled
}
func (disabledRuntime) Stop(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, errRuntimeDisabled
}
func (disabledRuntime) Restart(context.Context, Ref) (ActionResult, error) {
	return ActionResult{}, errRuntimeDisabled
}
func (disabledRuntime) Remove(context.Context, Ref, RemoveOptions) (ActionResult, error) {
	return ActionResult{}, errRuntimeDisabled
}
func (disabledRuntime) Close() error { return nil }

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
