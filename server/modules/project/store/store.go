// Package store defines Compose Project Management module persistence contracts.
package store

import (
	"context"
	"errors"
	"time"
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
	DisplayName                string
	CanonicalProjectName       string
	CanonicalProjectNameSource string
	SourceKind                 string
	HostScope                  string
	WorkingDirectory           string
	OwnershipMode              string
	LastRefreshStatus          string
	LastRefreshAt              *time.Time
	LastRefreshErrorCode       string
	LastRefreshErrorMessage    string
	LastRefreshConfigHash      string
	LastObservedConfigHash     string
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
	ID                  uint64
	ProjectID           uint64
	Kind                string
	Role                string
	AbsolutePath        string
	DisplayPath         string
	OrderIndex          int
	ExistsOnLastRefresh bool
	LastObservedHash    string
	CreatedAt           time.Time
	UpdatedAt           time.Time
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

// ProjectAggregate joins one project with its files and latest snapshot.
type ProjectAggregate struct {
	Project  Project
	Files    []ProjectFile
	Snapshot *Snapshot
}

// ListQuery describes project list filters.
type ListQuery struct {
	Limit             int
	Offset            int
	SourceKind        string
	DriftStatus       string
	LastRefreshStatus string
}

// ListResult returns a paginated project page.
type ListResult struct {
	Items []ProjectAggregate
	Total int
}

// ImportProjectInput creates or replaces one project registry entry.
type ImportProjectInput struct {
	DisplayName                string
	CanonicalProjectName       string
	CanonicalProjectNameSource string
	SourceKind                 string
	HostScope                  string
	WorkingDirectory           string
	OwnershipMode              string
	LastRefreshStatus          string
	LastRefreshAt              *time.Time
	LastRefreshErrorCode       string
	LastRefreshErrorMessage    string
	LastRefreshConfigHash      string
	LastObservedConfigHash     string
	LastDriftCheckedAt         *time.Time
	DriftStatus                string
	Files                      []ProjectFile
	Snapshot                   *Snapshot
	ActorID                    *uint64
}

// RefreshProjectInput updates one existing project refresh state.
type RefreshProjectInput struct {
	ProjectID               uint64
	LastRefreshStatus       string
	LastRefreshAt           *time.Time
	LastRefreshErrorCode    string
	LastRefreshErrorMessage string
	LastRefreshConfigHash   string
	LastObservedConfigHash  string
	LastDriftCheckedAt      *time.Time
	DriftStatus             string
	Files                   []ProjectFile
	Snapshot                *Snapshot
	ActorID                 *uint64
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
	UnregisterProject(ctx context.Context, input UnregisterProjectInput) error
}
