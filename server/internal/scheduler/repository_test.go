// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLRunRepositoryPersistsRunLifecycle(t *testing.T) {
	db := newSchedulerRepositoryTestDB(t)
	repo, err := NewSQLRunRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}

	startedAt := time.Date(2026, 6, 5, 8, 0, 0, 0, time.UTC)
	run, err := repo.CreateRun(context.Background(), TaskRun{
		TaskKey:     "audit.audit-log-retention-cleanup",
		JobKey:      "audit.audit-log-retention-cleanup",
		TaskName:    "audit.audit-log-retention-cleanup",
		Owner:       "audit",
		Module:      "audit",
		TriggerType: TriggerTypeManual,
		Status:      RunStatusRunning,
		StartedAt:   startedAt,
		CreatedAt:   startedAt,
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	finishedAt := startedAt.Add(1500 * time.Millisecond)
	finished, err := repo.FinishRun(context.Background(), RunFinishCommand{
		ID:            run.ID,
		Status:        RunStatusSuccess,
		FinishedAt:    finishedAt,
		ResultJSON:    `{"summary":"ok"}`,
		ResultSummary: "ok",
	})
	if err != nil {
		t.Fatalf("finish run: %v", err)
	}
	if finished.Status != RunStatusSuccess || finished.DurationMS == nil || *finished.DurationMS != 1500 {
		t.Fatalf("unexpected finished run: %#v", finished)
	}
	if finished.Result != "ok" || finished.Error != "" {
		t.Fatalf("expected result summary without error, got %#v", finished)
	}

	result, err := repo.ListRuns(context.Background(), RunListQuery{TaskKey: "audit.audit-log-retention-cleanup"})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("expected one run, got %#v", result)
	}

	latest, ok, err := repo.LatestRunByTask(context.Background(), "audit.audit-log-retention-cleanup")
	if err != nil {
		t.Fatalf("latest run: %v", err)
	}
	if !ok || latest.ID != run.ID {
		t.Fatalf("expected latest run %d, got ok=%v run=%#v", run.ID, ok, latest)
	}
}

func TestSQLJobDefinitionRepositorySyncsDefinitions(t *testing.T) {
	db := newSchedulerRepositoryTestDB(t)
	repo, err := NewSQLJobDefinitionRepository(db)
	if err != nil {
		t.Fatalf("new job definition repository: %v", err)
	}

	ctx := context.Background()
	definition := JobDefinition{
		JobKey:        "audit.retention.cleanup",
		ModuleKey:     "audit",
		Title:         "Audit retention",
		ConfigSchema:  "{}",
		DefaultConfig: "{}",
		DefaultCron:   "0 0 * * * *",
		Enabled:       true,
		CreatedAt:     time.Date(2026, 6, 5, 8, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 5, 8, 0, 0, 0, time.UTC),
	}
	if err := repo.SyncJobDefinitions(ctx, []JobDefinition{definition}); err != nil {
		t.Fatalf("sync job definition: %v", err)
	}
	definition.Title = "Audit retention updated"
	if err := repo.SyncJobDefinitions(ctx, []JobDefinition{definition}); err != nil {
		t.Fatalf("sync job definition again: %v", err)
	}
	got, err := repo.GetJobDefinition(ctx, definition.JobKey)
	if err != nil {
		t.Fatalf("get job definition: %v", err)
	}
	if got.Title != "Audit retention updated" || got.ModuleKey != "audit" {
		t.Fatalf("unexpected job definition: %#v", got)
	}
}

func TestSQLTaskRepositorySeedsBuiltinPreservesCronAndEnabledWhileRefreshingConfig(t *testing.T) {
	db := newSchedulerRepositoryTestDB(t)
	repo, err := NewSQLTaskRepository(db)
	if err != nil {
		t.Fatalf("new task repository: %v", err)
	}

	ctx := context.Background()
	seeded := TaskDefinition{
		TaskKey:        "audit.retention.cleanup",
		JobKey:         "audit.retention.cleanup",
		ModuleKey:      "audit",
		Title:          "scheduledTask.auditLogRetention.title",
		Description:    "scheduledTask.auditLogRetention.description",
		CronExpression: "0 0 * * * *",
		Enabled:        true,
		Builtin:        true,
		ConfigJSON:     "{}",
		CreatedAt:      time.Date(2026, 6, 5, 8, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 6, 5, 8, 0, 0, 0, time.UTC),
	}
	if err := repo.SeedBuiltinTasks(ctx, []TaskDefinition{seeded}); err != nil {
		t.Fatalf("seed builtin task: %v", err)
	}
	if _, err := repo.UpdateTask(ctx, seeded.TaskKey, TaskMutation{
		CronExpression: "0 */5 * * * *",
		Enabled:        false,
		EnabledSet:     true,
		ConfigJSON:     `{"retentionDays":90,"batchSize":500}`,
	}); err != nil {
		t.Fatalf("update builtin cron/enabled/config: %v", err)
	}
	seeded.Title = "audit.retention.updated"
	seeded.CronExpression = "0 0 1 * * *"
	seeded.Enabled = true
	seeded.ConfigJSON = `{"retentionDays":30,"batchSize":1000}`
	if err := repo.SeedBuiltinTasks(ctx, []TaskDefinition{seeded}); err != nil {
		t.Fatalf("seed builtin task again: %v", err)
	}

	task, err := repo.GetTask(ctx, seeded.TaskKey)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.Title != "audit.retention.updated" {
		t.Fatalf("expected metadata refresh, got %#v", task)
	}
	if task.CronExpression != "0 */5 * * * *" || task.Enabled {
		t.Fatalf("expected user-edited cron/enabled to survive reseed, got %#v", task)
	}
	if task.ConfigJSON != `{"retentionDays":30,"batchSize":1000}` {
		t.Fatalf("expected repository to accept runtime-selected builtin config, got %#v", task)
	}
	if task.ConfigSource != taskConfigSourceSystem {
		t.Fatalf("expected reseeded builtin config to use system source, got %#v", task)
	}
}

func TestSQLTaskRepositoryCreatesMultipleTasksForOneJobAndSoftDeletes(t *testing.T) {
	db := newSchedulerRepositoryTestDB(t)
	repo, err := NewSQLTaskRepository(db)
	if err != nil {
		t.Fatalf("new task repository: %v", err)
	}

	ctx := context.Background()
	task, err := repo.CreateTask(ctx, TaskDefinition{
		TaskKey:        "audit.retention.nightly",
		JobKey:         "audit.retention.cleanup",
		ModuleKey:      "audit",
		Title:          "Ping",
		CronExpression: "*/30 * * * * *",
		Enabled:        true,
		ConfigJSON:     `{"retention_days":30}`,
		CreatedAt:      time.Date(2026, 6, 5, 8, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 6, 5, 8, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.JobKey != "audit.retention.cleanup" || task.ConfigJSON == "" || !task.Enabled || task.Builtin {
		t.Fatalf("unexpected task: %#v", task)
	}
	if _, err := repo.CreateTask(ctx, TaskDefinition{
		TaskKey:        "audit.retention.weekly",
		JobKey:         "audit.retention.cleanup",
		ModuleKey:      "audit",
		Title:          "Weekly cleanup",
		CronExpression: "0 0 0 * * 0",
		Enabled:        true,
		ConfigJSON:     `{"retention_days":90}`,
	}); err != nil {
		t.Fatalf("create second task for same job: %v", err)
	}
	if err := repo.DeleteTask(ctx, task.TaskKey); err != nil {
		t.Fatalf("delete task: %v", err)
	}
	if _, err := repo.GetTask(ctx, task.TaskKey); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected soft-deleted task to be hidden, got %v", err)
	}
}

func TestSQLTaskRepositoryListTasksNormalizesPagination(t *testing.T) {
	db := newSchedulerRepositoryTestDB(t)
	repo, err := NewSQLTaskRepository(db)
	if err != nil {
		t.Fatalf("new task repository: %v", err)
	}

	ctx := context.Background()
	for i := range maxTaskListLimit + 5 {
		key := fmt.Sprintf("audit.retention.task-%03d", i)
		if _, err := repo.CreateTask(ctx, TaskDefinition{
			TaskKey:        key,
			JobKey:         "audit.retention.cleanup",
			ModuleKey:      "audit",
			Title:          key,
			CronExpression: "*/30 * * * * *",
			Enabled:        true,
			ConfigJSON:     "{}",
			CreatedAt:      time.Date(2026, 6, 5, 8, i, 0, 0, time.UTC),
			UpdatedAt:      time.Date(2026, 6, 5, 8, i, 0, 0, time.UTC),
		}); err != nil {
			t.Fatalf("create task %d: %v", i, err)
		}
	}

	items, total, err := repo.ListTasks(ctx, TaskListQuery{Limit: 0, Offset: -5})
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if total != maxTaskListLimit+5 {
		t.Fatalf("expected total %d, got %d", maxTaskListLimit+5, total)
	}
	if len(items) != defaultTaskListLimit {
		t.Fatalf("expected default page size %d, got %d", defaultTaskListLimit, len(items))
	}
	if items[0].TaskKey != "audit.retention.task-000" {
		t.Fatalf("expected negative offset to clamp to first item, got %#v", items[0])
	}

	items, total, err = repo.ListTasks(ctx, TaskListQuery{Limit: 3, Offset: 2})
	if err != nil {
		t.Fatalf("list paged tasks: %v", err)
	}
	if total != maxTaskListLimit+5 || len(items) != 3 {
		t.Fatalf("expected three of %d tasks, got total=%d items=%d", maxTaskListLimit+5, total, len(items))
	}
	if items[0].TaskKey != "audit.retention.task-002" {
		t.Fatalf("expected offset page to start at task-002, got %#v", items[0])
	}

	items, total, err = repo.ListTasks(ctx, TaskListQuery{Limit: maxTaskListLimit + 1})
	if err != nil {
		t.Fatalf("list capped tasks: %v", err)
	}
	if total != maxTaskListLimit+5 || len(items) != maxTaskListLimit {
		t.Fatalf("expected capped limit to keep bounded query and return available tasks, got total=%d items=%d", total, len(items))
	}
}

func newSchedulerRepositoryTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.Exec(`CREATE TABLE scheduled_tasks (
		id integer PRIMARY KEY AUTOINCREMENT,
		task_key text NOT NULL UNIQUE,
		job_key text NOT NULL,
		module_key text NOT NULL,
		title text NOT NULL DEFAULT '',
		description text NOT NULL DEFAULT '',
		cron_expression text NOT NULL,
		enabled boolean NOT NULL DEFAULT true,
			builtin boolean NOT NULL DEFAULT false,
			task_type text NOT NULL DEFAULT 'job',
			config_json text NOT NULL DEFAULT '{}',
			config_source text NOT NULL DEFAULT 'system',
			created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at datetime NULL
	);
	CREATE TABLE scheduler_job_definitions (
		id integer PRIMARY KEY AUTOINCREMENT,
		job_key text NOT NULL UNIQUE,
		module_key text NOT NULL,
		title_key text NOT NULL DEFAULT '',
		title text NOT NULL DEFAULT '',
		description_key text NOT NULL DEFAULT '',
		description text NOT NULL DEFAULT '',
		config_schema text NOT NULL DEFAULT '{}',
		default_config text NOT NULL DEFAULT '{}',
		default_cron text NOT NULL,
		enabled boolean NOT NULL DEFAULT true,
		created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at datetime NULL
	);
	CREATE TABLE scheduler_task_runs (
		id integer PRIMARY KEY AUTOINCREMENT,
		task_key text NOT NULL,
		job_key text NOT NULL DEFAULT '',
		task_name text NOT NULL DEFAULT '',
		owner text NOT NULL DEFAULT '',
		module text NOT NULL DEFAULT '',
		task_type text NOT NULL DEFAULT 'cron',
		trigger_type text NOT NULL,
		status text NOT NULL,
		error text NOT NULL DEFAULT '',
		result_summary text NOT NULL DEFAULT '',
		result_json text NOT NULL DEFAULT '{}',
		error_message text NOT NULL DEFAULT '',
		started_at datetime NOT NULL,
		finished_at datetime NULL,
		duration_ms integer NULL,
		created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}

	return db
}
