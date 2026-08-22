package container

import (
	"context"
	"strings"

	"graft/server/internal/httpx"
	"graft/server/internal/moduleapi"
)

func (s *service) lifecycleActionPolicyFailure(
	ctx context.Context,
	ref Ref,
	action string,
	options ActionOptions,
) (BatchLifecycleActionItem, bool) {
	if !isSupportedAction(action) {
		return BatchLifecycleActionItem{}, false
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		item := batchLifecycleActionFailure(ref.Value, action, err)
		s.publishLifecycleTaskSubmissionAudit(ctx, ref, action, options, moduleapi.TaskReceipt{}, err)
		return item, true
	}
	policy := s.effectiveActionPolicy(ctx)
	detail, detailErr := runtime.Detail(ctx, ref)
	orchestrator := actionAuditOrchestrator(detail, detailErr)
	orchestratorType := effectiveActionAuditOrchestratorType(orchestrator, detailErr)
	if policy.singleBlockedFor(orchestratorType) || policy.batchBlockedFor(orchestratorType) {
		result := blockedActionAuditResult(ref, detail, action, orchestrator)
		s.publishActionAudit(ctx, result, options, errDangerousActionsDisabled)
		return batchLifecycleActionFailure(ref.Value, action, errDangerousActionsDisabled), true
	}
	return BatchLifecycleActionItem{}, false
}

// actionAuditOrchestrator 在详情获取失败时返回空的编排器信息，否则返回容器详情中的编排器信息。
func actionAuditOrchestrator(detail Detail, detailErr error) OrchestratorInfo {
	if detailErr != nil {
		return OrchestratorInfo{}
	}
	return detail.Orchestrator
}

// effectiveActionAuditOrchestratorType 返回用于动作审计的编排器类型；当获取容器详情失败时返回未知类型。
func effectiveActionAuditOrchestratorType(orchestrator OrchestratorInfo, detailErr error) string {
	if detailErr != nil {
		return containerOrchestratorUnknown
	}
	return effectiveOrchestratorType(Summary{Orchestrator: orchestrator})
}

// blockedActionAuditResult 生成用于记录被阻止动作的容器快照。
func blockedActionAuditResult(ref Ref, detail Detail, action string, orchestrator OrchestratorInfo) actionAuditSnapshot {
	return actionAuditSnapshot{
		ID:           firstNonEmpty(ref.Value, detail.ID),
		Name:         detail.Name,
		Image:        detail.Image,
		Action:       action,
		Runtime:      runtimeNameDocker,
		Orchestrator: orchestrator,
	}
}

func (s *service) requireRuntimeAccess(ctx context.Context) error {
	if s == nil || !s.runtimeAccessEnabled(ctx) {
		return errRuntimeDisabled
	}
	return nil
}

func normalizeBatchActionCommand(command BatchActionCommand) (BatchActionCommand, error) {
	action := strings.TrimSpace(command.Action)
	if !isSupportedAction(action) {
		return BatchActionCommand{}, errInvalidBatchAction
	}
	if len(command.IDs) == 0 || len(command.IDs) > maxContainerBatchActionIDs {
		return BatchActionCommand{}, errInvalidBatchAction
	}
	normalizedIDs := make([]string, 0, len(command.IDs))
	for _, id := range command.IDs {
		if strings.TrimSpace(id) == "" {
			return BatchActionCommand{}, errInvalidBatchAction
		}
		normalizedIDs = append(normalizedIDs, strings.TrimSpace(id))
	}
	return BatchActionCommand{Action: action, IDs: normalizedIDs, Force: command.Force}, nil
}

func isSupportedAction(action string) bool {
	switch action {
	case containerActionStart, containerActionStop, containerActionRestart, containerActionRemove:
		return true
	default:
		return false
	}
}

func requestIDFromContext(ctx context.Context) string {
	if requestAudit, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		return requestAudit.RequestID
	}
	return ""
}
