package project

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"graft/server/internal/apperror"
	"graft/server/internal/contract/errorcode"
	messagecontract "graft/server/internal/contract/message"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/logger"
	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

// Up 异步提交应用启动意图，由 Task Runtime 通过外部执行租约调度。
func (s *Service) Up(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.submitLifecycleTask(ctx, projectID, actorID, generated.ApplicationActionResponseActionApplicationActionUp)
}

// Stop 异步提交应用停止意图，由 Task Runtime 通过外部执行租约调度。
func (s *Service) Stop(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.submitLifecycleTask(ctx, projectID, actorID, generated.ApplicationActionResponseActionApplicationActionStop)
}

// Restart 异步提交 Compose 恢复任务；已发现成员时执行 restart，准确确认无成员时执行 up -d。
func (s *Service) Restart(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.submitLifecycleTask(ctx, projectID, actorID, generated.ApplicationActionResponseActionApplicationActionRestart)
}

// Redeploy 按已保存的阶段配置异步提交 Compose 重部署任务；阶段顺序由任务计划固定。
func (s *Service) Redeploy(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.submitLifecycleTask(ctx, projectID, actorID, generated.ApplicationActionResponseActionApplicationActionRedeploy)
}

// BatchAction 为多个项目提交同一种生命周期动作，并返回逐项目结果；单项失败不会隐藏其它项目结果。
func (s *Service) BatchAction(ctx context.Context, request BatchActionRequest) (BatchActionResult, error) {
	actor, blocked, err := s.requireBatchActor(ctx, request)
	if err != nil {
		return blocked, nil
	}
	items := make([]BatchActionItemResult, 0, len(request.ApplicationRecordIDs))
	result := BatchActionResult{TotalCount: len(request.ApplicationRecordIDs)}
	for _, projectID := range request.ApplicationRecordIDs {
		item, itemErr := s.batchActionItem(ctx, projectID, request, actor)
		if itemErr != nil && !hasBatchActionItemResult(item) {
			return BatchActionResult{}, itemErr
		}
		items = append(items, item)
		switch {
		case item.Skipped:
			result.SkippedCount++
		case item.Result == generated.ApplicationActionResponseResultApplicationActionResultCompleted:
			result.CompletedCount++
		case item.Result == generated.ApplicationActionResponseResultApplicationActionResultAccepted:
			result.AcceptedCount++
		default:
			result.BlockedCount++
		}
	}
	result.Items = items
	s.publishProjectBatchAudit(ctx, actor, request, result)
	return result, nil
}

// Unregister 移除项目注册记录但不触碰宿主机文件。
func (s *Service) Unregister(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ActionResult{}, err
	}
	if err := s.ensureApplicationScope(ctx, aggregate, projectcontract.ApplicationDestroyPermission.String()); err != nil {
		return ActionResult{}, err
	}
	actor, blocked, err := s.requireActionActor(ctx, projectID, generated.ApplicationActionResponseActionApplicationActionUnregister, actorID)
	if err != nil {
		return blocked, err
	}
	result, actionErr := s.unregisterWithActor(ctx, aggregate, actor)
	s.publishApplicationActionAudit(ctx, aggregate, actor, result, actionErr)
	return result, actionErr
}

// Destroy 先执行受保护的拆除步骤，再注销项目记录；保护条件失败时不会改变宿主机或注册表状态。
func (s *Service) Destroy(ctx context.Context, projectID uint64, request DestroyRequest) (ActionResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ActionResult{}, err
	}
	if err := s.ensureApplicationScope(ctx, aggregate, projectcontract.ApplicationDestroyPermission.String()); err != nil {
		return ActionResult{}, err
	}
	actor, blocked, err := s.requireActionActor(ctx, projectID, generated.ApplicationActionResponseActionApplicationActionDestroy, request.ActorID)
	if err != nil {
		return blocked, err
	}
	if result, blockErr := validateDestroyRequest(projectID, aggregate, request); blockErr != nil {
		s.publishApplicationActionAudit(ctx, aggregate, actor, result, blockErr)
		return result, blockErr
	}
	result, actionErr := s.submitDestroyTask(ctx, aggregate, request, actor)
	s.publishApplicationActionAudit(ctx, aggregate, actor, result, actionErr)
	return result, actionErr
}

// validateDestroyRequest 校验销毁请求是否允许继续执行，并在违反保护条件时返回阻断结果。
// 当确认名称不匹配、请求删除命名卷，或请求删除工作目录但项目并非受控根专属所有权时，
// 返回带有相应守卫结果的阻断动作结果和销毁阻断错误。
// @param projectID 项目标识。
// @param aggregate 项目聚合数据。
// @param request 销毁请求。
// @returns 允许继续销毁时返回空动作结果和 nil；否则返回阻断动作结果和销毁阻断错误。
func validateDestroyRequest(
	projectID uint64,
	aggregate projectstore.ApplicationAggregate,
	request DestroyRequest,
) (ActionResult, error) {
	guardResults := []GuardResult{}
	if aggregate.Application.ApplicationID == "" || strings.TrimSpace(request.ConfirmComposeProjectName) != aggregate.Application.ApplicationID {
		return blockedActionResult(projectID, generated.ApplicationActionResponseActionApplicationActionDestroy, append(guardResults, guardCode("confirm_compose_project_name_mismatch"))), errProjectDestroyBlocked
	}
	guardResults = append(guardResults, guardCode("confirm_compose_project_name_matched"))

	if request.RemoveNamedVolumes {
		guardResults = append(guardResults, guardCode("remove_named_volumes_requested"))
	}

	if request.DeleteWorkspacePath && aggregate.Application.OwnershipMode != projectcontract.OwnershipModeManagedRootDedicated.String() {
		guardResults = append(guardResults, guardDetail("delete_workspace_path_blocked", "ownership_mode_external"))
		return blockedActionResult(projectID, generated.ApplicationActionResponseActionApplicationActionDestroy, guardResults), errProjectDestroyBlocked
	}
	return ActionResult{}, nil
}

func (s *Service) submitDestroyTask(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
	request DestroyRequest,
	actor actionActor,
) (ActionResult, error) {
	return s.submitLifecycleTaskWithActor(ctx, aggregate, actor, generated.ApplicationActionResponseActionApplicationActionDestroy, &request)
}

// UnsupportedLifecycleAction 返回明确标记为当前阶段阻断的生命周期动作结果。
func (s *Service) UnsupportedLifecycleAction(projectID uint64, action generated.ApplicationActionResponseAction) (ActionResult, error) {
	return ActionResult{
		ApplicationRecordID: projectID,
		Action:              action,
		Result:              generated.ApplicationActionResponseResultApplicationActionResultBlocked,
		MessageKey:          stringPointer(projectcontract.ApplicationLifecycleAccepted.String()),
		Message:             stringPointer(projectcontract.ApplicationLifecycleAccepted.String()),
		GuardResults:        []GuardResult{guardDetail("batch-2-scope", "lifecycle execution is deferred to phase-1-batch-3")},
	}, errProjectUnsupportedLifecycle
}

func (s *Service) submitLifecycleTask(ctx context.Context, projectID uint64, actorID *uint64, action generated.ApplicationActionResponseAction) (ActionResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ActionResult{}, err
	}
	if err := s.ensureApplicationScope(ctx, aggregate, projectcontract.ApplicationLifecyclePermission.String()); err != nil {
		return ActionResult{}, err
	}
	actor, blocked, err := s.requireActionActor(ctx, projectID, action, actorID)
	if err != nil {
		return blocked, err
	}
	result, actionErr := s.submitLifecycleTaskWithActor(ctx, aggregate, actor, action, nil)
	s.publishApplicationActionAudit(ctx, aggregate, actor, result, actionErr)
	return result, actionErr
}

func (s *Service) submitLifecycleTaskWithActor(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
	actor actionActor,
	action generated.ApplicationActionResponseAction,
	destroy *DestroyRequest,
) (ActionResult, error) {
	if err := ensureProjectLifecycleReady(aggregate); err != nil {
		result := lifecycleBlockedResult(aggregate, action, err)
		return result, err
	}
	target, err := s.lifecycleExecutionTarget(ctx, aggregate)
	if err != nil {
		result := lifecycleBlockedResult(aggregate, action, err)
		return result, err
	}
	if s.taskService == nil {
		err := errors.New("task service is unavailable")
		err = s.reportLifecycleTaskSubmissionFailure(ctx, aggregate, action, err)
		result := lifecycleBlockedResult(aggregate, action, err)
		return result, err
	}
	plan, err := lifecycleTaskPlan(aggregate, action, target, actor.id, destroy)
	if err != nil {
		result := lifecycleBlockedResult(aggregate, action, err)
		return result, err
	}
	receipt, err := s.taskService.Submit(ctx, moduleapi.SubmitTaskInput{Type: moduleapi.TaskType("application.compose." + strings.ToLower(string(action))), Owner: moduleapi.TaskOwner{Type: applicationTaskOwnerType, ID: aggregate.Application.ApplicationID}, RequestedBy: actor.id, Plan: plan})
	if err != nil {
		if errors.Is(err, moduleapi.ErrTaskOwnerBusy) {
			result := lifecycleBlockedResult(aggregate, action, err)
			return result, err
		}
		err = s.reportLifecycleTaskSubmissionFailure(ctx, aggregate, action, err)
		result := lifecycleBlockedResult(aggregate, action, err)
		return result, err
	}
	messageKey := projectcontract.ApplicationLifecycleAccepted.String()
	return ActionResult{ApplicationRecordID: aggregate.Application.ApplicationRecordID, Action: action, Result: generated.ApplicationActionResponseResultApplicationActionResultAccepted, MessageKey: &messageKey, Message: &messageKey, GuardResults: []GuardResult{guardDetail("task_id", fmt.Sprintf("%d", receipt.TaskID))}}, nil
}

// reportLifecycleTaskSubmissionFailure 在 Project 仍掌握应用和动作语义时记录一次任务提交失败。
func (s *Service) reportLifecycleTaskSubmissionFailure(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
	action generated.ApplicationActionResponseAction,
	err error,
) error {
	typedErr := apperror.Wrap(err, apperror.Descriptor{
		Kind:       apperror.KindInternal,
		Code:       errorcode.CommonInternalError,
		MessageKey: messagecontract.CommonInternalError,
	})
	return logger.ReportError(ctx, s.appLogger, "submit application lifecycle task failed", typedErr,
		logger.StringField(logger.FieldOperation, "submit_application_lifecycle_task"),
		logger.StringField("application_record_id", fmt.Sprintf("%d", aggregate.Application.ApplicationRecordID)),
		logger.StringField("application_id", aggregate.Application.ApplicationID),
		logger.StringField("lifecycle_action", string(action)),
		logger.StringField("task_type", "application.compose."+strings.ToLower(string(action))),
	)
}

// lifecycleTaskPlan 为指定生命周期动作创建异步任务计划，实际 Compose 执行由任务运行时负责。
func lifecycleTaskPlan(
	aggregate projectstore.ApplicationAggregate,
	action generated.ApplicationActionResponseAction,
	target moduleapi.ComposeRuntimeTargetSummary,
	actorID uint64,
	destroy *DestroyRequest,
) (moduleapi.TaskPlan, error) {
	if aggregate.Snapshot == nil {
		return moduleapi.TaskPlan{}, errProjectUnsupportedLifecycle
	}
	config := lifecycleConfigurationFromAggregate(aggregate)
	policy := composeExecutionPolicy{
		SnapshotDigest:     aggregate.Snapshot.ConfigHash,
		BuildBeforeUp:      config.Standard.BuildBeforeUp,
		ForceRecreate:      config.Standard.ForceRecreate,
		RemoveOrphans:      config.Standard.RemoveOrphans,
		WaitAfterUp:        config.Standard.WaitAfterUp,
		WaitTimeoutSeconds: config.Standard.WaitTimeoutSeconds,
		RenewAnonVolumes:   config.Standard.RenewAnonVolumes,
		ActorID:            actorID,
	}
	switch action {
	case generated.ApplicationActionResponseActionApplicationActionRedeploy:
		return redeployTaskPlan(aggregate, target, policy, config)
	case generated.ApplicationActionResponseActionApplicationActionDestroy:
		if destroy == nil {
			return moduleapi.TaskPlan{}, errProjectInvalidArgument
		}
		return destroyTaskPlan(aggregate, target, policy, *destroy)
	case generated.ApplicationActionResponseActionApplicationActionUp,
		generated.ApplicationActionResponseActionApplicationActionStop,
		generated.ApplicationActionResponseActionApplicationActionRestart:
		return taskPlanWithExternalStage(aggregate, target, strings.ToLower(string(action)), policy)
	default:
		return moduleapi.TaskPlan{}, errProjectInvalidArgument
	}
}

// redeployTaskPlan 按固定顺序构建重部署阶段，并根据配置追加停止、拉取和镜像清理等可选阶段。
func redeployTaskPlan(aggregate projectstore.ApplicationAggregate, target moduleapi.ComposeRuntimeTargetSummary, policy composeExecutionPolicy, config LifecycleConfiguration) (moduleapi.TaskPlan, error) {
	stages := make([]moduleapi.StagePlan, 0, projectLifecycleStageCapacity)
	if config.Standard.DownBeforeRedeploy {
		action := "down"
		if !lifecycleManagesAllServices(config) {
			action = "stop"
		}
		if err := appendExternalTaskStage(&stages, aggregate, target, action, policy); err != nil {
			return moduleapi.TaskPlan{}, err
		}
	}
	if config.Standard.PullBeforeRedeploy {
		if err := appendExternalTaskStage(&stages, aggregate, target, "pull", policy); err != nil {
			return moduleapi.TaskPlan{}, err
		}
	}
	if err := appendExternalTaskStage(&stages, aggregate, target, "up", policy); err != nil {
		return moduleapi.TaskPlan{}, err
	}
	if config.Standard.PruneImagesAfterRedeploy {
		if err := appendExternalTaskStage(&stages, aggregate, target, "image-prune", policy); err != nil {
			return moduleapi.TaskPlan{}, err
		}
	}
	return moduleapi.TaskPlan{Stages: stages}, nil
}

func destroyTaskPlan(
	aggregate projectstore.ApplicationAggregate,
	target moduleapi.ComposeRuntimeTargetSummary,
	policy composeExecutionPolicy,
	request DestroyRequest,
) (moduleapi.TaskPlan, error) {
	policy.RemoveNamedVolumes = request.RemoveNamedVolumes
	policy.DeleteWorkspacePath = request.DeleteWorkspacePath
	policy.AutoUnregister = request.AutoUnregister
	stages := make([]moduleapi.StagePlan, 0, projectDestroyStageCapacity)
	if err := appendExternalTaskStage(&stages, aggregate, target, "down", policy); err != nil {
		return moduleapi.TaskPlan{}, err
	}
	if request.ImagePrune {
		if err := appendExternalTaskStage(&stages, aggregate, target, "image-prune", policy); err != nil {
			return moduleapi.TaskPlan{}, err
		}
	}
	input, err := json.Marshal(composeStageInput{ApplicationID: aggregate.Application.ApplicationID, Policy: cleanupExecutionPolicy(policy)})
	if err != nil {
		return moduleapi.TaskPlan{}, err
	}
	stages = append(stages, moduleapi.StagePlan{
		Key: "cleanup", ExecutorType: moduleapi.StageExecutorType(destroyCleanupStageType), Input: input,
		RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: 1}, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile,
	})
	return moduleapi.TaskPlan{Stages: stages}, nil
}

const (
	projectLifecycleStageCapacity    = 4
	projectDestroyStageCapacity      = 3
	projectExternalExecutionDeadline = 2 * time.Hour
)

func taskPlanWithExternalStage(aggregate projectstore.ApplicationAggregate, target moduleapi.ComposeRuntimeTargetSummary, action string, policy composeExecutionPolicy) (moduleapi.TaskPlan, error) {
	stages := make([]moduleapi.StagePlan, 0, 1)
	if err := appendExternalTaskStage(&stages, aggregate, target, action, policy); err != nil {
		return moduleapi.TaskPlan{}, err
	}
	return moduleapi.TaskPlan{Stages: stages}, nil
}

func appendExternalTaskStage(stages *[]moduleapi.StagePlan, aggregate projectstore.ApplicationAggregate, target moduleapi.ComposeRuntimeTargetSummary, action string, policy composeExecutionPolicy) error {
	if !composeExternalOperationAllowed(action) {
		return errProjectInvalidArgument
	}
	input, err := json.Marshal(composeStageInput{ApplicationID: aggregate.Application.ApplicationID, Policy: externalOperationPolicy(action, policy)})
	if err != nil {
		return err
	}
	*stages = append(*stages, moduleapi.StagePlan{
		Key: action, ExecutorType: moduleapi.StageExecutorType(composeStagePrefix + action), Input: input,
		RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: 1}, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile,
		ExternalExecution: &moduleapi.ExternalExecutionExpectation{
			RuntimeTargetID: target.ID, ProviderID: target.Provider, Capability: composeExecutionCapability,
			CapabilityVersion: composeExecutionCapabilityVersion, Protocol: composeExecutionProtocol,
			OperationID: "application.compose." + action + ".v1",
			LeaseTTL:    time.Minute, AbsoluteDeadline: projectExternalExecutionDeadline,
		},
	})
	return nil
}

func composeExternalOperationAllowed(action string) bool {
	switch action {
	case "down", "pull", "up", "stop", "restart", "image-prune":
		return true
	default:
		return false
	}
}

func externalOperationPolicy(action string, policy composeExecutionPolicy) composeExecutionPolicy {
	result := composeExecutionPolicy{SnapshotDigest: policy.SnapshotDigest}
	switch action {
	case "up":
		result.BuildBeforeUp = policy.BuildBeforeUp
		result.ForceRecreate = policy.ForceRecreate
		result.RemoveOrphans = policy.RemoveOrphans
		result.WaitAfterUp = policy.WaitAfterUp
		if policy.WaitAfterUp {
			result.WaitTimeoutSeconds = policy.WaitTimeoutSeconds
		}
		result.RenewAnonVolumes = policy.RenewAnonVolumes
	case "down":
		result.RemoveNamedVolumes = policy.RemoveNamedVolumes
	}
	return result
}

func cleanupExecutionPolicy(policy composeExecutionPolicy) composeExecutionPolicy {
	return composeExecutionPolicy{
		SnapshotDigest: policy.SnapshotDigest, DeleteWorkspacePath: policy.DeleteWorkspacePath,
		AutoUnregister: policy.AutoUnregister, ActorID: policy.ActorID,
	}
}

func (s *Service) lifecycleExecutionTarget(ctx context.Context, aggregate projectstore.ApplicationAggregate) (moduleapi.ComposeRuntimeTargetSummary, error) {
	if s == nil || s.runtimeTargets == nil {
		return moduleapi.ComposeRuntimeTargetSummary{}, errProjectRuntimeUnavailable
	}
	if aggregate.Application.RuntimeTargetID == nil || *aggregate.Application.RuntimeTargetID == 0 {
		return moduleapi.ComposeRuntimeTargetSummary{}, errProjectRuntimeUnavailable
	}
	if *aggregate.Application.RuntimeTargetID > uint64(^uint64(0)>>1) {
		return moduleapi.ComposeRuntimeTargetSummary{}, errProjectRuntimeUnavailable
	}
	id := int64(*aggregate.Application.RuntimeTargetID) // #nosec G115 -- bounded by max signed int64 immediately above.
	target, err := s.runtimeTargets.ReadComposeTarget(ctx, &id)
	if err != nil || !target.Available || strings.TrimSpace(target.Provider) == "" || !slices.Contains(target.Capabilities, composeExecutionCapability) {
		return moduleapi.ComposeRuntimeTargetSummary{}, errProjectRuntimeUnavailable
	}
	return target, nil
}

// ensureProjectLifecycleReady 检查项目是否满足执行生命周期操作的条件。
func ensureProjectLifecycleReady(aggregate projectstore.ApplicationAggregate) error {
	if aggregate.Snapshot == nil {
		return errProjectUnsupportedLifecycle
	}
	if err := lifecycleReviewGuard(aggregate); err != nil {
		return err
	}
	return nil
}

// blockedActionResult 返回一个标记为阻止的项目操作结果，并保留给定的守卫结果。
//
// @param projectID 项目 ID。
// @param action 操作类型。
// @param guardResults 守卫结果列表。
// @returns 标记为 blocked 的 ActionResult，包含项目 ID、操作类型、阻止消息以及守卫结果副本。
func blockedActionResult(projectID uint64, action generated.ApplicationActionResponseAction, guardResults []GuardResult) ActionResult {
	messageKey := projectcontract.ApplicationLifecycleBlocked.String()
	return ActionResult{
		ApplicationRecordID: projectID,
		Action:              action,
		Result:              generated.ApplicationActionResponseResultApplicationActionResultBlocked,
		MessageKey:          &messageKey,
		Message:             &messageKey,
		GuardResults:        append([]GuardResult(nil), guardResults...),
	}
}

func (s *Service) unregisterWithActor(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
	actor actionActor,
) (ActionResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ActionResult{}, err
	}
	if err := repository.UnregisterApplication(ctx, projectstore.UnregisterApplicationInput{
		ApplicationRecordID: aggregate.Application.ApplicationRecordID,
		ActorID:             &actor.id,
	}); err != nil {
		return blockedActionResult(aggregate.Application.ApplicationRecordID, generated.ApplicationActionResponseActionApplicationActionUnregister, []GuardResult{guardDetail("registry_delete_failed", "persistence_error")}), mapStoreError(err)
	}
	messageKey := projectcontract.ApplicationUnregisterCompleted.String()
	return ActionResult{
		ApplicationRecordID: aggregate.Application.ApplicationRecordID,
		Action:              generated.ApplicationActionResponseActionApplicationActionUnregister,
		Result:              generated.ApplicationActionResponseResultApplicationActionResultCompleted,
		MessageKey:          &messageKey,
		Message:             &messageKey,
		GuardResults: []GuardResult{
			guardCode("registry_deleted"),
			guardCode("workspace_path_preserved"),
			guardCode("runtime_state_not_persisted"),
		},
	}, nil
}

func lifecycleBlockedResult(
	aggregate projectstore.ApplicationAggregate,
	action generated.ApplicationActionResponseAction,
	err error,
) ActionResult {
	return blockedActionResult(aggregate.Application.ApplicationRecordID, action, []GuardResult{guardDetail("lifecycle_blocked", lifecycleBlockedReason(err))})
}

func lifecycleBlockedReason(err error) string {
	switch {
	case errors.Is(err, moduleapi.ErrTaskOwnerBusy):
		return "task_already_active"
	case errors.Is(err, errProjectLifecycleReview):
		return "review_required"
	case errors.Is(err, errProjectInvalidArgument):
		return "invalid_policy"
	default:
		return "refresh_required"
	}
}

func (s *Service) batchActionItem(
	ctx context.Context,
	projectID uint64,
	request BatchActionRequest,
	actor actionActor,
) (itemResult BatchActionItemResult, itemErr error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return BatchActionItemResult{}, err
	}
	permission, ok := batchActionPermission(request.Action)
	if !ok {
		return BatchActionItemResult{}, errProjectInvalidArgument
	}
	if err := s.ensureApplicationScope(ctx, aggregate, permission); err != nil {
		return BatchActionItemResult{}, err
	}
	defer func() {
		itemResult.ApplicationID = aggregate.Application.ApplicationID
	}()
	if item, ok, err := s.batchLifecycleActionItem(ctx, aggregate, request, actor); ok {
		return item, err
	}
	switch request.Action {
	case generated.ApplicationBatchActionRequestActionUnregister:
		action, err := s.unregisterWithActor(ctx, aggregate, actor)
		return BatchActionItemResult{ActionResult: action}, err
	case generated.ApplicationBatchActionRequestActionDestroy:
		confirmName := ""
		if request.ConfirmComposeProjectName != nil {
			confirmName = strings.TrimSpace(*request.ConfirmComposeProjectName)
		}
		destroyReq := DestroyRequest{
			RemoveNamedVolumes:        request.RemoveNamedVolumes,
			AutoUnregister:            request.AutoUnregister,
			ImagePrune:                request.ImagePrune,
			DeleteWorkspacePath:       request.DeleteWorkspacePath,
			ConfirmComposeProjectName: confirmName,
			ActorID:                   &actor.id,
		}
		if _, blockErr := validateDestroyRequest(projectID, aggregate, destroyReq); blockErr != nil {
			return skippedBatchActionResult(projectID, generated.ApplicationActionResponseActionApplicationActionDestroy, "destroy_not_applicable"), nil
		}
		action, err := s.submitDestroyTask(ctx, aggregate, destroyReq, actor)
		return BatchActionItemResult{ActionResult: action}, err
	default:
		return BatchActionItemResult{}, errProjectInvalidArgument
	}
}

func (s *Service) batchLifecycleActionItem(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
	request BatchActionRequest,
	actor actionActor,
) (BatchActionItemResult, bool, error) {
	switch request.Action {
	case generated.ApplicationBatchActionRequestActionStart:
		action, err := s.submitLifecycleTaskWithActor(ctx, aggregate, actor, generated.ApplicationActionResponseActionApplicationActionUp, nil)
		return BatchActionItemResult{ActionResult: action}, true, err
	case generated.ApplicationBatchActionRequestActionStop:
		action, err := s.submitLifecycleTaskWithActor(ctx, aggregate, actor, generated.ApplicationActionResponseActionApplicationActionStop, nil)
		return BatchActionItemResult{ActionResult: action}, true, err
	case generated.ApplicationBatchActionRequestActionRestart:
		action, err := s.submitLifecycleTaskWithActor(ctx, aggregate, actor, generated.ApplicationActionResponseActionApplicationActionRestart, nil)
		return BatchActionItemResult{ActionResult: action}, true, err
	case generated.ApplicationBatchActionRequestActionRedeploy:
		action, err := s.submitLifecycleTaskWithActor(ctx, aggregate, actor, generated.ApplicationActionResponseActionApplicationActionRedeploy, nil)
		return BatchActionItemResult{ActionResult: action}, true, err
	default:
		return BatchActionItemResult{}, false, nil
	}
}

func (s *Service) applyDestroyWorkspacePathStep(
	aggregate projectstore.ApplicationAggregate,
	request DestroyRequest,
	guardResults []GuardResult,
) ([]GuardResult, bool, error) {
	autoUnregister := request.AutoUnregister
	if request.DeleteWorkspacePath {
		if err := deleteManagedWorkspacePath(aggregate.Application.WorkspacePath); err != nil {
			return nil, false, fmt.Errorf("%w: %w", errProjectUnsupportedLifecycle, err)
		}
		guardResults = append(guardResults, guardCode("workspace_path_deleted"))
		autoUnregister = true
		return guardResults, autoUnregister, nil
	}
	guardResults = append(guardResults, guardCode("workspace_path_preserved"))
	return guardResults, autoUnregister, nil
}

func (s *Service) applyDestroyUnregisterStep(
	ctx context.Context,
	projectID uint64,
	actor actionActor,
	guardResults []GuardResult,
	autoUnregister bool,
) ([]GuardResult, error) {
	if !autoUnregister {
		guardResults = append(guardResults, guardCode("registry_preserved"))
		return guardResults, nil
	}
	repository, err := s.repositoryOrErr()
	if err != nil {
		return nil, err
	}
	if err := repository.UnregisterApplication(ctx, projectstore.UnregisterApplicationInput{
		ApplicationRecordID: projectID,
		ActorID:             &actor.id,
	}); err != nil {
		return nil, mapStoreError(err)
	}
	guardResults = append(guardResults, guardCode("registry_deleted"))
	return guardResults, nil
}

func (s *Service) requireBatchActor(
	ctx context.Context,
	request BatchActionRequest,
) (actionActor, BatchActionResult, error) {
	requestAuth, ok := moduleapi.RequestAuthContextFromContext(ctx)
	reason := "actor_request_context_missing"
	if ok && requestAuth.User != nil {
		if request.ActorID != nil && *request.ActorID != requestAuth.User.ID {
			reason = "actor_identity_mismatch"
		} else {
			user := *requestAuth.User
			return actionActor{
				id:       user.ID,
				operator: &user,
			}, BatchActionResult{}, nil
		}
	}
	items := make([]BatchActionItemResult, 0, len(request.ApplicationRecordIDs))
	for _, projectID := range request.ApplicationRecordIDs {
		result := actorAttributionBlockedResult(projectID, batchActionToApplicationAction(request.Action), reason)
		items = append(items, BatchActionItemResult{ActionResult: result})
	}
	return actionActor{}, BatchActionResult{
		TotalCount:   len(request.ApplicationRecordIDs),
		BlockedCount: len(request.ApplicationRecordIDs),
		Items:        items,
	}, errProjectActorAttribution
}

func batchActionToApplicationAction(action generated.ApplicationBatchActionRequestAction) generated.ApplicationActionResponseAction {
	switch action {
	case generated.ApplicationBatchActionRequestActionStart:
		return generated.ApplicationActionResponseActionApplicationActionUp
	case generated.ApplicationBatchActionRequestActionStop:
		return generated.ApplicationActionResponseActionApplicationActionStop
	case generated.ApplicationBatchActionRequestActionRestart:
		return generated.ApplicationActionResponseActionApplicationActionRestart
	case generated.ApplicationBatchActionRequestActionRedeploy:
		return generated.ApplicationActionResponseActionApplicationActionRedeploy
	case generated.ApplicationBatchActionRequestActionUnregister:
		return generated.ApplicationActionResponseActionApplicationActionUnregister
	case generated.ApplicationBatchActionRequestActionDestroy:
		return generated.ApplicationActionResponseActionApplicationActionDestroy
	default:
		return generated.ApplicationActionResponseActionApplicationActionRestart
	}
}

func skippedBatchActionResult(projectID uint64, action generated.ApplicationActionResponseAction, reason string) BatchActionItemResult {
	messageKey := projectcontract.ApplicationLifecycleBlocked.String()
	return BatchActionItemResult{
		ActionResult: ActionResult{
			ApplicationRecordID: projectID,
			Action:              action,
			Result:              generated.ApplicationActionResponseResultApplicationActionResultBlocked,
			MessageKey:          &messageKey,
			Message:             &messageKey,
			GuardResults: []GuardResult{
				guardDetail("skipped", reason),
			},
		},
		Skipped: true,
	}
}

func hasBatchActionItemResult(item BatchActionItemResult) bool {
	return strings.TrimSpace(string(item.Action)) != ""
}
