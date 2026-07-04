package project

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/eventbus"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

const (
	projectResourceType      = "project"
	projectBatchResourceType = "project_batch"
	projectAuditTimeout      = 3 * time.Second
)

type actionActor struct {
	id       uint64
	operator *moduleapi.CurrentUser
}

func (s *Service) requireActionActor(
	ctx context.Context,
	projectID uint64,
	action generated.ProjectActionResponseAction,
	actorID *uint64,
) (actionActor, ActionResult, error) {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	if !ok || requestAuth.User == nil {
		return actionActor{}, actorAttributionBlockedResult(projectID, action, "actor_request_context_missing"), errProjectActorAttribution
	}
	user := *requestAuth.User
	if actorID != nil && *actorID != user.ID {
		return actionActor{}, actorAttributionBlockedResult(projectID, action, "actor_identity_mismatch"), errProjectActorAttribution
	}
	return actionActor{
		id:       user.ID,
		operator: &user,
	}, ActionResult{}, nil
}

func actorAttributionBlockedResult(
	projectID uint64,
	action generated.ProjectActionResponseAction,
	reason string,
) ActionResult {
	return blockedActionResult(projectID, action, []GuardResult{guardDetail("actor_attribution_required", reason)})
}

func (s *Service) publishProjectActionAudit(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	actor actionActor,
	result ActionResult,
	actionErr error,
) {
	if s == nil || s.auditBus == nil {
		return
	}
	auditCtx, cancel := s.detachedAuditContext(ctx)
	defer cancel()

	metadata := projectActionAuditMetadata(aggregate, result)
	enrichProjectAuditMetadata(auditCtx, metadata)
	event := moduleapi.AuditEvent{
		Kind:          moduleapi.AuditEventKindDomain,
		Operator:      actor.operator,
		Action:        projectAuditAction(result.Action).String(),
		ResourceType:  projectResourceType,
		ResourceID:    strconv.FormatUint(aggregate.Project.ID, 10),
		ResourceName:  aggregate.Project.CanonicalProjectName,
		RequestMethod: requestMethodFromContext(auditCtx),
		RequestPath:   requestPathFromContext(auditCtx),
		StatusCode:    projectActionAuditStatusCode(result, actionErr),
		Success:       actionErr == nil && result.Result == generated.ProjectActionResponseResultProjectActionResultCompleted,
		MessageKey:    trimmedStringValue(result.MessageKey),
		Message:       trimmedStringValue(result.Message),
		Metadata:      metadata,
	}
	s.publishAuditEvent(auditCtx, event, "publish project audit event failed")
}

func (s *Service) publishProjectBatchAudit(
	ctx context.Context,
	actor actionActor,
	request BatchActionRequest,
	result BatchActionResult,
) {
	if s == nil || s.auditBus == nil {
		return
	}
	auditCtx, cancel := s.detachedAuditContext(ctx)
	defer cancel()

	resourceID := batchAuditResourceID(request.Action, time.Now().UTC())
	metadata := map[string]any{
		"batch":                    true,
		"requested_total":          result.TotalCount,
		"requested_ids":            append([]uint64(nil), request.ProjectIDs...),
		"completed_count":          result.CompletedCount,
		"blocked_count":            result.BlockedCount,
		"skipped_count":            result.SkippedCount,
		"blocked_ids":              batchBlockedProjectIDs(result.Items),
		"skipped_ids":              batchSkippedProjectIDs(result.Items),
		"remove_named_volumes":     request.RemoveNamedVolumes,
		"auto_unregister":          request.AutoUnregister,
		"image_prune":              request.ImagePrune,
		"delete_working_directory": request.DeleteWorkingDirectory,
	}
	if request.ConfirmCanonicalProjectName != nil {
		metadata["confirm_canonical_project_name"] = strings.TrimSpace(*request.ConfirmCanonicalProjectName)
	}
	enrichProjectAuditMetadata(auditCtx, metadata)

	event := moduleapi.AuditEvent{
		Kind:          moduleapi.AuditEventKindDomain,
		Operator:      actor.operator,
		Action:        projectBatchAuditAction(request.Action).String(),
		ResourceType:  projectBatchResourceType,
		ResourceID:    resourceID,
		ResourceName:  strings.TrimSpace(string(request.Action)) + " x" + strconv.Itoa(result.TotalCount),
		RequestMethod: requestMethodFromContext(auditCtx),
		RequestPath:   requestPathFromContext(auditCtx),
		StatusCode:    projectBatchAuditStatusCode(result),
		Success:       result.BlockedCount == 0,
		MessageKey:    "",
		Message:       "",
		Metadata:      metadata,
	}
	s.publishAuditEvent(auditCtx, event, "publish project batch audit event failed")
}

func projectActionAuditMetadata(aggregate projectstore.ProjectAggregate, result ActionResult) map[string]any {
	metadata := map[string]any{
		"project_id":             aggregate.Project.ID,
		"canonical_project_name": aggregate.Project.CanonicalProjectName,
		"display_name":           aggregate.Project.DisplayName,
		"host_scope":             aggregate.Project.HostScope,
		"source_kind":            aggregate.Project.SourceKind,
		"ownership_mode":         aggregate.Project.OwnershipMode,
		"working_directory":      aggregate.Project.WorkingDirectory,
		"action_result":          result.Result,
		"guard_results":          guardResultsAuditMetadata(result.GuardResults),
	}
	return metadata
}

func guardResultsAuditMetadata(guards []GuardResult) []map[string]string {
	items := make([]map[string]string, 0, len(guards))
	for _, guard := range guards {
		items = append(items, map[string]string{
			"code":        guard.Code,
			"message_key": trimmedStringValue(guard.MessageKey),
			"detail":      trimmedStringValue(guard.Detail),
		})
	}
	return items
}

func batchBlockedProjectIDs(items []BatchActionItemResult) []uint64 {
	ids := make([]uint64, 0, len(items))
	for _, item := range items {
		if item.Result == generated.ProjectActionResponseResultProjectActionResultBlocked {
			ids = append(ids, item.ProjectID)
		}
	}
	return ids
}

func batchSkippedProjectIDs(items []BatchActionItemResult) []uint64 {
	ids := make([]uint64, 0, len(items))
	for _, item := range items {
		if item.Skipped {
			ids = append(ids, item.ProjectID)
		}
	}
	return ids
}

func projectActionAuditStatusCode(result ActionResult, err error) int {
	if err == nil && result.Result == generated.ProjectActionResponseResultProjectActionResultCompleted {
		return http.StatusOK
	}
	switch {
	case errors.Is(err, errProjectInvalidArgument), errors.Is(err, errProjectFileNotFound):
		return http.StatusBadRequest
	case errors.Is(err, errProjectNotFound):
		return http.StatusNotFound
	case errors.Is(err, errProjectDirectoryForbidden):
		return http.StatusForbidden
	case errors.Is(err, errProjectConflict),
		errors.Is(err, errProjectUnsupportedLifecycle),
		errors.Is(err, errProjectDestroyBlocked),
		errors.Is(err, errProjectManagedFlow),
		errors.Is(err, errProjectInspectionExpired),
		errors.Is(err, errProjectInspectionStale),
		errors.Is(err, errProjectActorAttribution):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func projectBatchAuditStatusCode(result BatchActionResult) int {
	if result.BlockedCount > 0 {
		return http.StatusConflict
	}
	return http.StatusOK
}

func projectAuditAction(action generated.ProjectActionResponseAction) projectcontract.AuditAction {
	switch action {
	case generated.ProjectActionResponseActionProjectActionUp:
		return projectcontract.ProjectAuditActionUp
	case generated.ProjectActionResponseActionProjectActionDown:
		return projectcontract.ProjectAuditActionDown
	case generated.ProjectActionResponseActionProjectActionRestart:
		return projectcontract.ProjectAuditActionRestart
	case generated.ProjectActionResponseActionProjectActionRedeploy:
		return projectcontract.ProjectAuditActionRedeploy
	case generated.ProjectActionResponseActionProjectActionUpdateDeploy:
		return projectcontract.ProjectAuditActionUpdateDeploy
	case generated.ProjectActionResponseActionProjectActionUnregister:
		return projectcontract.ProjectAuditActionUnregister
	case generated.ProjectActionResponseActionProjectActionDestroy:
		return projectcontract.ProjectAuditActionDestroy
	default:
		return projectcontract.AuditAction(strings.TrimSpace(string(action)))
	}
}

func projectBatchAuditAction(action generated.ProjectBatchActionRequestAction) projectcontract.AuditAction {
	switch action {
	case generated.ProjectBatchActionRequestActionStart:
		return projectcontract.ProjectAuditActionBatchStart
	case generated.ProjectBatchActionRequestActionStop:
		return projectcontract.ProjectAuditActionBatchStop
	case generated.ProjectBatchActionRequestActionRestart:
		return projectcontract.ProjectAuditActionBatchRestart
	case generated.ProjectBatchActionRequestActionRedeploy:
		return projectcontract.ProjectAuditActionBatchRedeploy
	case generated.ProjectBatchActionRequestActionUpdateDeploy:
		return projectcontract.ProjectAuditActionBatchUpdateDeploy
	case generated.ProjectBatchActionRequestActionUnregister:
		return projectcontract.ProjectAuditActionBatchUnregister
	case generated.ProjectBatchActionRequestActionDestroy:
		return projectcontract.ProjectAuditActionBatchDestroy
	default:
		return projectcontract.AuditAction(strings.TrimSpace(string(action)))
	}
}

func (s *Service) publishAuditEvent(ctx context.Context, event moduleapi.AuditEvent, failureMessage string) {
	if s == nil || s.auditBus == nil {
		return
	}
	if publishErr := s.auditBus.Publish(ctx, eventbus.Event{
		Name:    string(moduleapi.AuditRecordEventName),
		Source:  s.auditModuleName(),
		Payload: event,
	}); publishErr != nil && s.logger != nil {
		s.logger.Warn(failureMessage,
			zap.String("module", s.auditModuleName()),
			zap.String("action", event.Action),
			zap.Error(publishErr),
		)
	}
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
