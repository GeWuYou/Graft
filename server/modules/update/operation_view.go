package update

import "time"

// OperationView 是 server 对 runner 状态卷或终态历史的只读 API 投影。
// 活动操作始终由 RunnerState 生成；数据库只为已结算终态提供历史读取。
type OperationView struct {
	OperationID                string      `json:"operation_id"`
	Operation                  string      `json:"operation"`
	RunnerID                   string      `json:"runner_id"`
	SourceVersion              string      `json:"source_version"`
	TargetVersion              string      `json:"target_version"`
	DeploymentStrategy         string      `json:"deployment_strategy"`
	Phase                      RunnerPhase `json:"phase"`
	Progress                   int         `json:"progress"`
	Message                    string      `json:"message"`
	StartedAt                  time.Time   `json:"started_at"`
	UpdatedAt                  time.Time   `json:"updated_at"`
	FinishedAt                 *time.Time  `json:"finished_at,omitempty"`
	Error                      string      `json:"error,omitempty"`
	FailureDiagnosticAvailable bool        `json:"failure_diagnostic_available,omitempty"`
	StateSource                string      `json:"state_source"`
	StateAvailable             bool        `json:"state_available"`
}

// OperationLaunchAcknowledgement 只确认 runner 启动请求已被 Docker 接受。
// 调用方必须重新读取操作快照，不能把该确认当作生命周期状态。
type OperationLaunchAcknowledgement struct {
	OperationID string `json:"operation_id"`
	RunnerID    string `json:"runner_id"`
}

func updateOperationViewFromRunnerState(state RunnerState) OperationView {
	return OperationView{OperationID: state.OperationID, Operation: state.Operation, RunnerID: state.RunnerID, SourceVersion: state.SourceVersion, TargetVersion: state.TargetVersion, DeploymentStrategy: state.Strategy, Phase: state.Phase, Progress: state.Progress, Message: state.Message, StartedAt: state.StartedAt, UpdatedAt: state.UpdatedAt, FinishedAt: state.FinishedAt, Error: state.Error, StateSource: "runner_state", StateAvailable: true}
}

func updateOperationViewFromHistory(operation ComposeUpdateOperation) OperationView {
	phase := RunnerPhaseReady
	progress := 0
	message := "runner_starting"
	switch operation.Outcome {
	case ExecutionOutcomeSuccess:
		phase, progress, message = RunnerPhaseSuccess, 100, "update_completed"
	case ExecutionOutcomeRecovered:
		phase, progress, message = RunnerPhaseRollback, 100, "rollback_completed"
	case ExecutionOutcomeFailed, ExecutionOutcomeNeedsAttention:
		phase, progress, message = RunnerPhaseFailed, 100, "update_failed"
	}
	return OperationView{OperationID: operation.OperationID, Operation: "self_update", RunnerID: operation.RunnerID, SourceVersion: operation.SourceVersion, TargetVersion: operation.TargetVersion, DeploymentStrategy: string(operation.DeploymentStrategy), Phase: phase, Progress: progress, Message: message, StartedAt: operation.StartedAt, UpdatedAt: operation.UpdatedAt, FinishedAt: operation.FinishedAt, Error: operation.FailureCode, FailureDiagnosticAvailable: operation.FailureDiagnosticAvailable, StateSource: "terminal_history", StateAvailable: true}
}

func updateOperationViewFromUnavailableRunnerState(operation ComposeUpdateOperation) OperationView {
	return OperationView{OperationID: operation.OperationID, Operation: "self_update", RunnerID: operation.RunnerID, SourceVersion: operation.SourceVersion, TargetVersion: operation.TargetVersion, DeploymentStrategy: string(operation.DeploymentStrategy), Phase: RunnerPhaseReady, Progress: 0, Message: "runner_state_unavailable", StartedAt: operation.StartedAt, UpdatedAt: operation.UpdatedAt, StateSource: "runner_state_unavailable", StateAvailable: false}
}
