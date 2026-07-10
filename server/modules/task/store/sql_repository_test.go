package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
)

func TestSQLRepositoryCreatePersistsFrozenTaskPlanAndCreatedEvent(t *testing.T) {
	t.Parallel()

	repository, _ := newTestSQLRepository(t)
	created, stages, err := repository.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	assertCreatedTask(t, created, stages)
	assertStoredTaskPlan(t, repository, created.ID)
}

func assertCreatedTask(t *testing.T, created taskmodel.Task, stages []taskmodel.Stage) {
	t.Helper()
	if created.ID == 0 || created.Status != moduleapi.TaskStatusPending {
		t.Fatalf("unexpected created task: %#v", created)
	}
	if len(stages) != 2 || stages[0].ID == 0 || stages[0].TaskID != created.ID || stages[1].Sequence != 2 {
		t.Fatalf("unexpected persisted stages: %#v", stages)
	}
}

func assertStoredTaskPlan(t *testing.T, repository *SQLRepository, taskID uint64) {
	t.Helper()
	loaded, err := repository.Get(context.Background(), taskID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if loaded.Type != "project.compose.redeploy" || loaded.Owner.Type != "compose_project" || loaded.Owner.ID != "42" {
		t.Fatalf("unexpected loaded task: %#v", loaded)
	}
	assertStoredStages(t, repository, taskID)
	assertCreatedEvent(t, repository, taskID)
}

func assertStoredStages(t *testing.T, repository *SQLRepository, taskID uint64) {
	t.Helper()
	loadedStages, err := repository.ListStages(context.Background(), taskID)
	if err != nil {
		t.Fatalf("list stages: %v", err)
	}
	if len(loadedStages) != 2 || loadedStages[0].Key != "prepare" || loadedStages[1].Key != "up" {
		t.Fatalf("unexpected loaded stages: %#v", loadedStages)
	}
}

func assertCreatedEvent(t *testing.T, repository *SQLRepository, taskID uint64) {
	t.Helper()
	events, err := repository.ListEvents(context.Background(), taskID, 0, 10)
	if err != nil {
		t.Fatalf("list created event: %v", err)
	}
	if len(events) != 1 || events[0].Type != taskmodel.EventTypeCreated || events[0].Sequence != 1 {
		t.Fatalf("unexpected initial events: %#v", events)
	}
}

func TestSQLRepositoryTransitionsUseCompareAndSwap(t *testing.T) {
	t.Parallel()

	repository, _ := newTestSQLRepository(t)
	created, stages, err := repository.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	if err := repository.TransitionTask(context.Background(), TaskTransitionInput{
		TaskID: created.ID,
		From:   moduleapi.TaskStatusPending,
		To:     moduleapi.TaskStatusRunning,
	}); err != nil {
		t.Fatalf("transition task: %v", err)
	}
	if err := repository.TransitionTask(context.Background(), TaskTransitionInput{
		TaskID: created.ID,
		From:   moduleapi.TaskStatusPending,
		To:     moduleapi.TaskStatusRunning,
	}); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("expected stale task transition conflict, got %v", err)
	}

	if err := repository.TransitionStage(context.Background(), StageTransitionInput{
		StageID: stages[0].ID,
		From:    moduleapi.StageStatusPending,
		To:      moduleapi.StageStatusRunning,
		Attempt: 1,
	}); err != nil {
		t.Fatalf("transition stage: %v", err)
	}
	if err := repository.TransitionStage(context.Background(), StageTransitionInput{
		StageID: stages[0].ID,
		From:    moduleapi.StageStatusPending,
		To:      moduleapi.StageStatusRunning,
		Attempt: 1,
	}); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("expected stale stage transition conflict, got %v", err)
	}
}

func TestSQLRepositoryReplaysEventsAndLogsBySequence(t *testing.T) {
	t.Parallel()

	repository, _ := newTestSQLRepository(t)
	created, stages, err := repository.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if _, err := repository.AppendEvent(context.Background(), AppendEventInput{
		TaskID: created.ID, Sequence: 2, Type: taskmodel.EventTypeCancelRequested,
	}); err != nil {
		t.Fatalf("append event: %v", err)
	}
	stageID := stages[0].ID
	if _, err := repository.AppendLog(context.Background(), AppendLogInput{
		TaskID: created.ID, StageID: &stageID, Sequence: 1, Stream: "stdout", Level: "info", Line: "pulling image",
	}); err != nil {
		t.Fatalf("append log: %v", err)
	}
	if _, err := repository.AppendLog(context.Background(), AppendLogInput{
		TaskID: created.ID, Sequence: 2, Stream: "system", Level: "warn", Line: "cancellation requested",
	}); err != nil {
		t.Fatalf("append second log: %v", err)
	}

	events, err := repository.ListEvents(context.Background(), created.ID, 1, 10)
	if err != nil {
		t.Fatalf("list events after cursor: %v", err)
	}
	if len(events) != 1 || events[0].Sequence != 2 {
		t.Fatalf("unexpected replay events: %#v", events)
	}
	logs, err := repository.ListLogs(context.Background(), created.ID, 1, 10)
	if err != nil {
		t.Fatalf("list logs after cursor: %v", err)
	}
	if len(logs) != 1 || logs[0].Sequence != 2 || logs[0].StageID != nil {
		t.Fatalf("unexpected replay logs: %#v", logs)
	}
}

func TestSQLRepositoryRejectsNonSerialOrDuplicateStagePlan(t *testing.T) {
	t.Parallel()

	repository, _ := newTestSQLRepository(t)
	input := validCreateInput()
	input.Stages[1].Sequence = 3
	if _, _, err := repository.Create(context.Background(), input); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid serial plan, got %v", err)
	}
	input = validCreateInput()
	input.Stages[1].Key = input.Stages[0].Key
	if _, _, err := repository.Create(context.Background(), input); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected duplicate stage key rejection, got %v", err)
	}
}

func validCreateInput() CreateInput {
	return CreateInput{
		Task: taskmodel.Task{
			Type:     "project.compose.redeploy",
			Owner:    moduleapi.TaskOwner{Type: "compose_project", ID: "42"},
			Status:   moduleapi.TaskStatusPending,
			Input:    []byte(`{"project_id":42}`),
			Metadata: []byte(`{"display_name":"demo"}`),
			Plan:     []byte(`{"stages":["prepare","up"]}`),
		},
		Stages: []taskmodel.Stage{
			{Key: "prepare", Sequence: 1, ExecutorType: "project.compose.prepare", Status: moduleapi.StageStatusPending, MaxAttempts: 1, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile},
			{Key: "up", Sequence: 2, ExecutorType: "project.compose.up", Status: moduleapi.StageStatusPending, MaxAttempts: 2, RetryBackoffMS: 1000, RecoveryPolicy: moduleapi.StageRecoveryRetryIfIdempotent},
		},
	}
}

func newTestSQLRepository(t *testing.T) (*SQLRepository, *sql.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:task-store-%s?mode=memory&cache=private", t.Name())
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable sqlite foreign keys: %v", err)
	}
	createTaskStoreSchema(t, db)
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatalf("new task repository: %v", err)
	}
	return repository, db
}

func createTaskStoreSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, statement := range []string{
		`CREATE TABLE tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT, task_type TEXT NOT NULL, owner_type TEXT NOT NULL, owner_id TEXT NOT NULL,
			status TEXT NOT NULL, input_json BLOB NOT NULL, metadata_json BLOB NOT NULL, plan_json BLOB NOT NULL, state_json BLOB NOT NULL,
			current_stage_key TEXT NULL, created_by INTEGER NULL, scheduled_at TIMESTAMP NULL, cancel_requested_at TIMESTAMP NULL,
			started_at TIMESTAMP NULL, finished_at TIMESTAMP NULL, duration_ms INTEGER NULL, failure_code TEXT NULL, failure_message TEXT NULL,
			created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL
		)`,
		`CREATE TABLE task_stages (
			id INTEGER PRIMARY KEY AUTOINCREMENT, task_id INTEGER NOT NULL, stage_key TEXT NOT NULL, sequence INTEGER NOT NULL,
			executor_type TEXT NOT NULL, status TEXT NOT NULL, attempt INTEGER NOT NULL, max_attempts INTEGER NOT NULL,
			retry_backoff_ms INTEGER NOT NULL, next_retry_at TIMESTAMP NULL, input_json BLOB NOT NULL, recovery_policy TEXT NOT NULL,
			result_json BLOB NOT NULL, failure_code TEXT NULL, failure_message TEXT NULL, started_at TIMESTAMP NULL,
			finished_at TIMESTAMP NULL, duration_ms INTEGER NULL, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL,
			FOREIGN KEY(task_id) REFERENCES tasks(id)
		)`,
		`CREATE TABLE task_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT, task_id INTEGER NOT NULL, sequence INTEGER NOT NULL, event_type TEXT NOT NULL,
			payload_json BLOB NOT NULL, created_at TIMESTAMP NOT NULL, FOREIGN KEY(task_id) REFERENCES tasks(id)
		)`,
		`CREATE TABLE task_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT, task_id INTEGER NOT NULL, stage_id INTEGER NULL, sequence INTEGER NOT NULL,
			stream TEXT NOT NULL, level TEXT NOT NULL, line TEXT NOT NULL, occurred_at TIMESTAMP NOT NULL,
			FOREIGN KEY(task_id) REFERENCES tasks(id), FOREIGN KEY(stage_id) REFERENCES task_stages(id)
		)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create test task store schema: %v", err)
		}
	}
}
