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
			status TEXT NOT NULL, input_json BLOB NOT NULL, metadata_json BLOB NOT NULL, plan_json BLOB NOT NULL, state_json BLOB NOT NULL, activation_required BOOLEAN NOT NULL DEFAULT FALSE,
			current_stage_key TEXT NULL, created_by INTEGER NULL, idempotency_key_hash TEXT NULL, submission_fingerprint TEXT NULL,
			scheduled_at TIMESTAMP NULL, cancel_requested_at TIMESTAMP NULL,
			started_at TIMESTAMP NULL, finished_at TIMESTAMP NULL, duration_ms INTEGER NULL, failure_code TEXT NULL, failure_message TEXT NULL,
			created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL,
			CHECK (trim(task_type) <> ''), CHECK (trim(owner_type) <> ''), CHECK (trim(owner_id) <> ''),
			CHECK (status IN ('pending', 'ready', 'scheduled', 'running', 'success', 'failed', 'cancelled', 'needs_attention')),
			CHECK (idempotency_key_hash IS NULL OR (length(idempotency_key_hash) = 64 AND idempotency_key_hash NOT GLOB '*[^0-9a-f]*')),
			CHECK (submission_fingerprint IS NULL OR (length(submission_fingerprint) = 64 AND submission_fingerprint NOT GLOB '*[^0-9a-f]*')),
			CHECK (duration_ms IS NULL OR duration_ms >= 0)
		)`,
		`CREATE UNIQUE INDEX uq_tasks_idempotency_submission
			ON tasks (task_type, owner_type, owner_id, COALESCE(created_by, 0), idempotency_key_hash)
			WHERE idempotency_key_hash IS NOT NULL`,
		`CREATE UNIQUE INDEX uq_tasks_active_owner
			ON tasks (owner_type, owner_id)
			WHERE status IN ('pending', 'ready', 'scheduled', 'running', 'needs_attention')`,
		`CREATE TABLE task_submissions (
			id TEXT PRIMARY KEY, task_type TEXT NOT NULL, owner_type TEXT NOT NULL, owner_id TEXT NOT NULL, requested_by INTEGER NULL,
			idempotency_key_hash TEXT NULL, submission_fingerprint TEXT NULL, state TEXT NOT NULL, submission_version INTEGER NOT NULL, lease_ttl_ms INTEGER NOT NULL, lease_renewable BOOLEAN NOT NULL,
			lease_token_hash TEXT NOT NULL, lease_expires_at TIMESTAMP NOT NULL, absolute_deadline_at TIMESTAMP NOT NULL, prerequisite_kind TEXT NOT NULL,
			prerequisite_ref TEXT NULL, task_id INTEGER NULL, terminal_reason TEXT NULL, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL,
			activated_at TIMESTAMP NULL, terminal_at TIMESTAMP NULL, FOREIGN KEY(task_id) REFERENCES tasks(id),
			CHECK (state IN ('reserved', 'activated', 'discarded', 'expired')), CHECK (submission_version > 0), CHECK (lease_ttl_ms > 0), CHECK (lease_expires_at <= absolute_deadline_at)
		)`,
		`CREATE UNIQUE INDEX uq_task_submissions_reserved_owner ON task_submissions (owner_type, owner_id) WHERE state = 'reserved'`,
		`CREATE UNIQUE INDEX uq_task_submissions_idempotency ON task_submissions (task_type, owner_type, owner_id, COALESCE(requested_by, 0), idempotency_key_hash) WHERE idempotency_key_hash IS NOT NULL`,
		`CREATE TABLE task_stages (
			id INTEGER PRIMARY KEY AUTOINCREMENT, task_id INTEGER NOT NULL, stage_key TEXT NOT NULL, sequence INTEGER NOT NULL,
			executor_type TEXT NOT NULL, external_execution BOOLEAN NOT NULL DEFAULT FALSE, status TEXT NOT NULL, attempt INTEGER NOT NULL, max_attempts INTEGER NOT NULL,
			retry_backoff_ms INTEGER NOT NULL, next_retry_at TIMESTAMP NULL, input_json BLOB NOT NULL,
			coordination_group TEXT NOT NULL DEFAULT '', leg_id TEXT NOT NULL DEFAULT '', recovery_policy TEXT NOT NULL,
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
		`CREATE TABLE task_external_execution_leases (
			id TEXT PRIMARY KEY, task_id INTEGER NOT NULL, stage_id INTEGER NOT NULL, attempt INTEGER NOT NULL,
			executor_type TEXT NOT NULL, runtime_target_id INTEGER NOT NULL, provider_id TEXT NOT NULL, capability TEXT NOT NULL,
			receipt_protocol TEXT NOT NULL, operation_id TEXT NOT NULL, payload_sha256 TEXT NOT NULL, fence_token_hash TEXT NOT NULL,
			state TEXT NOT NULL, lease_ttl_ms INTEGER NOT NULL, lease_expires_at TIMESTAMP NOT NULL, absolute_deadline_at TIMESTAMP NOT NULL,
			cancel_observed_at TIMESTAMP NULL, settled_at TIMESTAMP NULL, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL,
			FOREIGN KEY(task_id) REFERENCES tasks(id), FOREIGN KEY(task_id, stage_id) REFERENCES task_stages(task_id, id),
			UNIQUE(task_id, stage_id, attempt), UNIQUE(task_id, operation_id, attempt),
			CHECK (attempt > 0), CHECK (runtime_target_id > 0), CHECK (state IN ('claimed', 'settled', 'expired')),
			CHECK (lease_ttl_ms > 0), CHECK (lease_expires_at <= absolute_deadline_at),
			CHECK ((state = 'settled' AND settled_at IS NOT NULL) OR (state <> 'settled' AND settled_at IS NULL))
		)`,
		`CREATE TABLE task_external_receipts (
			id INTEGER PRIMARY KEY AUTOINCREMENT, lease_id TEXT NULL, task_id INTEGER NOT NULL, stage_id INTEGER NOT NULL,
			attempt INTEGER NOT NULL DEFAULT 1, executor_type TEXT NOT NULL,
			receipt_protocol TEXT NOT NULL, operation_id TEXT NOT NULL, outcome TEXT NOT NULL, failure_code TEXT NULL,
			integrity_sha256 TEXT NOT NULL, settled_stage_status TEXT GENERATED ALWAYS AS (
				CASE outcome WHEN 'success' THEN 'success' WHEN 'failed' THEN 'failed' ELSE 'unknown' END
			) STORED, settled_task_status TEXT NOT NULL, created_at TIMESTAMP NOT NULL,
			FOREIGN KEY(task_id) REFERENCES tasks(id), FOREIGN KEY(task_id, stage_id) REFERENCES task_stages(task_id, id),
			FOREIGN KEY(lease_id) REFERENCES task_external_execution_leases(id),
			UNIQUE(task_id, operation_id, attempt), CHECK (outcome IN ('success', 'failed', 'needs_attention')),
			UNIQUE(lease_id), CHECK (attempt > 0), CHECK (settled_stage_status IN ('success', 'failed', 'unknown')),
			CHECK (trim(executor_type) <> ''), CHECK (trim(receipt_protocol) <> ''), CHECK (trim(operation_id) <> ''),
			CHECK ((outcome = 'success' AND failure_code IS NULL) OR (outcome IN ('failed', 'needs_attention') AND failure_code IS NOT NULL AND trim(failure_code) <> '')),
			CHECK (settled_task_status IN ('running', 'success', 'failed', 'cancelled', 'needs_attention'))
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
