package contract

// DeploymentAdapterKind 标识应用定义格式及其部署适配器。
// 运行目标和 Provider 只声明能力，不能替代此契约。
type DeploymentAdapterKind string

// SourceType 标识稳定的应用来源契约；规范 owner 为 server/modules/project/contract。
type SourceType string

// OwnershipMode 标识应用生命周期守卫使用的稳定所有权契约。
type OwnershipMode string

// ManagedRootStatus 标识受管创建使用的稳定受管根目录就绪契约。
type ManagedRootStatus string

// DriftStatus 标识应用配置检查使用的稳定漂移状态契约。
type DriftStatus string

// ComposeProjectNameSource 标识 Compose 运行时名称的稳定来源契约。
type ComposeProjectNameSource string

// FileKind 标识稳定的应用文件类型契约。
type FileKind string

// FileRole 标识稳定的应用文件角色契约。
type FileRole string

// LifecycleStrategyKind 标识稳定的应用生命周期执行策略契约。
type LifecycleStrategyKind string

// LifecycleReviewStatus 标识稳定的生命周期配置审核状态契约。
type LifecycleReviewStatus string

const (
	// DeploymentAdapterKindCompose 表示由 Compose Specification 驱动的应用定义。
	DeploymentAdapterKindCompose DeploymentAdapterKind = "compose"

	// SourceTypeImported 表示导入 Graft 的外部 Compose 应用。
	SourceTypeImported SourceType = "imported"
	// SourceTypeManaged 表示由 Graft 创建的受管应用。
	SourceTypeManaged SourceType = "managed"
	// SourceTypeTemplate 表示由模板派生的应用来源。
	SourceTypeTemplate SourceType = "template"

	// OwnershipModeExternal 表示项目由外部工作目录提供内容。
	OwnershipModeExternal OwnershipMode = "external"
	// OwnershipModeManagedRootDedicated 表示项目工作目录由受管根目录拥有。
	OwnershipModeManagedRootDedicated OwnershipMode = "managed-root-dedicated"

	// ManagedRootStatusUnconfigured 表示缺少受管根目录系统配置。
	ManagedRootStatusUnconfigured ManagedRootStatus = "unconfigured"
	// ManagedRootStatusReady 表示受管根目录配置已通过有界权威校验。
	ManagedRootStatusReady ManagedRootStatus = "ready"
	// ManagedRootStatusInvalid 表示受管根目录配置未通过有界权威校验。
	ManagedRootStatusInvalid ManagedRootStatus = "invalid"

	// DriftStatusUnknown 表示项目漂移状态尚未确定。
	DriftStatusUnknown DriftStatus = "unknown"
	// DriftStatusClean 表示当前观察到的文件与最近成功快照一致。
	DriftStatusClean DriftStatus = "clean"
	// DriftStatusChanged 表示观察到的文件与最近成功快照不同。
	DriftStatusChanged DriftStatus = "changed"
	// DriftStatusMissing 表示项目所需文件缺失。
	DriftStatusMissing DriftStatus = "missing"

	// ComposeProjectNameSourceComputed 表示根据 Compose 输入计算出的运行时身份。
	ComposeProjectNameSourceComputed ComposeProjectNameSource = "computed"
	// ComposeProjectNameSourceOverride 表示导入时显式覆盖的运行时身份。
	ComposeProjectNameSourceOverride ComposeProjectNameSource = "override"

	// FileKindCompose 表示 Compose 定义文件。
	FileKindCompose FileKind = "compose"
	// FileKindEnv 表示环境文件。
	FileKindEnv FileKind = "env"

	// FileRolePrimary 表示主 Compose 定义文件。
	FileRolePrimary FileRole = "primary"
	// FileRoleOverride 表示按顺序应用的 Compose 覆盖文件。
	FileRoleOverride FileRole = "override"
	// FileRoleEnv 表示解析时使用的环境文件。
	FileRoleEnv FileRole = "env"

	// LifecycleStrategyKindStandard 表示项目拥有的标准 Compose 生命周期策略。
	LifecycleStrategyKindStandard LifecycleStrategyKind = "standard"

	// LifecycleReviewStatusReviewRequired 表示导入或变更后的生命周期配置仍需操作员确认。
	LifecycleReviewStatusReviewRequired LifecycleReviewStatus = "review_required"
	// LifecycleReviewStatusConfirmed 表示生命周期配置已确认，可以执行 Compose 动作。
	LifecycleReviewStatusConfirmed LifecycleReviewStatus = "confirmed"
)

// String 返回线格式值，供跨边界契约序列化。
func (v DeploymentAdapterKind) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v SourceType) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v OwnershipMode) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v ManagedRootStatus) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v DriftStatus) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v ComposeProjectNameSource) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v FileKind) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v FileRole) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v LifecycleStrategyKind) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v LifecycleReviewStatus) String() string { return string(v) }
