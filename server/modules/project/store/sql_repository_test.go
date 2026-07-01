package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	projectcontract "graft/server/modules/project/contract"
)

func TestSQLRepositoryGetFileSkipsDeletedProject(t *testing.T) {
	t.Parallel()

	repo, db := newTestSQLRepository(t)
	ctx := context.Background()

	mustExec(t, db, `INSERT INTO compose_projects (
		id, display_name, canonical_project_name, canonical_project_name_source, source_kind, host_scope,
		working_directory, ownership_mode, last_refresh_status, drift_status, created_at, updated_at, deleted_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		1, "demo", "demo", projectcontract.CanonicalProjectNameSourceComputed.String(), projectcontract.SourceKindImported.String(),
		projectcontract.HostScopeLocal.String(), "/srv/demo", projectcontract.OwnershipModeExternal.String(),
		projectcontract.RefreshStatusSuccess.String(), projectcontract.DriftStatusClean.String(), time.Now().UTC(), time.Now().UTC(), 1,
	)
	mustExec(t, db, `INSERT INTO compose_project_files (
		id, project_id, kind, role, absolute_path, display_path, order_index, exists_on_last_refresh, last_observed_hash, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		10, 1, projectcontract.FileKindCompose.String(), projectcontract.FileRolePrimary.String(), "/srv/demo/compose.yml", "compose.yml", 0, 1, "hash", time.Now().UTC(), time.Now().UTC(),
	)

	_, err := repo.GetFile(ctx, 1, 10)
	if !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("expected ErrFileNotFound, got %v", err)
	}
}

func TestSQLRepositoryListAggregatesFilesAndSnapshots(t *testing.T) {
	t.Parallel()

	repo, db := newTestSQLRepository(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	insertProjectRow(t, db, 1, "alpha", now, 0)
	insertProjectRow(t, db, 2, "beta", now.Add(time.Second), 0)
	insertProjectFileRow(t, db, 11, 1, "compose.yml", 0)
	insertProjectFileRow(t, db, 12, 1, ".env", 1)
	insertProjectFileRow(t, db, 21, 2, "compose.yml", 0)
	mustExec(t, db, `INSERT INTO compose_project_snapshots (
		project_id, normalized_compose_json, config_hash, declared_service_count, declared_services_digest, refreshed_at
	) VALUES (?, ?, ?, ?, ?, ?)`,
		2, []byte(`{"services":{"web":{}}}`), "cfg-beta", 1, "digest-beta", now,
	)

	result, err := repo.List(ctx, ListQuery{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].Project.ID != 2 || len(result.Items[0].Files) != 1 || result.Items[0].Snapshot == nil {
		t.Fatalf("unexpected first aggregate: %#v", result.Items[0])
	}
	if result.Items[1].Project.ID != 1 || len(result.Items[1].Files) != 2 || result.Items[1].Snapshot != nil {
		t.Fatalf("unexpected second aggregate: %#v", result.Items[1])
	}
}

func TestNormalizeListQueryRejectsInvalidTypedContract(t *testing.T) {
	t.Parallel()

	_, err := normalizeListQuery(ListQuery{SourceKind: "bogus"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestValidateImportInputRejectsInvalidTypedContract(t *testing.T) {
	t.Parallel()

	_, err := validateImportInput(ImportProjectInput{
		DisplayName:                "demo",
		CanonicalProjectName:       "demo",
		CanonicalProjectNameSource: "bogus",
		SourceKind:                 projectcontract.SourceKindImported.String(),
		HostScope:                  projectcontract.HostScopeLocal.String(),
		WorkingDirectory:           "/srv/demo",
		OwnershipMode:              projectcontract.OwnershipModeExternal.String(),
		LastRefreshStatus:          projectcontract.RefreshStatusSuccess.String(),
		DriftStatus:                projectcontract.DriftStatusClean.String(),
		Files: []ProjectFile{
			{
				Kind:         projectcontract.FileKindCompose.String(),
				Role:         projectcontract.FileRolePrimary.String(),
				AbsolutePath: "/srv/demo/compose.yml",
				DisplayPath:  "compose.yml",
			},
		},
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func newTestSQLRepository(t *testing.T) (*SQLRepository, *sql.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:project-store-%s?mode=memory&cache=private", t.Name())
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	createProjectStoreSchema(t, db)

	repo, err := NewSQLRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	return repo, db
}

func createProjectStoreSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	mustExec(t, db, `CREATE TABLE compose_projects (
		id INTEGER PRIMARY KEY,
		display_name TEXT NOT NULL,
		canonical_project_name TEXT NOT NULL,
		canonical_project_name_source TEXT NOT NULL,
		source_kind TEXT NOT NULL,
		host_scope TEXT NOT NULL,
		working_directory TEXT NOT NULL,
		ownership_mode TEXT NOT NULL,
		last_refresh_status TEXT NOT NULL,
		last_refresh_at TIMESTAMP NULL,
		last_refresh_error_code TEXT NOT NULL DEFAULT '',
		last_refresh_error_message TEXT NOT NULL DEFAULT '',
		last_refresh_config_hash TEXT NOT NULL DEFAULT '',
		last_observed_config_hash TEXT NOT NULL DEFAULT '',
		last_drift_checked_at TIMESTAMP NULL,
		drift_status TEXT NOT NULL,
		created_by INTEGER NULL,
		updated_by INTEGER NULL,
		deleted_by INTEGER NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		deleted_at INTEGER NOT NULL DEFAULT 0
	)`)
	mustExec(t, db, `CREATE TABLE compose_project_files (
		id INTEGER PRIMARY KEY,
		project_id INTEGER NOT NULL,
		kind TEXT NOT NULL,
		role TEXT NOT NULL,
		absolute_path TEXT NOT NULL,
		display_path TEXT NOT NULL,
		order_index INTEGER NOT NULL,
		exists_on_last_refresh BOOLEAN NOT NULL,
		last_observed_hash TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`)
	mustExec(t, db, `CREATE TABLE compose_project_snapshots (
		project_id INTEGER PRIMARY KEY,
		normalized_compose_json BLOB NOT NULL,
		config_hash TEXT NOT NULL,
		declared_service_count INTEGER NOT NULL,
		declared_services_digest TEXT NOT NULL,
		refreshed_at TIMESTAMP NOT NULL
	)`)
}

func insertProjectRow(t *testing.T, db *sql.DB, id int, name string, updatedAt time.Time, deletedAt int64) {
	t.Helper()
	mustExec(t, db, `INSERT INTO compose_projects (
		id, display_name, canonical_project_name, canonical_project_name_source, source_kind, host_scope,
		working_directory, ownership_mode, last_refresh_status, drift_status, created_at, updated_at, deleted_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, name, name, projectcontract.CanonicalProjectNameSourceComputed.String(), projectcontract.SourceKindImported.String(),
		projectcontract.HostScopeLocal.String(), "/srv/"+name, projectcontract.OwnershipModeExternal.String(),
		projectcontract.RefreshStatusSuccess.String(), projectcontract.DriftStatusClean.String(), updatedAt, updatedAt, deletedAt,
	)
}

func insertProjectFileRow(t *testing.T, db *sql.DB, id int, projectID int, displayPath string, orderIndex int) {
	t.Helper()
	mustExec(t, db, `INSERT INTO compose_project_files (
		id, project_id, kind, role, absolute_path, display_path, order_index, exists_on_last_refresh, last_observed_hash, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, projectID, projectcontract.FileKindCompose.String(), projectcontract.FileRolePrimary.String(),
		"/srv/project/"+displayPath, displayPath, orderIndex, 1, "hash-"+displayPath, time.Now().UTC(), time.Now().UTC(),
	)
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q failed: %v", query, err)
	}
}
