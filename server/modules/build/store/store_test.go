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
	first := BuilderReservationFence("plan_1", 42, 1)
	retry := BuilderReservationFence("plan_1", 42, 2)
	if first == retry {
		t.Fatal("retry reservation fence must differ from the first attempt")
	}
	if first != BuilderReservationFence("plan_1", 42, 1) {
		t.Fatal("reservation fence must be deterministic for one attempt")
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
	reservation := moduleapi.BuilderReservation{ID: "reservation_plan_1", InstanceID: "builder_1", PlanID: "plan_1", TaskID: 42, Attempt: 1, FenceToken: BuilderReservationFence("plan_1", 42, 1), State: moduleapi.BuilderReservationAccepted, LeaseExpiresAt: expiresAt}
	mock.ExpectBegin()
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("UPDATE build_builder_reservations SET state = 'expired'").WithArgs("builder_1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("INSERT INTO build_builder_reservations").WithArgs(reservation.ID, reservation.InstanceID, reservation.PlanID, reservation.TaskID, reservation.Attempt, reservation.FenceToken, reservation.State, reservation.LeaseExpiresAt).WillReturnRows(sqlmock.NewRows([]string{"reservation_id", "builder_instance_id", "plan_id", "task_id", "attempt", "fence_token", "state", "lease_expires_at", "created_at", "updated_at"}).AddRow(reservation.ID, reservation.InstanceID, reservation.PlanID, reservation.TaskID, reservation.Attempt, reservation.FenceToken, reservation.State, expiresAt, expiresAt.Add(-time.Minute), expiresAt.Add(-time.Minute)))
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
	mock.ExpectQuery("SELECT workspace_id, display_name, source_kind, source_reference").WithArgs(uint64(7)).WillReturnRows(
		sqlmock.NewRows([]string{"workspace_id", "display_name", "source_kind", "source_reference", "retention_policy", "created_by", "created_at", "updated_at"}).
			AddRow("workspace_shared", "Shared", moduleapi.WorkspaceSourceApplication, "app_shared", "workspace", nil, time.Now(), time.Now()).
			AddRow("workspace_owned", "Owned", moduleapi.WorkspaceSourceGit, "git:main", "workspace", uint64(7), time.Now(), time.Now()),
	)
	items, err := repository.ListWorkspaces(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != "workspace_shared" || items[1].CreatedBy != 7 {
		t.Fatalf("unexpected workspace selector projection: %#v", items)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
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
	columns := []string{"plan_id", "plan_digest", "snapshot_id", "source_kind", "source_reference", "content_digest", "materialization_ref", "snapshot_created_at", "builder_pool_id", "builder_instance_id", "runtime_target_id", "driver", "template_ref", "platforms_json", "builder_placements_json", "destination_json", "created_at"}
	mock.ExpectQuery("SELECT p.plan_id").WithArgs(uint64(42)).WillReturnRows(sqlmock.NewRows(columns).AddRow("plan_test", "sha256:plan", "snapshot_test", "application_workspace", "app_test", "sha256:source", "/managed/snapshot", time.Now(), "pool_builders", "builder-amd64", int64(4), "docker-buildx@v1", "oci-dockerfile/default@v1", `["linux/amd64","linux/arm64"]`, placements, `{"kind":"oci_registry","connection_ref":"registry","repository_ref":"team/app","reference":"v1"}`, time.Now()))
	plan, err := repository.GetExecutionPlanByTaskID(context.Background(), 42)
	if err != nil {
		t.Fatal(err)
	}
	if placement, ok := plan.PlacementForPlatform("linux/arm64"); !ok || placement.BuilderInstanceID != "builder-arm64" || placement.RuntimeTargetID != 5 {
		t.Fatalf("placement = %#v ok=%t", placement, ok)
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
	columns := []string{"build_id", "task_id", "application_id", "application_record_id", "application_name_snapshot", "workspace_context_path", "workspace_root", "dockerfile_path", "runtime_target_id", "runtime_target_name", "runtime_provider", "image_repository", "image_tag", "created_by", "created_at", "artifact_id", "image_id", "digest", "repository", "tag", "size_bytes", "platform"}
	mock.ExpectQuery("SELECT j.build_id").WithArgs("build_test").WillReturnRows(sqlmock.NewRows(columns).AddRow("build_test", uint64(42), "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", uint64(9), "app", "src", "/workspace/app", "Dockerfile", uint64(4), "Local Docker", "docker", "example/app", "v1", uint64(7), time.Now(), nil, nil, "", nil, nil, nil, ""))
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
