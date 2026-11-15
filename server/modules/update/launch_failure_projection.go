package update

import (
	"context"
	"fmt"

	"graft/server/internal/moduleapi"
)

// failedLaunchOperationView 投影 Task 已确认的启动前失败，保留持久化操作和材料供受控重试。
func (s *RolloutService) failedLaunchOperationView(ctx context.Context, operation ComposeUpdateOperation) (OperationView, bool, error) {
	if s.taskQuery == nil {
		return OperationView{}, false, nil
	}
	task, err := s.taskQuery.GetTask(ctx, operation.TaskID)
	if err != nil {
		return OperationView{}, false, fmt.Errorf("read update launch task: %w", err)
	}
	if !knownFailedLaunchTask(operation, task) {
		return OperationView{}, false, nil
	}
	stages, err := s.taskQuery.ListTaskStages(ctx, task.ID)
	if err != nil {
		return OperationView{}, false, fmt.Errorf("read update launch stages: %w", err)
	}
	if len(stages) != 2 || !failedControllerLaunchStage(stages[0], *task.FailureCode) || !unstartedControllerResultStage(stages[1]) {
		return OperationView{}, false, nil
	}
	operation.Outcome, operation.FailureCode = ExecutionOutcomeFailed, *task.FailureCode
	view := updateOperationViewFromHistory(operation)
	view.StateSource = "task_recovery"
	return view, true, nil
}

func knownFailedLaunchTask(operation ComposeUpdateOperation, task moduleapi.TaskView) bool {
	return task.ID == operation.TaskID && task.Type == composeUpdateTaskType &&
		task.Owner.Type == platformUpdateTaskOwnerType && task.Owner.ID == operation.OperationID &&
		task.Status == moduleapi.TaskStatusFailed && task.FailureCode != nil &&
		knownPrelaunchFailureCode(*task.FailureCode)
}

func knownPrelaunchFailureCode(code string) bool {
	switch code {
	case "invalid_execution_intent", "unsupported_execution_operation", "local_material_resolve_failed", "local_controller_pull_failed":
		return true
	default:
		return false
	}
}

func failedControllerLaunchStage(stage moduleapi.TaskStageView, code string) bool {
	return stage.ID != 0 && stage.Key == "controller_launch" && stage.Sequence == 1 &&
		stage.ExecutorType == composeUpdateLaunchExecutor && stage.Status == moduleapi.StageStatusFailed &&
		stage.Attempt > 0 && stage.FailureCode != nil && *stage.FailureCode == code && supportedLaunchExecution(stage)
}

func supportedLaunchExecution(stage moduleapi.TaskStageView) bool {
	switch stage.ExecutionMode {
	case "local":
		return stage.ExecutionVersion == "local-sdk/v1"
	case "agent":
		return stage.ExecutionVersion == composeUpdateCapabilityVersion
	default:
		return false
	}
}

func unstartedControllerResultStage(stage moduleapi.TaskStageView) bool {
	return stage.ID != 0 && stage.Key == "controller_result" && stage.Sequence == 2 &&
		stage.ExecutorType == composeUpdateExecutor && stage.Status == moduleapi.StageStatusPending && stage.Attempt == 0 &&
		stage.ExecutionMode == "controller" && stage.ExecutionVersion == runnerProtocol
}
