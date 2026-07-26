package backup

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestBackupRecordTaskExecutorVerifiesFrozenArtifactsAndAttachesTask(t *testing.T) {
	input, err := json.Marshal(testBackupTaskInput())
	if err != nil {
		t.Fatalf("marshal task input: %v", err)
	}
	writer := &taskPlanWriter{verified: moduleapi.CreateBackupInput{Purpose: backupTaskPurpose, ConfigSnapshot: moduleapi.BackupArtifact{StorageRef: "config", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, DatabaseDump: moduleapi.BackupArtifact{StorageRef: "dump", SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}, RetainUntil: time.Now().UTC().Add(time.Hour)}}
	repository := &serviceTestRepository{}
	executor := backupRecordTaskExecutor{service: NewService(repository)}
	executor.service.setArtifactWriter(writer)
	if err := executor.Execute(context.Background(), &backupStageRun{taskID: 88, input: input}); err != nil {
		t.Fatalf("record backup artifacts: %v", err)
	}
	if writer.createCalls != 0 || writer.verifyCalls != 1 || writer.verifyCtx == nil || repository.created.TaskID != 88 {
		t.Fatalf("record stage must only verify and attach task: writer=%#v input=%#v", writer, repository.created)
	}
}

func TestBackupArtifactExecutionErrorHidesArtifactReference(t *testing.T) {
	artifactError := errors.New("open backup artifact: /var/lib/graft/backups/backup-42/database.dump: permission denied")
	if got := backupArtifactExecutionError(artifactError); !errors.Is(got, errBackupArtifactsUnavailable) || got.Error() != "backup artifacts unavailable" {
		t.Fatalf("expected sanitized artifact failure, got %v", got)
	}
	if got := backupArtifactExecutionError(context.Canceled); !errors.Is(got, context.Canceled) {
		t.Fatalf("expected cancellation to propagate, got %v", got)
	}
}

type taskPlanWriter struct {
	verified    moduleapi.CreateBackupInput
	createCalls int
	verifyCalls int
	verifyCtx   context.Context
}

func (w *taskPlanWriter) Create(context.Context, backupTaskInput) (moduleapi.CreateBackupInput, error) {
	w.createCalls++
	return w.verified, nil
}

func (w *taskPlanWriter) Verify(ctx context.Context, _ backupTaskInput) (moduleapi.CreateBackupInput, error) {
	w.verifyCalls++
	w.verifyCtx = ctx
	return w.verified, nil
}

type backupStageRun struct {
	taskID uint64
	input  json.RawMessage
}

func (r *backupStageRun) TaskID() uint64                                        { return r.taskID }
func (*backupStageRun) StageID() uint64                                         { return 1 }
func (*backupStageRun) Attempt() int                                            { return 1 }
func (r *backupStageRun) Input() json.RawMessage                                { return r.input }
func (*backupStageRun) CancellationRequested(context.Context) bool              { return false }
func (*backupStageRun) AppendLog(context.Context, moduleapi.TaskLogEntry) error { return nil }
