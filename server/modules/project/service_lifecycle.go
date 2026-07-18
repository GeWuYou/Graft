package project

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"graft/server/internal/apperror"
	"graft/server/internal/contract/errorcode"
	messagecontract "graft/server/internal/contract/message"
	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/logger"
	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

// Up 异步提交 docker compose up -d，并限定在项目已登记的工作目录执行。
func (s *Service) Up(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.submitLifecycleTask(ctx, projectID, actorID, generated.ApplicationActionResponseActionApplicationActionUp)
}

// Stop 异步提交 docker compose stop，并限定在项目已登记的工作目录执行。
func (s *Service) Stop(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.submitLifecycleTask(ctx, projectID, actorID, generated.ApplicationActionResponseActionApplicationActionStop)
}

// Restart 异步提交 docker compose restart，并限定在项目已登记的工作目录执行。
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
	actor, blocked, err := s.requireActionActor(ctx, projectID, generated.ApplicationActionResponseActionApplicationActionDestroy, request.ActorID)
	if err != nil {
		return blocked, err
	}
	if result, blockErr := validateDestroyRequest(projectID, aggregate, request); blockErr != nil {
		s.publishApplicationActionAudit(ctx, aggregate, actor, result, blockErr)
		return result, blockErr
	}
	result, actionErr := s.destroyAfterGuard(ctx, aggregate, request, actor)
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

func (s *Service) destroyAfterGuard(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
	request DestroyRequest,
	actor actionActor,
) (ActionResult, error) {
	projectID := aggregate.Application.ApplicationRecordID
	guardResults := []GuardResult{guardCode("confirm_compose_project_name_matched")}
	downArgs, err := destroyDownArgs(aggregate, request.RemoveNamedVolumes)
	if err != nil {
		return lifecycleBlockedResult(aggregate, generated.ApplicationActionResponseActionApplicationActionDestroy, err), err
	}
	downResult, err := s.executeLifecycleActionWithAggregate(
		ctx,
		aggregate,
		generated.ApplicationActionResponseActionApplicationActionDestroy,
		downArgs,
	)
	if err != nil {
		return downResult, err
	}
	guardResults = appendDestroyDownGuards(guardResults, request.RemoveNamedVolumes)
	nextGuards, autoUnregister, err := s.applyDestroyWorkspacePathStep(aggregate, request, guardResults)
	if err != nil {
		return destroyCleanupBlockedResult(projectID, guardResults, "workspace_path_delete_failed", "filesystem_error"), err
	}
	guardResults = nextGuards
	blockedResult, nextGuards, err := s.applyDestroyImagePruneStep(ctx, aggregate, guardResults, request.ImagePrune)
	if err != nil {
		return blockedResult, err
	}
	guardResults = nextGuards
	nextGuards, err = s.applyDestroyUnregisterStep(ctx, projectID, actor, guardResults, autoUnregister)
	if err != nil {
		return destroyCleanupBlockedResult(projectID, guardResults, "registry_delete_failed", "persistence_error"), err
	}
	guardResults = nextGuards
	messageKey := projectcontract.ApplicationDestroyCompleted.String()
	return ActionResult{
		ApplicationRecordID: projectID,
		Action:              generated.ApplicationActionResponseActionApplicationActionDestroy,
		Result:              generated.ApplicationActionResponseResultApplicationActionResultCompleted,
		MessageKey:          &messageKey,
		Message:             &messageKey,
		GuardResults:        guardResults,
	}, nil
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
	actor, blocked, err := s.requireActionActor(ctx, projectID, action, actorID)
	if err != nil {
		return blocked, err
	}
	if err := ensureProjectLifecycleReady(aggregate); err != nil {
		result := lifecycleBlockedResult(aggregate, action, err)
		s.publishApplicationActionAudit(ctx, aggregate, actor, result, err)
		return result, err
	}
	if err := s.ensureLifecycleRuntimeTargetAvailable(ctx, aggregate); err != nil {
		result := lifecycleBlockedResult(aggregate, action, err)
		s.publishApplicationActionAudit(ctx, aggregate, actor, result, err)
		return result, err
	}
	if s.taskService == nil {
		err := errors.New("task service is unavailable")
		err = s.reportLifecycleTaskSubmissionFailure(ctx, aggregate, action, err)
		result := lifecycleBlockedResult(aggregate, action, err)
		s.publishApplicationActionAudit(ctx, aggregate, actor, result, err)
		return result, err
	}
	plan, err := lifecycleTaskPlan(aggregate, action)
	if err != nil {
		result := lifecycleBlockedResult(aggregate, action, err)
		s.publishApplicationActionAudit(ctx, aggregate, actor, result, err)
		return result, err
	}
	receipt, err := s.taskService.Submit(ctx, moduleapi.SubmitTaskInput{Type: moduleapi.TaskType("application.compose." + strings.ToLower(string(action))), Owner: moduleapi.TaskOwner{Type: applicationTaskOwnerType, ID: aggregate.Application.ApplicationID}, RequestedBy: actor.id, Plan: plan})
	if err != nil {
		err = s.reportLifecycleTaskSubmissionFailure(ctx, aggregate, action, err)
		result := lifecycleBlockedResult(aggregate, action, err)
		s.publishApplicationActionAudit(ctx, aggregate, actor, result, err)
		return result, err
	}
	messageKey := projectcontract.ApplicationLifecycleAccepted.String()
	result := ActionResult{ApplicationRecordID: projectID, Action: action, Result: generated.ApplicationActionResponseResultApplicationActionResultAccepted, MessageKey: &messageKey, Message: &messageKey, GuardResults: []GuardResult{guardDetail("task_id", fmt.Sprintf("%d", receipt.TaskID))}}
	s.publishApplicationActionAudit(ctx, aggregate, actor, result, nil)
	return result, nil
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
func lifecycleTaskPlan(aggregate projectstore.ApplicationAggregate, action generated.ApplicationActionResponseAction) (moduleapi.TaskPlan, error) {
	if action == generated.ApplicationActionResponseActionApplicationActionRedeploy {
		return redeployTaskPlan(aggregate)
	}
	args, err := lifecycleCommandArgs(aggregate, action)
	if err != nil {
		return moduleapi.TaskPlan{}, err
	}
	return taskPlanWithStage(aggregate, strings.ToLower(string(action)), args)
}

// redeployTaskPlan 按固定顺序构建重部署阶段，并根据配置追加停止、拉取和镜像清理等可选阶段。
func redeployTaskPlan(aggregate projectstore.ApplicationAggregate) (moduleapi.TaskPlan, error) {
	config := lifecycleConfigurationFromAggregate(aggregate)
	stages := make([]moduleapi.StagePlan, 0, projectLifecycleStageCapacity)
	if err := appendOptionalRedeployStages(&stages, aggregate, config); err != nil {
		return moduleapi.TaskPlan{}, err
	}
	return moduleapi.TaskPlan{Stages: stages}, nil
}

const projectLifecycleStageCapacity = 4

// appendOptionalRedeployStages 将重新部署所需的 Compose 阶段追加到任务计划中，并根据配置可选地包含停止、拉取和镜像清理阶段。
func appendOptionalRedeployStages(stages *[]moduleapi.StagePlan, aggregate projectstore.ApplicationAggregate, config LifecycleConfiguration) error {
	if config.Standard.DownBeforeRedeploy {
		args, err := lifecycleRedeployDownArgs(aggregate, config)
		if err != nil {
			return err
		}
		if err := appendTaskPlanStage(stages, aggregate, "down", args); err != nil {
			return err
		}
	}
	if config.Standard.PullBeforeRedeploy {
		args, err := lifecyclePullArgs(aggregate, config)
		if err != nil {
			return err
		}
		if err := appendTaskPlanStage(stages, aggregate, "pull", args); err != nil {
			return err
		}
	}
	up, err := lifecycleUpArgs(aggregate, config)
	if err != nil {
		return err
	}
	if err := appendTaskPlanStage(stages, aggregate, "up", up); err != nil {
		return err
	}
	if config.Standard.PruneImagesAfterRedeploy {
		return appendTaskPlanStage(stages, aggregate, "image-prune", []string{"image", "prune", "-f"})
	}
	return nil
}

// taskPlanWithStage 创建只包含一个 Compose 执行阶段的任务计划，并保留手动恢复策略。
func taskPlanWithStage(aggregate projectstore.ApplicationAggregate, key string, args []string) (moduleapi.TaskPlan, error) {
	stages := make([]moduleapi.StagePlan, 0, 1)
	if err := appendTaskPlanStage(&stages, aggregate, key, args); err != nil {
		return moduleapi.TaskPlan{}, err
	}
	return moduleapi.TaskPlan{Stages: stages}, nil
}

// appendTaskPlanStage 将已校验的 Compose 执行阶段追加到计划；参数无效或阶段输入无法序列化时返回错误。
func appendTaskPlanStage(stages *[]moduleapi.StagePlan, aggregate projectstore.ApplicationAggregate, key string, args []string) error {
	if err := ensureLifecycleCommandArgs(args); err != nil {
		return err
	}
	input, err := json.Marshal(composeStageInput{WorkspacePath: aggregate.Application.WorkspacePath, Args: args})
	if err != nil {
		return err
	}
	*stages = append(*stages, moduleapi.StagePlan{Key: key, ExecutorType: moduleapi.StageExecutorType(composeStagePrefix + key), Input: input, RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: 1}, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile})
	return nil
}

func (s *Service) executeLifecycleActionWithAggregate(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
	action generated.ApplicationActionResponseAction,
	args []string,
) (ActionResult, error) {
	if err := ensureProjectLifecycleReady(aggregate); err != nil {
		return lifecycleBlockedResult(aggregate, action, err), err
	}
	if err := s.ensureLifecycleRuntimeTargetAvailable(ctx, aggregate); err != nil {
		return lifecycleBlockedResult(aggregate, action, err), err
	}
	if err := ensureLifecycleCommandArgs(args); err != nil {
		return blockedActionResult(aggregate.Application.ApplicationRecordID, action, []GuardResult{guardDetail("lifecycle_blocked", "invalid_command")}), err
	}
	commandOutput, err := s.runComposeCommand(ctx, aggregate, args)
	if err != nil {
		result := blockedActionResult(aggregate.Application.ApplicationRecordID, action, []GuardResult{guardDetail("lifecycle_failed", summarizeCommandOutput(commandOutput))})
		return result, fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
	}
	messageKey := lifecycleMessageKey(action).String()
	return ActionResult{
		ApplicationRecordID: aggregate.Application.ApplicationRecordID,
		Action:              action,
		Result:              generated.ApplicationActionResponseResultApplicationActionResultCompleted,
		MessageKey:          &messageKey,
		Message:             &messageKey,
		GuardResults: []GuardResult{
			guardDetail("command", strings.Join(args, " ")),
		},
	}, nil
}

func (s *Service) runComposeCommand(ctx context.Context, aggregate projectstore.ApplicationAggregate, args []string) (string, error) {
	return s.runDockerCommand(ctx, aggregate.Application.WorkspacePath, args)
}

// ensureLifecycleRuntimeTargetAvailable 校验选定运行目标在项目提交或执行生命周期任务前是否可达。
// Compose 名称占用检查只属于创建阶段；已登记项目通常拥有使用自身 Compose 名称的运行时资源。
func (s *Service) ensureLifecycleRuntimeTargetAvailable(ctx context.Context, aggregate projectstore.ApplicationAggregate) error {
	if s == nil || s.runtimeTargets == nil {
		return nil
	}
	if aggregate.Application.RuntimeTargetID == nil || *aggregate.Application.RuntimeTargetID == 0 {
		return errProjectRuntimeUnavailable
	}
	if *aggregate.Application.RuntimeTargetID > uint64(^uint64(0)>>1) {
		return errProjectRuntimeUnavailable
	}
	id := int64(*aggregate.Application.RuntimeTargetID) // #nosec G115 -- bounded by max signed int64 immediately above.
	target, err := s.runtimeTargets.ReadComposeTarget(ctx, &id)
	if err != nil || !target.Available {
		return errProjectRuntimeUnavailable
	}
	return nil
}

func (s *Service) runDockerCommand(ctx context.Context, workingDirectory string, args []string) (string, error) {
	commandCtx, cancel := withComposeCommandTimeout(ctx)
	defer cancel()
	// #nosec G204 -- 可执行文件固定为 docker，参数均为已校验的命令片段且不经过 shell 展开。
	command := exec.CommandContext(commandCtx, "docker", args...)
	command.Dir = workingDirectory
	command.Env = os.Environ()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return strings.TrimSpace(stdout.String() + "\n" + stderr.String()), err
}

func withComposeCommandTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, projectComposeTimeout)
}

// ensureLifecycleCommandArgs 校验生命周期命令参数。
// 只有当参数数量满足要求且每个参数都包含非空白内容时才通过。
func ensureLifecycleCommandArgs(args []string) error {
	if len(args) < minLifecycleArgCount {
		return errProjectInvalidArgument
	}
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			return errProjectInvalidArgument
		}
	}
	return nil
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

func destroyCleanupBlockedResult(projectID uint64, guardResults []GuardResult, code string, detail string) ActionResult {
	return blockedActionResult(
		projectID,
		generated.ApplicationActionResponseActionApplicationActionDestroy,
		append(append([]GuardResult(nil), guardResults...), guardDetail(code, detail)),
	)
}

// lifecycleMessageKey 返回指定生命周期动作对应的完成消息键。
func lifecycleMessageKey(action generated.ApplicationActionResponseAction) projectcontract.MessageKey {
	switch action {
	case generated.ApplicationActionResponseActionApplicationActionUp:
		return projectcontract.ApplicationUpCompleted
	case generated.ApplicationActionResponseActionApplicationActionStop:
		return projectcontract.ApplicationStopCompleted
	case generated.ApplicationActionResponseActionApplicationActionRestart:
		return projectcontract.ApplicationRestartCompleted
	case generated.ApplicationActionResponseActionApplicationActionRedeploy:
		return projectcontract.ApplicationRedeployCompleted
	case generated.ApplicationActionResponseActionApplicationActionDestroy:
		return projectcontract.ApplicationDestroyCompleted
	case generated.ApplicationActionResponseActionApplicationActionUnregister:
		return projectcontract.ApplicationUnregisterCompleted
	default:
		return projectcontract.ApplicationLifecycleAccepted
	}
}

func (s *Service) redeployWithActor(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
	_ actionActor,
) (ActionResult, error) {
	if err := s.ensureRedeployReady(ctx, aggregate); err != nil {
		return lifecycleBlockedResult(aggregate, generated.ApplicationActionResponseActionApplicationActionRedeploy, err), err
	}
	config := lifecycleConfigurationFromAggregate(aggregate)
	guards := []GuardResult{}
	if config.Standard.DownBeforeRedeploy {
		var err error
		guards, err = s.runRedeployComposeStep(ctx, aggregate, config, guards, lifecycleRedeployDownArgs, "compose_down_completed")
		if err != nil {
			return blockedActionResult(aggregate.Application.ApplicationRecordID, generated.ApplicationActionResponseActionApplicationActionRedeploy, guards), err
		}
	}
	if config.Standard.PullBeforeRedeploy {
		var err error
		guards, err = s.runRedeployComposeStep(ctx, aggregate, config, guards, lifecyclePullArgs, "compose_pull_completed")
		if err != nil {
			return blockedActionResult(aggregate.Application.ApplicationRecordID, generated.ApplicationActionResponseActionApplicationActionRedeploy, guards), err
		}
	}
	upArgs, err := lifecycleUpArgs(aggregate, config)
	if err != nil {
		return lifecycleBlockedResult(aggregate, generated.ApplicationActionResponseActionApplicationActionRedeploy, err), err
	}
	output, err := s.runComposeCommand(ctx, aggregate, upArgs)
	if err != nil {
		return blockedActionResult(aggregate.Application.ApplicationRecordID, generated.ApplicationActionResponseActionApplicationActionRedeploy, append(guards, guardDetail("lifecycle_failed", summarizeCommandOutput(output)))), fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
	}
	guards = append(guards, guardDetail("command", strings.Join(upArgs, " ")))
	if config.Standard.PruneImagesAfterRedeploy {
		output, err = s.runDockerCommand(ctx, aggregate.Application.WorkspacePath, []string{"image", "prune", "-f"})
		if err != nil {
			return blockedActionResult(aggregate.Application.ApplicationRecordID, generated.ApplicationActionResponseActionApplicationActionRedeploy, append(guards, guardDetail("image_prune_failed", summarizeCommandOutput(output)))), fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
		}
		guards = append(guards, guardCode("image_prune_completed"))
	}
	messageKey := lifecycleMessageKey(generated.ApplicationActionResponseActionApplicationActionRedeploy).String()
	return ActionResult{
		ApplicationRecordID: aggregate.Application.ApplicationRecordID,
		Action:              generated.ApplicationActionResponseActionApplicationActionRedeploy,
		Result:              generated.ApplicationActionResponseResultApplicationActionResultCompleted,
		MessageKey:          &messageKey,
		Message:             &messageKey,
		GuardResults:        guards,
	}, nil
}

func (s *Service) ensureRedeployReady(ctx context.Context, aggregate projectstore.ApplicationAggregate) error {
	if err := ensureProjectLifecycleReady(aggregate); err != nil {
		return err
	}
	return s.ensureLifecycleRuntimeTargetAvailable(ctx, aggregate)
}

func (s *Service) runRedeployComposeStep(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
	config LifecycleConfiguration,
	guards []GuardResult,
	argsBuilder func(projectstore.ApplicationAggregate, LifecycleConfiguration) ([]string, error),
	successCode string,
) ([]GuardResult, error) {
	args, err := argsBuilder(aggregate, config)
	if err != nil {
		return lifecycleBlockedGuardResults(guards, err), err
	}
	output, err := s.runComposeCommand(ctx, aggregate, args)
	if err != nil {
		return append(guards, guardDetail("lifecycle_failed", summarizeCommandOutput(output))), fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
	}
	return append(guards, guardCode(successCode)), nil
}

func lifecycleBlockedGuardResults(guards []GuardResult, err error) []GuardResult {
	if err == nil {
		return append([]GuardResult(nil), guards...)
	}
	return append(append([]GuardResult(nil), guards...), guardDetail("lifecycle_blocked", err.Error()))
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

// composeProjectArgs 根据项目聚合数据和生命周期配置构建 Docker Compose 命令参数；缺少 Compose 文件或项目规范名称无效时返回错误。
func composeProjectArgs(aggregate projectstore.ApplicationAggregate, config LifecycleConfiguration) ([]string, error) {
	composeFiles := filterFiles(aggregate.Files, projectcontract.FileKindCompose.String())
	if len(composeFiles) == 0 {
		return nil, errProjectInvalidArgument
	}
	base := []string{"compose"}
	for _, file := range composeFiles {
		if strings.TrimSpace(file.AbsolutePath) == "" {
			return nil, errProjectInvalidArgument
		}
		path := file.AbsolutePath
		if !filepath.IsAbs(path) {
			path = filepath.Join(aggregate.Application.WorkspacePath, path)
		}
		base = append(base, "-f", path)
	}
	for _, profile := range config.Standard.Profiles {
		base = append(base, "--profile", profile)
	}
	if strings.TrimSpace(config.ApplicationName) == "" {
		return nil, errProjectInvalidCanonicalName
	}
	canonicalProjectName, err := validateExplicitComposeProjectName(config.ApplicationName)
	if err != nil {
		return nil, err
	}
	base = append(base, "-p", canonicalProjectName)
	return base, nil
}

func lifecycleCommandArgs(aggregate projectstore.ApplicationAggregate, action generated.ApplicationActionResponseAction) ([]string, error) {
	config := lifecycleConfigurationFromAggregate(aggregate)
	switch action {
	case generated.ApplicationActionResponseActionApplicationActionUp:
		return lifecycleUpArgs(aggregate, config)
	case generated.ApplicationActionResponseActionApplicationActionStop:
		base, err := composeProjectArgs(aggregate, config)
		if err != nil {
			return nil, err
		}
		return append(base, "stop"), nil
	case generated.ApplicationActionResponseActionApplicationActionRestart:
		base, err := composeProjectArgs(aggregate, config)
		if err != nil {
			return nil, err
		}
		return append(base, "restart"), nil
	default:
		return nil, errProjectInvalidArgument
	}
}

// lifecycleUpArgs 构建用于启动项目的 Docker Compose 参数，并根据配置添加构建、重建、孤立容器清理、匿名卷更新及等待选项。
// lifecycleUpArgs 构建用于启动项目的 Docker Compose 参数列表。
// 返回包含配置选项和附加参数的命令参数；配置无效时返回错误。
func lifecycleUpArgs(aggregate projectstore.ApplicationAggregate, config LifecycleConfiguration) ([]string, error) {
	base, err := composeProjectArgs(aggregate, config)
	if err != nil {
		return nil, err
	}
	args := append(base, "up", "-d")
	if config.Standard.BuildBeforeUp {
		args = append(args, "--build")
	}
	if config.Standard.ForceRecreate {
		args = append(args, "--force-recreate")
	}
	if config.Standard.RemoveOrphans {
		args = append(args, "--remove-orphans")
	}
	if config.Standard.RenewAnonVolumes {
		args = append(args, "--renew-anon-volumes")
	}
	if config.Standard.WaitAfterUp {
		args = append(args, "--wait")
		args = append(args, "--wait-timeout", fmt.Sprintf("%d", config.Standard.WaitTimeoutSeconds))
	}
	args = append(args, config.Standard.AdditionalArgs...)
	return args, nil
}

func lifecyclePullArgs(aggregate projectstore.ApplicationAggregate, config LifecycleConfiguration) ([]string, error) {
	base, err := composeProjectArgs(aggregate, config)
	if err != nil {
		return nil, err
	}
	return append(base, "pull"), nil
}

func lifecycleRedeployDownArgs(aggregate projectstore.ApplicationAggregate, config LifecycleConfiguration) ([]string, error) {
	base, err := composeProjectArgs(aggregate, config)
	if err != nil {
		return nil, err
	}
	return append(base, "down"), nil
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
	case errors.Is(err, errProjectLifecycleReview):
		return "review_required"
	case errors.Is(err, errProjectInvalidArgument):
		return "invalid_command"
	default:
		return "refresh_required"
	}
}

// summarizeCommandOutput 归一化并截断命令输出摘要。
// 它会去除首尾空白，空输出返回 "command_failed"，并将过长内容截断到最大摘要长度。
// @param output 原始命令输出。
// @returns 处理后的输出摘要。
func summarizeCommandOutput(output string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return "command_failed"
	}
	if len(trimmed) > maxCommandOutputSummary {
		return trimmed[:maxCommandOutputSummary]
	}
	return trimmed
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
	defer func() {
		itemResult.ApplicationID = aggregate.Application.ApplicationID
	}()
	if item, ok, err := s.batchLifecycleActionItem(ctx, aggregate, projectID, request, actor); ok {
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
		action, err := s.destroyAfterGuard(ctx, aggregate, destroyReq, actor)
		return BatchActionItemResult{ActionResult: action}, err
	default:
		return BatchActionItemResult{}, errProjectInvalidArgument
	}
}

func (s *Service) batchLifecycleActionItem(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
	projectID uint64,
	request BatchActionRequest,
	actor actionActor,
) (BatchActionItemResult, bool, error) {
	switch request.Action {
	case generated.ApplicationBatchActionRequestActionStart:
		item, err := s.batchLifecycleItem(
			ctx,
			aggregate,
			generated.ApplicationActionResponseActionApplicationActionUp,
		)
		return item, true, err
	case generated.ApplicationBatchActionRequestActionStop:
		item, err := s.batchLifecycleItem(
			ctx,
			aggregate,
			generated.ApplicationActionResponseActionApplicationActionStop,
		)
		return item, true, err
	case generated.ApplicationBatchActionRequestActionRestart:
		item, err := s.batchLifecycleItem(
			ctx,
			aggregate,
			generated.ApplicationActionResponseActionApplicationActionRestart,
		)
		return item, true, err
	case generated.ApplicationBatchActionRequestActionRedeploy:
		if err := ensureProjectLifecycleReady(aggregate); err != nil {
			return skippedBatchActionResult(projectID, generated.ApplicationActionResponseActionApplicationActionRedeploy, lifecycleBlockedReason(err)), true, nil
		}
		action, err := s.redeployWithActor(ctx, aggregate, actor)
		return BatchActionItemResult{ActionResult: action}, true, err
	default:
		return BatchActionItemResult{}, false, nil
	}
}

func (s *Service) batchLifecycleItem(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
	action generated.ApplicationActionResponseAction,
) (BatchActionItemResult, error) {
	if err := ensureProjectLifecycleReady(aggregate); err != nil {
		return BatchActionItemResult{ActionResult: lifecycleBlockedResult(aggregate, action, err)}, nil
	}
	if err := s.ensureLifecycleRuntimeTargetAvailable(ctx, aggregate); err != nil {
		return BatchActionItemResult{ActionResult: lifecycleBlockedResult(aggregate, action, err)}, nil
	}
	runtimeSummary, runtimeErr := s.runtimeSummary(ctx, aggregate)
	if skipReason, shouldSkip := skipBatchLifecycleAction(action, &runtimeSummary, runtimeErr); shouldSkip {
		return skippedBatchActionResult(aggregate.Application.ApplicationRecordID, action, skipReason), nil
	}
	args, err := lifecycleCommandArgs(aggregate, action)
	if err != nil {
		return BatchActionItemResult{ActionResult: lifecycleBlockedResult(aggregate, action, err)}, nil
	}
	result, err := s.executeLifecycleActionWithAggregate(ctx, aggregate, action, args)
	return BatchActionItemResult{ActionResult: result}, err
}

func skipBatchLifecycleAction(
	action generated.ApplicationActionResponseAction,
	runtimeSummary *moduleapi.ContainerProjectRuntimeSummary,
	runtimeErr error,
) (string, bool) {
	runtimeStatus := deriveProjectRuntimeStatus(runtimeSummary, runtimeErr)
	if runtimeStatus == nil {
		return "", false
	}
	switch action {
	case generated.ApplicationActionResponseActionApplicationActionUp:
		return skipBatchStartForStatus(*runtimeStatus)
	case generated.ApplicationActionResponseActionApplicationActionStop:
		return skipBatchStopForStatus(*runtimeStatus)
	case generated.ApplicationActionResponseActionApplicationActionRestart:
		return skipBatchRestartForStatus(*runtimeStatus)
	default:
		return "", false
	}
}

func skipBatchStartForStatus(status generated.ApplicationRuntimeStatus) (string, bool) {
	switch status {
	case generated.ApplicationRuntimeStatusRunning:
		return "already_running", true
	case generated.ApplicationRuntimeStatusDegraded:
		return "already_partially_running", true
	case generated.ApplicationRuntimeStatusTransitioning:
		return "currently_transitioning", true
	default:
		return "", false
	}
}

func skipBatchStopForStatus(status generated.ApplicationRuntimeStatus) (string, bool) {
	switch status {
	case generated.ApplicationRuntimeStatusStopped:
		return "already_stopped", true
	case generated.ApplicationRuntimeStatusUnknown:
		return "runtime_status_unknown", true
	default:
		return "", false
	}
}

func skipBatchRestartForStatus(status generated.ApplicationRuntimeStatus) (string, bool) {
	switch status {
	case generated.ApplicationRuntimeStatusRunning, generated.ApplicationRuntimeStatusDegraded, generated.ApplicationRuntimeStatusStopped:
		return "", false
	case generated.ApplicationRuntimeStatusTransitioning:
		return "currently_transitioning", true
	case generated.ApplicationRuntimeStatusUnknown:
		return "runtime_status_unknown", true
	default:
		return "runtime_status_unknown", true
	}
}

func destroyDownArgs(aggregate projectstore.ApplicationAggregate, removeNamedVolumes bool) ([]string, error) {
	base, err := composeProjectArgs(aggregate, lifecycleConfigurationFromAggregate(aggregate))
	if err != nil {
		return nil, err
	}
	downArgs := append(base, "down")
	if removeNamedVolumes {
		downArgs = append(downArgs, "--volumes")
	}
	return downArgs, nil
}

func appendDestroyDownGuards(guardResults []GuardResult, removeNamedVolumes bool) []GuardResult {
	guardResults = append(guardResults, guardCode("compose_down_completed"))
	if removeNamedVolumes {
		guardResults = append(guardResults, guardCode("named_volumes_removed"))
	}
	return guardResults
}

func (s *Service) applyDestroyWorkspacePathStep(
	aggregate projectstore.ApplicationAggregate,
	request DestroyRequest,
	guardResults []GuardResult,
) ([]GuardResult, bool, error) {
	autoUnregister := request.AutoUnregister
	if request.DeleteWorkspacePath {
		if err := deleteManagedWorkspacePath(aggregate.Application.WorkspacePath); err != nil {
			return nil, false, fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
		}
		guardResults = append(guardResults, guardCode("workspace_path_deleted"))
		autoUnregister = true
		return guardResults, autoUnregister, nil
	}
	guardResults = append(guardResults, guardCode("workspace_path_preserved"))
	return guardResults, autoUnregister, nil
}

func (s *Service) applyDestroyImagePruneStep(
	ctx context.Context,
	aggregate projectstore.ApplicationAggregate,
	guardResults []GuardResult,
	imagePrune bool,
) (ActionResult, []GuardResult, error) {
	if !imagePrune {
		return ActionResult{}, guardResults, nil
	}
	output, err := s.runDockerCommand(ctx, aggregate.Application.WorkspacePath, []string{"image", "prune", "-f"})
	if err != nil {
		return blockedActionResult(
				aggregate.Application.ApplicationRecordID,
				generated.ApplicationActionResponseActionApplicationActionDestroy,
				append(guardResults, guardDetail("image_prune_failed", summarizeCommandOutput(output))),
			),
			nil,
			fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
	}
	guardResults = append(guardResults, guardCode("image_prune_completed"))
	return ActionResult{}, guardResults, nil
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
