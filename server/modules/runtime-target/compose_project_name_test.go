package runtimetarget

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"graft/server/internal/moduleapi"
	store "graft/server/modules/runtime-target/store"
)

type stubComposeProjectNameProbe struct {
	occupied bool
	err      error
}

func (s stubComposeProjectNameProbe) Occupied(context.Context, store.Target, string) (bool, error) {
	return s.occupied, s.err
}

func TestCheckComposeProjectNameReportsRuntimeAuthorityStates(t *testing.T) {
	db := openComposeProjectNameTestDB(t)
	if _, err := db.Exec(`INSERT INTO runtime_targets (id, provider, display_name, endpoint_label, connection_kind, capabilities_json, availability, last_error, deleted_at) VALUES (1, 'docker', 'Local Docker', 'unix:///var/run/docker.sock', 'unix_socket', '["compose_execution", "workspace_access"]', true, '', 0)`); err != nil {
		t.Fatalf("insert target: %v", err)
	}
	repository := store.NewSQLRepository(db)
	for _, tc := range []struct {
		name  string
		probe stubComposeProjectNameProbe
		want  moduleapi.ComposeProjectNameState
	}{
		{name: "occupied", probe: stubComposeProjectNameProbe{occupied: true}, want: moduleapi.ComposeProjectNameStateOccupied},
		{name: "available", probe: stubComposeProjectNameProbe{}, want: moduleapi.ComposeProjectNameStateAvailable},
		{name: "probe error", probe: stubComposeProjectNameProbe{err: errors.New("docker unavailable")}, want: moduleapi.ComposeProjectNameStateError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader := runtimeTargetReader{repository: repository, composeProjectNameProbe: tc.probe}
			result, err := reader.CheckComposeProjectName(context.Background(), 1, "demo")
			if err != nil {
				t.Fatalf("check compose project name: %v", err)
			}
			if result.State != tc.want {
				t.Fatalf("state = %q, want %q", result.State, tc.want)
			}
		})
	}
}

func TestCheckComposeProjectNameReportsUnavailableTarget(t *testing.T) {
	db := openComposeProjectNameTestDB(t)
	reader := runtimeTargetReader{repository: store.NewSQLRepository(db)}
	result, err := reader.CheckComposeProjectName(context.Background(), 999, "demo")
	if err != nil {
		t.Fatalf("check compose project name: %v", err)
	}
	if result.State != moduleapi.ComposeProjectNameStateUnavailable {
		t.Fatalf("state = %q, want unavailable", result.State)
	}
}

func TestReadDockerTargetPropagatesRepositoryErrors(t *testing.T) {
	db := openComposeProjectNameTestDB(t)
	if err := db.Close(); err != nil {
		t.Fatalf("close test database: %v", err)
	}
	reader := runtimeTargetReader{repository: store.NewSQLRepository(db)}
	id := int64(1)

	for _, testID := range []*int64{&id, nil} {
		_, err := reader.ReadDockerTarget(context.Background(), testID)
		if err == nil || errors.Is(err, store.ErrNotFound) {
			t.Fatalf("expected repository error, got %v", err)
		}
	}
}

func openComposeProjectNameTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE runtime_targets (
		id INTEGER PRIMARY KEY,
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
		t.Fatalf("create runtime_targets: %v", err)
	}
	return db
}
