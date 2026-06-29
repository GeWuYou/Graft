package container

import (
	"context"
	"strings"

	"graft/server/internal/httpx"
	containercontract "graft/server/modules/container/contract"
)

func (s *service) batchActionPolicyFailure(
	ctx context.Context,
	ref Ref,
	action string,
	options ActionOptions,
) (BatchActionItem, bool) {
	if !isSupportedAction(action) {
		return BatchActionItem{}, false
	}
	runtime, err := s.runtimeForRequest()
	if err != nil {
		return batchActionFailure(ref.Value, action, err), true
	}
	policy := s.effectiveActionPolicy(ctx)
	detail, detailErr := runtime.Detail(ctx, ref)
	orchestrator := actionAuditOrchestrator(detail, detailErr)
	orchestratorType := effectiveActionAuditOrchestratorType(orchestrator, detailErr)
	if policy.singleBlockedFor(orchestratorType) || policy.batchBlockedFor(orchestratorType) {
		result := blockedActionAuditResult(ref, detail, action, orchestrator)
		s.publishActionAudit(ctx, result, options, errDangerousActionsDisabled)
		return batchActionItem(ref.Value, action, result, errDangerousActionsDisabled), true
	}
	return BatchActionItem{}, false
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

// blockedActionAuditResult 生成用于记录被阻止动作的结果信息。
// 返回包含容器标识、镜像、动作、运行时和编排器信息的 ActionResult。
func blockedActionAuditResult(ref Ref, detail Detail, action string, orchestrator OrchestratorInfo) ActionResult {
	return ActionResult{
		ID:           firstNonEmpty(ref.Value, detail.ID),
		Name:         detail.Name,
		Image:        detail.Image,
		Action:       action,
		Runtime:      runtimeNameDocker,
		Orchestrator: orchestrator,
	}
}

// shouldBackfillActionAuditOrchestrator 判断是否需要回填动作审计中的编排器信息。
// 当详情读取成功但当前编排器字段仍全部为空时，返回 true。
func shouldBackfillActionAuditOrchestrator(orchestrator OrchestratorInfo, detailErr error) bool {
	if detailErr != nil {
		return false
	}
	return orchestrator.Type == "" &&
		orchestrator.GroupScopeKind == "" &&
		orchestrator.MemberScopeKind == "" &&
		orchestrator.GroupValue == "" &&
		orchestrator.MemberValue == ""
}

func (s *service) requireRuntimeAccess(ctx context.Context) error {
	if s == nil || !s.runtimeAccessEnabled(ctx) {
		return errRuntimeDisabled
	}
	return nil
}

func runWithRuntime(ctx context.Context, ref Ref, action string, options ActionOptions, runtime Runtime) (ActionResult, error) {
	switch action {
	case containerActionStart:
		return runtime.Start(ctx, ref)
	case containerActionStop:
		return runtime.Stop(ctx, ref)
	case containerActionRemove:
		return runtime.Remove(ctx, ref, RemoveOptions(options))
	case containerActionRestart:
		return runtime.Restart(ctx, ref)
	default:
		return ActionResult{ID: ref.Value, Action: action, Runtime: runtimeNameDocker}, errInvalidBatchAction
	}
}

func withActionMessage(result ActionResult) ActionResult {
	if result.MessageKey != "" {
		return result
	}
	key := actionSuccessMessageKey(result.Action)
	result.MessageKey = key.String()
	result.Message = key.String()
	return result
}

func actionSuccessMessageKey(action string) containercontract.MessageKey {
	switch action {
	case containerActionStart:
		return containercontract.ContainerActionStartCompleted
	case containerActionStop:
		return containercontract.ContainerActionStopCompleted
	case containerActionRemove:
		return containercontract.ContainerActionRemoveCompleted
	default:
		return containercontract.ContainerActionRestartCompleted
	}
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

func batchActionFailure(id string, action string, err error) BatchActionItem {
	messageKey := messageKeyForError(err).String()
	return BatchActionItem{
		ID:         id,
		Action:     action,
		Success:    false,
		ErrorCode:  messageKey,
		MessageKey: messageKey,
		Message:    fallbackMessageForError(err),
		Result: ActionResult{
			ID:      id,
			Action:  action,
			Runtime: runtimeNameDocker,
		},
	}
}

func batchActionItem(id string, action string, result ActionResult, err error) BatchActionItem {
	if err != nil {
		if result.ID == "" {
			result.ID = id
		}
		if result.Action == "" {
			result.Action = action
		}
		if result.Runtime == "" {
			result.Runtime = runtimeNameDocker
		}
		item := batchActionFailure(firstNonEmpty(result.ID, id), result.Action, err)
		item.Name = result.Name
		item.Result = result
		return item
	}
	return BatchActionItem{
		ID:         firstNonEmpty(result.ID, id),
		Name:       result.Name,
		Action:     result.Action,
		Success:    true,
		MessageKey: result.MessageKey,
		Message:    result.Message,
		Result:     result,
	}
}

func withBatchActionMessage(result BatchActionResult) BatchActionResult {
	key := containercontract.ContainerBatchActionCompleted
	switch {
	case result.SuccessCount == 0:
		key = containercontract.ContainerBatchActionFailed
	case result.FailedCount > 0:
		key = containercontract.ContainerBatchActionPartial
	}
	result.MessageKey = key.String()
	result.Message = key.String()
	return result
}

func requestIDFromContext(ctx context.Context) string {
	if requestAudit, ok := httpx.RequestAuditContextFromContext(ctx); ok {
		return requestAudit.RequestID
	}
	return ""
}
