package store

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"graft/server/internal/moduleapi"
)

func TestCreateBuildInputSnapshotReusesDigestAndGrantsAccessWithoutChangingOwner(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	createdAt := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO build_workspace_snapshots").WithArgs("snapshot-new", moduleapi.WorkspaceSourceArchive, "upload:source", "sha256:digest", "build://materialized", uint64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"snapshot_id", "source_kind", "source_reference", "content_digest", "materialization_ref", "created_at"}).
			AddRow("snapshot-original", moduleapi.WorkspaceSourceArchive, "upload:original", "sha256:digest", "build://original", createdAt))
	mock.ExpectExec("INSERT INTO build_workspace_snapshot_access").WithArgs("snapshot-original", uint64(7)).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	got, err := repository.CreateBuildInputSnapshot(context.Background(), "snapshot-new", "upload:source", "sha256:digest", "build://materialized", 7)
	if err != nil {
		t.Fatalf("create reused snapshot: %v", err)
	}
	if got.ID != "snapshot-original" || got.ContentDigest != "sha256:digest" || got.CreatedAt != createdAt {
		t.Fatalf("reused snapshot = %#v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

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

func TestBuilderReservationFenceChangesForRetryAttempt(t *testing.T) {
	first := BuilderReservationFence("plan_1", 42, "single", 1)
	retry := BuilderReservationFence("plan_1", 42, "single", 2)
	if first == retry {
		t.Fatal("retry reservation fence must differ from the first attempt")
	}
	if first != BuilderReservationFence("plan_1", 42, "single", 1) {
		t.Fatal("reservation fence must be deterministic for one attempt")
	}
	if first == BuilderReservationFence("plan_1", 42, "linux/arm64", 1) {
		t.Fatal("reservation fence must differ for independent platform legs")
	}
}

func TestPlacementReservationCapacityRequiresDynamicObservationTime(t *testing.T) {
	observedAt := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	placement := moduleapi.BuilderPlacement{SchedulingPolicy: "capacity", SchedulingEvidence: []byte(`{"reservation_slot_budget":4,"reservation_observed_at":"2026-08-08T12:00:00Z"}`)}
	slotBudget, frozenObservedAt, err := placementReservationCapacity(placement)
	if err != nil || slotBudget != 4 || !frozenObservedAt.Equal(observedAt) {
		t.Fatalf("dynamic reservation capacity = (%d,%s,%v)", slotBudget, frozenObservedAt, err)
	}
	placement.SchedulingEvidence = []byte(`{"reservation_slot_budget":4}`)
	if _, _, err := placementReservationCapacity(placement); err == nil {
		t.Fatal("dynamic reservation without observation time unexpectedly passed")
	}
}

func TestReserveBuilderExpiresAcceptedLeaseBeforeAcquiringSameInstance(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Date(2026, time.August, 7, 12, 5, 0, 0, time.UTC)
	reservation := moduleapi.BuilderReservation{ID: "reservation_plan_1", InstanceID: "builder_1", PlanID: "plan_1", TaskID: 42, Attempt: 1, LegID: "single", FenceToken: BuilderReservationFence("plan_1", 42, "single", 1), State: moduleapi.BuilderReservationAccepted, LeaseExpiresAt: expiresAt}
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(reservation.InstanceID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE build_builder_reservations SET state = 'expired'").WithArgs("builder_1").WillReturnResult(sqlmock.NewResult(0, 1))
	expectBuilderReservationCapacity(mock, reservation, expiresAt, 1)
	stored, err := repository.ReserveBuilder(context.Background(), tx, reservation)
	if err != nil {
		t.Fatalf("reserve builder: %v", err)
	}
	if stored.ID != reservation.ID || stored.LeaseExpiresAt != expiresAt {
		t.Fatalf("stored reservation = %#v", stored)
	}
	mock.ExpectCommit()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReserveBuilderCapacityRejectsUnitBeyondFrozenSlotBudget(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	reservation := moduleapi.BuilderReservation{ID: "reservation_plan_2", InstanceID: "builder_1", PlanID: "plan_2", TaskID: 43, Attempt: 1, LegID: "single", FenceToken: BuilderReservationFence("plan_2", 43, "single", 1), State: moduleapi.BuilderReservationAccepted, LeaseExpiresAt: time.Now().UTC().Add(time.Minute)}
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(reservation.InstanceID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE build_builder_reservations SET state = 'expired'").WithArgs(reservation.InstanceID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT reservation_id, builder_instance_id, plan_id, task_id, attempt, leg_id, fence_token, state").WithArgs(reservation.PlanID, reservation.TaskID, reservation.Attempt, reservation.LegID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(capacity_units\\), 0\\)").WithArgs(reservation.InstanceID, nil).
		WillReturnRows(sqlmock.NewRows([]string{"used_units"}).AddRow(4))
	if _, err := repository.reserveBuilderCapacity(context.Background(), tx, reservation, 4, time.Time{}); !errors.Is(err, ErrConflict) {
		t.Fatalf("reserve builder capacity error = %v, want conflict", err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReserveBuilderCapacityReplaysMatchingReservationBeforeCapacityCheck(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Date(2026, time.August, 8, 12, 5, 0, 0, time.UTC)
	reservation := moduleapi.BuilderReservation{ID: "reservation_plan_replay", InstanceID: "builder_1", PlanID: "plan_replay", TaskID: 43, Attempt: 1, LegID: "single", FenceToken: BuilderReservationFence("plan_replay", 43, "single", 1), State: moduleapi.BuilderReservationAccepted, LeaseExpiresAt: expiresAt}
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(reservation.InstanceID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE build_builder_reservations SET state = 'expired'").WithArgs(reservation.InstanceID).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT reservation_id, builder_instance_id, plan_id, task_id, attempt, leg_id, fence_token, state").WithArgs(reservation.PlanID, reservation.TaskID, reservation.Attempt, reservation.LegID).
		WillReturnRows(sqlmock.NewRows([]string{"reservation_id", "builder_instance_id", "plan_id", "task_id", "attempt", "leg_id", "fence_token", "state", "lease_expires_at", "created_at", "updated_at"}).
			AddRow(reservation.ID, reservation.InstanceID, reservation.PlanID, reservation.TaskID, reservation.Attempt, reservation.LegID, reservation.FenceToken, reservation.State, expiresAt, expiresAt.Add(-time.Minute), expiresAt.Add(-time.Minute)))
	stored, err := repository.ReserveBuilder(context.Background(), tx, reservation)
	if err != nil || stored.ID != reservation.ID {
		t.Fatalf("reserve replay = %#v, %v", stored, err)
	}
	mock.ExpectCommit()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRenewBuilderReservationRequiresMatchingRunningFenceAndLeg(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Now().UTC().Add(time.Minute)
	mock.ExpectExec("UPDATE build_builder_reservations SET lease_expires_at").WithArgs(uint64(42), "linux/amd64", "fence-amd64", expiresAt).WillReturnResult(sqlmock.NewResult(0, 1))
	if err := repository.RenewBuilderReservation(context.Background(), 42, "linux/amd64", "fence-amd64", expiresAt); err != nil {
		t.Fatalf("renew builder reservation: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReserveBuilderAttemptAbandonsOnlyOlderAttempts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Date(2026, time.August, 7, 12, 5, 0, 0, time.UTC)
	reservation := moduleapi.BuilderReservation{ID: "reservation_plan_1_3", InstanceID: "builder_1", PlanID: "plan_1", TaskID: 42, Attempt: 3, LegID: "linux/amd64", FenceToken: BuilderReservationFence("plan_1", 42, "linux/amd64", 3), State: moduleapi.BuilderReservationRunning, LeaseExpiresAt: expiresAt}
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE build_builder_reservations SET state = 'abandoned'").WithArgs(uint64(42), "linux/amd64", 3).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("SELECT pg_advisory_xact_lock").WithArgs(reservation.InstanceID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE build_builder_reservations SET state = 'expired'").WithArgs("builder_1").WillReturnResult(sqlmock.NewResult(0, 0))
	expectBuilderReservationCapacity(mock, reservation, expiresAt, 1)
	mock.ExpectCommit()
	if _, err := repository.ReserveBuilderAttempt(context.Background(), reservation); err != nil {
		t.Fatalf("reserve builder retry: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func expectBuilderReservationCapacity(mock sqlmock.Sqlmock, reservation moduleapi.BuilderReservation, timestamp time.Time, slotBudget int) {
	mock.ExpectQuery("SELECT reservation_id, builder_instance_id, plan_id, task_id, attempt, leg_id, fence_token, state").WithArgs(reservation.PlanID, reservation.TaskID, reservation.Attempt, reservation.LegID).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(capacity_units\\), 0\\)").WithArgs(reservation.InstanceID, nil).
		WillReturnRows(sqlmock.NewRows([]string{"used_units"}).AddRow(0))
	mock.ExpectQuery("INSERT INTO build_builder_reservations").
		WithArgs(reservation.ID, reservation.InstanceID, reservation.PlanID, reservation.TaskID, reservation.Attempt, reservation.LegID, reservation.FenceToken, reservation.State, reservation.LeaseExpiresAt, slotBudget).
		WillReturnRows(sqlmock.NewRows([]string{"reservation_id", "builder_instance_id", "plan_id", "task_id", "attempt", "leg_id", "fence_token", "state", "lease_expires_at", "created_at", "updated_at"}).
			AddRow(reservation.ID, reservation.InstanceID, reservation.PlanID, reservation.TaskID, reservation.Attempt, reservation.LegID, reservation.FenceToken, reservation.State, timestamp, timestamp.Add(-time.Minute), timestamp.Add(-time.Minute)))
}

func TestListV2ArtifactsReturnsDigestAddressedProjection(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	createdAt := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM build_v2_artifacts").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT artifact_id, artifact_digest").WithArgs(20, 0).WillReturnRows(sqlmock.NewRows([]string{"artifact_id", "artifact_digest", "media_type", "platforms_json", "size_bytes", "created_at"}).AddRow("artifact_1", "sha256:abc", "application/vnd.oci.image.manifest.v1+json", `["linux/amd64"]`, int64(12), createdAt))
	result, err := repository.ListV2Artifacts(context.Background(), 20, 0)
	if err != nil {
		t.Fatalf("ListV2Artifacts() error = %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].Digest != "sha256:abc" || !reflect.DeepEqual(result.Items[0].Platforms, []string{"linux/amd64"}) {
		t.Fatalf("unexpected artifact result: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListArtifactPublicationSourcesUsesArtifactIdentityAndNeverReturnsMutableTag(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT artifact.artifact_id, publication.publication_id").WithArgs("artifact_1").WillReturnRows(sqlmock.NewRows([]string{"artifact_id", "publication_id", "artifact_digest", "media_type", "destination_kind", "connection_ref", "repository_ref"}).AddRow("artifact_1", "publication_1", "sha256:abc", "application/vnd.oci.image.manifest.v1+json", "oci_registry", "registry:primary", "team/api"))
	items, err := repository.ListArtifactPublicationSources(context.Background(), "artifact_1")
	if err != nil {
		t.Fatalf("ListArtifactPublicationSources() error = %v", err)
	}
	if len(items) != 1 || items[0].PublicationID != "publication_1" || items[0].Digest != "sha256:abc" || items[0].RepositoryRef != "team/api" {
		t.Fatalf("unexpected publication sources: %#v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListArtifactPublicationsReturnsEmptyListWhenArtifactHasNoPublication(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT publication.publication_id, artifact.artifact_id").WithArgs("artifact_1").WillReturnRows(
		sqlmock.NewRows([]string{"publication_id", "artifact_id", "artifact_digest", "media_type", "destination_kind", "connection_ref", "repository_ref", "mutable_reference", "credential_execution_mode", "created_at"}).
			AddRow(nil, "artifact_1", "sha256:abc", "application/vnd.oci.image.manifest.v1+json", nil, nil, nil, nil, nil, nil),
	)
	items, err := repository.ListArtifactPublications(context.Background(), "artifact_1")
	if err != nil {
		t.Fatalf("ListArtifactPublications() error = %v", err)
	}
	if items == nil || len(items) != 0 {
		t.Fatalf("ListArtifactPublications() = %#v, want non-nil empty list", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListArtifactPublicationsReturnsNotFoundWhenArtifactDoesNotExist(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT publication.publication_id, artifact.artifact_id").WithArgs("missing").WillReturnRows(
		sqlmock.NewRows([]string{"publication_id", "artifact_id", "artifact_digest", "media_type", "destination_kind", "connection_ref", "repository_ref", "mutable_reference", "credential_execution_mode", "created_at"}),
	)
	items, err := repository.ListArtifactPublications(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListArtifactPublications() error = %v, want ErrNotFound", err)
	}
	if items != nil {
		t.Fatalf("ListArtifactPublications() items = %#v, want nil on missing artifact", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSettleArtifactPromotionRequiresMatchingImmutableArtifactAndPreservesDestinationHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	digest := "sha256:" + strings.Repeat("a", 64)
	input := moduleapi.OCIArtifactCopyInput{
		Source: moduleapi.ArtifactPublicationSource{
			ArtifactID: "artifact_1", PublicationID: "publication_source", Digest: digest,
			MediaType: "application/vnd.oci.image.manifest.v1+json", DestinationKind: "oci_registry",
			ConnectionRef: "registry:source", RepositoryRef: "team/api",
		},
		Destination: moduleapi.AuthorizedArtifactDestination{Kind: "oci_registry", ConnectionRef: "registry:target", RepositoryRef: "team/api", Reference: "promoted"},
	}
	result := moduleapi.OCIArtifactCopyResult{Digest: digest, MediaType: input.Source.MediaType, SizeBytes: 42}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM build_v2_artifacts WHERE artifact_id = \\$1 AND artifact_digest = \\$2").WithArgs("artifact_1", digest).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7)))
	mock.ExpectExec("INSERT INTO build_publications").WithArgs(sqlmock.AnyArg(), int64(7), "oci_registry", "registry:target", "team/api", "promoted", "ephemeral-credential").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	if err := repository.SettleArtifactPromotion(context.Background(), input, result, moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}); err != nil {
		t.Fatalf("SettleArtifactPromotion() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSettleArtifactPromotionRejectsProviderDigestMismatch(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	digest := "sha256:" + strings.Repeat("a", 64)
	input := moduleapi.OCIArtifactCopyInput{Source: moduleapi.ArtifactPublicationSource{ArtifactID: "artifact_1", Digest: digest, MediaType: "application/vnd.oci.image.manifest.v1+json", DestinationKind: "oci_registry"}, Destination: moduleapi.AuthorizedArtifactDestination{Kind: "oci_registry", ConnectionRef: "registry:target", RepositoryRef: "team/api", Reference: "promoted"}}
	result := moduleapi.OCIArtifactCopyResult{Digest: "sha256:" + strings.Repeat("b", 64), MediaType: input.Source.MediaType}
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.SettleArtifactPromotion(context.Background(), input, result, moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}); !errors.Is(err, ErrConflict) {
		t.Fatalf("SettleArtifactPromotion() error = %v, want %v", err, ErrConflict)
	}
}

func TestJobListFiltersMapProductStatusGroupsToTaskStatuses(t *testing.T) {
	cases := []struct {
		name   string
		filter StatusFilter
		want   []any
	}{
		{name: "queued", filter: StatusFilterQueued, want: []any{moduleapi.TaskStatusPending, moduleapi.TaskStatusReady, moduleapi.TaskStatusScheduled}},
		{name: "running", filter: StatusFilterRunning, want: []any{moduleapi.TaskStatusRunning}},
		{name: "success", filter: StatusFilterSuccess, want: []any{moduleapi.TaskStatusSuccess}},
		{name: "failed", filter: StatusFilterFailed, want: []any{moduleapi.TaskStatusFailed, moduleapi.TaskStatusNeedsAttention}},
		{name: "cancelled", filter: StatusFilterCancelled, want: []any{moduleapi.TaskStatusCancelled}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			where, args := jobListFilters(ListQuery{BuildStatus: &testCase.filter})
			if got := where[len(where)-1]; !strings.HasPrefix(got, `EXISTS (SELECT 1 FROM tasks t WHERE t.id = j.task_id AND t.status IN ($1`) {
				t.Fatalf("status where clause = %q, want Task status projection", got)
			}
			if !reflect.DeepEqual(args, testCase.want) {
				t.Fatalf("status arguments = %#v, want %#v", args, testCase.want)
			}
		})
	}
}

func TestListJobsUsesStatusProjectionForExactTotalAndOffset(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	status := StatusFilterFailed
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM build_jobs j WHERE 1 = 1 AND EXISTS").WithArgs(moduleapi.TaskStatusFailed, moduleapi.TaskStatusNeedsAttention).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	columns := []string{"build_id", "task_id", "application_id", "application_record_id", "application_name_snapshot", "workspace_context_path", "workspace_root", "dockerfile_path", "runtime_target_id", "runtime_target_name", "runtime_provider", "image_repository", "image_tag", "created_by", "created_at", "artifact_id", "image_id", "digest", "repository", "tag", "size_bytes", "platform"}
	mock.ExpectQuery("SELECT j.build_id").WithArgs(moduleapi.TaskStatusFailed, moduleapi.TaskStatusNeedsAttention, 1, 1).WillReturnRows(sqlmock.NewRows(columns).AddRow("build_failed", uint64(42), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "Local Docker", "docker", "example/app", "v1", uint64(7), time.Now(), nil, nil, "", nil, nil, nil, ""))

	result, err := repository.ListJobs(context.Background(), ListQuery{Limit: 1, Offset: 1, BuildStatus: &status})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 3 || len(result.Items) != 1 || result.Items[0].BuildID != "build_failed" {
		t.Fatalf("status page = %#v, want exact total and requested offset page", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListWorkspacesScopesToCallerAndSharedSources(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM build_workspaces").WithArgs(moduleapi.WorkspaceSourceApplication, uint64(7), "app").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery("SELECT workspace_id, display_name, source_kind, source_reference").WithArgs(moduleapi.WorkspaceSourceApplication, uint64(7), "app", 20, 20).WillReturnRows(
		sqlmock.NewRows([]string{"workspace_id", "display_name", "source_kind", "source_reference", "retention_policy", "created_by", "created_at", "updated_at"}).
			AddRow("workspace_shared", "Shared", moduleapi.WorkspaceSourceApplication, "app_shared", "workspace", nil, time.Now(), time.Now()).
			AddRow("workspace_owned", "Owned", moduleapi.WorkspaceSourceApplication, "app_owned", "workspace", uint64(7), time.Now(), time.Now()),
	)
	search := "app"
	result, err := repository.ListWorkspaces(context.Background(), 7, WorkspaceListQuery{Limit: 20, Offset: 20, Search: &search})
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || len(result.Items) != 2 || result.Items[0].ID != "workspace_shared" || result.Items[1].CreatedBy != 7 {
		t.Fatalf("unexpected workspace selector projection: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBuildWorkspaceListFilterRestrictsToExecutableApplicationSources(t *testing.T) {
	where, args, err := buildWorkspaceListFilter(7, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(where, "source_kind = $1") {
		t.Fatalf("workspace filter = %q, want executable source restriction", where)
	}
	if len(args) != 2 || args[0] != moduleapi.WorkspaceSourceApplication || args[1] != uint64(7) {
		t.Fatalf("workspace filter args = %#v", args)
	}
}

func TestListWorkspacesEnforcesUnicodeSearchLimitBeforeQuery(t *testing.T) {
	accepted := strings.Repeat("界", 255)
	_, args, err := buildWorkspaceListFilter(7, &accepted)
	if err != nil {
		t.Fatalf("255-rune workspace query: %v", err)
	}
	if len(args) != 3 || args[0] != moduleapi.WorkspaceSourceApplication || args[1] != uint64(7) || args[2] != accepted {
		t.Fatalf("255-rune workspace query arguments = %#v", args)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	rejected := strings.Repeat("界", 256)
	if _, err := repository.ListWorkspaces(context.Background(), 7, WorkspaceListQuery{Search: &rejected}); err == nil || err.Error() != "invalid build workspace query" {
		t.Fatalf("256-rune workspace query error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("over-limit workspace query reached SQL: %v", err)
	}
}

func TestListBuilderPoolsReturnsOnlyLivePoolProjection(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT pool_id, display_name, scheduling_policy, selector_json").WillReturnRows(
		sqlmock.NewRows([]string{"pool_id", "display_name", "scheduling_policy", "selector_json"}).
			AddRow("pool:default", "Default", "round_robin", []byte(`{"labels":{}}`)),
	)
	items, err := repository.ListBuilderPools(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "pool:default" || items[0].SchedulingPolicy != "round_robin" {
		t.Fatalf("unexpected pool selector projection: %#v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
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

func TestSettleBuildArtifactReturnsNotFoundWhenJobDoesNotExist(t *testing.T) {
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
	if err := repository.SettleBuildArtifact(context.Background(), 42, moduleapi.BuildArtifactResult{ImageID: "sha256:image", Repository: "example/app", Tag: "v1"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("SettleBuildArtifact error = %v, want ErrNotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListPlatformArtifactsUsesFrozenPlanAndTaskIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	createdAt := time.Date(2026, time.August, 6, 0, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT link.leg_id").WithArgs("plan_123", uint64(42)).WillReturnRows(sqlmock.NewRows([]string{"leg_id", "platform", "artifact_digest", "media_type", "size_bytes", "created_at"}).AddRow("amd64", "linux/amd64", "sha256:amd64", "application/vnd.oci.image.manifest.v1+json", int64(12), createdAt))
	items, err := repository.ListPlatformArtifacts(context.Background(), 42, moduleapi.BuildExecutionPlan{ID: "plan_123", Platforms: []string{"linux/amd64"}})
	if err != nil {
		t.Fatalf("ListPlatformArtifacts: %v", err)
	}
	if len(items) != 1 || items[0].LegID != "amd64" || items[0].Digest != "sha256:amd64" {
		t.Fatalf("platform artifacts = %#v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareOCIManifestPublicationRejectsIncompletePlatformArtifacts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT link.leg_id").WithArgs("plan_123", uint64(42)).WillReturnRows(sqlmock.NewRows([]string{"leg_id", "platform", "artifact_digest", "media_type", "size_bytes", "created_at"}).AddRow("amd64", "linux/amd64", "sha256:amd64", "application/vnd.oci.image.manifest.v1+json", int64(12), time.Now()))
	_, err = repository.PrepareOCIManifestPublication(context.Background(), 42, moduleapi.BuildExecutionPlan{ID: "plan_123", Platforms: []string{"linux/amd64", "linux/arm64"}})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("PrepareOCIManifestPublication error = %v, want %v", err, ErrConflict)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetExecutionPlanByTaskIDReadsFrozenPlatformPlacements(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	placements := `[{
        "Platform":"linux/amd64","BuilderInstanceID":"builder-amd64","RuntimeTargetID":4,"SchedulingPolicy":"round_robin"
      },{
        "Platform":"linux/arm64","BuilderInstanceID":"builder-arm64","RuntimeTargetID":5,"SchedulingPolicy":"round_robin"
      }]`
	columns := []string{"plan_id", "plan_digest", "snapshot_id", "source_kind", "source_reference", "content_digest", "materialization_ref", "snapshot_created_at", "builder_pool_id", "builder_instance_id", "runtime_target_id", "driver", "template_ref", "cache_policy", "security_policy", "platforms_json", "builder_placements_json", "destination_json", "created_at"}
	mock.ExpectQuery("SELECT p.plan_id").WithArgs(uint64(42)).WillReturnRows(sqlmock.NewRows(columns).AddRow("plan_test", "sha256:plan", "snapshot_test", "application_workspace", "app_test", "sha256:source", "/managed/snapshot", time.Now(), "pool_builders", "builder-amd64", int64(4), "docker-buildx@v1", "oci-dockerfile/default@v1", "disabled", "default", `["linux/amd64","linux/arm64"]`, placements, `{"kind":"oci_registry","connection_ref":"registry","repository_ref":"team/app","reference":"v1"}`, time.Now()))
	plan, err := repository.GetExecutionPlanByTaskID(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if placement, ok := plan.PlacementForPlatform("linux/arm64"); !ok || placement.BuilderInstanceID != "builder-arm64" || placement.RuntimeTargetID != 5 {
		t.Fatalf("placement = %#v ok=%t", placement, ok)
	}
	if plan.CachePolicy != "disabled" || plan.SecurityPolicy != "default" {
		t.Fatalf("resolved policies = %#v", plan)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestClaimExpiredSnapshotMaterializationsUsesRecoverableLease(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 6, 12, 0, 0, 0, time.UTC)
	claimBefore := now.Add(-10 * time.Minute)
	mock.ExpectQuery("WITH candidates AS").WithArgs(now, claimBefore, 10).WillReturnRows(sqlmock.NewRows([]string{"snapshot_id", "materialization_ref"}).AddRow("snapshot_expired", "/tmp/graft-build-snapshots/snapshot-a"))
	items, err := repository.ClaimExpiredSnapshotMaterializations(context.Background(), now, claimBefore, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].SnapshotID != "snapshot_expired" || items[0].MaterializationRef != "/tmp/graft-build-snapshots/snapshot-a" {
		t.Fatalf("claimed materializations = %#v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGetJobByBuildIDReturnsNoPersistedBuildArgs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	columns := []string{"build_id", "task_id", "application_id", "application_record_id", "application_name_snapshot", "workspace_context_path", "workspace_root", "dockerfile_path", "runtime_target_id", "runtime_target_name", "runtime_provider", "image_repository", "image_tag", "created_by", "created_at", "artifact_id", "image_id", "digest", "repository", "tag", "size_bytes", "platform"}
	mock.ExpectQuery("SELECT j.build_id").WithArgs("build_test").WillReturnRows(sqlmock.NewRows(columns).AddRow("build_test", uint64(42), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "Local Docker", "docker", "example/app", "v1", uint64(7), time.Now(), nil, nil, "", nil, nil, nil, ""))
	mock.ExpectQuery("SELECT name, value FROM build_job_args").WithArgs(uint64(42)).WillReturnRows(sqlmock.NewRows([]string{"name", "value"}))

	job, err := repository.GetJobByBuildID(context.Background(), "build_test")
	if err != nil {
		t.Fatal(err)
	}
	if len(job.BuildArgs) != 0 {
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
	mock.ExpectQuery("INSERT INTO build_jobs").WithArgs("build_test", uint64(42), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "", "docker", "dockerfile", "example/app", "v1", uint64(0)).WillReturnRows(sqlmock.NewRows([]string{"id", "xmax = 0"}).AddRow(uint64(42), false))
	mock.ExpectQuery("SELECT build_id, task_id").WithArgs(uint64(42)).WillReturnRows(sqlmock.NewRows([]string{"build_id", "task_id", "application_id", "application_record_id", "application_name_snapshot", "workspace_context_path", "workspace_root", "dockerfile_path", "runtime_target_id", "runtime_target_name", "runtime_provider", "image_repository", "image_tag", "created_by"}).AddRow("build_test", uint64(42), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "", "docker", "example/app", "v1", uint64(0)))
	mock.ExpectQuery("SELECT name, value FROM build_job_args").WithArgs(uint64(42)).WillReturnRows(sqlmock.NewRows([]string{"name", "value"}))
	mock.ExpectRollback()

	if err := repository.CreateJob(context.Background(), snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.BuildID != "build_test" || snapshot.TaskID != 42 {
		t.Fatalf("snapshot changed during replay: %#v", snapshot)
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
	mock.ExpectQuery("INSERT INTO build_jobs").WithArgs("build_retry", uint64(43), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "", "docker", "dockerfile", "example/app", "v1", uint64(0)).WillReturnRows(sqlmock.NewRows([]string{"id", "xmax = 0"}).AddRow(uint64(43), false))
	mock.ExpectQuery("SELECT build_id, task_id").WithArgs(uint64(43)).WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO build_jobs").WithArgs("build_retry", uint64(43), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "", "docker", "dockerfile", "example/app", "v1", uint64(0)).WillReturnRows(sqlmock.NewRows([]string{"id", "xmax = 0"}).AddRow(uint64(7), true))
	mock.ExpectCommit()

	if err := repository.CreateJob(context.Background(), snapshot); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateJobReturnsConflictAfterRetryBudgetIsExhausted(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := JobSnapshot{BuildID: "build_retry_exhausted", TaskID: 45, ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ApplicationRecordID: 9, ApplicationName: "app", WorkspaceRoot: "/workspace/app", ContextPath: "src", DockerfilePath: "Dockerfile", RuntimeTargetID: 4, RuntimeProvider: "docker", ImageRepository: "example/app", ImageTag: "v1"}
	for range 2 {
		mock.ExpectBegin()
		mock.ExpectQuery("INSERT INTO build_jobs").WithArgs("build_retry_exhausted", uint64(45), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "", "docker", "dockerfile", "example/app", "v1", uint64(0)).WillReturnRows(sqlmock.NewRows([]string{"id", "xmax = 0"}).AddRow(uint64(45), false))
		mock.ExpectQuery("SELECT build_id, task_id").WithArgs(uint64(45)).WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()
	}
	if err := repository.CreateJob(context.Background(), snapshot); !errors.Is(err, ErrConflict) {
		t.Fatalf("CreateJob error = %v, want ErrConflict", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateJobRejectsConflictingReplay(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := JobSnapshot{BuildID: "build_conflict", TaskID: 44, ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ApplicationRecordID: 9, ApplicationName: "app", WorkspaceRoot: "/workspace/app", ContextPath: "src", DockerfilePath: "Dockerfile", RuntimeTargetID: 4, RuntimeProvider: "docker", ImageRepository: "example/app", ImageTag: "v1"}
	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO build_jobs").WithArgs("build_conflict", uint64(44), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "", "docker", "dockerfile", "example/app", "v1", uint64(0)).WillReturnRows(sqlmock.NewRows([]string{"id", "xmax = 0"}).AddRow(uint64(44), false))
	mock.ExpectQuery("SELECT build_id, task_id").WithArgs(uint64(44)).WillReturnRows(sqlmock.NewRows([]string{"build_id", "task_id", "application_id", "application_record_id", "application_name_snapshot", "workspace_context_path", "workspace_root", "dockerfile_path", "runtime_target_id", "runtime_target_name", "runtime_provider", "image_repository", "image_tag", "created_by"}).AddRow("build_other", uint64(44), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "", "docker", "other/app", "v1", uint64(0)))
	mock.ExpectQuery("SELECT name, value FROM build_job_args").WithArgs(uint64(44)).WillReturnRows(sqlmock.NewRows([]string{"name", "value"}))
	mock.ExpectRollback()

	if err := repository.CreateJob(context.Background(), snapshot); err == nil {
		t.Fatal("CreateJob conflict error = nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordProviderExecutionEvidenceUsesPlanScopedTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	plan := moduleapi.BuildExecutionPlan{ID: "plan_evidence", RuntimeTargetID: 4}
	conformance := moduleapi.ProviderExecutionConformanceResult{ProviderID: "docker-target", ConformanceVersion: "v1", Executable: true, SnapshotDeliveryProof: true, DriverExecutionProof: true, PublicationProof: true, CancellationProof: true, CleanupProof: true}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM build_execution_plans").WithArgs("plan_evidence", uint64(42)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(9)))
	mock.ExpectExec("INSERT INTO build_provider_execution_evidence").WithArgs(int64(9), uint64(42), uint64(7), int64(4), "linux/amd64", "docker-target", "v1", true, true, true, true, true).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery("SELECT runtime_target_id, platform, provider_id, conformance_version").WithArgs(int64(9), uint64(7)).WillReturnRows(sqlmock.NewRows([]string{"runtime_target_id", "platform", "provider_id", "conformance_version", "snapshot_delivery_proof", "driver_execution_proof", "publication_proof", "cancellation_proof", "cleanup_proof"}).AddRow(int64(4), "linux/amd64", "docker-target", "v1", true, true, true, true, true))
	mock.ExpectCommit()
	if err := repository.RecordProviderExecutionEvidence(context.Background(), plan, moduleapi.ProviderExecutionEvidence{TaskID: 42, StageID: 7, TargetID: 4, Platform: "linux/amd64", Conformance: conformance}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRecordProviderExecutionEvidenceRejectsConflictingReplay(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	plan := moduleapi.BuildExecutionPlan{ID: "plan_evidence_conflict"}
	conformance := moduleapi.ProviderExecutionConformanceResult{ProviderID: "docker-target", ConformanceVersion: "v1", Executable: true, SnapshotDeliveryProof: true, DriverExecutionProof: true, PublicationProof: true, CancellationProof: true, CleanupProof: true}
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id FROM build_execution_plans").WithArgs("plan_evidence_conflict", uint64(42)).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(9)))
	mock.ExpectExec("INSERT INTO build_provider_execution_evidence").WillReturnResult(sqlmock.NewResult(1, 0))
	mock.ExpectQuery("SELECT runtime_target_id, platform, provider_id, conformance_version").WithArgs(int64(9), uint64(7)).WillReturnRows(sqlmock.NewRows([]string{"runtime_target_id", "platform", "provider_id", "conformance_version", "snapshot_delivery_proof", "driver_execution_proof", "publication_proof", "cancellation_proof", "cleanup_proof"}).AddRow(int64(5), "linux/amd64", "docker-target", "v1", true, true, true, true, true))
	mock.ExpectRollback()
	err = repository.RecordProviderExecutionEvidence(context.Background(), plan, moduleapi.ProviderExecutionEvidence{TaskID: 42, StageID: 7, TargetID: 4, Platform: "linux/amd64", Conformance: conformance})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("conflicting replay error = %v, want ErrConflict", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
