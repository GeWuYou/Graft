// Package project defines Compose Project Management authority-owned data-model artifacts.
package project

import "time"

// Record is the module-owned persistence model for one Compose project registry row.
//
// Phase 1 keeps runtime state out of this record; it only stores registry, ownership, file,
// snapshot, and drift metadata.
type Record struct {
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

// FileRecord stores one ordered Compose or environment file reference for a project.
type FileRecord struct {
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

// SnapshotRecord stores the latest successful normalized Compose snapshot for one project.
type SnapshotRecord struct {
	ProjectID              uint64
	NormalizedComposeJSON  []byte
	ConfigHash             string
	RefreshedAt            time.Time
	DeclaredServiceCount   int
	DeclaredServicesDigest string
}
