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
	"graft/server/internal/event"
	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

const (
	applicationResourceType      = "application"
	applicationBatchResourceType = "application_batch"
	projectAuditTimeout          = 3 * time.Second
)

var (
	projectAuditActions = map[generated.ApplicationActionResponseAction]projectcontract.AuditAction{
		generated.ApplicationActionResponseActionApplicationActionUp:         projectcontract.ApplicationAuditActionUp,
		generated.ApplicationActionResponseActionApplicationActionStop:       projectcontract.ApplicationAuditActionStop,
		generated.ApplicationActionResponseActionApplicationActionRestart:    projectcontract.ApplicationAuditActionRestart,
		generated.ApplicationActionResponseActionApplicationActionRedeploy:   projectcontract.ApplicationAuditActionRedeploy,
		generated.ApplicationActionResponseActionApplicationActionUnregister: projectcontract.ApplicationAuditActionUnregister,
		generated.ApplicationActionResponseActionApplicationActionDestroy:    projectcontract.ApplicationAuditActionDestroy,
	}
	projectBatchAuditActions = map[generated.ApplicationBatchActionRequestAction]projectcontract.AuditAction{
		generated.ApplicationBatchActionRequestActionStart:      projectcontract.ApplicationAuditActionBatchStart,
		generated.ApplicationBatchActionRequestActionStop:       projectcontract.ApplicationAuditActionBatchStop,
		generated.ApplicationBatchActionRequestActionRestart:    projectcontract.ApplicationAuditActionBatchRestart,
		generated.ApplicationBatchActionRequestActionRedeploy:   projectcontract.ApplicationAuditActionBatchRedeploy,
		generated.ApplicationBatchActionRequestActionUnregister: projectcontract.ApplicationAuditActionBatchUnregister,
		generated.ApplicationBatchActionRequestActionDestroy:    projectcontract.ApplicationAuditActionBatchDestroy,
	}
)

type actionActor struct {
	id       uint64
	operator *moduleapi.CurrentUser
}

func (s *Service) requireActionActor(
	ctx context.Context,
	projectID uint64,
	action generated.ApplicationActionResponseAction,
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
	action generated.ApplicationActionResponseAction,
	reason string,
) ActionResult {
	return blockedActionResult(projectID, action, []GuardResult{guardDetail("actor_attribution_required", reason)})
}

func (s *Service) publishApplicationActionAudit(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
	actor actionActor,
	result ActionResult,
	actionErr error,
) {
	result.ApplicationID = aggregate.Application.ApplicationID
	metadata := projectActionAuditMetadata(aggregate, result)
	s.publishProjectAudit(ctx, false, "publish project audit event failed", func(auditCtx context.Context) moduleapi.AuditEvent {
		enrichProjectAuditMetadata(auditCtx, metadata)
		event := newProjectAuditEvent(
			auditCtx,
			actor,
			projectAuditAction(result.Action).String(),
			applicationResourceType,
			aggregate.Application.ApplicationID,
			aggregate.Application.DisplayName,
		)
		event.StatusCode = projectActionAuditStatusCode(result, actionErr)
		event.Success = actionErr == nil && result.Result == generated.ApplicationActionResponseResultApplicationActionResultCompleted
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
			applicationBatchResourceType,
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

func projectActionAuditMetadata(aggregate projectstore.ApplicationAggregate, result ActionResult) map[string]any {
	metadata := map[string]any{
		"application_id":          aggregate.Application.ApplicationID,
		"application_record_id":   aggregate.Application.ApplicationRecordID,
		"compose_project_name":    aggregate.Application.ComposeProjectName,
		"display_name":            aggregate.Application.DisplayName,
		"deployment_adapter_kind": projectcontract.DeploymentAdapterKindCompose.String(),
		"source_type":             aggregate.Application.SourceType,
		"ownership_mode":          aggregate.Application.OwnershipMode,
		"workspace_path":          aggregate.Application.WorkspacePath,
		"action_result":           result.Result,
		"guard_results":           guardResultsAuditMetadata(result.GuardResults),
	}
	return metadata
}

func projectBatchAuditMetadata(request BatchActionRequest, result BatchActionResult) map[string]any {
	metadata := map[string]any{
		"batch":                     true,
		"requested_total":           result.TotalCount,
		"requested_application_ids": batchApplicationIDs(result.Items),
		"completed_count":           result.CompletedCount,
		"blocked_count":             result.BlockedCount,
		"skipped_count":             result.SkippedCount,
		"blocked_application_ids":   batchBlockedApplicationIDs(result.Items),
		"skipped_application_ids":   batchSkippedApplicationIDs(result.Items),
		"remove_named_volumes":      request.RemoveNamedVolumes,
		"auto_unregister":           request.AutoUnregister,
		"image_prune":               request.ImagePrune,
		"delete_workspace_path":     request.DeleteWorkspacePath,
	}
	if request.ConfirmComposeProjectName != nil {
		metadata["confirm_compose_project_name"] = strings.TrimSpace(*request.ConfirmComposeProjectName)
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

func batchApplicationIDs(items []BatchActionItemResult) []string {
	return collectBatchApplicationIDs(items, func(BatchActionItemResult) bool { return true })
}

func batchBlockedApplicationIDs(items []BatchActionItemResult) []string {
	return collectBatchApplicationIDs(items, func(item BatchActionItemResult) bool {
		return item.Result == generated.ApplicationActionResponseResultApplicationActionResultBlocked
	})
}

func batchSkippedApplicationIDs(items []BatchActionItemResult) []string {
	return collectBatchApplicationIDs(items, func(item BatchActionItemResult) bool {
		return item.Skipped
	})
}

func collectBatchApplicationIDs(items []BatchActionItemResult, include func(BatchActionItemResult) bool) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if include(item) {
			ids = append(ids, item.ApplicationID)
		}
	}
	return ids
}

// 冲突及操作阻止类错误返回 409；其他错误返回 500。
func projectActionAuditStatusCode(result ActionResult, err error) int {
	if err == nil && result.Result == generated.ApplicationActionResponseResultApplicationActionResultCompleted {
		return http.StatusOK
	}
	switch {
	case errors.Is(err, errProjectInvalidArgument), errors.Is(err, errProjectInvalidCanonicalName), errors.Is(err, errProjectFileNotFound):
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

func projectAuditAction(action generated.ApplicationActionResponseAction) projectcontract.AuditAction {
	return resolveAuditAction(action, projectAuditActions)
}

func projectBatchAuditAction(action generated.ApplicationBatchActionRequestAction) projectcontract.AuditAction {
	return resolveAuditAction(action, projectBatchAuditActions)
}

func resolveAuditAction[T ~string](action T, mappings map[T]projectcontract.AuditAction) projectcontract.AuditAction {
	if resolved, ok := mappings[action]; ok {
		return resolved
	}
	return projectcontract.AuditAction(strings.TrimSpace(string(action)))
}

func (s *Service) publishAuditEvent(ctx context.Context, payload moduleapi.AuditEvent, failureMessage string) {
	if s == nil || s.auditPublisher == nil {
		return
	}
	envelope, encodeErr := httpx.NewAuditEvent(s.auditModuleName(), payload)
	if encodeErr != nil {
		if s.logger != nil {
			s.logger.Warn(failureMessage,
				zap.String("module", s.auditModuleName()),
				zap.String("action", payload.Action),
				zap.Error(encodeErr),
			)
		}
		return
	}
	if _, publishErr := s.auditPublisher.Publish(ctx, envelope, event.PublishOptions{Delivery: event.DeliveryDurable}); publishErr != nil && s.logger != nil {
		s.logger.Warn(failureMessage,
			zap.String("module", s.auditModuleName()),
			zap.String("action", payload.Action),
			zap.Error(publishErr),
		)
	}
}
