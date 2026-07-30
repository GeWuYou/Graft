package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
	"graft/server/modules/task/testschema"
)

func TestSQLRepositoryCreatePersistsFrozenTaskPlanAndCreatedEvent(t *testing.T) {
	t.Parallel()

	repository, _ := newTestSQLRepository(t)
	created, stages, _, err := repository.Create(context.Background(), validCreateInput())
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
	if loaded.Type != "application.compose.redeploy" || loaded.Owner.Type != "application" || loaded.Owner.ID != "app_01ARZ3NDEKTSV4RRFFQ69G5FAV" {
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
	created, stages, _, err := repository.Create(context.Background(), validCreateInput())
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

func TestSQLRepositoryCancelsUntrackedRunningStageWithDistinctDurations(t *testing.T) {
	t.Parallel()
	repository, _ := newTestSQLRepository(t)
	created, stages, _, err := repository.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	startedAt := time.Now().UTC().Add(-time.Minute)
	if err := repository.TransitionTask(context.Background(), TaskTransitionInput{TaskID: created.ID, From: moduleapi.TaskStatusPending, To: moduleapi.TaskStatusRunning, StartedAt: &startedAt}); err != nil {
		t.Fatalf("transition task to running: %v", err)
	}
	if err := repository.TransitionStage(context.Background(), StageTransitionInput{StageID: stages[0].ID, From: moduleapi.StageStatusPending, To: moduleapi.StageStatusRunning, Attempt: 1, StartedAt: &startedAt}); err != nil {
		t.Fatalf("transition stage to running: %v", err)
	}
	if _, err := repository.RequestCancellation(context.Background(), created.ID, time.Now().UTC()); err != nil {
		t.Fatalf("request cancellation: %v", err)
	}
	stageDurationMS, taskDurationMS := int64(1_000), int64(60_000)
	if err := repository.CancelUntrackedRunningStage(context.Background(), created.ID, stages[0].ID, time.Now().UTC(), &stageDurationMS, &taskDurationMS); err != nil {
		t.Fatalf("cancel untracked running stage: %v", err)
	}
	task, err := repository.Get(context.Background(), created.ID)
	if err != nil || task.DurationMS == nil || *task.DurationMS != taskDurationMS {
		t.Fatalf("cancelled task duration: task=%#v err=%v", task, err)
	}
	storedStages, err := repository.ListStages(context.Background(), created.ID)
	if err != nil || len(storedStages) != 2 || storedStages[0].DurationMS == nil || *storedStages[0].DurationMS != stageDurationMS {
		t.Fatalf("cancelled stage duration: stages=%#v err=%v", storedStages, err)
	}
}

func TestSQLRepositoryReplaysEventsAndLogsBySequence(t *testing.T) {
	t.Parallel()

	repository, _ := newTestSQLRepository(t)
	created, stages, _, err := repository.Create(context.Background(), validCreateInput())
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

func TestSQLRepositoryListsLogPagesInAscendingSequence(t *testing.T) {
	t.Parallel()

	repository, _ := newTestSQLRepository(t)
	first, _, _, err := repository.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("create first task: %v", err)
	}
	secondInput := validCreateInput()
	secondInput.Task.Owner.ID = "another-owner"
	second, _, _, err := repository.Create(context.Background(), secondInput)
	if err != nil {
		t.Fatalf("create second task: %v", err)
	}
	for _, sequence := range []int64{1, 2, 3, 4, 5} {
		if _, err := repository.AppendLog(context.Background(), AppendLogInput{TaskID: first.ID, Sequence: sequence, Stream: "stdout", Level: "info", Line: fmt.Sprintf("first-%d", sequence)}); err != nil {
			t.Fatalf("append first task log %d: %v", sequence, err)
		}
	}
	if _, err := repository.AppendLog(context.Background(), AppendLogInput{TaskID: second.ID, Sequence: 1, Stream: "stdout", Level: "info", Line: "other-task"}); err != nil {
		t.Fatalf("append second task log: %v", err)
	}

	logs, err := repository.ListLogs(context.Background(), first.ID, 2, 2)
	assertLogSequences(t, logs, err, []int64{3, 4})
	logs, err = repository.ListLatestLogs(context.Background(), first.ID, 2)
	assertLogSequences(t, logs, err, []int64{4, 5})
	logs, err = repository.ListLogsBefore(context.Background(), first.ID, 4, 3)
	assertLogSequences(t, logs, err, []int64{1, 2, 3})
	logs, err = repository.ListLatestLogs(context.Background(), second.ID, 10)
	assertLogSequences(t, logs, err, []int64{1})
}

func assertLogSequences(t *testing.T, logs []taskmodel.Log, err error, want []int64) {
	t.Helper()
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(logs) != len(want) {
		t.Fatalf("log count = %d, want %d: %#v", len(logs), len(want), logs)
		return
	}
	for index := 0; index < len(logs) && index < len(want); index += 1 {
		log := logs[index]
		if log.Sequence != want[index] {
			t.Fatalf("log sequence[%d] = %d, want %d", index, log.Sequence, want[index])
		}
	}
}

func TestSQLRepositoryListsOwnerScopedPageAndTotal(t *testing.T) {
	t.Parallel()
	repository, _ := newTestSQLRepository(t)
	for index, ownerID := range []string{"owner-a", "owner-a", "owner-b"} {
		input := validCreateInput()
		input.Task.Owner = moduleapi.TaskOwner{Type: "application", ID: ownerID}
		created, _, _, err := repository.Create(context.Background(), input)
		if err != nil {
			t.Fatalf("create %q task: %v", ownerID, err)
		}
		if index == 0 {
			if _, err := repository.db.Exec(`UPDATE tasks SET status = ? WHERE id = ?`, moduleapi.TaskStatusFailed, created.ID); err != nil {
				t.Fatalf("complete first owner task: %v", err)
			}
		}
	}
	items, total, err := repository.List(context.Background(), moduleapi.TaskListFilter{Owner: moduleapi.TaskOwner{Type: "application", ID: "owner-a"}}, 1, 1)
	if err != nil {
		t.Fatalf("list owner tasks: %v", err)
	}
	if total != 2 || len(items) != 1 || items[0].Owner.ID != "owner-a" {
		t.Fatalf("owner page = %#v total=%d", items, total)
	}
}

func TestSQLRepositoryListsOwnerScopedTypeAndStatusFilter(t *testing.T) {
	t.Parallel()

	repository, _ := newTestSQLRepository(t)
	inputs := []struct {
		taskType moduleapi.TaskType
		status   moduleapi.TaskStatus
	}{
		{taskType: "application.compose.redeploy", status: moduleapi.TaskStatusFailed},
		{taskType: "application.compose.retry", status: moduleapi.TaskStatusFailed},
		{taskType: "application.compose.retry", status: moduleapi.TaskStatusSuccess},
	}
	for _, spec := range inputs {
		input := validCreateInput()
		input.Task.Owner = moduleapi.TaskOwner{Type: "application", ID: "owner-filtered"}
		created, _, _, err := repository.Create(context.Background(), input)
		if err != nil {
			t.Fatalf("create task: %v", err)
		}
		if _, err := repository.db.Exec(`UPDATE tasks SET task_type = ?, status = ? WHERE id = ?`, spec.taskType, spec.status, created.ID); err != nil {
			t.Fatalf("seed task filters: %v", err)
		}
	}
	taskType := moduleapi.TaskType("application.compose.retry")
	status := moduleapi.TaskStatusFailed
	items, total, err := repository.List(context.Background(), moduleapi.TaskListFilter{
		Owner:  moduleapi.TaskOwner{Type: "application", ID: "owner-filtered"},
		Type:   &taskType,
		Status: &status,
	}, 20, 0)
	if err != nil {
		t.Fatalf("list filtered tasks: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Type != taskType || items[0].Status != status {
		t.Fatalf("filtered tasks = %#v total=%d", items, total)
	}
}

func TestJSONValuePlaceholderMatchesSQLDialect(t *testing.T) {
	if got := (&SQLRepository{placeholder: placeholderDollar}).jsonValuePlaceholder(); got != "?::jsonb" {
		t.Fatalf("postgres json placeholder = %q, want ?::jsonb", got)
	}
	if got := (&SQLRepository{placeholder: placeholderQuestion}).jsonValuePlaceholder(); got != "?" {
		t.Fatalf("sqlite json placeholder = %q, want ?", got)
	}
	if got := (&SQLRepository{placeholder: placeholderDollar}).timestampValuePlaceholder(); got != "?::timestamptz" {
		t.Fatalf("postgres timestamp placeholder = %q, want ?::timestamptz", got)
	}
	if got := (&SQLRepository{placeholder: placeholderQuestion}).timestampValuePlaceholder(); got != "?" {
		t.Fatalf("sqlite timestamp placeholder = %q, want ?", got)
	}
}

func TestPlaceholderRebindPreservesEscapedQuestionMarks(t *testing.T) {
	query := `SELECT data ?? 'key' FROM tasks WHERE id = ? AND note = ??`
	if got, want := placeholderDollar.rebind(query), `SELECT data ? 'key' FROM tasks WHERE id = $1 AND note = ?`; got != want {
		t.Fatalf("rebound query = %q, want %q", got, want)
	}
}

func TestPlaceholderStyleForDialect(t *testing.T) {
	if got, err := placeholderStyleForDialect(SQLDialectPostgres); err != nil || got != placeholderDollar {
		t.Fatalf("postgres placeholder style = %d, %v", got, err)
	}
	if got, err := placeholderStyleForDialect(SQLDialectSQLite); err != nil || got != placeholderQuestion {
		t.Fatalf("sqlite placeholder style = %d, %v", got, err)
	}
	if _, err := placeholderStyleForDialect("unsupported"); err == nil {
		t.Fatal("unsupported dialect succeeded")
	}
}

func TestSQLRepositoryRejectsNonSerialOrDuplicateStagePlan(t *testing.T) {
	t.Parallel()

	repository, _ := newTestSQLRepository(t)
	input := validCreateInput()
	input.Stages[1].Sequence = 3
	if _, _, _, err := repository.Create(context.Background(), input); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid serial plan, got %v", err)
	}
	input = validCreateInput()
	input.Stages[1].Key = input.Stages[0].Key
	if _, _, _, err := repository.Create(context.Background(), input); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected duplicate stage key rejection, got %v", err)
	}
}

func validCreateInput() CreateInput {
	return CreateInput{
		Task: taskmodel.Task{
			Type:     "application.compose.redeploy",
			Owner:    moduleapi.TaskOwner{Type: "application", ID: "app_01ARZ3NDEKTSV4RRFFQ69G5FAV"},
			Status:   moduleapi.TaskStatusPending,
			Input:    []byte(`{"application_id":"app_01ARZ3NDEKTSV4RRFFQ69G5FAV"}`),
			Metadata: []byte(`{"display_name":"demo"}`),
			Plan:     []byte(`{"stages":["prepare","up"]}`),
		},
		Stages: []taskmodel.Stage{
			{Key: "prepare", Sequence: 1, ExecutorType: "application.compose.prepare", Status: moduleapi.StageStatusPending, MaxAttempts: 1, RecoveryPolicy: moduleapi.StageRecoveryManualReconcile},
			{Key: "up", Sequence: 2, ExecutorType: "application.compose.up", Status: moduleapi.StageStatusPending, MaxAttempts: 2, RetryBackoffMS: 1000, RecoveryPolicy: moduleapi.StageRecoveryRetryIfIdempotent},
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
	if err := testschema.CreateSQLite(db); err != nil {
		t.Fatalf("create task store schema: %v", err)
	}
	repository, err := NewSQLRepository(db, SQLDialectSQLite)
	if err != nil {
		t.Fatalf("new task repository: %v", err)
	}
	return repository, db
}
