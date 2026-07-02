package project

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	generated "graft/server/internal/contract/openapi/generated"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

// Up executes docker compose up -d within the project's registered working directory.
func (s *Service) Up(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.runLifecycleAction(ctx, projectID, actorID, generated.ProjectActionUp, []string{"compose", "up", "-d"})
}

// Down executes docker compose down for the registered project without removing volumes by default.
func (s *Service) Down(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.runLifecycleAction(ctx, projectID, actorID, generated.ProjectActionDown, []string{"compose", "down"})
}

// Restart executes docker compose restart for the registered project.
func (s *Service) Restart(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	return s.runLifecycleAction(ctx, projectID, actorID, generated.ProjectActionRestart, []string{"compose", "restart"})
}

// Unregister removes the project registry record without touching host files.
func (s *Service) Unregister(ctx context.Context, projectID uint64, actorID *uint64) (ActionResult, error) {
	if _, err := s.getAggregate(ctx, projectID); err != nil {
		return ActionResult{}, err
	}
	repository, err := s.repositoryOrErr()
	if err != nil {
		return ActionResult{}, err
	}
	if err := repository.UnregisterProject(ctx, projectstore.UnregisterProjectInput{
		ProjectID: projectID,
		ActorID:   actorID,
	}); err != nil {
		return ActionResult{}, mapStoreError(err)
	}
	messageKey := projectcontract.ProjectUnregisterCompleted.String()
	return ActionResult{
		ProjectID:  projectID,
		Action:     generated.ProjectActionUnregister,
		Result:     generated.ProjectActionResultCompleted,
		MessageKey: &messageKey,
		Message:    &messageKey,
		GuardResults: []GuardResult{
			guardCode("registry_deleted"),
			guardCode("working_directory_preserved"),
			guardCode("runtime_state_not_persisted"),
		},
	}, nil
}

// Destroy executes guarded teardown steps and then unregisters the project record.
func (s *Service) Destroy(ctx context.Context, projectID uint64, request DestroyRequest) (ActionResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ActionResult{}, err
	}
	if result, blockErr := validateDestroyRequest(projectID, aggregate, request); blockErr != nil {
		return result, blockErr
	}
	return s.destroyAfterGuard(ctx, aggregate, request)
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
		return blockedActionResult(projectID, generated.ProjectActionDestroy, append(guardResults, guardCode("confirm_canonical_project_name_mismatch"))), errProjectDestroyBlocked
	}
	guardResults = append(guardResults, guardCode("confirm_canonical_project_name_matched"))

	if request.RemoveNamedVolumes {
		guardResults = append(guardResults, guardDetail("remove_named_volumes_blocked", "phase1_volume_exclusivity_not_proven"))
		return blockedActionResult(projectID, generated.ProjectActionDestroy, guardResults), errProjectDestroyBlocked
	}

	if request.DeleteWorkingDirectory && aggregate.Project.OwnershipMode != projectcontract.OwnershipModeManagedRootDedicated.String() {
		guardResults = append(guardResults, guardDetail("delete_working_directory_blocked", "ownership_mode_external"))
		return blockedActionResult(projectID, generated.ProjectActionDestroy, guardResults), errProjectDestroyBlocked
	}
	return ActionResult{}, nil
}

func (s *Service) destroyAfterGuard(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	request DestroyRequest,
) (ActionResult, error) {
	projectID := aggregate.Project.ID
	guardResults := []GuardResult{guardCode("confirm_canonical_project_name_matched")}
	if _, err := s.runLifecycleActionWithAggregate(ctx, aggregate, request.ActorID, generated.ProjectActionDown, []string{"compose", "down"}); err != nil {
		return ActionResult{}, err
	}
	guardResults = append(guardResults, guardCode("compose_down_completed"))

	if request.DeleteWorkingDirectory {
		if err := deleteManagedWorkingDirectory(aggregate.Project.WorkingDirectory); err != nil {
			return ActionResult{}, fmt.Errorf("%w: %v", errProjectUnsupportedLifecycle, err)
		}
		guardResults = append(guardResults, guardCode("working_directory_deleted"))
	} else {
		guardResults = append(guardResults, guardCode("working_directory_preserved"))
	}

	repository, err := s.repositoryOrErr()
	if err != nil {
		return ActionResult{}, err
	}
	if err := repository.UnregisterProject(ctx, projectstore.UnregisterProjectInput{
		ProjectID: projectID,
		ActorID:   request.ActorID,
	}); err != nil {
		return ActionResult{}, mapStoreError(err)
	}
	guardResults = append(guardResults, guardCode("registry_deleted"))
	messageKey := projectcontract.ProjectDestroyCompleted.String()
	return ActionResult{
		ProjectID:    projectID,
		Action:       generated.ProjectActionDestroy,
		Result:       generated.ProjectActionResultCompleted,
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
		Result:       generated.ProjectActionResultBlocked,
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
	args []string,
) (ActionResult, error) {
	aggregate, err := s.getAggregate(ctx, projectID)
	if err != nil {
		return ActionResult{}, err
	}
	return s.runLifecycleActionWithAggregate(ctx, aggregate, actorID, action, args)
}

func (s *Service) runLifecycleActionWithAggregate(
	ctx context.Context,
	aggregate projectstore.ProjectAggregate,
	_ *uint64,
	action generated.ProjectActionResponseAction,
	args []string,
) (ActionResult, error) {
	if err := ensureProjectLifecycleReady(aggregate); err != nil {
		return blockedActionResult(aggregate.Project.ID, action, []GuardResult{guardDetail("lifecycle_blocked", "refresh_required")}), err
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
		Result:     generated.ProjectActionResultCompleted,
		MessageKey: &messageKey,
		Message:    &messageKey,
		GuardResults: []GuardResult{
			guardDetail("command", strings.Join(args, " ")),
			guardDetail("host_scope", aggregate.Project.HostScope),
		},
	}, nil
}

func (s *Service) runComposeCommand(ctx context.Context, aggregate projectstore.ProjectAggregate, args []string) (string, error) {
	// #nosec G204 -- binary is fixed to docker and args are validated command fragments, not shell-expanded input.
	command := exec.CommandContext(ctx, "docker", args...)
	command.Dir = aggregate.Project.WorkingDirectory
	command.Env = os.Environ()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	return strings.TrimSpace(stdout.String() + "\n" + stderr.String()), err
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
	if aggregate.Project.LastRefreshStatus != projectcontract.RefreshStatusSuccess.String() {
		return errProjectUnsupportedLifecycle
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
		Result:       generated.ProjectActionResultBlocked,
		MessageKey:   &messageKey,
		Message:      &messageKey,
		GuardResults: append([]GuardResult(nil), guardResults...),
	}
}

// lifecycleMessageKey 返回指定生命周期动作对应的完成消息键。
func lifecycleMessageKey(action generated.ProjectActionResponseAction) projectcontract.MessageKey {
	switch action {
	case generated.ProjectActionUp:
		return projectcontract.ProjectUpCompleted
	case generated.ProjectActionDown:
		return projectcontract.ProjectDownCompleted
	case generated.ProjectActionRestart:
		return projectcontract.ProjectRestartCompleted
	case generated.ProjectActionDestroy:
		return projectcontract.ProjectDestroyCompleted
	case generated.ProjectActionDeploy:
		return projectcontract.ProjectDeployCompleted
	case generated.ProjectActionUnregister:
		return projectcontract.ProjectUnregisterCompleted
	default:
		return projectcontract.ProjectLifecycleAccepted
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
