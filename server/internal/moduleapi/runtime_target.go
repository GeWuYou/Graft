package moduleapi

import "context"

// RuntimeTargetSummary 是运行时资源向调用方暴露的最小目标身份投影。
type RuntimeTargetSummary struct {
	ID          int64
	DisplayName string
	Provider    string
}

// RuntimeTargetReader 是 provider 模块使用的窄化运行时目标能力。
// 该接口有意不暴露 endpoint 或凭据，避免调用方越过目标模块的信任边界。
type RuntimeTargetReader interface {
	ReadDockerTarget(context.Context, *int64) (RuntimeTargetSummary, error)
	ListDockerTargets(context.Context) ([]RuntimeTargetSummary, error)
}

// ComposeRuntimeTargetSummary 是 Compose 应用可消费的、按能力限定的运行时目标身份投影。
type ComposeRuntimeTargetSummary struct {
	ID           int64
	DisplayName  string
	Provider     string
	Capabilities []string
	Available    bool
}

// ComposeProjectNameState 表示针对一个运行时目标检查 Compose 项目名后的 provider 无关结果。
type ComposeProjectNameState string

const (
	// ComposeProjectNameStateAvailable 表示目标可访问且未占用该项目名。
	ComposeProjectNameStateAvailable ComposeProjectNameState = "available"
	// ComposeProjectNameStateOccupied 表示目标已经拥有同名 Compose 资源。
	ComposeProjectNameStateOccupied ComposeProjectNameState = "occupied"
	// ComposeProjectNameStateUnavailable 表示当前无法查询目标。
	ComposeProjectNameStateUnavailable ComposeProjectNameState = "unavailable"
	// ComposeProjectNameStateError 表示 provider 查询发生预期外失败。
	ComposeProjectNameStateError ComposeProjectNameState = "error"
)

// ComposeProjectNameAvailability 是 provider 无关的项目名占用结果。
type ComposeProjectNameAvailability struct {
	State ComposeProjectNameState
}

// ComposeRuntimeTargetReader 解析具备 Compose 执行能力且可访问 workspace 的运行时目标。
type ComposeRuntimeTargetReader interface {
	ReadComposeTarget(context.Context, *int64) (ComposeRuntimeTargetSummary, error)
	ListComposeTargets(context.Context) ([]ComposeRuntimeTargetSummary, error)
	CheckComposeProjectName(context.Context, int64, string) (ComposeProjectNameAvailability, error)
}

// RuntimeTargetDeploymentAssignmentReader 仅暴露部署安全的运行目标使用范围。
// 此边界刻意不暴露端点和凭据，避免应用模块将部署授权扩大为目标管理权限。
type RuntimeTargetDeploymentAssignmentReader interface {
	ListAssignedComposeTargets(ctx context.Context, userID uint64) ([]ComposeRuntimeTargetSummary, error)
	CanUseComposeTarget(ctx context.Context, userID uint64, targetID uint64) (bool, error)
}
