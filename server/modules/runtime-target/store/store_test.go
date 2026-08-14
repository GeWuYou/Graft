package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/mattn/go-sqlite3"
)

func TestSQLRepositoryListReturnsEveryLiveTargetWhileListPageRemainsBounded(t *testing.T) {
	db := openRuntimeTargetTestDB(t)
	for index := range 101 {
		if _, err := db.Exec(`INSERT INTO runtime_targets (provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, 0)`, "docker", fmt.Sprintf("Target %03d", index), "unix:///var/run/docker.sock", "unix_socket", `["containers"]`, true, ""); err != nil {
			t.Fatalf("insert runtime target %d: %v", index, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO runtime_targets (provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, checked_at, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, NULL, 1)`, "docker", "Deleted target", "unix:///var/run/docker.sock", "unix_socket", `["containers"]`, false, ""); err != nil {
		t.Fatalf("insert deleted runtime target: %v", err)
	}
	if _, err := db.Exec(`UPDATE runtime_targets SET availability = false WHERE id = 1`); err != nil {
		t.Fatalf("mark runtime target unavailable: %v", err)
	}

	repository := NewSQLRepository(db)
	items, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list runtime targets: %v", err)
	}
	if len(items) != 101 {
		t.Fatalf("List returned %d targets, want all 101 live targets", len(items))
	}

	page, err := repository.ListPage(context.Background(), 10, 90)
	if err != nil {
		t.Fatalf("list runtime-target page: %v", err)
	}
	if page.Total != 101 {
		t.Fatalf("ListPage total = %d, want 101", page.Total)
	}
	if len(page.Items) != 10 {
		t.Fatalf("ListPage returned %d targets, want 10", len(page.Items))
	}
	if page.Summary != (Summary{Total: 101, Healthy: 100, Unavailable: 1}) {
		t.Fatalf("ListPage summary = %+v, want total=101 healthy=100 unavailable=1", page.Summary)
	}
}

func TestSQLRepositoryRunInTransactionReusesContextTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("open sql mock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository := NewSQLRepository(db)

	mock.ExpectBegin()
	mock.ExpectCommit()
	var outerTx *sql.Tx
	err = repository.RunInTransaction(context.Background(), func(ctx context.Context, tx *sql.Tx) error {
		outerTx = tx
		return repository.RunInTransaction(ctx, func(nestedCtx context.Context, nestedTx *sql.Tx) error {
			if nestedCtx != ctx {
				t.Fatal("nested transaction callback received a different context")
			}
			if nestedTx != outerTx {
				t.Fatal("nested transaction callback received a different transaction")
			}
			return nil
		})
	})
	if err != nil {
		t.Fatalf("run nested transaction: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("nested transaction should not begin or commit independently: %v", err)
	}
}

func TestSQLRepositoryUserAssignmentsRestoreAndRestrictDeploymentCandidates(t *testing.T) {
	db := openRuntimeTargetTestDB(t)
	if _, err := db.Exec(`CREATE TABLE runtime_target_user_assignments (runtime_target_id INTEGER NOT NULL, user_id INTEGER NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, created_by INTEGER NOT NULL, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_by INTEGER NOT NULL, deleted_at INTEGER NOT NULL DEFAULT 0, deleted_by INTEGER NOT NULL DEFAULT 0, UNIQUE(runtime_target_id, user_id))`); err != nil {
		t.Fatalf("create assignment table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE runtime_target_assignment_revisions (runtime_target_id INTEGER PRIMARY KEY, revision INTEGER NOT NULL DEFAULT 1, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("create assignment revision table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO runtime_targets (id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, deleted_at) VALUES (7, 'docker', 'Assigned', 'unix:///var/run/docker.sock', 'unix_socket', '["compose_execution","workspace_access"]', true, '', 0), (8, 'docker', 'Other', 'unix:///var/run/docker.sock', 'unix_socket', '["compose_execution","workspace_access"]', true, '', 0)`); err != nil {
		t.Fatalf("seed runtime targets: %v", err)
	}
	repository := NewSQLRepository(db)
	assignment, err := repository.GrantUserAssignment(context.Background(), 7, 11, 3)
	if err != nil || assignment.TargetID != 7 || assignment.UserID != 11 || assignment.CreatedBy != 3 {
		t.Fatalf("grant assignment = %#v, %v", assignment, err)
	}
	allowed, err := repository.HasActiveUserAssignment(context.Background(), 7, 11)
	if err != nil || !allowed {
		t.Fatalf("assigned target allowed = %v, %v", allowed, err)
	}
	allowed, err = repository.HasActiveUserAssignment(context.Background(), 8, 11)
	if err != nil || allowed {
		t.Fatalf("unassigned target allowed = %v, %v", allowed, err)
	}
	candidates, err := repository.ListAssignedComposeTargets(context.Background(), 11)
	if err != nil || len(candidates) != 1 || candidates[0].ID != 7 {
		t.Fatalf("assigned candidates = %#v, %v", candidates, err)
	}
}

func TestSQLRepositoryReplaceUserAssignmentsUsesRevisionCAS(t *testing.T) {
	db := openRuntimeTargetTestDB(t)
	if _, err := db.Exec(`CREATE TABLE runtime_target_user_assignments (runtime_target_id INTEGER NOT NULL, user_id INTEGER NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, created_by INTEGER NOT NULL, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_by INTEGER NOT NULL, deleted_at INTEGER NOT NULL DEFAULT 0, deleted_by INTEGER NOT NULL DEFAULT 0, UNIQUE(runtime_target_id, user_id))`); err != nil {
		t.Fatalf("create assignment table: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE runtime_target_assignment_revisions (runtime_target_id INTEGER PRIMARY KEY, revision INTEGER NOT NULL DEFAULT 1, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("create assignment revision table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO runtime_targets (id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, deleted_at) VALUES (7, 'docker', 'Assigned', 'unix:///var/run/docker.sock', 'unix_socket', '[]', true, '', 0)`); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	repository := NewSQLRepository(db)
	items, revision, err := repository.ReplaceUserAssignmentsTx(context.Background(), 7, []uint64{11}, 1, 3)
	if err != nil || revision != 2 || len(items) != 1 || items[0].UserID != 11 {
		t.Fatalf("first replacement = %#v revision=%d err=%v", items, revision, err)
	}
	if _, _, err := repository.ReplaceUserAssignmentsTx(context.Background(), 7, []uint64{12}, 1, 3); !errors.Is(err, ErrAssignmentRevisionConflict) {
		t.Fatalf("stale replacement error = %v, want revision conflict", err)
	}
}

func openRuntimeTargetTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+strings.ReplaceAll(t.Name(), "/", "-")+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open runtime-target test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE runtime_targets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		provider TEXT NOT NULL,
		display_name TEXT NOT NULL,
		endpoint_label TEXT NOT NULL,
		connection_kind TEXT NOT NULL,
		capabilities_json BLOB NOT NULL,
		availability BOOLEAN NOT NULL,
		last_error TEXT NOT NULL,
		checked_at DATETIME NULL,
		deleted_at INTEGER NOT NULL
	)`); err != nil {
		t.Fatalf("create runtime-target test table: %v", err)
	}
	return db
}
