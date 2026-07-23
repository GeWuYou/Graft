package testschema

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestCreateSQLiteEnforcesTaskStageUniqueness(t *testing.T) {
	db := openSQLite(t, "task-testschema-uniqueness")
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

// TestCreateSQLiteEnforcesExternalReceiptIntegrity 验证外部执行回执的 SQLite 完整性约束。
func TestCreateSQLiteEnforcesExternalReceiptIntegrity(t *testing.T) {
	db := openSQLite(t, "task-testschema-external-receipt")
	insertTask(t, db, "first")
	insertTask(t, db, "second")
	insertStage(t, db, 1, "first-stage")
	insertStage(t, db, 2, "second-stage")

	assertExternalReceiptRejected(t, db, "failed receipt without failure code", externalReceiptInput{taskID: 1, stageID: 1, outcome: "failed"})
	assertExternalReceiptRejected(t, db, "needs_attention receipt without failure code", externalReceiptInput{taskID: 1, stageID: 1, outcome: "needs_attention"})
	assertExternalReceiptRejected(t, db, "blank executor type", externalReceiptInput{taskID: 1, stageID: 1, outcome: "success", executorType: "   "})
	assertExternalReceiptRejected(t, db, "task and stage ownership mismatch", externalReceiptInput{taskID: 1, stageID: 2, outcome: "success"})
}

func openSQLite(t *testing.T, name string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", "file:"+name+"?mode=memory&cache=private&_foreign_keys=on")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	t.Cleanup(func() { _ = db.Close() })
	if err := CreateSQLite(db); err != nil {
		t.Fatalf("create task test schema: %v", err)
	}
	return db
}

func insertTask(t *testing.T, db *sql.DB, ownerID string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO tasks (
		task_type, owner_type, owner_id, status, input_json, metadata_json, plan_json, state_json, created_at, updated_at
	) VALUES ('test', 'owner', ?, 'pending', '{}', '{}', '{}', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, ownerID); err != nil {
		t.Fatalf("insert task: %v", err)
	}
}

func insertStage(t *testing.T, db *sql.DB, taskID int, stageKey string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO task_stages (
		task_id, stage_key, sequence, executor_type, status, attempt, max_attempts, retry_backoff_ms, input_json, recovery_policy, result_json, created_at, updated_at
	) VALUES (?, ?, 1, 'test.executor', 'pending', 0, 1, 0, '{}', 'manual_reconcile', '{}', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, taskID, stageKey); err != nil {
		t.Fatalf("insert task stage: %v", err)
	}
}

type externalReceiptInput struct {
	taskID       int
	stageID      int
	outcome      string
	failureCode  any
	executorType string
}

func assertExternalReceiptRejected(t *testing.T, db *sql.DB, name string, input externalReceiptInput) {
	t.Helper()
	executorType := input.executorType
	if executorType == "" {
		executorType = "test.executor"
	}
	if _, err := db.Exec(`INSERT INTO task_external_receipts (
		task_id, stage_id, executor_type, receipt_protocol, operation_id, outcome, failure_code, integrity_sha256, settled_task_status, created_at
	) VALUES (?, ?, ?, 'test/v1', ?, ?, ?, '0123456789012345678901234567890123456789012345678901234567890123', 'failed', CURRENT_TIMESTAMP)`, input.taskID, input.stageID, executorType, name, input.outcome, input.failureCode); err == nil {
		t.Fatalf("%s insert succeeded", name)
	}
}

func assertDuplicateStageRejected(t *testing.T, db *sql.DB, name string, query string) {
	t.Helper()
	if _, err := db.Exec(query); err == nil {
		t.Fatalf("%s insert succeeded", name)
	}
}
