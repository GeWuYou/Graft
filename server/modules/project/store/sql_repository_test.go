package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	projectcontract "graft/server/modules/project/contract"
)

const completeLifecycleConfigJSON = `{"profiles":[],"down_before_redeploy":true,"pull_before_redeploy":false,"build_before_up":false,"force_recreate":false,"remove_orphans":true,"wait_after_up":false,"wait_timeout_seconds":120,"renew_anon_volumes":false,"prune_images_after_redeploy":false}`

func TestSQLRepositoryGetFileSkipsDeletedProject(t *testing.T) {
	t.Parallel()

	repo, db := newTestSQLRepository(t)
	ctx := context.Background()

	mustExec(t, db, `INSERT INTO applications (
		application_record_id, display_name, compose_project_name, compose_project_name_source, source_type,
		workspace_path, ownership_mode, source_metadata_json, lifecycle_strategy_kind, lifecycle_review_status, lifecycle_config_json,
		last_observed_config_hash, workspace_annotations_json, drift_status, created_at, updated_at, deleted_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		1, "demo", "demo", projectcontract.ComposeProjectNameSourceComputed.String(), projectcontract.SourceTypeImported.String(),
		"/srv/demo", projectcontract.OwnershipModeExternal.String(), `{}`,
		projectcontract.LifecycleStrategyKindStandard.String(), projectcontract.LifecycleReviewStatusReviewRequired.String(), completeLifecycleConfigJSON,
		"hash-demo", `{}`, projectcontract.DriftStatusClean.String(), time.Now().UTC(), time.Now().UTC(), 1,
	)
	mustExec(t, db, `INSERT INTO application_files (
		id, application_record_id, kind, role, absolute_path, display_path, order_index, last_observed_hash, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		10, 1, projectcontract.FileKindCompose.String(), projectcontract.FileRolePrimary.String(), "/srv/demo/compose.yml", "compose.yml", 0, "hash", time.Now().UTC(), time.Now().UTC(),
	)

	_, err := repo.GetFile(ctx, 1, 10)
	if !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("expected ErrFileNotFound, got %v", err)
	}
}

func TestSQLRepositoryGetRecordIDsByApplicationIDsIgnoresBlankValues(t *testing.T) {
	t.Parallel()

	repo, db := newTestSQLRepository(t)
	now := time.Now().UTC()
	insertProjectRow(t, db, 1, "alpha", now, 0)
	mustExec(t, db, `UPDATE applications SET application_id = ? WHERE application_record_id = ?`, "app_01ARZ3NDEKTSV4RRFFQ69G5FAV", 1)

	result, err := repo.GetRecordIDsByApplicationIDs(context.Background(), []string{"  ", " app_01ARZ3NDEKTSV4RRFFQ69G5FAV ", ""})
	if err != nil {
		t.Fatalf("get application IDs: %v", err)
	}
	if len(result) != 1 || result["app_01ARZ3NDEKTSV4RRFFQ69G5FAV"] != 1 {
		t.Fatalf("unexpected resolved IDs: %#v", result)
	}
}

func TestSQLRepositoryListAggregatesFilesAndSnapshots(t *testing.T) {
	t.Parallel()

	repo, db := newTestSQLRepository(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	insertProjectRow(t, db, 1, "alpha", now, 0)
	insertProjectRow(t, db, 2, "beta", now.Add(time.Second), 0)
	insertApplicationFileRow(t, db, 11, 1, "compose.yml", 0)
	insertApplicationFileRow(t, db, 12, 1, ".env", 1)
	insertApplicationFileRow(t, db, 21, 2, "compose.yml", 0)
	mustExec(t, db, `INSERT INTO application_snapshots (
		application_record_id, normalized_compose_json, config_hash, declared_service_count, declared_services_digest, refreshed_at
	) VALUES (?, ?, ?, ?, ?, ?)`,
		2, []byte(`{"services":{"web":{}}}`), "cfg-beta", 1, "digest-beta", now,
	)

	result, err := repo.List(ctx, ListQuery{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("list applications: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if result.Items[0].Application.ApplicationRecordID != 2 || len(result.Items[0].Files) != 1 || result.Items[0].Snapshot == nil {
		t.Fatalf("unexpected first aggregate: %#v", result.Items[0])
	}
	if result.Items[1].Application.ApplicationRecordID != 1 || len(result.Items[1].Files) != 2 || result.Items[1].Snapshot != nil {
		t.Fatalf("unexpected second aggregate: %#v", result.Items[1])
	}
}

func TestNormalizeListQueryRejectsInvalidTypedContract(t *testing.T) {
	t.Parallel()

	_, err := normalizeListQuery(ListQuery{SourceType: "bogus"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestValidateImportInputRejectsInvalidTypedContract(t *testing.T) {
	t.Parallel()

	_, err := validateImportInput(ImportApplicationInput{
		DisplayName:              "demo",
		ComposeProjectName:       "demo",
		ComposeProjectNameSource: "bogus",
		SourceType:               projectcontract.SourceTypeImported.String(),
		WorkspacePath:            "/srv/demo",
		OwnershipMode:            projectcontract.OwnershipModeExternal.String(),
		DriftStatus:              projectcontract.DriftStatusClean.String(),
		Files: []ApplicationFile{
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

func TestSQLRepositoryUpdateWorkspaceAnnotationRejectsOversizedAnnotation(t *testing.T) {
	t.Parallel()

	repo, db := newTestSQLRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertProjectRow(t, db, 1, "demo", now, 0)

	annotation := strings.Repeat("a", projectcontract.ApplicationWorkspaceAnnotationMaxLength+1)
	_, err := repo.UpdateWorkspaceAnnotation(ctx, UpdateWorkspaceAnnotationInput{
		ApplicationRecordID: 1,
		RelativePath:        "compose.yml",
		Annotation:          &annotation,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSQLRepositoryUpdateWorkspaceAnnotationPreservesConcurrentChanges(t *testing.T) {
	t.Parallel()

	repo, db := newTestSQLRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	insertProjectRow(t, db, 1, "demo", now, 0)

	note := "compose note"
	aggregate, err := repo.UpdateWorkspaceAnnotation(ctx, UpdateWorkspaceAnnotationInput{
		ApplicationRecordID: 1,
		RelativePath:        "compose.yml",
		Annotation:          &note,
	})
	if err != nil {
		t.Fatalf("first annotation update: %v", err)
	}
	if aggregate.Application.WorkspaceAnnotations["compose.yml"] != note {
		t.Fatalf("expected compose annotation, got %#v", aggregate.Application.WorkspaceAnnotations)
	}

	mustExec(t, db, `UPDATE applications SET workspace_annotations_json = ? WHERE application_record_id = ?`, `{"README.md":"doc note"}`, 1)

	envNote := "env note"
	aggregate, err = repo.UpdateWorkspaceAnnotation(ctx, UpdateWorkspaceAnnotationInput{
		ApplicationRecordID: 1,
		RelativePath:        ".env",
		Annotation:          &envNote,
	})
	if err != nil {
		t.Fatalf("second annotation update: %v", err)
	}
	if aggregate.Application.WorkspaceAnnotations["README.md"] != "doc note" {
		t.Fatalf("expected concurrent README annotation to be preserved, got %#v", aggregate.Application.WorkspaceAnnotations)
	}
	if aggregate.Application.WorkspaceAnnotations[".env"] != envNote {
		t.Fatalf("expected env annotation to be added, got %#v", aggregate.Application.WorkspaceAnnotations)
	}
}

func TestComposeProjectsUpsertSQLPreservesApplicationNameWhenExcludedValueIsNull(t *testing.T) {
	t.Parallel()

	if !strings.Contains(applicationsUpsertSQL(), "application_name = COALESCE(excluded.application_name, applications.application_name)") {
		t.Fatal("expected upsert SQL to preserve an existing application name when the import omits it")
	}
}

func newTestSQLRepository(t *testing.T) (*SQLRepository, *sql.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:application-store-%s?mode=memory&cache=private", t.Name())
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
	createApplicationStoreSchema(t, db)
	createApplicationTemplateStoreSchema(t, db)

	repo, err := NewSQLRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	return repo, db
}

func createApplicationTemplateStoreSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	mustExec(t, db, `CREATE TABLE application_templates (
		template_id TEXT PRIMARY KEY, display_name TEXT NOT NULL UNIQUE, description TEXT NOT NULL DEFAULT '', deployment_adapter_kind TEXT NOT NULL,
		archived_at TIMESTAMP NULL, created_by INTEGER NULL, updated_by INTEGER NULL, deleted_by INTEGER NULL,
		created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL, deleted_at INTEGER NOT NULL DEFAULT 0
	)`)
	mustExec(t, db, `CREATE TABLE application_template_versions (
		template_version_id TEXT PRIMARY KEY, template_id TEXT NOT NULL, version_number INTEGER NOT NULL, status TEXT NOT NULL,
		definition_schema_version INTEGER NOT NULL, definition_json BLOB NOT NULL, published_at TIMESTAMP NULL, published_by INTEGER NULL, withdrawn_at TIMESTAMP NULL, withdrawn_by INTEGER NULL,
		created_by INTEGER NULL, updated_by INTEGER NULL, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL, deleted_at INTEGER NOT NULL DEFAULT 0,
		UNIQUE(template_id, version_number), UNIQUE(template_id, status)
	)`)
}

func createApplicationStoreSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	mustExec(t, db, `CREATE TABLE applications (
		application_record_id INTEGER PRIMARY KEY,
		application_id TEXT NOT NULL DEFAULT 'app_00000000000000000000000000',
		deployment_adapter_kind TEXT NOT NULL DEFAULT 'compose',
		application_name TEXT NULL,
		workspace_path TEXT NOT NULL DEFAULT '',
		compose_project_name TEXT NOT NULL DEFAULT '',
		compose_project_name_source TEXT NOT NULL DEFAULT 'computed',
		runtime_target_id INTEGER NULL,
		display_name TEXT NOT NULL,
		source_type TEXT NOT NULL,
		ownership_mode TEXT NOT NULL,
		source_metadata_json TEXT NOT NULL DEFAULT '{}',
		lifecycle_strategy_kind TEXT NOT NULL,
		lifecycle_review_status TEXT NOT NULL,
		lifecycle_config_json TEXT NOT NULL DEFAULT '{}',
		last_observed_config_hash TEXT NOT NULL DEFAULT '',
		workspace_annotations_json TEXT NOT NULL DEFAULT '{}',
		last_drift_checked_at TIMESTAMP NULL,
		drift_status TEXT NOT NULL,
		created_by INTEGER NULL,
		updated_by INTEGER NULL,
		deleted_by INTEGER NULL,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		deleted_at INTEGER NOT NULL DEFAULT 0
	)`)
	mustExec(t, db, `CREATE TABLE application_files (
		id INTEGER PRIMARY KEY,
		application_record_id INTEGER NOT NULL,
		kind TEXT NOT NULL,
		role TEXT NOT NULL,
		absolute_path TEXT NOT NULL,
		display_path TEXT NOT NULL,
		order_index INTEGER NOT NULL,
		last_observed_hash TEXT NOT NULL DEFAULT '',
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL
	)`)
	mustExec(t, db, `CREATE TABLE application_snapshots (
		application_record_id INTEGER PRIMARY KEY,
		normalized_compose_json BLOB NOT NULL,
		config_hash TEXT NOT NULL,
		declared_service_count INTEGER NOT NULL,
		declared_services_digest TEXT NOT NULL,
		refreshed_at TIMESTAMP NOT NULL
	)`)
}

func insertProjectRow(t *testing.T, db *sql.DB, id int, name string, updatedAt time.Time, deletedAt int64) {
	t.Helper()
	mustExec(t, db, `INSERT INTO applications (
		application_record_id, display_name, compose_project_name, compose_project_name_source, source_type,
		workspace_path, ownership_mode, source_metadata_json, lifecycle_strategy_kind, lifecycle_review_status, lifecycle_config_json,
		last_observed_config_hash, workspace_annotations_json, drift_status, created_at, updated_at, deleted_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, name, name, projectcontract.ComposeProjectNameSourceComputed.String(), projectcontract.SourceTypeImported.String(),
		"/srv/"+name, projectcontract.OwnershipModeExternal.String(), `{}`,
		projectcontract.LifecycleStrategyKindStandard.String(), projectcontract.LifecycleReviewStatusReviewRequired.String(), completeLifecycleConfigJSON,
		"hash-"+name, `{}`, projectcontract.DriftStatusClean.String(), updatedAt, updatedAt, deletedAt,
	)
}

func insertApplicationFileRow(t *testing.T, db *sql.DB, id int, applicationRecordID int, displayPath string, orderIndex int) {
	t.Helper()
	mustExec(t, db, `INSERT INTO application_files (
		id, application_record_id, kind, role, absolute_path, display_path, order_index, last_observed_hash, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, applicationRecordID, projectcontract.FileKindCompose.String(), projectcontract.FileRolePrimary.String(),
		"/srv/application/"+displayPath, displayPath, orderIndex, "hash-"+displayPath, time.Now().UTC(), time.Now().UTC(),
	)
}

func mustExec(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q failed: %v", query, err)
	}
}
