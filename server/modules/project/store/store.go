// Package store 定义 Compose 项目管理模块的持久化契约。
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

	// ProjectListSortCreatedAtDesc orders projects from newest to oldest.
	ProjectListSortCreatedAtDesc = "created_at:desc"
	// ProjectListSortCreatedAtAsc orders projects from oldest to newest.
	ProjectListSortCreatedAtAsc = "created_at:asc"
)

var (
	// ErrInvalidInput indicates the repository input violates the module persistence contract.
	ErrInvalidInput = errors.New("project invalid input")
	// ErrProjectNotFound indicates no live project matches the requested id.
	ErrProjectNotFound = errors.New("project not found")
	// ErrProjectConflict indicates the requested import or update conflicts with a live record.
	ErrProjectConflict = errors.New("project conflict")
	// ErrFileNotFound indicates the requested project file record does not exist.
	ErrFileNotFound = errors.New("project file not found")
)

// Project 保存一条 Compose 项目注册记录。
type Project struct {
	ID                         uint64
	ApplicationID              string
	ApplicationName            *string
	WorkspacePath              string
	ComposeProjectName         string
	ComposeProjectNameSource   string
	RuntimeTargetID            *uint64
	DisplayName                string
	CanonicalProjectName       string
	CanonicalProjectNameSource string
	SourceKind                 string
	HostScope                  string
	WorkingDirectory           string
	OwnershipMode              string
	SourceMetadata             map[string]string
	LifecycleStrategyKind      string
	LifecycleReviewStatus      string
	LifecycleConfig            LifecycleConfig
	LastObservedConfigHash     string
	WorkspaceAnnotations       map[string]string
	LastDriftCheckedAt         *time.Time
	DriftStatus                string
	CreatedBy                  *uint64
	UpdatedBy                  *uint64
	DeletedBy                  *uint64
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
	DeletedAt                  int64
}

// ProjectFile 保存一个有序项目文件引用。
type ProjectFile struct {
	ID               uint64
	ProjectID        uint64
	Kind             string
	Role             string
	AbsolutePath     string
	DisplayPath      string
	OrderIndex       int
	LastObservedHash string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// Snapshot 保存最近一次成功规范化的 Compose 快照。
type Snapshot struct {
	ProjectID              uint64
	NormalizedComposeJSON  []byte
	ConfigHash             string
	DeclaredServiceCount   int
	DeclaredServicesDigest string
	RefreshedAt            time.Time
}

// LifecycleConfig 保存项目拥有的生命周期执行语义。
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

// ProjectAggregate 将项目、其文件和最近快照组合为一个读取聚合。
type ProjectAggregate struct {
	Project  Project
	Files    []ProjectFile
	Snapshot *Snapshot
}

// ListQuery 描述项目列表筛选条件。
type ListQuery struct {
	Limit   int
	Offset  int
	Keyword string
	// Sort accepts the restricted project-list sort expressions; an empty value uses ProjectListSortCreatedAtDesc.
	Sort            string
	RuntimeTargetID *int64
	SourceKind      string
	DriftStatus     string
}

// ListResult 返回一页分页项目结果。
type ListResult struct {
	Items []ProjectAggregate
	Total int
}

// ImportProjectInput 描述创建或替换一条项目注册记录的输入。
type ImportProjectInput struct {
	ApplicationID              string
	ApplicationName            *string
	WorkspacePath              string
	ComposeProjectName         string
	ComposeProjectNameSource   string
	StrictCreate               bool
	RuntimeTargetID            uint64
	DisplayName                string
	CanonicalProjectName       string
	CanonicalProjectNameSource string
	SourceKind                 string
	HostScope                  string
	WorkingDirectory           string
	OwnershipMode              string
	SourceMetadata             map[string]string
	LifecycleStrategyKind      string
	LifecycleReviewStatus      string
	LifecycleConfig            LifecycleConfig
	LastObservedConfigHash     string
	LastDriftCheckedAt         *time.Time
	DriftStatus                string
	Files                      []ProjectFile
	Snapshot                   *Snapshot
	ActorID                    *uint64
}

// RefreshProjectInput 描述更新现有项目刷新状态的输入。
type RefreshProjectInput struct {
	ProjectID              uint64
	LastObservedConfigHash string
	LastDriftCheckedAt     *time.Time
	DriftStatus            string
	Files                  []ProjectFile
	Snapshot               *Snapshot
	ActorID                *uint64
}

// UpdateLifecycleConfigInput 描述更新单个项目已保存生命周期执行语义的输入。
type UpdateLifecycleConfigInput struct {
	ProjectID             uint64
	LifecycleStrategyKind string
	LifecycleReviewStatus string
	LifecycleConfig       LifecycleConfig
	ActorID               *uint64
}

// UpdateWorkspaceAnnotationInput 描述更新或删除一个项目工作区注释的输入。
type UpdateWorkspaceAnnotationInput struct {
	ProjectID    uint64
	RelativePath string
	Annotation   *string
	ActorID      *uint64
}

// UnregisterProjectInput 描述软删除一条现有项目注册记录且不触碰宿主机文件的输入。
type UnregisterProjectInput struct {
	ProjectID uint64
	ActorID   *uint64
}

// Repository 持久化项目注册表、文件清单和快照。
type Repository interface {
	List(ctx context.Context, query ListQuery) (ListResult, error)
	Get(ctx context.Context, projectID uint64) (ProjectAggregate, error)
	GetFile(ctx context.Context, projectID uint64, fileID uint64) (ProjectFile, error)
	ImportProject(ctx context.Context, input ImportProjectInput) (ProjectAggregate, error)
	RefreshProject(ctx context.Context, input RefreshProjectInput) (ProjectAggregate, error)
	UpdateLifecycleConfig(ctx context.Context, input UpdateLifecycleConfigInput) (ProjectAggregate, error)
	UpdateWorkspaceAnnotation(ctx context.Context, input UpdateWorkspaceAnnotationInput) (ProjectAggregate, error)
	UnregisterProject(ctx context.Context, input UnregisterProjectInput) error
	BackfillRuntimeTarget(ctx context.Context, runtimeTargetID uint64) error
}

// ApplicationLookupRepository 将公开 Application ID 解析为模块私有的数字注册键。
// 接口保持窄边界，避免现有测试存储和仓储适配器被迫承担额外的公开查询契约。
type ApplicationLookupRepository interface {
	GetByApplicationID(ctx context.Context, applicationID string) (ProjectAggregate, error)
}

// ApplicationNameLookupRepository 将受管应用名称解析为存活项目，仅由受管创建预检边界使用。
type ApplicationNameLookupRepository interface {
	GetByApplicationName(ctx context.Context, applicationName string) (ProjectAggregate, error)
}

// ApplicationIDBatchLookupRepository 批量解析公开标识，不加载完整项目聚合。
type ApplicationIDBatchLookupRepository interface {
	GetIDsByApplicationIDs(ctx context.Context, applicationIDs []string) (map[string]uint64, error)
}
