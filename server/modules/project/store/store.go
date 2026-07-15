// Package store defines Compose Project Management module persistence contracts.
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

// Project stores one Compose project registry record.
type Project struct {
	ID                         uint64
	ApplicationID              string
	ApplicationName               *string
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

// ProjectFile stores one ordered project file reference.
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

// Snapshot stores the latest successful normalized compose snapshot.
type Snapshot struct {
	ProjectID              uint64
	NormalizedComposeJSON  []byte
	ConfigHash             string
	DeclaredServiceCount   int
	DeclaredServicesDigest string
	RefreshedAt            time.Time
}

// LifecycleConfig stores project-owned lifecycle execution semantics.
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

// ProjectAggregate joins one project with its files and latest snapshot.
type ProjectAggregate struct {
	Project  Project
	Files    []ProjectFile
	Snapshot *Snapshot
}

// ListQuery describes project list filters.
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

// ListResult returns a paginated project page.
type ListResult struct {
	Items []ProjectAggregate
	Total int
}

// ImportProjectInput creates or replaces one project registry entry.
type ImportProjectInput struct {
	ApplicationID              string
	ApplicationName               *string
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

// RefreshProjectInput updates one existing project refresh state.
type RefreshProjectInput struct {
	ProjectID              uint64
	LastObservedConfigHash string
	LastDriftCheckedAt     *time.Time
	DriftStatus            string
	Files                  []ProjectFile
	Snapshot               *Snapshot
	ActorID                *uint64
}

// UpdateLifecycleConfigInput updates the saved lifecycle execution semantics for one project.
type UpdateLifecycleConfigInput struct {
	ProjectID             uint64
	LifecycleStrategyKind string
	LifecycleReviewStatus string
	LifecycleConfig       LifecycleConfig
	ActorID               *uint64
}

// UpdateWorkspaceAnnotationInput updates or removes one project workspace annotation.
type UpdateWorkspaceAnnotationInput struct {
	ProjectID    uint64
	RelativePath string
	Annotation   *string
	ActorID      *uint64
}

// UnregisterProjectInput soft-deletes one existing project registry row without touching host files.
type UnregisterProjectInput struct {
	ProjectID uint64
	ActorID   *uint64
}

// Repository persists project registry, file inventory, and snapshots.
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

// ApplicationLookupRepository resolves the external Application ID to the
// module-private numeric registry key. It is deliberately narrow so existing
// test stores and repository adapters do not gain an accidental public lookup
// obligation.
type ApplicationLookupRepository interface {
	GetByApplicationID(ctx context.Context, applicationID string) (ProjectAggregate, error)
}

// ApplicationIDBatchLookupRepository resolves public IDs without aggregate loads.
type ApplicationIDBatchLookupRepository interface {
	GetIDsByApplicationIDs(ctx context.Context, applicationIDs []string) (map[string]uint64, error)
}
