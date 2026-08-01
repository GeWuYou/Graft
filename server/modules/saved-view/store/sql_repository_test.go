package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
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

func TestSQLRepositoryUpdateReplacesOwnedViewAndRejectsConflicts(t *testing.T) {
	t.Parallel()
	repository, _ := newTestRepository(t)
	ctx := context.Background()
	first, err := repository.Create(ctx, createInput(7, "project.list", "first"))
	if err != nil {
		t.Fatalf("create first view: %v", err)
	}
	second, err := repository.Create(ctx, createInput(7, "project.list", "second"))
	if err != nil {
		t.Fatalf("create second view: %v", err)
	}

	input := updateInput(second, "renamed")
	updated, err := repository.Update(ctx, input)
	if err != nil {
		t.Fatalf("update view: %v", err)
	}
	if updated.ID != second.ID || updated.Name != input.Name || updated.PageSize != input.PageSize || string(updated.QueryState) != string(input.QueryState) || len(updated.VisibleColumns) != len(input.VisibleColumns) || updated.VisibleColumns[0] != input.VisibleColumns[0] {
		t.Fatalf("updated view mismatch: %#v", updated)
	}

	conflict := updateInput(second, first.Name)
	if _, err := repository.Update(ctx, conflict); !errors.Is(err, moduleapi.ErrSavedViewConflict) {
		t.Fatalf("expected rename conflict, got %v", err)
	}
	foreign := updateInput(second, "foreign")
	foreign.OwnerUserID = 8
	if _, err := repository.Update(ctx, foreign); !errors.Is(err, moduleapi.ErrSavedViewNotFound) {
		t.Fatalf("expected foreign update not found, got %v", err)
	}
}

func TestSQLRepositoryDefaultViewIsUniqueWithinOwnerAndSurface(t *testing.T) {
	t.Parallel()
	repository, _ := newTestRepository(t)
	ctx := context.Background()
	firstInput := createInput(7, "project.list", "first")
	firstInput.IsDefault = true
	first, err := repository.Create(ctx, firstInput)
	if err != nil {
		t.Fatalf("create first default view: %v", err)
	}
	if !first.IsDefault {
		t.Fatalf("first view must be returned as default: %#v", first)
	}
	secondInput := createInput(7, "project.list", "second")
	secondInput.IsDefault = true
	second, err := repository.Create(ctx, secondInput)
	if err != nil {
		t.Fatalf("create replacement default view: %v", err)
	}
	items, err := repository.List(ctx, 7, "project.list")
	if err != nil {
		t.Fatalf("list default views: %v", err)
	}
	if len(items) != 2 || countDefaults(items) != 1 || !findView(items, second.ID).IsDefault || findView(items, first.ID).IsDefault {
		t.Fatalf("expected only second view as default: %#v", items)
	}
}

func TestSQLRepositoryUpdateDefaultReplacesExistingDefault(t *testing.T) {
	t.Parallel()
	repository, _ := newTestRepository(t)
	ctx := context.Background()
	defaultInput := createInput(7, "project.list", "default")
	defaultInput.IsDefault = true
	created, err := repository.Create(ctx, defaultInput)
	if err != nil {
		t.Fatalf("create default view: %v", err)
	}
	second, err := repository.Create(ctx, createInput(7, "project.list", "second"))
	if err != nil {
		t.Fatalf("create second view: %v", err)
	}
	update := updateInput(second, "second")
	update.IsDefault = true
	if _, err := repository.Update(ctx, update); err != nil {
		t.Fatalf("replace default via update: %v", err)
	}
	items, err := repository.List(ctx, 7, "project.list")
	if err != nil || len(items) != 2 || countDefaults(items) != 1 || findView(items, created.ID).IsDefault || !findView(items, second.ID).IsDefault {
		t.Fatalf("expected update to replace default atomically: %#v, %v", items, err)
	}
}

func TestSQLRepositoryMissingDefaultUpdateRollsBackPriorDefault(t *testing.T) {
	t.Parallel()
	repository, _ := newTestRepository(t)
	ctx := context.Background()
	defaultInput := createInput(7, "project.list", "default")
	defaultInput.IsDefault = true
	created, err := repository.Create(ctx, defaultInput)
	if err != nil {
		t.Fatalf("create default view: %v", err)
	}
	missing := updateInput(created, "missing")
	missing.ID = created.ID + 100
	missing.IsDefault = true
	if _, err := repository.Update(ctx, missing); !errors.Is(err, moduleapi.ErrSavedViewNotFound) {
		t.Fatalf("expected missing view error, got %v", err)
	}
	items, err := repository.List(ctx, 7, "project.list")
	if err != nil || len(items) != 1 || !items[0].IsDefault {
		t.Fatalf("failed update must retain prior default: %#v, %v", items, err)
	}
}

func countDefaults(items []moduleapi.SavedView) int {
	count := 0
	for _, item := range items {
		if item.IsDefault {
			count++
		}
	}
	return count
}

func findView(items []moduleapi.SavedView, id uint64) moduleapi.SavedView {
	for _, item := range items {
		if item.ID == id {
			return item
		}
	}
	return moduleapi.SavedView{}
}

func TestColumnsValueScanAcceptsDatabaseJSONValues(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		value any
		want  []string
	}{
		{name: "bytes", value: []byte(`["name"]`), want: []string{"name"}},
		{name: "string", value: `["runtime_status"]`, want: []string{"runtime_status"}},
		{name: "nil", value: nil, want: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			columns := []string{"previous"}
			if err := columnScanner(&columns).Scan(tc.value); err != nil {
				t.Fatalf("scan columns: %v", err)
			}
			if len(columns) != len(tc.want) {
				t.Fatalf("columns length = %d, want %d", len(columns), len(tc.want))
			}
			for index := range columns {
				if columns[index] != tc.want[index] {
					t.Fatalf("columns[%d] = %q, want %q", index, columns[index], tc.want[index])
				}
			}
		})
	}
	if err := columnScanner(new([]string)).Scan(1); err == nil {
		t.Fatal("expected unsupported database value error")
	}
}

func TestMapWriteErrorRecognizesPostgresUniqueViolation(t *testing.T) {
	t.Parallel()
	err := mapWriteError(&pgconn.PgError{Code: "23505"})
	if !errors.Is(err, moduleapi.ErrSavedViewConflict) {
		t.Fatalf("expected saved view conflict, got %v", err)
	}
}

func createInput(owner uint64, surface, name string) moduleapi.SavedViewCreateInput {
	state, _ := json.Marshal(map[string]string{"source_kind": "managed"})
	return moduleapi.SavedViewCreateInput{OwnerUserID: owner, SurfaceKey: surface, Name: name, QueryState: state, PageSize: 20, VisibleColumns: []string{"name", "runtime_status"}}
}

func updateInput(view moduleapi.SavedView, name string) moduleapi.SavedViewUpdateInput {
	return moduleapi.SavedViewUpdateInput{ID: view.ID, OwnerUserID: view.OwnerUserID, SurfaceKey: view.SurfaceKey, Name: name, QueryState: view.QueryState, PageSize: 50, VisibleColumns: []string{"name"}}
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
		is_default BOOLEAN NOT NULL DEFAULT FALSE,
		created_at DATETIME NOT NULL, created_by INTEGER NOT NULL, updated_at DATETIME NOT NULL, updated_by INTEGER NOT NULL,
		deleted_at INTEGER NOT NULL DEFAULT 0, deleted_by INTEGER NOT NULL DEFAULT 0
	); CREATE UNIQUE INDEX uq_saved_views_owner_surface_name_live ON saved_views (owner_user_id, surface_key, name) WHERE deleted_at = 0;
	CREATE UNIQUE INDEX uq_saved_views_owner_surface_default_live ON saved_views (owner_user_id, surface_key) WHERE deleted_at = 0 AND is_default = TRUE;`); err != nil {
		t.Fatalf("create saved view test table: %v", err)
	}
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	return repository, db
}
