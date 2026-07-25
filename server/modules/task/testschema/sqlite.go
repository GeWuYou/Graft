// Package testschema 提供 Task 测试使用的模块私有 SQLite schema。
package testschema

import (
	"database/sql"
	"fmt"
)

// CreateSQLite 创建 SQLite 单元测试使用的 Task Runtime schema；该 schema 不是生产迁移来源。
func CreateSQLite(db *sql.DB) error {
	for _, statement := range []string{
		`CREATE TABLE tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT, task_type TEXT NOT NULL, owner_type TEXT NOT NULL, owner_id TEXT NOT NULL,
			status TEXT NOT NULL, input_json BLOB NOT NULL, metadata_json BLOB NOT NULL, plan_json BLOB NOT NULL, state_json BLOB NOT NULL,
			current_stage_key TEXT NULL, created_by INTEGER NULL, idempotency_key_hash TEXT NULL, submission_fingerprint TEXT NULL,
			scheduled_at TIMESTAMP NULL, cancel_requested_at TIMESTAMP NULL,
			started_at TIMESTAMP NULL, finished_at TIMESTAMP NULL, duration_ms INTEGER NULL, failure_code TEXT NULL, failure_message TEXT NULL,
			created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL,
			CHECK (trim(task_type) <> ''), CHECK (trim(owner_type) <> ''), CHECK (trim(owner_id) <> ''),
			CHECK (status IN ('pending', 'scheduled', 'running', 'success', 'failed', 'cancelled', 'needs_attention')),
			CHECK (idempotency_key_hash IS NULL OR (length(idempotency_key_hash) = 64 AND idempotency_key_hash NOT GLOB '*[^0-9a-f]*')),
			CHECK (submission_fingerprint IS NULL OR (length(submission_fingerprint) = 64 AND submission_fingerprint NOT GLOB '*[^0-9a-f]*')),
			CHECK (duration_ms IS NULL OR duration_ms >= 0)
		)`,
		`CREATE UNIQUE INDEX uq_tasks_idempotency_submission
			ON tasks (task_type, owner_type, owner_id, COALESCE(created_by, 0), idempotency_key_hash)
			WHERE idempotency_key_hash IS NOT NULL`,
		`CREATE TABLE task_stages (
			id INTEGER PRIMARY KEY AUTOINCREMENT, task_id INTEGER NOT NULL, stage_key TEXT NOT NULL, sequence INTEGER NOT NULL,
			executor_type TEXT NOT NULL, status TEXT NOT NULL, attempt INTEGER NOT NULL, max_attempts INTEGER NOT NULL,
			retry_backoff_ms INTEGER NOT NULL, next_retry_at TIMESTAMP NULL, input_json BLOB NOT NULL, recovery_policy TEXT NOT NULL,
			result_json BLOB NOT NULL, failure_code TEXT NULL, failure_message TEXT NULL, started_at TIMESTAMP NULL,
			finished_at TIMESTAMP NULL, duration_ms INTEGER NULL, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL,
			FOREIGN KEY(task_id) REFERENCES tasks(id),
			UNIQUE(task_id, id),
			UNIQUE(task_id, sequence),
			UNIQUE(task_id, stage_key),
			CHECK (trim(stage_key) <> ''), CHECK (trim(executor_type) <> ''), CHECK (sequence > 0),
			CHECK (status IN ('pending', 'running', 'success', 'failed', 'skipped', 'cancelled', 'unknown')),
			CHECK (attempt >= 0), CHECK (max_attempts >= 1), CHECK (retry_backoff_ms >= 0),
			CHECK (recovery_policy IN ('manual_reconcile', 'retry_if_idempotent')),
			CHECK (duration_ms IS NULL OR duration_ms >= 0)
		)`,
		`CREATE TABLE task_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT, task_id INTEGER NOT NULL, sequence INTEGER NOT NULL, event_type TEXT NOT NULL,
			payload_json BLOB NOT NULL, created_at TIMESTAMP NOT NULL, FOREIGN KEY(task_id) REFERENCES tasks(id),
			UNIQUE(task_id, sequence), CHECK (sequence > 0),
			CHECK (event_type IN ('created', 'cancel_requested', 'cancelled', 'retry_requested', 'retry_scheduled', 'recovery_required', 'recovery_resolved', 'external_receipt_settled'))
		)`,
		`CREATE TABLE task_external_receipts (
			id INTEGER PRIMARY KEY AUTOINCREMENT, task_id INTEGER NOT NULL, stage_id INTEGER NOT NULL, executor_type TEXT NOT NULL,
			receipt_protocol TEXT NOT NULL, operation_id TEXT NOT NULL, outcome TEXT NOT NULL, failure_code TEXT NULL,
			integrity_sha256 TEXT NOT NULL, settled_task_status TEXT NOT NULL, created_at TIMESTAMP NOT NULL,
			FOREIGN KEY(task_id) REFERENCES tasks(id), FOREIGN KEY(task_id, stage_id) REFERENCES task_stages(task_id, id),
			UNIQUE(task_id, operation_id), CHECK (outcome IN ('success', 'failed', 'needs_attention')),
			CHECK (trim(executor_type) <> ''), CHECK (trim(receipt_protocol) <> ''), CHECK (trim(operation_id) <> ''),
			CHECK ((outcome = 'success' AND failure_code IS NULL) OR (outcome IN ('failed', 'needs_attention') AND failure_code IS NOT NULL AND trim(failure_code) <> '')),
			CHECK (settled_task_status IN ('success', 'failed', 'needs_attention'))
		)`,
		`CREATE TABLE task_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT, task_id INTEGER NOT NULL, stage_id INTEGER NULL, sequence INTEGER NOT NULL,
			stream TEXT NOT NULL, level TEXT NOT NULL, line TEXT NOT NULL, occurred_at TIMESTAMP NOT NULL,
			FOREIGN KEY(task_id) REFERENCES tasks(id), FOREIGN KEY(stage_id) REFERENCES task_stages(id),
			UNIQUE(task_id, sequence), CHECK (sequence > 0),
			CHECK (stream IN ('stdout', 'stderr', 'system')), CHECK (level IN ('info', 'warn', 'error')),
			CHECK (trim(line) <> '')
		)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("create task test schema: %w", err)
		}
	}
	return nil
}
