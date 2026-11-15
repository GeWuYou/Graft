package store

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestRetrySuccessfulLaunchLeavesFailedTaskUnchanged(t *testing.T) {
	repository, db := newTestSQLRepository(t)
	created, stages, _, err := repository.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE tasks SET status = ?, current_stage_key = ?, failure_code = ?, finished_at = ? WHERE id = ?`, moduleapi.TaskStatusFailed, "up", "controller_receipt_failed", time.Now().UTC(), created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE task_stages SET status = ?, attempt = 1 WHERE id = ?`, moduleapi.StageStatusSuccess, stages[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE task_stages SET status = ?, attempt = 1 WHERE id = ?`, moduleapi.StageStatusFailed, stages[1].ID); err != nil {
		t.Fatal(err)
	}
	beforeTask, err := repository.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	beforeStages, err := repository.ListStages(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.RetryStage(context.Background(), created.ID, stages[0].ID, time.Now().UTC()); !errors.Is(err, ErrStateConflict) {
		t.Fatalf("retry successful launch error = %v, want state conflict", err)
	}
	afterTask, err := repository.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	afterStages, err := repository.ListStages(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeTask, afterTask) || !reflect.DeepEqual(beforeStages, afterStages) {
		t.Fatalf("rejected stage retry changed Task or attempts: task status before=%s after=%s", beforeTask.Status, afterTask.Status)
	}
}

func TestConcurrentStageRetryResumesTaskOnceWithoutChangingAttempt(t *testing.T) {
	for _, taskStatus := range []moduleapi.TaskStatus{moduleapi.TaskStatusFailed, moduleapi.TaskStatusNeedsAttention} {
		for _, stageStatus := range []moduleapi.StageStatus{moduleapi.StageStatusFailed, moduleapi.StageStatusUnknown} {
			t.Run(string(taskStatus)+"/"+string(stageStatus), func(t *testing.T) {
				assertConcurrentStageRetry(t, taskStatus, stageStatus)
			})
		}
	}
}

func assertConcurrentStageRetry(t *testing.T, taskStatus moduleapi.TaskStatus, stageStatus moduleapi.StageStatus) {
	t.Helper()
	repository, db := newTestSQLRepository(t)
	db.SetMaxOpenConns(1)
	created, stages, _, err := repository.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE tasks SET status = ? WHERE id = ?`, taskStatus, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE task_stages SET status = ?, attempt = 1 WHERE id = ?`, stageStatus, stages[0].ID); err != nil {
		t.Fatal(err)
	}
	retryAt := time.Now().UTC()
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, retryErr := repository.RetryStage(context.Background(), created.ID, stages[0].ID, retryAt)
			results <- retryErr
		}()
	}
	var successful int
	for range 2 {
		retryErr := <-results
		if retryErr == nil {
			successful++
		} else if !errors.Is(retryErr, ErrStateConflict) {
			t.Fatalf("concurrent retry error = %v", retryErr)
		}
	}
	if successful != 1 {
		t.Fatalf("successful retries = %d, want one", successful)
	}
	task, err := repository.Get(context.Background(), created.ID)
	if err != nil || task.Status != moduleapi.TaskStatusRunning {
		t.Fatalf("resumed Task status = %s, error = %v", task.Status, err)
	}
	stages, err = repository.ListStages(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stages[0].Status != moduleapi.StageStatusPending || stages[0].Attempt != 1 || !reflect.DeepEqual(stages[0].NextRetryAt, &retryAt) {
		t.Fatalf("resumed Stage = %#v", stages[0])
	}
}
