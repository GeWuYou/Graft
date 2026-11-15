package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"graft/server/internal/moduleapi"
)

func directExecutionPlan() moduleapi.BuildExecutionPlan {
	return moduleapi.BuildExecutionPlan{
		ID: "plan_direct", Digest: "sha256:plan", RuntimeTargetID: 4, BuilderInstanceID: "runtime-target:4",
		Workspace: moduleapi.WorkspaceSnapshot{ID: "snapshot_1", SourceKind: moduleapi.WorkspaceSourceArchive, ContentDigest: "sha256:source", MaterializationRef: "snapshot-materialization://snapshot_1"},
		Driver:    "docker-engine@v1", TemplateRef: "oci-dockerfile/default@v1", CachePolicy: "disabled", SecurityPolicy: "default", Platforms: []string{"linux/amd64"},
		Destination:       moduleapi.BuildDestination{Kind: "oci_registry", ConnectionRef: "registry:primary", RepositoryRef: "team/app", Reference: "latest"},
		BuilderPlacements: []moduleapi.BuilderPlacement{{Platform: "linux/amd64", RuntimeTargetID: 4, BuilderInstanceID: "runtime-target:4", SchedulingPolicy: "manual", SchedulingEvidence: []byte(`{"reservation_slot_budget":1}`)}},
	}
}

func TestMaterializeExecutionPlanAcceptsDirectTargetCapacityBinding(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	plan := directExecutionPlan()
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO build_workspace_snapshots").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec("INSERT INTO build_execution_plans").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs("runtime-target:4").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE build_builder_reservations").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT reservation_id").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"capacity"}).AddRow(0))
	now := time.Now().UTC()
	mock.ExpectQuery("INSERT INTO build_builder_reservations").WillReturnRows(sqlmock.NewRows([]string{"reservation_id", "builder_instance_id", "plan_id", "task_id", "attempt", "leg_id", "fence_token", "state", "lease_expires_at", "created_at", "updated_at"}).AddRow("reservation_plan_direct_linux/amd64", "runtime-target:4", plan.ID, 42, 1, "linux/amd64", BuilderReservationFence(plan.ID, 42, "linux/amd64", 1), "accepted", now.Add(time.Minute), now, now))
	mock.ExpectRollback()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback() }()
	taskID := uint64(42)
	if id, err := repository.MaterializeExecutionPlan(context.Background(), tx, moduleapi.TaskSubmission{TaskID: &taskID}, plan, 7); err != nil || id != plan.ID {
		t.Fatalf("materialize direct target plan = %q, %v", id, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestExecutionPlanBuilderBindingPreservesDirectAndPoolSemantics(t *testing.T) {
	for _, testCase := range []struct {
		pool, instance string
		valid          bool
	}{
		{valid: true},
		{instance: "runtime-target:4", valid: true},
		{instance: "runtime-target:5"},
		{instance: "builder_pool_member"},
		{pool: "pool", instance: "builder_pool_member", valid: true},
		{pool: "pool"},
	} {
		plan := directExecutionPlan()
		plan.BuilderPoolID, plan.BuilderInstanceID = testCase.pool, testCase.instance
		if validExecutionPlan(plan) != testCase.valid {
			t.Fatalf("unexpected builder binding verdict for pool=%q instance=%q", testCase.pool, testCase.instance)
		}
	}
}
