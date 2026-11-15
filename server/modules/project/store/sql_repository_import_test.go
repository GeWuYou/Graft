package store

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSQLRepositoryUpsertApplicationBindsDeclaredColumns(t *testing.T) {
	t.Parallel()

	columns := strings.Join(strings.Fields(extractBetween(applicationsUpsertSQL(), "INSERT INTO applications (", ") VALUES")), " ")
	const expectedColumns = "application_id, deployment_adapter_kind, application_name, workspace_path, compose_project_name, compose_project_name_source, display_name, runtime_target_id, source_type, ownership_mode, source_metadata_json, lifecycle_strategy_kind, lifecycle_review_status, lifecycle_config_json, last_observed_config_hash, workspace_annotations_json, last_drift_checked_at, drift_status, created_by, updated_by, created_at, updated_at"
	if columns != expectedColumns {
		t.Fatalf("unexpected application insert columns: %s", columns)
	}

	applicationName := "billing-application"
	actorID := uint64(73)
	checkedAt := time.Date(2026, time.November, 15, 10, 0, 0, 0, time.UTC)
	for _, testCase := range []struct {
		name            string
		targetID        uint64
		applicationName *string
		actorID         *uint64
		checkedAt       *time.Time
		expectedTarget  driver.Value
	}{
		{name: "explicit runtime binding", targetID: 42, applicationName: &applicationName, actorID: &actorID, checkedAt: &checkedAt, expectedTarget: int64(42)},
		{name: "optional values absent"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assertApplicationUpsertBindings(t, ImportApplicationInput{
				ApplicationID: "app_01ARZ3NDEKTSV4RRFFQ69G5FAV", DeploymentAdapterKind: "compose",
				ApplicationName: testCase.applicationName, WorkspacePath: "/srv/billing-workspace",
				ComposeProjectName: "billing-compose", ComposeProjectNameSource: "computed",
				DisplayName: "Billing display", RuntimeTargetID: testCase.targetID,
				SourceType: "imported", OwnershipMode: "external", SourceMetadata: map[string]string{"origin": "import"},
				LifecycleStrategyKind: "standard", LifecycleReviewStatus: "review_required",
				LifecycleConfig:        LifecycleConfig{WaitTimeoutSeconds: 120, RemoveOrphans: true},
				LastObservedConfigHash: "billing-config-hash", LastDriftCheckedAt: testCase.checkedAt,
				DriftStatus: "clean", ActorID: testCase.actorID,
			}, testCase.expectedTarget)
		})
	}
}

func TestApplicationsUpsertSQLConflictAssignments(t *testing.T) {
	t.Parallel()

	query := applicationsUpsertSQL()
	assignments := extractBetween(query, "ON CONFLICT (runtime_target_id, compose_project_name) WHERE deleted_at = 0 DO UPDATE SET", "RETURNING application_record_id")
	const expected = `deployment_adapter_kind = excluded.deployment_adapter_kind,
		display_name = excluded.display_name,
		application_name = COALESCE(excluded.application_name, applications.application_name),
		runtime_target_id = excluded.runtime_target_id,
		compose_project_name_source = excluded.compose_project_name_source,
		source_type = excluded.source_type,
		workspace_path = excluded.workspace_path,
		ownership_mode = excluded.ownership_mode,
		source_metadata_json = excluded.source_metadata_json,
		lifecycle_strategy_kind = excluded.lifecycle_strategy_kind,
		lifecycle_review_status = excluded.lifecycle_review_status,
		lifecycle_config_json = excluded.lifecycle_config_json,
		last_observed_config_hash = excluded.last_observed_config_hash,
		last_drift_checked_at = excluded.last_drift_checked_at,
		drift_status = excluded.drift_status,
		updated_by = excluded.updated_by,
		updated_at = excluded.updated_at`
	if strings.Join(strings.Fields(assignments), " ") != strings.Join(strings.Fields(expected), " ") {
		t.Fatalf("unexpected application conflict assignments: %s", assignments)
	}
}

func assertApplicationUpsertBindings(t *testing.T, input ImportApplicationInput, expectedTarget driver.Value) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("create SQL mock: %v", err)
	}
	defer func() { _ = db.Close() }()
	repo, err := NewSQLRepository(db)
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	configJSON, err := encodeLifecycleConfigJSON(input.LifecycleConfig)
	if err != nil {
		t.Fatalf("encode lifecycle configuration: %v", err)
	}
	now := time.Date(2026, time.November, 15, 11, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery(applicationsUpsertSQL()).WithArgs(
		input.ApplicationID, input.DeploymentAdapterKind, input.ApplicationName, input.WorkspacePath,
		input.ComposeProjectName, input.ComposeProjectNameSource, input.DisplayName, expectedTarget,
		input.SourceType, input.OwnershipMode, `{"origin":"import"}`, input.LifecycleStrategyKind,
		input.LifecycleReviewStatus, string(configJSON), input.LastObservedConfigHash, `{}`,
		input.LastDriftCheckedAt, input.DriftStatus, input.ActorID, input.ActorID, now, now,
	).WillReturnRows(sqlmock.NewRows([]string{"application_record_id"}).AddRow(int64(17)))
	mock.ExpectCommit()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin application import: %v", err)
	}
	defer rollbackTx(tx)
	recordID, err := repo.upsertApplication(context.Background(), tx, input, now)
	if err != nil {
		t.Fatalf("upsert application with declared column bindings: %v", err)
	}
	if recordID != 17 {
		t.Fatalf("expected returned record ID 17, got %d", recordID)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit application upsert: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("verify application bindings: %v", err)
	}
}
