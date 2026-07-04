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
	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

const (
	projectResourceType      = "project"
	projectBatchResourceType = "project_batch"
	projectAuditTimeout      = 3 * time.Second
)

var (
	projectAuditActions = map[generated.ProjectActionResponseAction]projectcontract.AuditAction{
		generated.ProjectActionResponseActionProjectActionUp:           projectcontract.ProjectAuditActionUp,
		generated.ProjectActionResponseActionProjectActionDown:         projectcontract.ProjectAuditActionDown,
		generated.ProjectActionResponseActionProjectActionRestart:      projectcontract.ProjectAuditActionRestart,
		generated.ProjectActionResponseActionProjectActionRedeploy:     projectcontract.ProjectAuditActionRedeploy,
		generated.ProjectActionResponseActionProjectActionUpdateDeploy: projectcontract.ProjectAuditActionUpdateDeploy,
		generated.ProjectActionResponseActionProjectActionUnregister:   projectcontract.ProjectAuditActionUnregister,
		generated.ProjectActionResponseActionProjectActionDestroy:      projectcontract.ProjectAuditActionDestroy,
	}
	projectBatchAuditActions = map[generated.ProjectBatchActionRequestAction]projectcontract.AuditAction{
		generated.ProjectBatchActionRequestActionStart:        projectcontract.ProjectAuditActionBatchStart,
		generated.ProjectBatchActionRequestActionStop:         projectcontract.ProjectAuditActionBatchStop,
		generated.ProjectBatchActionRequestActionRestart:      projectcontract.ProjectAuditActionBatchRestart,
		generated.ProjectBatchActionRequestActionRedeploy:     projectcontract.ProjectAuditActionBatchRedeploy,
		generated.ProjectBatchActionRequestActionUpdateDeploy: projectcontract.ProjectAuditActionBatchUpdateDeploy,
		generated.ProjectBatchActionRequestActionUnregister:   projectcontract.ProjectAuditActionBatchUnregister,
		generated.ProjectBatchActionRequestActionDestroy:      projectcontract.ProjectAuditActionBatchDestroy,
	}
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
	metadata := projectActionAuditMetadata(aggregate, result)
	s.publishProjectAudit(ctx, false, "publish project audit event failed", func(auditCtx context.Context) moduleapi.AuditEvent {
		enrichProjectAuditMetadata(auditCtx, metadata)
		event := newProjectAuditEvent(
			auditCtx,
			actor,
			projectAuditAction(result.Action).String(),
			projectResourceType,
			strconv.FormatUint(aggregate.Project.ID, 10),
			aggregate.Project.CanonicalProjectName,
		)
		event.StatusCode = projectActionAuditStatusCode(result, actionErr)
		event.Success = actionErr == nil && result.Result == generated.ProjectActionResponseResultProjectActionResultCompleted
		event.MessageKey = trimmedStringValue(result.MessageKey)
		event.Message = trimmedStringValue(result.Message)
		event.Metadata = metadata
		return event
	})
}

func (s *Service) publishProjectBatchAudit(
	ctx context.Context,
	actor actionActor,
	request BatchActionRequest,
	result BatchActionResult,
) {
	resourceID := batchAuditResourceID(request.Action, time.Now().UTC())
	metadata := projectBatchAuditMetadata(request, result)
	s.publishProjectAudit(ctx, true, "publish project batch audit event failed", func(auditCtx context.Context) moduleapi.AuditEvent {
		enrichProjectAuditMetadata(auditCtx, metadata)
		event := newProjectAuditEvent(
			auditCtx,
			actor,
			projectBatchAuditAction(request.Action).String(),
			projectBatchResourceType,
			resourceID,
			strings.TrimSpace(string(request.Action))+" x"+strconv.Itoa(result.TotalCount),
		)
		event.StatusCode = projectBatchAuditStatusCode(result)
		event.Success = result.BlockedCount == 0
		event.Metadata = metadata
		return event
	})
}

func (s *Service) publishProjectAudit(
	ctx context.Context,
	async bool,
	failureMessage string,
	build func(context.Context) moduleapi.AuditEvent,
) {
	auditCtx, cancel, ok := s.prepareAuditContext(ctx)
	if !ok {
		return
	}
	event := build(auditCtx)
	if async {
		s.publishDetachedAuditEvent(auditCtx, cancel, event, failureMessage)
		return
	}
	defer cancel()
	s.publishAuditEvent(auditCtx, event, failureMessage)
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

func projectBatchAuditMetadata(request BatchActionRequest, result BatchActionResult) map[string]any {
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
	return collectBatchProjectIDs(items, func(item BatchActionItemResult) bool {
		return item.Result == generated.ProjectActionResponseResultProjectActionResultBlocked
	})
}

func batchSkippedProjectIDs(items []BatchActionItemResult) []uint64 {
	return collectBatchProjectIDs(items, func(item BatchActionItemResult) bool {
		return item.Skipped
	})
}

func collectBatchProjectIDs(items []BatchActionItemResult, include func(BatchActionItemResult) bool) []uint64 {
	ids := make([]uint64, 0, len(items))
	for _, item := range items {
		if include(item) {
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
	return resolveAuditAction(action, projectAuditActions)
}

func projectBatchAuditAction(action generated.ProjectBatchActionRequestAction) projectcontract.AuditAction {
	return resolveAuditAction(action, projectBatchAuditActions)
}

func resolveAuditAction[T ~string](action T, mappings map[T]projectcontract.AuditAction) projectcontract.AuditAction {
	if resolved, ok := mappings[action]; ok {
		return resolved
	}
	return projectcontract.AuditAction(strings.TrimSpace(string(action)))
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
