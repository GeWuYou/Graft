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
	ID                        int64
	DisplayName               string
	Provider                  string
	Available                 bool
	ProviderCapabilityProfile string
	ProviderCapabilityVersion string
	SupportedDrivers          []string
	SupportedPlatforms        []string
	WorkspaceLocalities       []string
	SnapshotDeliveryModes     []string
	BuildFeatures             []string
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
	TargetID              int64
	BuilderScope          string
	ProviderID            string
	CapabilityProfile     string
	CapabilityVersion     string
	Available             bool
	Running               int
	Queued                int
	AllocatableSlots      int
	ObservedAt            time.Time
	ExpiresAt             time.Time
	SourceRef             string
	Region                string
	AffinityKey           string
	Provenance            string
	Integrity             string
	UnsupportedDimensions []string
}

// FreshAt 判断调度器在指定时刻是否可以使用该快照；失效或不自洽的快照必须 fail-closed。
func (s BuilderTelemetrySnapshot) FreshAt(now time.Time) bool {
	return s.validIdentity() && s.validCapacity() && s.validWindow(now)
}

func (s BuilderTelemetrySnapshot) validIdentity() bool {
	return s.TargetID > 0 && s.Available && s.SourceRef != ""
}

func (s BuilderTelemetrySnapshot) validCapacity() bool {
	return s.Running >= 0 && s.Queued >= 0 && s.AllocatableSlots >= 0
}

func (s BuilderTelemetrySnapshot) validWindow(now time.Time) bool {
	return !s.ObservedAt.IsZero() && !s.ExpiresAt.IsZero() && now.Before(s.ExpiresAt) && !s.ObservedAt.After(now)
}

// Conformant 判断快照是否具备进入动态 Placement 所需的 provider 证明。
// 该门槛与 FreshAt 分离，使现有静态诊断读取不会被误当作动态调度 authority。
func (s BuilderTelemetrySnapshot) Conformant() bool {
	return s.BuilderScope != "" && s.ProviderID != "" && s.CapabilityProfile != "" && s.CapabilityVersion != "" && s.Provenance != "" && s.Integrity != ""
}

// DynamicPlacementConformantAt 仅在每个调度基础维度都有可信值时允许动态 Placement。
// 未知维度必须由 provider 显式列出；缺失的运行、排队、容量或健康事实一律拒绝。
func (s BuilderTelemetrySnapshot) DynamicPlacementConformantAt(now time.Time) bool {
	if !s.FreshAt(now) || !s.Conformant() {
		return false
	}
	for _, dimension := range s.UnsupportedDimensions {
		switch dimension {
		case "cache_state":
			// 当前 OCI 准入契约将缓存遥测列为可选维度。
		case "running_builds", "queue", "allocatable_slots", "health", "capability_profile", "capability_version", "provenance", "integrity":
			return false
		default:
			return false
		}
	}
	return true
}

// RuntimeTargetBuilderTelemetryReader 是 Runtime Target 对构建调度器的窄化遥测边界。
// 实现必须返回带 freshness 和来源证明的目标事实，不能把端点、凭据或运行时内部对象泄漏给 Build。
type RuntimeTargetBuilderTelemetryReader interface {
	ListBuilderTelemetry(context.Context, []int64) ([]BuilderTelemetrySnapshot, error)
	ConformBuilderTelemetry(context.Context, []int64) (bool, error)
}

// BuilderTelemetryProvider 是 Runtime Target facade 下方的 provider-owned 观察适配器。
// Provider 必须返回 Builder 范围事实；Docker/host 指标、UI 投影和 Task JSON 不得实现此接口。
type BuilderTelemetryProvider interface {
	ListBuilderTelemetry(context.Context, []int64) ([]BuilderTelemetrySnapshot, error)
	ConformBuilderTelemetry(context.Context, []int64) (bool, error)
}

// BuilderTelemetryReport 是 Builder Agent 向 Runtime Target 控制平面提交的已签名观测。
// Signature 覆盖除自身外的全部字段；控制平面验证后自行生成完整性摘要。
type BuilderTelemetryReport struct {
	AgentID               string
	TargetID              int64
	Sequence              int64
	BuilderScope          string
	ProviderID            string
	CapabilityProfile     string
	CapabilityVersion     string
	Available             bool
	Running               int
	Queued                int
	AllocatableSlots      int
	ObservedAt            time.Time
	ExpiresAt             time.Time
	SourceRef             string
	Provenance            string
	UnsupportedDimensions []string
	Signature             []byte
}

// RuntimeTargetBuilderTelemetryControlPlane 是 Runtime Target 提供给已绑定 Builder Agent 的私有写入边界。
// 它不是 Build API；未经 Agent 公钥验证的报告不会进入持久化遥测账本。
type RuntimeTargetBuilderTelemetryControlPlane interface {
	ProvisionBuilderTelemetryAgent(context.Context, BuilderTelemetryAgentRegistration) error
	SubmitBuilderTelemetry(context.Context, BuilderTelemetryReport) error
}

// BuilderTelemetryAgentRegistration 是 Runtime Target 控制平面为已绑定 Agent 保存的验证公钥。
// 它只能由 provider/控制平面装配调用，不能由 Build、UI 或普通 Task 元数据建立。
type BuilderTelemetryAgentRegistration struct {
	TargetID          int64
	AgentID           string
	ProviderID        string
	BuilderScope      string
	CapabilityProfile string
	CapabilityVersion string
	PublicKey         []byte
	Enabled           bool
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
