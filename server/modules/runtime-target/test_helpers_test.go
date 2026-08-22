package runtimetarget

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func openRuntimeTargetTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
