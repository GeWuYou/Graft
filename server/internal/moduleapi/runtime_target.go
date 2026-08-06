package moduleapi

import (
	"context"
	"time"
)

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

// BuildRuntimeTargetSummary 是 Build domain 可消费的运行目标构建能力投影。
// 它只公开调度所需的能力事实，不公开连接端点或凭据。
type BuildRuntimeTargetSummary struct {
	ID                    int64
	DisplayName           string
	Provider              string
	Available             bool
	SupportedDrivers      []string
	SupportedPlatforms    []string
	WorkspaceLocalities   []string
	SnapshotDeliveryModes []string
}

const (
	// SnapshotDeliveryModeTargetLocal 表示 Provider 在目标自身可直接读取 Build-owned Snapshot 物化内容。
	SnapshotDeliveryModeTargetLocal = "target-local"
	// SnapshotDeliveryModeProviderTransfer 预留给经过 Provider 证明的跨目标 Snapshot 传输适配器。
	SnapshotDeliveryModeProviderTransfer = "provider-transfer"
)

// BuildRuntimeTargetReader 解析具备构建能力的运行目标，不携带调用者授权范围。
// Build 执行端必须结合 RuntimeTargetBuildAssignmentReader 复核调用者权限。
type BuildRuntimeTargetReader interface {
	ReadBuildTarget(ctx context.Context, targetID int64) (BuildRuntimeTargetSummary, error)
}

// RuntimeTargetBuildAssignmentReader 仅暴露构建安全的运行目标使用范围。
// 该边界避免 Build 将目标选择授权扩大为目标管理或连接访问权限。
type RuntimeTargetBuildAssignmentReader interface {
	ListAssignedBuildTargets(ctx context.Context, userID uint64) ([]BuildRuntimeTargetSummary, error)
	CanUseBuildTarget(ctx context.Context, userID uint64, targetID int64) (bool, error)
}

// BuilderTelemetrySnapshot 是 Runtime/Infrastructure 提供给 Build Scheduler 的目标级事实。
// 它必须带有来源、观察时间和过期时间；UI 资源摘要、主机负载或静态 Builder 标签不能替代该事实。
type BuilderTelemetrySnapshot struct {
	TargetID    int64
	Available   bool
	Capacity    int
	Running     int
	Queued      int
	ObservedAt  time.Time
	ExpiresAt   time.Time
	SourceRef   string
	Region      string
	AffinityKey string
}

// FreshAt 判断调度器在指定时刻是否可以使用该快照；失效或不自洽的快照必须 fail-closed。
func (s BuilderTelemetrySnapshot) FreshAt(now time.Time) bool {
	return s.validIdentity() && s.validCapacity() && s.validWindow(now)
}

func (s BuilderTelemetrySnapshot) validIdentity() bool {
	return s.TargetID > 0 && s.Available && s.SourceRef != ""
}

func (s BuilderTelemetrySnapshot) validCapacity() bool {
	return s.Capacity > 0 && s.Running >= 0 && s.Queued >= 0 && s.Running <= s.Capacity
}

func (s BuilderTelemetrySnapshot) validWindow(now time.Time) bool {
	return !s.ObservedAt.IsZero() && !s.ExpiresAt.IsZero() && now.Before(s.ExpiresAt) && !s.ObservedAt.After(now)
}

// RuntimeTargetBuilderTelemetryReader 是 Runtime Target 对构建调度器的窄化遥测边界。
// 实现必须返回带 freshness 和来源证明的目标事实，不能把端点、凭据或运行时内部对象泄漏给 Build。
type RuntimeTargetBuilderTelemetryReader interface {
	ListBuilderTelemetry(context.Context, []int64) ([]BuilderTelemetrySnapshot, error)
}

// RuntimeTargetProviderConnection 是 provider 私有执行边界使用的连接事实；不得进入 HTTP、Build Plan 或 Task metadata。
type RuntimeTargetProviderConnection struct {
	TargetID       int64
	Provider       string
	Endpoint       string
	ConnectionKind string
}

// RuntimeTargetProviderConnectionReader 由 Runtime Target 提供给具体 provider，统一负责可用性和 build capability 校验。
type RuntimeTargetProviderConnectionReader interface {
	GetProviderConnection(context.Context, int64) (RuntimeTargetProviderConnection, error)
}
