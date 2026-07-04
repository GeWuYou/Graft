package project

import (
	"context"
	"strconv"
	"strings"
	"time"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
)

func (s *Service) publishDetachedAuditEvent(
	ctx context.Context,
	cancel context.CancelFunc,
	event moduleapi.AuditEvent,
	failureMessage string,
) {
	if cancel == nil {
		return
	}
	if s == nil || s.auditBus == nil {
		cancel()
		return
	}
	// Batch actions already aggregate to one audit event; detach its publish so
	// slow synchronous handlers do not extend the user-visible batch request.
	go func() {
		defer cancel()
		s.publishAuditEvent(ctx, event, failureMessage)
	}()
}

func (s *Service) prepareAuditContext(ctx context.Context) (context.Context, context.CancelFunc, bool) {
	if s == nil || s.auditBus == nil {
		return nil, nil, false
	}
	auditCtx, cancel := s.detachedAuditContext(ctx)
	return auditCtx, cancel, true
}

func (s *Service) detachedAuditContext(ctx context.Context) (context.Context, context.CancelFunc) {
	auditCtx, cancel := context.WithTimeout(context.Background(), projectAuditTimeout)
	if requestAudit, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		auditCtx = httpx.WithRequestAuditContext(auditCtx, requestAudit)
	}
	if requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx); ok {
		auditCtx = moduleapi.WithRequestAuthContext(auditCtx, requestAuth)
	}
	return auditCtx, cancel
}

func (s *Service) auditModuleName() string {
	if s == nil {
		return moduleID
	}
	if strings.TrimSpace(s.moduleName) == "" {
		return moduleID
	}
	return s.moduleName
}

func newProjectAuditEvent(
	ctx context.Context,
	actor actionActor,
	action string,
	resourceType string,
	resourceID string,
	resourceName string,
) moduleapi.AuditEvent {
	return moduleapi.AuditEvent{
		Kind:          moduleapi.AuditEventKindDomain,
		Operator:      actor.operator,
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		ResourceName:  resourceName,
		RequestMethod: requestMethodFromContext(ctx),
		RequestPath:   requestPathFromContext(ctx),
	}
}

func enrichProjectAuditMetadata(ctx context.Context, metadata map[string]any) {
	if metadata == nil {
		return
	}
	if requestAudit, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		metadata["requestId"] = requestAudit.RequestID
		metadata["traceId"] = requestAudit.TraceID
		metadata["route"] = requestAudit.Route
		metadata["method"] = requestAudit.Method
		metadata["client_ip"] = requestAudit.ClientIP
	}
}

func requestMethodFromContext(ctx context.Context) string {
	if requestAudit, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		return requestAudit.Method
	}
	return ""
}

func requestPathFromContext(ctx context.Context) string {
	if requestAudit, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		return requestAudit.Route
	}
	return ""
}

func batchAuditResourceID(action generated.ProjectBatchActionRequestAction, now time.Time) string {
	return "batch:" + strings.TrimSpace(string(action)) + ":" + strconv.FormatInt(now.UnixNano(), 10)
}

func trimmedStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
