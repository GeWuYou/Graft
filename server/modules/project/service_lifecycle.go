package project

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

// Up executes docker compose up -d within the project's registered working directory.
func (s *Service) Up(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.runLifecycleAction(ctx, projectID, actorID, generated.ProjectActionResponseActionProjectActionUp)
}

// Stop executes docker compose stop for the registered project.
func (s *Service) Stop(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.runLifecycleAction(ctx, projectID, actorID, generated.ProjectActionResponseActionProjectActionStop)
}

// Restart executes docker compose restart for the registered project.
func (s *Service) Restart(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.runLifecycleAction(ctx, projectID, actorID, generated.ProjectActionResponseActionProjectActionRestart)
}

// Redeploy executes docker compose down then docker compose up -d for the registered project.
func (s *Service) Redeploy(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ActionResult{}, err
	}
	actor, blocked, err := s.requireActionActor(ctx, projectID, generated.ProjectActionResponseActionProjectActionRedeploy, actorID)
	if err != nil {
		return blocked, err
	}
	result, actionErr := s.redeployWithActor(ctx, aggregate, actor)
	s.publishProjectActionAudit(ctx, aggregate, actor, result, actionErr)
	return result, actionErr
}

// BatchAction executes one action for multiple projects and returns per-item results.
func (s *Service) BatchAction(ctx context.Context, request BatchActionRequest) (BatchActionResult, error) {
	actor, blocked, err := s.requireBatchActor(ctx, request)
	if err != nil {
		return blocked, nil
	}
	items := make([]BatchActionItemResult, 0, len(request.ProjectIDs))
	result := BatchActionResult{TotalCount: len(request.ProjectIDs)}
	for _, projectID := range request.ProjectIDs {
		item, itemErr := s.batchActionItem(ctx, projectID, request, actor)
		if itemErr != nil && !hasBatchActionItemResult(item) {
			return BatchActionResult{}, itemErr
		}
		items = append(items, item)
		switch {
		case item.Skipped:
			result.SkippedCount++
		case item.Result == generated.ProjectActionResponseResultProjectActionResultCompleted:
			result.CompletedCount++
		default:
			result.BlockedCount++
		}
	}
	result.Items = items
	s.publishProjectBatchAudit(ctx, actor, request, result)
	return result, nil
}

// Unregister removes the project registry record without touching host files.
func (s *Service) Unregister(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ActionResult{}, err
	}
	actor, blocked, err := s.requireActionActor(ctx, projectID, generated.ProjectActionResponseActionProjectActionUnregister, actorID)
	if err != nil {
		return blocked, err
	}
	result, actionErr := s.unregisterWithActor(ctx, aggregate, actor)
	s.publishProjectActionAudit(ctx, aggregate, actor, result, actionErr)
	return result, actionErr
}

// Destroy executes guarded teardown steps and then unregisters the project record.
func (s *Service) Destroy(ctx context.Context, projectID uint64, request DestroyRequest) (ActionResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ActionResult{}, err
	}
	actor, blocked, err := s.requireActionActor(ctx, projectID, generated.ProjectActionResponseActionProjectActionDestroy, request.ActorID)
	if err != nil {
		return blocked, err
	}
	if result, blockErr := validateDestroyRequest(projectID, aggregate, request); blockErr != nil {
		s.publishProjectActionAudit(ctx, aggregate, actor, result, blockErr)
		return result, blockErr
	}
	result, actionErr := s.destroyAfterGuard(ctx, aggregate, request, actor)
	s.publishProjectActionAudit(ctx, aggregate, actor, result, actionErr)
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
	aggregate projectstore.ProjectAggregate,
	request DestroyRequest,
) (ActionResult, error) {
	guardResults := []GuardResult{}
	if strings.TrimSpace(request.ConfirmCanonicalProjectName) != aggregate.Project.CanonicalProjectName {
		return blockedActionResult(projectID, generated.ProjectActionResponseActionProjectActionDestroy, append(guardResults, guardCode("confirm_canonical_project_name_mismatch"))), errProjectDestroyBlocked
	}
	guardResults = append(guardResults, guardCode("confirm_canonical_project_name_matched"))

	if request.RemoveNamedVolumes {
		guardResults = append(guardResults, guardCode("remove_named_volumes_requested"))
	}

	if request.DeleteWorkingDirectory && aggregate.Project.OwnershipMode != projectcontract.OwnershipModeManagedRootDedicated.String() {
		guardResults = append(guardResults, guardDetail("delete_working_directory_blocked", "ownership_mode_external"))
		return blockedActionResult(projectID, generated.ProjectActionResponseActionProjectActionDestroy, guardResults), errProjectDestroyBlocked
	}
	return ActionResult{}, nil
}

func (s *Service) destroyAfterGuard(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	request DestroyRequest,
	actor actionActor,
) (ActionResult, error) {
	projectID := aggregate.Project.ID
	guardResults := []GuardResult{guardCode("confirm_canonical_project_name_matched")}
	downArgs, err := destroyDownArgs(aggregate, request.RemoveNamedVolumes)
	if err != nil {
		return lifecycleBlockedResult(aggregate, generated.ProjectActionResponseActionProjectActionDestroy, err), err
	}
	downResult, err := s.executeLifecycleActionWithAggregate(
		ctx,
		aggregate,
		generated.ProjectActionResponseActionProjectActionDestroy,
		downArgs,
	)
	if err != nil {
		return downResult, err
	}
	guardResults = appendDestroyDownGuards(guardResults, request.RemoveNamedVolumes)
	nextGuards, autoUnregister, err := s.applyDestroyWorkingDirectoryStep(aggregate, request, guardResults)
	if err != nil {
		return destroyCleanupBlockedResult(projectID, guardResults, "working_directory_delete_failed", "filesystem_error"), err
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
	messageKey := projectcontract.ProjectDestroyCompleted.String()
	return ActionResult{
		ProjectID:    projectID,
		Action:       generated.ProjectActionResponseActionProjectActionDestroy,
		Result:       generated.ProjectActionResponseResultProjectActionResultCompleted,
		MessageKey:   &messageKey,
		Message:      &messageKey,
		GuardResults: guardResults,
	}, nil
}

// UnsupportedLifecycleAction returns an explicit batch-2 blocked action result.
func (s *Service) UnsupportedLifecycleAction(projectID uint64, action generated.ProjectActionResponseAction) (ActionResult, error) {
	return ActionResult{
		ProjectID:    projectID,
		Action:       action,
		Result:       generated.ProjectActionResponseResultProjectActionResultBlocked,
		MessageKey:   stringPointer(projectcontract.ProjectLifecycleAccepted.String()),
		Message:      stringPointer(projectcontract.ProjectLifecycleAccepted.String()),
		GuardResults: []GuardResult{guardDetail("batch-2-scope", "lifecycle execution is deferred to phase-1-batch-3")},
	}, errProjectUnsupportedLifecycle
}

func (s *Service) runLifecycleAction(
	ctx context.Context,
	projectID uint64,
	actorID *uint64,
	action generated.ProjectActionResponseAction,
) (ActionResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ActionResult{}, err
	}
	actor, blocked, err := s.requireActionActor(ctx, projectID, action, actorID)
	if err != nil {
		return blocked, err
	}
	args, err := lifecycleCommandArgs(aggregate, action)
	if err != nil {
		result := lifecycleBlockedResult(aggregate, action, err)
		s.publishProjectActionAudit(ctx, aggregate, actor, result, err)
		return result, err
	}
	result, actionErr := s.executeLifecycleActionWithAggregate(ctx, aggregate, action, args)
	s.publishProjectActionAudit(ctx, aggregate, actor, result, actionErr)
	return result, actionErr
}

func (s *Service) executeLifecycleActionWithAggregate(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	action generated.ProjectActionResponseAction,
	args []string,
) (ActionResult, error) {
	if err := ensureProjectLifecycleReady(aggregate); err != nil {
		return lifecycleBlockedResult(aggregate, action, err), err
	}
	if err := ensureLifecycleCommandArgs(args); err != nil {
		return blockedActionResult(aggregate.Project.ID, action, []GuardResult{guardDetail("lifecycle_blocked", "invalid_command")}), err
	}
	commandOutput, err := s.runComposeCommand(ctx, aggregate, args)
	if err != nil {
		result := blockedActionResult(aggregate.Project.ID, action, []GuardResult{guardDetail("lifecycle_failed", summarizeCommandOutput(commandOutput))})
		return result, fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
	}
	messageKey := lifecycleMessageKey(action).String()
	return ActionResult{
		ProjectID:  aggregate.Project.ID,
		Action:     action,
		Result:     generated.ProjectActionResponseResultProjectActionResultCompleted,
		MessageKey: &messageKey,
		Message:    &messageKey,
		GuardResults: []GuardResult{
			guardDetail("command", strings.Join(args, " ")),
			guardDetail("host_scope", aggregate.Project.HostScope),
		},
	}, nil
}

func (s *Service) runComposeCommand(ctx context.Context, aggregate projectstore.ProjectAggregate, args []string) (string, error) {
	return s.runDockerCommand(ctx, aggregate.Project.WorkingDirectory, args)
}

func (s *Service) runDockerCommand(ctx context.Context, workingDirectory string, args []string) (string, error) {
	commandCtx, cancel := withComposeCommandTimeout(ctx)
	defer cancel()
	// #nosec G204 -- binary is fixed to docker and args are validated command fragments, not shell-expanded input.
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
func ensureProjectLifecycleReady(aggregate projectstore.ProjectAggregate) error {
	if strings.TrimSpace(aggregate.Project.HostScope) != projectcontract.HostScopeLocal.String() {
		return errProjectUnsupportedLifecycle
	}
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
func blockedActionResult(projectID uint64, action generated.ProjectActionResponseAction, guardResults []GuardResult) ActionResult {
	messageKey := projectcontract.ProjectLifecycleBlocked.String()
	return ActionResult{
		ProjectID:    projectID,
		Action:       action,
		Result:       generated.ProjectActionResponseResultProjectActionResultBlocked,
		MessageKey:   &messageKey,
		Message:      &messageKey,
		GuardResults: append([]GuardResult(nil), guardResults...),
	}
}

func destroyCleanupBlockedResult(projectID uint64, guardResults []GuardResult, code string, detail string) ActionResult {
	return blockedActionResult(
		projectID,
		generated.ProjectActionResponseActionProjectActionDestroy,
		append(append([]GuardResult(nil), guardResults...), guardDetail(code, detail)),
	)
}

// lifecycleMessageKey 返回指定生命周期动作对应的完成消息键。
func lifecycleMessageKey(action generated.ProjectActionResponseAction) projectcontract.MessageKey {
	switch action {
	case generated.ProjectActionResponseActionProjectActionUp:
		return projectcontract.ProjectUpCompleted
	case generated.ProjectActionResponseActionProjectActionStop:
		return projectcontract.ProjectStopCompleted
	case generated.ProjectActionResponseActionProjectActionRestart:
		return projectcontract.ProjectRestartCompleted
	case generated.ProjectActionResponseActionProjectActionRedeploy:
		return projectcontract.ProjectRedeployCompleted
	case generated.ProjectActionResponseActionProjectActionDestroy:
		return projectcontract.ProjectDestroyCompleted
	case generated.ProjectActionResponseActionProjectActionDeploy:
		return projectcontract.ProjectDeployCompleted
	case generated.ProjectActionResponseActionProjectActionUnregister:
		return projectcontract.ProjectUnregisterCompleted
	default:
		return projectcontract.ProjectLifecycleAccepted
	}
}

func (s *Service) redeployWithActor(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	_ actionActor,
) (ActionResult, error) {
	if err := ensureProjectLifecycleReady(aggregate); err != nil {
		return lifecycleBlockedResult(aggregate, generated.ProjectActionResponseActionProjectActionRedeploy, err), err
	}
	config := lifecycleConfigurationFromAggregate(aggregate)
	guards := []GuardResult{guardDetail("host_scope", aggregate.Project.HostScope)}
	if config.Standard.DownBeforeRedeploy {
		var err error
		guards, err = s.runRedeployComposeStep(ctx, aggregate, config, guards, lifecycleRedeployDownArgs, "compose_down_completed")
		if err != nil {
			return blockedActionResult(aggregate.Project.ID, generated.ProjectActionResponseActionProjectActionRedeploy, guards), err
		}
	}
	if config.Standard.PullBeforeRedeploy {
		var err error
		guards, err = s.runRedeployComposeStep(ctx, aggregate, config, guards, lifecyclePullArgs, "compose_pull_completed")
		if err != nil {
			return blockedActionResult(aggregate.Project.ID, generated.ProjectActionResponseActionProjectActionRedeploy, guards), err
		}
	}
	upArgs, err := lifecycleUpArgs(aggregate, config)
	if err != nil {
		return lifecycleBlockedResult(aggregate, generated.ProjectActionResponseActionProjectActionRedeploy, err), err
	}
	output, err := s.runComposeCommand(ctx, aggregate, upArgs)
	if err != nil {
		return blockedActionResult(aggregate.Project.ID, generated.ProjectActionResponseActionProjectActionRedeploy, append(guards, guardDetail("lifecycle_failed", summarizeCommandOutput(output)))), fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
	}
	guards = append(guards, guardDetail("command", strings.Join(upArgs, " ")))
	if config.Standard.PruneImagesAfterRedeploy {
		output, err = s.runDockerCommand(ctx, aggregate.Project.WorkingDirectory, []string{"image", "prune", "-f"})
		if err != nil {
			return blockedActionResult(aggregate.Project.ID, generated.ProjectActionResponseActionProjectActionRedeploy, append(guards, guardDetail("image_prune_failed", summarizeCommandOutput(output)))), fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
		}
		guards = append(guards, guardCode("image_prune_completed"))
	}
	messageKey := lifecycleMessageKey(generated.ProjectActionResponseActionProjectActionRedeploy).String()
	return ActionResult{
		ProjectID:    aggregate.Project.ID,
		Action:       generated.ProjectActionResponseActionProjectActionRedeploy,
		Result:       generated.ProjectActionResponseResultProjectActionResultCompleted,
		MessageKey:   &messageKey,
		Message:      &messageKey,
		GuardResults: guards,
	}, nil
}

func (s *Service) runRedeployComposeStep(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	config LifecycleConfiguration,
	guards []GuardResult,
	argsBuilder func(projectstore.ProjectAggregate, LifecycleConfiguration) ([]string, error),
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
	aggregate projectstore.ProjectAggregate,
	actor actionActor,
) (ActionResult, error) {
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ActionResult{}, err
	}
	if err := repository.UnregisterProject(ctx, projectstore.UnregisterProjectInput{
		ProjectID: aggregate.Project.ID,
		ActorID:   &actor.id,
	}); err != nil {
		return blockedActionResult(aggregate.Project.ID, generated.ProjectActionResponseActionProjectActionUnregister, []GuardResult{guardDetail("registry_delete_failed", "persistence_error")}), mapStoreError(err)
	}
	messageKey := projectcontract.ProjectUnregisterCompleted.String()
	return ActionResult{
		ProjectID:  aggregate.Project.ID,
		Action:     generated.ProjectActionResponseActionProjectActionUnregister,
		Result:     generated.ProjectActionResponseResultProjectActionResultCompleted,
		MessageKey: &messageKey,
		Message:    &messageKey,
		GuardResults: []GuardResult{
			guardCode("registry_deleted"),
			guardCode("working_directory_preserved"),
			guardCode("runtime_state_not_persisted"),
		},
	}, nil
}

// composeProjectArgs 根据项目聚合数据和生命周期配置构建 Docker Compose 命令参数；缺少 Compose 文件或项目规范名称无效时返回错误。
func composeProjectArgs(aggregate projectstore.ProjectAggregate, config LifecycleConfiguration) ([]string, error) {
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
			path = filepath.Join(aggregate.Project.WorkingDirectory, path)
		}
		base = append(base, "-f", path)
	}
	for _, profile := range config.Standard.Profiles {
		base = append(base, "--profile", profile)
	}
	if strings.TrimSpace(config.ProjectName) == "" {
		return nil, errProjectInvalidCanonicalName
	}
	if _, err := validateExplicitCanonicalProjectName(config.ProjectName); err != nil {
		return nil, err
	}
	base = append(base, "-p", config.ProjectName)
	return base, nil
}

func lifecycleCommandArgs(aggregate projectstore.ProjectAggregate, action generated.ProjectActionResponseAction) ([]string, error) {
	config := lifecycleConfigurationFromAggregate(aggregate)
	switch action {
	case generated.ProjectActionResponseActionProjectActionUp:
		return lifecycleUpArgs(aggregate, config)
	case generated.ProjectActionResponseActionProjectActionStop:
		base, err := composeProjectArgs(aggregate, config)
		if err != nil {
			return nil, err
		}
		return append(base, "stop"), nil
	case generated.ProjectActionResponseActionProjectActionRestart:
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
// 返回参数列表；配置无效时返回错误。
func lifecycleUpArgs(aggregate projectstore.ProjectAggregate, config LifecycleConfiguration) ([]string, error) {
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
	return args, nil
}

func lifecyclePullArgs(aggregate projectstore.ProjectAggregate, config LifecycleConfiguration) ([]string, error) {
	base, err := composeProjectArgs(aggregate, config)
	if err != nil {
		return nil, err
	}
	return append(base, "pull"), nil
}

func lifecycleRedeployDownArgs(aggregate projectstore.ProjectAggregate, config LifecycleConfiguration) ([]string, error) {
	base, err := composeProjectArgs(aggregate, config)
	if err != nil {
		return nil, err
	}
	return append(base, "down"), nil
}

func lifecycleBlockedResult(
	aggregate projectstore.ProjectAggregate,
	action generated.ProjectActionResponseAction,
	err error,
) ActionResult {
	return blockedActionResult(aggregate.Project.ID, action, []GuardResult{guardDetail("lifecycle_blocked", lifecycleBlockedReason(err))})
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
) (BatchActionItemResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return BatchActionItemResult{}, err
	}
	if item, ok, err := s.batchLifecycleActionItem(ctx, aggregate, projectID, request, actor); ok {
		return item, err
	}
	switch request.Action {
	case generated.ProjectBatchActionRequestActionUnregister:
		action, err := s.unregisterWithActor(ctx, aggregate, actor)
		return BatchActionItemResult{ActionResult: action}, err
	case generated.ProjectBatchActionRequestActionDestroy:
		confirmName := ""
		if request.ConfirmCanonicalProjectName != nil {
			confirmName = strings.TrimSpace(*request.ConfirmCanonicalProjectName)
		}
		destroyReq := DestroyRequest{
			RemoveNamedVolumes:          request.RemoveNamedVolumes,
			AutoUnregister:              request.AutoUnregister,
			ImagePrune:                  request.ImagePrune,
			DeleteWorkingDirectory:      request.DeleteWorkingDirectory,
			ConfirmCanonicalProjectName: confirmName,
			ActorID:                     &actor.id,
		}
		if _, blockErr := validateDestroyRequest(projectID, aggregate, destroyReq); blockErr != nil {
			return skippedBatchActionResult(projectID, generated.ProjectActionResponseActionProjectActionDestroy, "destroy_not_applicable"), nil
		}
		action, err := s.destroyAfterGuard(ctx, aggregate, destroyReq, actor)
		return BatchActionItemResult{ActionResult: action}, err
	default:
		return BatchActionItemResult{}, errProjectInvalidArgument
	}
}

func (s *Service) batchLifecycleActionItem(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	projectID uint64,
	request BatchActionRequest,
	actor actionActor,
) (BatchActionItemResult, bool, error) {
	switch request.Action {
	case generated.ProjectBatchActionRequestActionStart:
		item, err := s.batchLifecycleItem(
			ctx,
			aggregate,
			generated.ProjectActionResponseActionProjectActionUp,
		)
		return item, true, err
	case generated.ProjectBatchActionRequestActionStop:
		item, err := s.batchLifecycleItem(
			ctx,
			aggregate,
			generated.ProjectActionResponseActionProjectActionStop,
		)
		return item, true, err
	case generated.ProjectBatchActionRequestActionRestart:
		item, err := s.batchLifecycleItem(
			ctx,
			aggregate,
			generated.ProjectActionResponseActionProjectActionRestart,
		)
		return item, true, err
	case generated.ProjectBatchActionRequestActionRedeploy:
		if err := ensureProjectLifecycleReady(aggregate); err != nil {
			return skippedBatchActionResult(projectID, generated.ProjectActionResponseActionProjectActionRedeploy, lifecycleBlockedReason(err)), true, nil
		}
		action, err := s.redeployWithActor(ctx, aggregate, actor)
		return BatchActionItemResult{ActionResult: action}, true, err
	default:
		return BatchActionItemResult{}, false, nil
	}
}

func (s *Service) batchLifecycleItem(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	action generated.ProjectActionResponseAction,
) (BatchActionItemResult, error) {
	if err := ensureProjectLifecycleReady(aggregate); err != nil {
		return BatchActionItemResult{ActionResult: lifecycleBlockedResult(aggregate, action, err)}, nil
	}
	runtimeSummary, runtimeErr := s.runtimeSummary(ctx, aggregate)
	if skipReason, shouldSkip := skipBatchLifecycleAction(action, &runtimeSummary, runtimeErr); shouldSkip {
		return skippedBatchActionResult(aggregate.Project.ID, action, skipReason), nil
	}
	args, err := lifecycleCommandArgs(aggregate, action)
	if err != nil {
		return BatchActionItemResult{ActionResult: lifecycleBlockedResult(aggregate, action, err)}, nil
	}
	result, err := s.executeLifecycleActionWithAggregate(ctx, aggregate, action, args)
	return BatchActionItemResult{ActionResult: result}, err
}

func skipBatchLifecycleAction(
	action generated.ProjectActionResponseAction,
	runtimeSummary *moduleapi.ContainerProjectRuntimeSummary,
	runtimeErr error,
) (string, bool) {
	runtimeStatus := deriveProjectRuntimeStatus(runtimeSummary, runtimeErr)
	if runtimeStatus == nil {
		return "", false
	}
	switch action {
	case generated.ProjectActionResponseActionProjectActionUp:
		return skipBatchStartForStatus(*runtimeStatus)
	case generated.ProjectActionResponseActionProjectActionStop:
		return skipBatchStopForStatus(*runtimeStatus)
	case generated.ProjectActionResponseActionProjectActionRestart:
		return skipBatchRestartForStatus(*runtimeStatus)
	default:
		return "", false
	}
}

func skipBatchStartForStatus(status generated.ProjectRuntimeStatus) (string, bool) {
	switch status {
	case generated.ProjectRuntimeStatusRunning:
		return "already_running", true
	case generated.ProjectRuntimeStatusDegraded:
		return "already_partially_running", true
	case generated.ProjectRuntimeStatusTransitioning:
		return "currently_transitioning", true
	default:
		return "", false
	}
}

func skipBatchStopForStatus(status generated.ProjectRuntimeStatus) (string, bool) {
	switch status {
	case generated.ProjectRuntimeStatusStopped:
		return "already_stopped", true
	case generated.ProjectRuntimeStatusUnknown:
		return "runtime_status_unknown", true
	default:
		return "", false
	}
}

func skipBatchRestartForStatus(status generated.ProjectRuntimeStatus) (string, bool) {
	switch status {
	case generated.ProjectRuntimeStatusRunning, generated.ProjectRuntimeStatusDegraded, generated.ProjectRuntimeStatusStopped:
		return "", false
	case generated.ProjectRuntimeStatusTransitioning:
		return "currently_transitioning", true
	case generated.ProjectRuntimeStatusUnknown:
		return "runtime_status_unknown", true
	default:
		return "runtime_status_unknown", true
	}
}

func destroyDownArgs(aggregate projectstore.ProjectAggregate, removeNamedVolumes bool) ([]string, error) {
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

func (s *Service) applyDestroyWorkingDirectoryStep(
	aggregate projectstore.ProjectAggregate,
	request DestroyRequest,
	guardResults []GuardResult,
) ([]GuardResult, bool, error) {
	autoUnregister := request.AutoUnregister
	if request.DeleteWorkingDirectory {
		if err := deleteManagedWorkingDirectory(aggregate.Project.WorkingDirectory); err != nil {
			return nil, false, fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
		}
		guardResults = append(guardResults, guardCode("working_directory_deleted"))
		autoUnregister = true
		return guardResults, autoUnregister, nil
	}
	guardResults = append(guardResults, guardCode("working_directory_preserved"))
	return guardResults, autoUnregister, nil
}

func (s *Service) applyDestroyImagePruneStep(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	guardResults []GuardResult,
	imagePrune bool,
) (ActionResult, []GuardResult, error) {
	if !imagePrune {
		return ActionResult{}, guardResults, nil
	}
	output, err := s.runDockerCommand(ctx, aggregate.Project.WorkingDirectory, []string{"image", "prune", "-f"})
	if err != nil {
		return blockedActionResult(
				aggregate.Project.ID,
				generated.ProjectActionResponseActionProjectActionDestroy,
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
	if err := repository.UnregisterProject(ctx, projectstore.UnregisterProjectInput{
		ProjectID: projectID,
		ActorID:   &actor.id,
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
	items := make([]BatchActionItemResult, 0, len(request.ProjectIDs))
	for _, projectID := range request.ProjectIDs {
		result := actorAttributionBlockedResult(projectID, batchActionToProjectAction(request.Action), reason)
		items = append(items, BatchActionItemResult{ActionResult: result})
	}
	return actionActor{}, BatchActionResult{
		TotalCount:   len(request.ProjectIDs),
		BlockedCount: len(request.ProjectIDs),
		Items:        items,
	}, errProjectActorAttribution
}

func batchActionToProjectAction(action generated.ProjectBatchActionRequestAction) generated.ProjectActionResponseAction {
	switch action {
	case generated.ProjectBatchActionRequestActionStart:
		return generated.ProjectActionResponseActionProjectActionUp
	case generated.ProjectBatchActionRequestActionStop:
		return generated.ProjectActionResponseActionProjectActionStop
	case generated.ProjectBatchActionRequestActionRestart:
		return generated.ProjectActionResponseActionProjectActionRestart
	case generated.ProjectBatchActionRequestActionRedeploy:
		return generated.ProjectActionResponseActionProjectActionRedeploy
	case generated.ProjectBatchActionRequestActionUnregister:
		return generated.ProjectActionResponseActionProjectActionUnregister
	case generated.ProjectBatchActionRequestActionDestroy:
		return generated.ProjectActionResponseActionProjectActionDestroy
	default:
		return generated.ProjectActionResponseActionProjectActionRestart
	}
}

func skippedBatchActionResult(projectID uint64, action generated.ProjectActionResponseAction, reason string) BatchActionItemResult {
	messageKey := projectcontract.ProjectLifecycleBlocked.String()
	return BatchActionItemResult{
		ActionResult: ActionResult{
			ProjectID:  projectID,
			Action:     action,
			Result:     generated.ProjectActionResponseResultProjectActionResultBlocked,
			MessageKey: &messageKey,
			Message:    &messageKey,
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
