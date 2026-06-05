package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/cronx"
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
		TaskName:    "audit.audit-log-retention-cleanup",
		Owner:       "audit",
		Module:      "audit",
		TaskType:    cronx.TaskTypeSystem,
		TriggerType: TriggerTypeManual,
		Status:      RunStatusRunning,
		StartedAt:   startedAt,
		CreatedAt:   startedAt,
	})
	if err != nil {
		t.Fatalf("create run: %v", err)
	}

	finishedAt := startedAt.Add(1500 * time.Millisecond)
	finished, err := repo.FinishRun(context.Background(), run.ID, RunStatusSuccess, finishedAt, "ok", "")
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

func TestSQLTaskRepositorySeedsBuiltinWithoutOverwritingCronOrEnabled(t *testing.T) {
	db := newSchedulerRepositoryTestDB(t)
	repo, err := NewSQLTaskRepository(db)
	if err != nil {
		t.Fatalf("new task repository: %v", err)
	}

	ctx := context.Background()
	seeded := TaskDefinition{
		TaskKey:        "audit.retention.cleanup",
		TaskType:       cronx.TaskTypeSystem,
		Title:          "audit.retention.title",
		Description:    "audit.retention.description",
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
	}); err != nil {
		t.Fatalf("update builtin cron/enabled: %v", err)
	}
	seeded.Title = "audit.retention.updated"
	seeded.CronExpression = "0 0 1 * * *"
	seeded.Enabled = true
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
}

func TestSQLTaskRepositoryCreatesAndSoftDeletesHTTPTask(t *testing.T) {
	db := newSchedulerRepositoryTestDB(t)
	repo, err := NewSQLTaskRepository(db)
	if err != nil {
		t.Fatalf("new task repository: %v", err)
	}

	ctx := context.Background()
	task, err := repo.CreateTask(ctx, TaskDefinition{
		TaskKey:        "http.ping",
		TaskType:       cronx.TaskTypeHTTP,
		Title:          "Ping",
		CronExpression: "*/30 * * * * *",
		Enabled:        true,
		ConfigJSON:     `{"method":"GET","url":"https://example.com","timeout_seconds":30}`,
		CreatedAt:      time.Date(2026, 6, 5, 8, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 6, 5, 8, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create http task: %v", err)
	}
	if task.TaskType != cronx.TaskTypeHTTP || !task.Enabled || task.Builtin {
		t.Fatalf("unexpected task: %#v", task)
	}
	if err := repo.DeleteTask(ctx, task.TaskKey); err != nil {
		t.Fatalf("delete http task: %v", err)
	}
	if _, err := repo.GetTask(ctx, task.TaskKey); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected soft-deleted task to be hidden, got %v", err)
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
		task_type text NOT NULL,
		title text NOT NULL DEFAULT '',
		description text NOT NULL DEFAULT '',
		cron_expression text NOT NULL,
		enabled boolean NOT NULL DEFAULT true,
		builtin boolean NOT NULL DEFAULT false,
		config_json text NOT NULL DEFAULT '{}',
		created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at datetime NULL
	);
	CREATE TABLE scheduler_task_runs (
		id integer PRIMARY KEY AUTOINCREMENT,
		task_key text NOT NULL,
		task_name text NOT NULL DEFAULT '',
		owner text NOT NULL DEFAULT '',
		module text NOT NULL DEFAULT '',
		task_type text NOT NULL DEFAULT 'cron',
		trigger_type text NOT NULL,
		status text NOT NULL,
		error text NOT NULL DEFAULT '',
		result_summary text NOT NULL DEFAULT '',
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
