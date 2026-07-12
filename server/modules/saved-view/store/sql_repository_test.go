package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/moduleapi"
)

func TestSQLRepositoryIsolatesOwnerAndSurfaceAndAllowsReuseAfterDelete(t *testing.T) {
	t.Parallel()
	repository, db := newTestRepository(t)
	ctx := context.Background()
	first := createInput(7, "project.list", "running")
	created, err := repository.Create(ctx, first)
	if err != nil {
		t.Fatalf("create first view: %v", err)
	}
	if _, err := repository.Create(ctx, first); !errors.Is(err, moduleapi.ErrSavedViewConflict) {
		t.Fatalf("expected duplicate conflict, got %v", err)
	}
	if _, err := repository.Create(ctx, createInput(8, "project.list", "running")); err != nil {
		t.Fatalf("create other owner view: %v", err)
	}
	if _, err := repository.Create(ctx, createInput(7, "audit.list", "running")); err != nil {
		t.Fatalf("create other surface view: %v", err)
	}
	items, err := repository.List(ctx, 7, "project.list")
	if err != nil || len(items) != 1 || items[0].ID != created.ID {
		t.Fatalf("owner/surface list mismatch: %#v, %v", items, err)
	}
	if err := repository.Delete(ctx, 8, "project.list", created.ID); !errors.Is(err, moduleapi.ErrSavedViewNotFound) {
		t.Fatalf("foreign delete must not find view: %v", err)
	}
	if err := repository.Delete(ctx, 7, "project.list", created.ID); err != nil {
		t.Fatalf("delete view: %v", err)
	}
	if _, err := repository.Create(ctx, first); err != nil {
		t.Fatalf("reuse deleted name: %v", err)
	}
	_ = db
}

func TestSQLRepositoryRejectsCurrentPageAndDuplicateColumnsOnlyThroughConsumerState(t *testing.T) {
	t.Parallel()
	repository, _ := newTestRepository(t)
	input := createInput(7, "project.list", "view")
	input.PageSize = 0
	if _, err := repository.Create(context.Background(), input); !errors.Is(err, moduleapi.ErrSavedViewInvalidInput) {
		t.Fatalf("expected invalid page size, got %v", err)
	}
	input = createInput(7, "project.list", "view")
	input.VisibleColumns = []string{"name", "name"}
	if _, err := repository.Create(context.Background(), input); !errors.Is(err, moduleapi.ErrSavedViewInvalidInput) {
		t.Fatalf("expected duplicate columns error, got %v", err)
	}
}

func createInput(owner uint64, surface, name string) moduleapi.SavedViewCreateInput {
	state, _ := json.Marshal(map[string]string{"source_kind": "managed"})
	return moduleapi.SavedViewCreateInput{OwnerUserID: owner, SurfaceKey: surface, Name: name, QueryState: state, PageSize: 20, VisibleColumns: []string{"name", "runtime_status"}}
}

func newTestRepository(t *testing.T) (*SQLRepository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE saved_views (
		id INTEGER PRIMARY KEY AUTOINCREMENT, owner_user_id INTEGER NOT NULL, surface_key TEXT NOT NULL, name TEXT NOT NULL,
		query_state_json BLOB NOT NULL, page_size INTEGER NOT NULL, visible_columns_json BLOB NOT NULL,
		created_at DATETIME NOT NULL, created_by INTEGER NOT NULL, updated_at DATETIME NOT NULL, updated_by INTEGER NOT NULL,
		deleted_at INTEGER NOT NULL DEFAULT 0, deleted_by INTEGER NOT NULL DEFAULT 0
	); CREATE UNIQUE INDEX uq_saved_views_owner_surface_name_live ON saved_views (owner_user_id, surface_key, name) WHERE deleted_at = 0;`); err != nil {
		t.Fatalf("create saved view test table: %v", err)
	}
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	return repository, db
}
