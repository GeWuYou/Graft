package store

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

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

func openRuntimeTargetTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:runtime-target-store?mode=memory&cache=shared")
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
