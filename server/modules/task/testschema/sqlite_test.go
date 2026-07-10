package testschema

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateSQLiteEnforcesTaskStageUniqueness(t *testing.T) {
	db, err := sql.Open("sqlite3", "file:task-testschema-uniqueness?mode=memory&cache=private")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := CreateSQLite(db); err != nil {
		t.Fatalf("create task test schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO tasks (
		task_type, owner_type, owner_id, status, input_json, metadata_json, plan_json, state_json, created_at, updated_at
	) VALUES ('test', 'owner', '1', 'pending', '{}', '{}', '{}', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert task: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO task_stages (
		task_id, stage_key, sequence, executor_type, status, attempt, max_attempts, retry_backoff_ms, input_json, recovery_policy, result_json, created_at, updated_at
	) VALUES (1, 'first', 1, 'test.executor', 'pending', 0, 1, 0, '{}', 'manual_reconcile', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`); err != nil {
		t.Fatalf("insert task stage: %v", err)
	}
	assertDuplicateStageRejected(t, db, "duplicate sequence", `INSERT INTO task_stages (
		task_id, stage_key, sequence, executor_type, status, attempt, max_attempts, retry_backoff_ms, input_json, recovery_policy, result_json, created_at, updated_at
	) VALUES (1, 'second', 1, 'test.executor', 'pending', 0, 1, 0, '{}', 'manual_reconcile', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	assertDuplicateStageRejected(t, db, "duplicate key", `INSERT INTO task_stages (
		task_id, stage_key, sequence, executor_type, status, attempt, max_attempts, retry_backoff_ms, input_json, recovery_policy, result_json, created_at, updated_at
	) VALUES (1, 'first', 2, 'test.executor', 'pending', 0, 1, 0, '{}', 'manual_reconcile', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
	assertDuplicateStageRejected(t, db, "non-positive sequence", `INSERT INTO task_stages (
		task_id, stage_key, sequence, executor_type, status, attempt, max_attempts, retry_backoff_ms, input_json, recovery_policy, result_json, created_at, updated_at
	) VALUES (1, 'third', 0, 'test.executor', 'pending', 0, 1, 0, '{}', 'manual_reconcile', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
}

func assertDuplicateStageRejected(t *testing.T, db *sql.DB, name string, query string) {
	t.Helper()
	if _, err := db.Exec(query); err == nil {
		t.Fatalf("%s insert succeeded", name)
	}
}
