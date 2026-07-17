// Package store 定义 Compose Application 管理模块的持久化契约。
package store

import (
	"context"
	"errors"
	"time"
)

const (
	defaultLifecycleWaitTimeoutSeconds = 120
	minLifecycleWaitTimeoutSeconds     = 1
	maxLifecycleWaitTimeoutSeconds     = 3600

	// ApplicationListSortCreatedAtDesc 按创建时间从新到旧排列应用。
	ApplicationListSortCreatedAtDesc = "created_at:desc"
	// ApplicationListSortCreatedAtAsc 按创建时间从旧到新排列应用。
	ApplicationListSortCreatedAtAsc = "created_at:asc"
)

var (
	// ErrInvalidInput 表示仓储输入违反模块持久化契约。
	ErrInvalidInput = errors.New("application invalid input")
	// ErrApplicationNotFound 表示没有存活应用匹配给定标识。
	ErrApplicationNotFound = errors.New("application not found")
	// ErrApplicationConflict 表示导入或更新与存活应用记录冲突。
	ErrApplicationConflict = errors.New("application conflict")
	// ErrFileNotFound 表示应用范围内不存在给定文件记录。
	ErrFileNotFound = errors.New("application file not found")
)

// Application 保存一条 Compose Application 注册记录。
type Application struct {
	ApplicationRecordID      uint64
	ApplicationID            string
	DeploymentAdapterKind    string
	ApplicationName          *string
	WorkspacePath            string
	ComposeProjectName       string
	ComposeProjectNameSource string
	RuntimeTargetID          *uint64
	DisplayName              string
	SourceType               string
	OwnershipMode            string
	SourceMetadata           map[string]string
	LifecycleStrategyKind    string
	LifecycleReviewStatus    string
	LifecycleConfig          LifecycleConfig
	LastObservedConfigHash   string
	WorkspaceAnnotations     map[string]string
	LastDriftCheckedAt       *time.Time
	DriftStatus              string
	CreatedBy                *uint64
	UpdatedBy                *uint64
	DeletedBy                *uint64
	CreatedAt                time.Time
	UpdatedAt                time.Time
	DeletedAt                int64
}

// ApplicationFile 保存一个有序应用文件引用。
type ApplicationFile struct {
	ID                  uint64
	ApplicationRecordID uint64
	Kind                string
	Role                string
	AbsolutePath        string
	DisplayPath         string
	OrderIndex          int
	LastObservedHash    string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Snapshot 保存最近一次成功规范化的 Compose 快照。
type Snapshot struct {
	ApplicationRecordID    uint64
	NormalizedComposeJSON  []byte
	ConfigHash             string
	DeclaredServiceCount   int
	DeclaredServicesDigest string
	RefreshedAt            time.Time
}

// LifecycleConfig 保存应用拥有的生命周期执行语义。
type LifecycleConfig struct {
	Profiles                 []string `json:"profiles"`
	DownBeforeRedeploy       bool     `json:"down_before_redeploy"`
	PullBeforeRedeploy       bool     `json:"pull_before_redeploy"`
	BuildBeforeUp            bool     `json:"build_before_up"`
	ForceRecreate            bool     `json:"force_recreate"`
	RemoveOrphans            bool     `json:"remove_orphans"`
	WaitAfterUp              bool     `json:"wait_after_up"`
	WaitTimeoutSeconds       int      `json:"wait_timeout_seconds"`
	RenewAnonVolumes         bool     `json:"renew_anon_volumes"`
	PruneImagesAfterRedeploy bool     `json:"prune_images_after_redeploy"`
	AdditionalArgs           []string `json:"additional_args"`
}

// ApplicationAggregate 将应用、文件和最近快照组合为读取聚合。
type ApplicationAggregate struct {
	Application Application
	Files       []ApplicationFile
	Snapshot    *Snapshot
}

// ListQuery 描述应用列表筛选条件。
type ListQuery struct {
	Limit   int
	Offset  int
	Keyword string
	// Sort 只接受应用列表白名单排序表达式；空值使用 ApplicationListSortCreatedAtDesc。
	Sort            string
	RuntimeTargetID *int64
	SourceType      string
	DriftStatus     string
}

// ListResult 返回一页应用分页结果。
type ListResult struct {
	Items []ApplicationAggregate
	Total int
}

// ImportApplicationInput 描述创建或替换一条应用注册记录的输入。
type ImportApplicationInput struct {
	ApplicationID            string
	DeploymentAdapterKind    string
	ApplicationName          *string
	WorkspacePath            string
	ComposeProjectName       string
	ComposeProjectNameSource string
	StrictCreate             bool
	RuntimeTargetID          uint64
	DisplayName              string
	SourceType               string
	OwnershipMode            string
	SourceMetadata           map[string]string
	LifecycleStrategyKind    string
	LifecycleReviewStatus    string
	LifecycleConfig          LifecycleConfig
	LastObservedConfigHash   string
	LastDriftCheckedAt       *time.Time
	DriftStatus              string
	Files                    []ApplicationFile
	Snapshot                 *Snapshot
	ActorID                  *uint64
}

// RefreshApplicationInput 描述更新现有应用刷新状态的输入。
type RefreshApplicationInput struct {
	ApplicationRecordID    uint64
	LastObservedConfigHash string
	LastDriftCheckedAt     *time.Time
	DriftStatus            string
	Files                  []ApplicationFile
	Snapshot               *Snapshot
	ActorID                *uint64
}

// UpdateLifecycleConfigInput 描述更新单个应用已保存生命周期执行语义的输入。
type UpdateLifecycleConfigInput struct {
	ApplicationRecordID   uint64
	LifecycleStrategyKind string
	LifecycleReviewStatus string
	LifecycleConfig       LifecycleConfig
	ActorID               *uint64
}

// UpdateWorkspaceAnnotationInput 描述更新或删除一个应用工作区注释的输入。
type UpdateWorkspaceAnnotationInput struct {
	ApplicationRecordID uint64
	RelativePath        string
	Annotation          *string
	ActorID             *uint64
}

// UnregisterApplicationInput 描述软删除应用注册记录且不触碰工作区文件的输入。
type UnregisterApplicationInput struct {
	ApplicationRecordID uint64
	ActorID             *uint64
}

// Repository 持久化应用注册表、文件清单和快照。
type Repository interface {
	List(ctx context.Context, query ListQuery) (ListResult, error)
	Get(ctx context.Context, applicationRecordID uint64) (ApplicationAggregate, error)
	GetFile(ctx context.Context, applicationRecordID uint64, fileID uint64) (ApplicationFile, error)
	ImportApplication(ctx context.Context, input ImportApplicationInput) (ApplicationAggregate, error)
	RefreshApplication(ctx context.Context, input RefreshApplicationInput) (ApplicationAggregate, error)
	UpdateLifecycleConfig(ctx context.Context, input UpdateLifecycleConfigInput) (ApplicationAggregate, error)
	UpdateWorkspaceAnnotation(ctx context.Context, input UpdateWorkspaceAnnotationInput) (ApplicationAggregate, error)
	UnregisterApplication(ctx context.Context, input UnregisterApplicationInput) error
	BackfillRuntimeTarget(ctx context.Context, runtimeTargetID uint64) error
}

// ApplicationLookupRepository 将公开 Application ID 解析为模块私有的数字注册键。
// 接口保持窄边界，避免现有测试存储和仓储适配器被迫承担额外的公开查询契约。
type ApplicationLookupRepository interface {
	GetByApplicationID(ctx context.Context, applicationID string) (ApplicationAggregate, error)
}

// ApplicationNameLookupRepository 将受管应用名称解析为存活应用，仅由受管创建预检边界使用。
type ApplicationNameLookupRepository interface {
	GetByApplicationName(ctx context.Context, applicationName string) (ApplicationAggregate, error)
}

// ApplicationIDBatchLookupRepository 批量解析公开标识，不加载完整应用聚合。
type ApplicationIDBatchLookupRepository interface {
	GetRecordIDsByApplicationIDs(ctx context.Context, applicationIDs []string) (map[string]uint64, error)
}

// ComposeContext 是容器运行时到 Application 注册表的规范关联键。
type ComposeContext struct {
	RuntimeTargetID    int64
	ComposeProjectName string
}

// ComposeApplicationReference 是可安全暴露给关联消费者的应用标识与显示名。
type ComposeApplicationReference struct {
	ComposeContext
	ApplicationID string
	DisplayName   string
}

// ComposeContextReferenceRepository 以规范的运行目标和 Compose 项目名批量解析存活 Application 引用。
type ComposeContextReferenceRepository interface {
	ResolveComposeContexts(ctx context.Context, contexts []ComposeContext) ([]ComposeApplicationReference, error)
}
