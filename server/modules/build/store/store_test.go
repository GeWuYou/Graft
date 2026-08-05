package store

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"graft/server/internal/moduleapi"
)

func TestJobListFiltersPreserveExactFilterArgumentOrder(t *testing.T) {
	repository := "example/app"
	tag := "v1"
	applicationID := "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X"
	after := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	before := after.Add(24 * time.Hour)

	where, args := jobListFilters(ListQuery{ApplicationID: &applicationID, ImageRepository: &repository, ImageTag: &tag, CreatedAfter: &after, CreatedBefore: &before})
	wantWhere := []string{"1 = 1", "j.application_id = $1", "j.image_repository = $2", "j.image_tag = $3", "j.created_at >= $4", "j.created_at <= $5"}
	if !reflect.DeepEqual(where, wantWhere) {
		t.Fatalf("unexpected where clauses: %#v", where)
	}
	wantArgs := []any{applicationID, repository, tag, after, before}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("unexpected filter arguments: %#v", args)
	}
}

func TestSQLRepositoryRejectsUnavailableReadOperations(t *testing.T) {
	var repository *SQLRepository
	if _, err := repository.GetJobByTaskID(context.Background(), 1); err == nil {
		t.Fatal("nil repository GetJobByTaskID succeeded")
	}
	if _, err := repository.GetJobByBuildID(context.Background(), "build_test"); err == nil {
		t.Fatal("nil repository GetJobByBuildID succeeded")
	}
}

func TestSettleDockerArtifactReturnsNotFoundWhenJobDoesNotExist(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("INSERT INTO build_artifacts").WithArgs(uint64(42), "sha256:image", "", "example/app", "v1", int64(0), "", "", "").WillReturnResult(sqlmock.NewResult(0, 0))
	if err := repository.SettleDockerArtifact(context.Background(), 42, moduleapi.DockerImageBuildResult{ImageID: "sha256:image", Repository: "example/app", Tag: "v1"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("SettleDockerArtifact error = %v, want ErrNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetJobByBuildIDLoadsPersistedBuildArgs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	columns := []string{"build_id", "task_id", "application_id", "application_record_id", "application_name_snapshot", "workspace_context_path", "workspace_root", "dockerfile_path", "runtime_target_id", "runtime_provider", "image_repository", "image_tag", "created_by", "created_at", "artifact_id", "image_id", "digest", "repository", "tag", "size_bytes", "platform"}
	mock.ExpectQuery("SELECT j.build_id").WithArgs("build_test").WillReturnRows(sqlmock.NewRows(columns).AddRow("build_test", uint64(42), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "docker", "example/app", "v1", uint64(7), time.Now(), nil, nil, "", nil, nil, nil, ""))
	mock.ExpectQuery("SELECT name, value FROM build_job_args").WithArgs(uint64(42)).WillReturnRows(sqlmock.NewRows([]string{"name", "value"}).AddRow("MODE", "release"))

	job, err := repository.GetJobByBuildID(context.Background(), "build_test")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(job.BuildArgs, []moduleapi.DockerImageBuildArg{{Name: "MODE", Value: "release"}}) {
		t.Fatalf("build args = %#v", job.BuildArgs)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateJobVerifiesConflictReplayWithinTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := JobSnapshot{BuildID: "build_test", TaskID: 42, ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ApplicationRecordID: 9, ApplicationName: "app", WorkspaceRoot: "/workspace/app", ContextPath: "src", DockerfilePath: "Dockerfile", RuntimeTargetID: 4, RuntimeProvider: "docker", ImageRepository: "example/app", ImageTag: "v1"}
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO build_jobs").WithArgs("build_test", uint64(42), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "docker", "dockerfile", "example/app", "v1", uint64(0)).WillReturnRows(sqlmock.NewRows([]string{"id", "xmax = 0"}).AddRow(uint64(42), false))
	mock.ExpectQuery("SELECT build_id, task_id").WithArgs(uint64(42)).WillReturnRows(sqlmock.NewRows([]string{"build_id", "task_id", "application_id", "application_record_id", "application_name_snapshot", "workspace_context_path", "workspace_root", "dockerfile_path", "runtime_target_id", "runtime_provider", "image_repository", "image_tag", "created_by"}).AddRow("build_test", uint64(42), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "docker", "example/app", "v1", uint64(0)))
	mock.ExpectQuery("SELECT name, value FROM build_job_args").WithArgs(uint64(42)).WillReturnRows(sqlmock.NewRows([]string{"name", "value"}))
	mock.ExpectRollback()

	if err := repository.CreateJob(context.Background(), snapshot); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateJobRetriesAfterConcurrentConflictRollsBack(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := JobSnapshot{BuildID: "build_retry", TaskID: 43, ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ApplicationRecordID: 9, ApplicationName: "app", WorkspaceRoot: "/workspace/app", ContextPath: "src", DockerfilePath: "Dockerfile", RuntimeTargetID: 4, RuntimeProvider: "docker", ImageRepository: "example/app", ImageTag: "v1"}
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO build_jobs").WithArgs("build_retry", uint64(43), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "docker", "dockerfile", "example/app", "v1", uint64(0)).WillReturnRows(sqlmock.NewRows([]string{"id", "xmax = 0"}).AddRow(uint64(43), false))
	mock.ExpectQuery("SELECT build_id, task_id").WithArgs(uint64(43)).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO build_jobs").WithArgs("build_retry", uint64(43), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "docker", "dockerfile", "example/app", "v1", uint64(0)).WillReturnRows(sqlmock.NewRows([]string{"id", "xmax = 0"}).AddRow(uint64(7), true))
	mock.ExpectCommit()

	if err := repository.CreateJob(context.Background(), snapshot); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
