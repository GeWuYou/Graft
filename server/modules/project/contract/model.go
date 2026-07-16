package contract

// SourceKind 标识稳定的项目来源契约；规范 owner 为 server/modules/project/contract。
type SourceKind string

// HostScope 标识项目注册使用的稳定宿主范围契约。
type HostScope string

// OwnershipMode 标识项目生命周期守卫使用的稳定所有权契约。
type OwnershipMode string

// ManagedRootStatus 标识受管创建使用的稳定受管根目录就绪契约。
type ManagedRootStatus string

// DriftStatus 标识项目配置检查使用的稳定漂移状态契约。
type DriftStatus string

// CanonicalProjectNameSource 标识规范项目名称的稳定来源契约。
type CanonicalProjectNameSource string

// FileKind 标识稳定的项目文件类型契约。
type FileKind string

// FileRole 标识稳定的项目文件角色契约。
type FileRole string

// LifecycleStrategyKind 标识稳定的项目生命周期执行策略契约。
type LifecycleStrategyKind string

// LifecycleReviewStatus 标识稳定的生命周期配置审核状态契约。
type LifecycleReviewStatus string

const (
	// SourceKindImported 表示导入 Graft 的外部创建 Compose 项目。
	SourceKindImported SourceKind = "imported"
	// SourceKindManaged 表示由 Graft 创建的受管项目。
	SourceKindManaged SourceKind = "managed"
	// SourceKindTemplate 表示由模板派生的项目来源。
	SourceKindTemplate SourceKind = "template"

	// HostScopeLocal 表示第一阶段仅支持本地主机的项目范围。
	HostScopeLocal HostScope = "local"

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

	// CanonicalProjectNameSourceComputed 表示根据 Compose 输入计算出的运行时身份。
	CanonicalProjectNameSourceComputed CanonicalProjectNameSource = "computed"
	// CanonicalProjectNameSourceOverride 表示导入时显式覆盖的运行时身份。
	CanonicalProjectNameSourceOverride CanonicalProjectNameSource = "override"

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
func (v SourceKind) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v HostScope) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v OwnershipMode) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v ManagedRootStatus) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v DriftStatus) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v CanonicalProjectNameSource) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v FileKind) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v FileRole) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v LifecycleStrategyKind) String() string { return string(v) }

// String 返回线格式值，供跨边界契约序列化。
func (v LifecycleReviewStatus) String() string { return string(v) }
