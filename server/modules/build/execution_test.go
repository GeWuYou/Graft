package build

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

type recordingBuildRepository struct {
	created            buildstore.JobSnapshot
	snapshot           buildstore.JobSnapshot
	settledID          uint64
	settleCanceled     bool
	settleDeadline     bool
	getBuildIDErr      error
	createErr          error
	listResult         buildstore.ListResult
	listQuery          buildstore.ListQuery
	artifactResult     buildstore.V2ArtifactListResult
	v2Plan             moduleapi.BuildExecutionPlan
	workspaces         []moduleapi.BuildWorkspace
	publicationSources []moduleapi.ArtifactPublicationSource
	promotionInput     moduleapi.OCIArtifactCopyInput
	promotionResult    moduleapi.OCIArtifactCopyResult
	promotionAuth      moduleapi.RegistryAuthExecution
	promotionSettled   bool
	platformArtifacts  []moduleapi.PlatformArtifact
	manifestInput      moduleapi.OCIManifestPublicationInput
	manifestSettled    bool
}

func (r *recordingBuildRepository) RecordPlatformArtifact(_ context.Context, _ uint64, _ moduleapi.BuildExecutionPlan, artifact moduleapi.PlatformArtifact) error {
	r.platformArtifacts = append(r.platformArtifacts, artifact)
	return nil
}

func (r *recordingBuildRepository) ListPlatformArtifacts(context.Context, uint64, moduleapi.BuildExecutionPlan) ([]moduleapi.PlatformArtifact, error) {
	return append([]moduleapi.PlatformArtifact(nil), r.platformArtifacts...), nil
}

func (r *recordingBuildRepository) PrepareOCIManifestPublication(_ context.Context, _ uint64, plan moduleapi.BuildExecutionPlan) (moduleapi.OCIManifestPublicationInput, error) {
	r.manifestInput = moduleapi.OCIManifestPublicationInput{Destination: moduleapi.AuthorizedArtifactDestination(plan.Destination), PlatformArtifacts: append([]moduleapi.PlatformArtifact(nil), r.platformArtifacts...)}
	return r.manifestInput, nil
}

func (r *recordingBuildRepository) SettleOCIManifestPublication(context.Context, uint64, moduleapi.BuildExecutionPlan, moduleapi.OCIManifestPublicationResult, moduleapi.RegistryAuthExecution) error {
	r.manifestSettled = true
	return nil
}

func (r *recordingBuildRepository) CreateWorkspace(context.Context, moduleapi.BuildWorkspace) error {
	return nil
}

func (r *recordingBuildRepository) GetWorkspace(context.Context, string) (moduleapi.BuildWorkspace, error) {
	return moduleapi.BuildWorkspace{ID: "workspace_app", Name: "Application", SourceKind: moduleapi.WorkspaceSourceApplication, SourceReference: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X"}, nil
}

func (r *recordingBuildRepository) ListWorkspaces(context.Context, uint64) ([]moduleapi.BuildWorkspace, error) {
	return append([]moduleapi.BuildWorkspace(nil), r.workspaces...), nil
}

func (r *recordingBuildRepository) MaterializeExecutionPlan(_ context.Context, _ *sql.Tx, submission moduleapi.TaskSubmission, plan moduleapi.BuildExecutionPlan, _ uint64) (string, error) {
	if r.createErr != nil {
		return "", r.createErr
	}
	if submission.TaskID == nil {
		return "", errors.New("task id is required")
	}
	r.v2Plan = plan
	return plan.ID, nil
}

func (r *recordingBuildRepository) CreateJob(_ context.Context, value buildstore.JobSnapshot) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.created = value
	r.snapshot = value
	return nil
}
func (r *recordingBuildRepository) MaterializeSubmissionSnapshot(_ context.Context, _ *sql.Tx, submission moduleapi.TaskSubmission, value buildstore.JobSnapshot) (string, error) {
	if r.createErr != nil {
		return "", r.createErr
	}
	value.SubmissionID = submission.ID
	if submission.TaskID != nil {
		value.TaskID = *submission.TaskID
	}
	r.created = value
	r.snapshot = value
	return value.BuildID, nil
}
func (r *recordingBuildRepository) GetJobByTaskID(_ context.Context, taskID uint64) (buildstore.JobSnapshot, error) {
	if taskID != r.snapshot.TaskID {
		return buildstore.JobSnapshot{}, buildstore.ErrNotFound
	}
	return r.snapshot, nil
}
func (r *recordingBuildRepository) SettleDockerArtifact(ctx context.Context, taskID uint64, _ moduleapi.DockerImageBuildResult) error {
	r.settledID = taskID
	r.settleCanceled = errors.Is(ctx.Err(), context.Canceled)
	_, r.settleDeadline = ctx.Deadline()
	return nil
}
func (r *recordingBuildRepository) ListJobs(_ context.Context, query buildstore.ListQuery) (buildstore.ListResult, error) {
	r.listQuery = query
	return r.listResult, nil
}

func (r *recordingBuildRepository) ListV2Artifacts(context.Context, int, int) (buildstore.V2ArtifactListResult, error) {
	return r.artifactResult, nil
}

func (r *recordingBuildRepository) ListArtifactPublicationSources(context.Context, string) ([]moduleapi.ArtifactPublicationSource, error) {
	return append([]moduleapi.ArtifactPublicationSource(nil), r.publicationSources...), nil
}

func (r *recordingBuildRepository) SettleArtifactPromotion(_ context.Context, input moduleapi.OCIArtifactCopyInput, result moduleapi.OCIArtifactCopyResult, auth moduleapi.RegistryAuthExecution) error {
	r.promotionInput, r.promotionResult, r.promotionAuth, r.promotionSettled = input, result, auth, true
	return nil
}

func (r *recordingBuildRepository) GetJobByBuildID(context.Context, string) (buildstore.JobProjection, error) {
	if r.getBuildIDErr != nil {
		return buildstore.JobProjection{}, r.getBuildIDErr
	}
	return buildstore.JobProjection{}, buildstore.ErrNotFound
}

type recordingBuildContexts struct{ calls int }

func (r *recordingBuildContexts) ResolveApplicationBuildContext(context.Context, string) (moduleapi.ApplicationBuildContext, error) {
	r.calls++
	return moduleapi.ApplicationBuildContext{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ApplicationRecordID: 9, DisplayName: "app", WorkspaceRoot: "/workspace/app", RuntimeTargetID: 4, RuntimeProvider: "docker", CanBuild: true}, nil
}

type recordingBuildTasks struct {
	input            moduleapi.SubmitTaskInput
	beginCalls       int
	materializeCalls int
	discardCalls     int
	err              error
	views            []moduleapi.TaskView
	batchErr         error
	batchIDs         []uint64
	beginSubmission  *moduleapi.TaskSubmission
	submitCalls      int
}

func (r *recordingBuildTasks) Submit(_ context.Context, input moduleapi.SubmitTaskInput) (moduleapi.TaskReceipt, error) {
	r.submitCalls++
	r.input = input
	if r.err != nil {
		return moduleapi.TaskReceipt{}, r.err
	}
	return moduleapi.TaskReceipt{TaskID: 42, Status: moduleapi.TaskStatusReady}, nil
}

func (*recordingBuildTasks) SettleExternalReceipt(context.Context, moduleapi.ExternalTaskReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	return moduleapi.ExternalReceiptSettlement{}, errors.New("not implemented")
}

func (*recordingBuildTasks) Cancel(context.Context, uint64) error { return nil }

func (*recordingBuildTasks) RetryStage(context.Context, uint64, uint64) error { return nil }

func (r *recordingBuildTasks) GetTasksByIDs(_ context.Context, taskIDs []uint64) ([]moduleapi.TaskView, error) {
	r.batchIDs = append([]uint64(nil), taskIDs...)
	if r.batchErr != nil {
		return nil, r.batchErr
	}
	return r.views, nil
}

func (r *recordingBuildTasks) BeginSubmission(_ context.Context, input moduleapi.BeginTaskSubmissionInput) (moduleapi.TaskSubmissionHandle, error) {
	r.beginCalls++
	r.input = input.Task
	if r.err != nil {
		return moduleapi.TaskSubmissionHandle{}, r.err
	}
	if r.beginSubmission != nil {
		return moduleapi.TaskSubmissionHandle{Submission: *r.beginSubmission, LeaseToken: "lease"}, nil
	}
	return moduleapi.TaskSubmissionHandle{Submission: moduleapi.TaskSubmission{ID: "submission_test", TaskType: input.Task.Type, Owner: input.Task.Owner, State: moduleapi.TaskSubmissionStateReserved, SubmissionVersion: 1}, LeaseToken: "lease"}, nil
}
func (r *recordingBuildTasks) RenewSubmissionLease(context.Context, moduleapi.TaskSubmissionHandle) (moduleapi.TaskSubmissionHandle, error) {
	return moduleapi.TaskSubmissionHandle{}, errors.New("not implemented")
}
func (r *recordingBuildTasks) MaterializeSubmission(ctx context.Context, handle moduleapi.TaskSubmissionHandle, input moduleapi.SubmitTaskInput, writer moduleapi.TaskSubmissionWriter) (moduleapi.TaskReceipt, error) {
	r.materializeCalls++
	r.input = input
	if r.err != nil {
		return moduleapi.TaskReceipt{}, r.err
	}
	taskID := uint64(42)
	if _, err := writer.MaterializeTaskSubmission(ctx, nil, moduleapi.TaskSubmission{ID: handle.Submission.ID, TaskID: &taskID}); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	return moduleapi.TaskReceipt{TaskID: taskID, Status: moduleapi.TaskStatusReady}, nil
}
func (r *recordingBuildTasks) DiscardSubmission(context.Context, moduleapi.TaskSubmissionHandle, string) error {
	r.discardCalls++
	return nil
}
func (*recordingBuildTasks) ExpireSubmissions(context.Context, int) (int, error) { return 0, nil }
func (*recordingBuildTasks) GetSubmission(context.Context, string) (moduleapi.TaskSubmission, error) {
	return moduleapi.TaskSubmission{}, errors.New("not implemented")
}

type recordingBuildDocker struct {
	input moduleapi.DockerImageBuildInput
}

func TestListJobsAddsBatchedTaskExecutionProjectionAfterRepositoryStatusFilter(t *testing.T) {
	repository := &recordingBuildRepository{listResult: buildstore.ListResult{Items: []buildstore.JobProjection{{JobSnapshot: buildstore.JobSnapshot{BuildID: "build_test", TaskID: 42}}}, Total: 1}}
	tasks := &recordingBuildTasks{views: []moduleapi.TaskView{{ID: 42, Status: moduleapi.TaskStatusRunning, CurrentStageKey: stringPtr("dockerfile-build")}}}
	service, err := NewService(&recordingBuildContexts{}, tasks, tasks, &recordingBuildDocker{}, repository)
	if err != nil {
		t.Fatal(err)
	}
	status := buildstore.StatusFilterRunning
	result, err := service.ListJobs(context.Background(), buildstore.ListQuery{Limit: 1, BuildStatus: &status})
	if err != nil || len(result.Items) != 1 {
		t.Fatalf("list jobs = %#v err=%v", result, err)
	}
	if result.Items[0].Execution.Status != moduleapi.TaskStatusRunning || result.Items[0].Execution.StageCount != 1 || result.Items[0].Execution.CompletedStageCount != 0 {
		t.Fatalf("unexpected execution projection: %#v", result.Items[0].Execution)
	}
	if result.Total != 1 {
		t.Fatalf("repository filtered total = %d, want 1", result.Total)
	}
	if repository.listQuery.BuildStatus == nil || *repository.listQuery.BuildStatus != status || repository.listQuery.Limit != 1 {
		t.Fatalf("repository query = %#v, want running status and requested pagination", repository.listQuery)
	}
}

func TestBuildStatusFilterGroupsTaskRuntimeStates(t *testing.T) {
	cases := []struct {
		filter buildstore.StatusFilter
		status moduleapi.TaskStatus
		want   bool
	}{
		{filter: buildstore.StatusFilterQueued, status: moduleapi.TaskStatusPending, want: true},
		{filter: buildstore.StatusFilterQueued, status: moduleapi.TaskStatusReady, want: true},
		{filter: buildstore.StatusFilterQueued, status: moduleapi.TaskStatusScheduled, want: true},
		{filter: buildstore.StatusFilterFailed, status: moduleapi.TaskStatusNeedsAttention, want: true},
		{filter: buildstore.StatusFilterFailed, status: moduleapi.TaskStatusRunning, want: false},
	}
	for _, item := range cases {
		if got := item.filter.MatchesTaskStatus(item.status); got != item.want {
			t.Fatalf("filter %q status %q = %t, want %t", item.filter, item.status, got, item.want)
		}
	}
}

func stringPtr(value string) *string { return &value }

func TestNormalizePlatformDigest(t *testing.T) {
	valid := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	for _, value := range []string{valid, "registry.example/team/app@" + valid} {
		if got, ok := normalizePlatformDigest(value); !ok || got != valid {
			t.Fatalf("normalizePlatformDigest(%q) = %q, %t", value, got, ok)
		}
	}
	for _, value := range []string{"", "sha256:short", "registry.example/team/app:v1"} {
		if got, ok := normalizePlatformDigest(value); ok || got != "" {
			t.Fatalf("normalizePlatformDigest(%q) = %q, %t, want rejection", value, got, ok)
		}
	}
}

type snapshotDeliveryCapabilityStub struct {
	result moduleapi.WorkspaceSnapshotDeliveryResult
	err    error
	last   moduleapi.WorkspaceSnapshotDeliveryRequest
}

type providerConformanceCapabilityStub struct {
	result moduleapi.ProviderExecutionConformanceResult
	err    error
}

type builderReservationRepositoryStub struct {
	markedLeg, markedFence   string
	renewLeg, renewFence     string
	releaseLeg, releaseFence string
	releaseState             string
	retryReservation         moduleapi.BuilderReservation
	renewErr                 error
}

func (s *builderReservationRepositoryStub) ReserveBuilder(context.Context, *sql.Tx, moduleapi.BuilderReservation) (moduleapi.BuilderReservation, error) {
	return moduleapi.BuilderReservation{}, errors.New("not implemented")
}

func (s *builderReservationRepositoryStub) ReserveBuilderAttempt(_ context.Context, reservation moduleapi.BuilderReservation) (moduleapi.BuilderReservation, error) {
	s.retryReservation = reservation
	return reservation, nil
}

func (s *builderReservationRepositoryStub) MarkBuilderReservationRunning(_ context.Context, _ uint64, legID, fence string) error {
	s.markedLeg, s.markedFence = legID, fence
	return nil
}

func (s *builderReservationRepositoryStub) RenewBuilderReservation(_ context.Context, _ uint64, legID, fence string, _ time.Time) error {
	s.renewLeg, s.renewFence = legID, fence
	return s.renewErr
}

func (s *builderReservationRepositoryStub) ReleaseBuilderReservation(_ context.Context, _ uint64, legID, fence, state string) error {
	s.releaseLeg, s.releaseFence, s.releaseState = legID, fence, state
	return nil
}

func TestBeginBuilderReservationUsesSinglePlatformPlacementLeg(t *testing.T) {
	plan := moduleapi.BuildExecutionPlan{ID: "plan_1", BuilderInstanceID: "fallback", BuilderPlacements: []moduleapi.BuilderPlacement{{Platform: "linux/amd64", BuilderInstanceID: "builder-amd64", RuntimeTargetID: 4}}}
	placement, found := plan.PlacementForPlatform("linux/amd64")
	if !found {
		t.Fatal("single-platform placement was not found")
	}
	repository := &builderReservationRepositoryStub{}
	cleanup, err := beginBuilderReservation(context.Background(), repository, builderReservationStart{planID: plan.ID, taskID: 42, instanceID: placement.BuilderInstanceID, legID: placement.Platform, attempt: 1})
	if err != nil {
		t.Fatalf("begin builder reservation: %v", err)
	}
	wantFence := buildstore.BuilderReservationFence(plan.ID, 42, "linux/amd64", 1)
	if repository.markedLeg != "linux/amd64" || repository.markedFence != wantFence || repository.renewLeg != "linux/amd64" || repository.renewFence != wantFence {
		t.Fatalf("reservation calls = mark(%q,%q) renew(%q,%q)", repository.markedLeg, repository.markedFence, repository.renewLeg, repository.renewFence)
	}
	var executionErr error
	cleanup(&executionErr)
	if repository.releaseState != moduleapi.BuilderReservationReleased || repository.releaseLeg != "linux/amd64" || repository.releaseFence != wantFence {
		t.Fatalf("reservation release = (%q,%q,%q)", repository.releaseLeg, repository.releaseFence, repository.releaseState)
	}
}

func TestBeginBuilderReservationAbandonsRunningReservationWhenRenewalFails(t *testing.T) {
	repository := &builderReservationRepositoryStub{renewErr: errors.New("renew failed")}
	cleanup, err := beginBuilderReservation(context.Background(), repository, builderReservationStart{planID: "plan_1", taskID: 42, instanceID: "builder-amd64", legID: "linux/amd64", attempt: 1})
	if cleanup != nil || err == nil || !strings.Contains(err.Error(), "renew builder reservation") {
		t.Fatalf("begin builder reservation = cleanup:%v err:%v", cleanup != nil, err)
	}
	if repository.releaseState != moduleapi.BuilderReservationAbandoned || repository.releaseLeg != "linux/amd64" || repository.releaseFence != buildstore.BuilderReservationFence("plan_1", 42, "linux/amd64", 1) {
		t.Fatalf("renewal-failure release = (%q,%q,%q)", repository.releaseLeg, repository.releaseFence, repository.releaseState)
	}
}

func TestBeginBuilderReservationRetryPreservesFrozenBuilderIdentityAndUsesNewFence(t *testing.T) {
	repository := &builderReservationRepositoryStub{}
	cleanup, err := beginBuilderReservation(context.Background(), repository, builderReservationStart{planID: "plan_frozen", taskID: 42, instanceID: "builder-amd64", legID: "linux/amd64", attempt: 2})
	if err != nil {
		t.Fatalf("begin retry reservation: %v", err)
	}
	wantFence := buildstore.BuilderReservationFence("plan_frozen", 42, "linux/amd64", 2)
	if repository.retryReservation.InstanceID != "builder-amd64" || repository.retryReservation.PlanID != "plan_frozen" || repository.retryReservation.FenceToken != wantFence || repository.retryReservation.Attempt != 2 {
		t.Fatalf("retry reservation = %#v", repository.retryReservation)
	}
	var executionErr error
	cleanup(&executionErr)
	if repository.releaseFence != wantFence || repository.releaseState != moduleapi.BuilderReservationReleased {
		t.Fatalf("retry release = (%q,%q)", repository.releaseFence, repository.releaseState)
	}
}

func TestReconfirmFrozenDynamicPlacementDoesNotReselectTarget(t *testing.T) {
	now := time.Now().UTC()
	executor := v2ExecutionPlanExecutor{builderTelemetry: builderTelemetryReaderStub{
		admitted:  map[int64]bool{4: true},
		snapshots: map[int64]moduleapi.BuilderTelemetrySnapshot{4: {TargetID: 4, BuilderScope: "builder:frozen", ProviderID: "agent", CapabilityProfile: "buildkit", CapabilityVersion: "v1", Available: true, ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute), SourceRef: "report:retry", Provenance: "agent", Integrity: "sha256:retry"}},
	}}
	if err := executor.reconfirmFrozenDynamicPlacement(context.Background(), moduleapi.BuilderPlacement{RuntimeTargetID: 4, SchedulingPolicy: "least_load"}); err != nil {
		t.Fatalf("reconfirm frozen placement: %v", err)
	}
	if err := executor.reconfirmFrozenDynamicPlacement(context.Background(), moduleapi.BuilderPlacement{RuntimeTargetID: 5, SchedulingPolicy: "least_load"}); err == nil {
		t.Fatal("expected the frozen target to fail closed rather than be reselected")
	}
}

func (s providerConformanceCapabilityStub) ConformProviderExecution(context.Context, moduleapi.ProviderExecutionConformanceRequest) (moduleapi.ProviderExecutionConformanceResult, error) {
	return s.result, s.err
}

func (s *snapshotDeliveryCapabilityStub) DeliverWorkspaceSnapshot(_ context.Context, request moduleapi.WorkspaceSnapshotDeliveryRequest) (moduleapi.WorkspaceSnapshotDeliveryResult, error) {
	s.last = request
	return s.result, s.err
}

func TestVerifySnapshotDeliveryRequiresMatchingFrozenIdentity(t *testing.T) {
	snapshot := moduleapi.WorkspaceSnapshot{ID: "snapshot-1", ContentDigest: "sha256:source", MaterializedRoot: "/tmp/graft-build-snapshots/snapshot-1"}
	capability := &snapshotDeliveryCapabilityStub{result: moduleapi.WorkspaceSnapshotDeliveryResult{TargetID: 4, SnapshotID: snapshot.ID, ContentDigest: snapshot.ContentDigest, DeliveryMode: moduleapi.SnapshotDeliveryModeTargetLocal}}
	if err := verifySnapshotDelivery(context.Background(), capability, 4, snapshot, moduleapi.SnapshotDeliveryModeTargetLocal); err != nil {
		t.Fatal(err)
	}
	if capability.last.TargetID != 4 || capability.last.SnapshotID != snapshot.ID || capability.last.ContentDigest != snapshot.ContentDigest || capability.last.DeliveryMode != moduleapi.SnapshotDeliveryModeTargetLocal {
		t.Fatalf("delivery request = %#v", capability.last)
	}
	capability.result.ContentDigest = "sha256:other"
	if err := verifySnapshotDelivery(context.Background(), capability, 4, snapshot, moduleapi.SnapshotDeliveryModeTargetLocal); err == nil {
		t.Fatal("mismatched Snapshot proof unexpectedly succeeded")
	}
}

func TestVerifyProviderConformanceFailsClosedOnIncompleteEvidence(t *testing.T) {
	snapshot := moduleapi.WorkspaceSnapshot{ID: "snapshot-1", ContentDigest: "sha256:source"}
	capability := providerConformanceCapabilityStub{result: moduleapi.ProviderExecutionConformanceResult{ProviderID: "docker-target", ConformanceVersion: "v1", Executable: true}}
	if _, err := verifyProviderConformance(context.Background(), capability, moduleapi.ProviderExecutionConformanceRequest{TargetID: 4, DriverRef: "docker-engine", Platform: "linux/amd64", SnapshotID: snapshot.ID, ContentDigest: snapshot.ContentDigest, DeliveryMode: moduleapi.SnapshotDeliveryModeTargetLocal}); err == nil {
		t.Fatal("incomplete provider conformance unexpectedly succeeded")
	}
}

func TestTemporaryPlatformTagIsDeterministicAndScoped(t *testing.T) {
	if got := temporaryPlatformTag("v1", "linux/amd64"); got != "v1-graft-linux-amd64" {
		t.Fatalf("temporaryPlatformTag = %q", got)
	}
}

func (r *recordingBuildDocker) BuildImage(_ context.Context, input moduleapi.DockerImageBuildInput, _ moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error) {
	r.input = input
	return moduleapi.DockerImageBuildResult{ImageID: "sha256:image", Repository: input.ImageRepository, Tag: input.ImageTag}, nil
}

type buildStageRun struct{ input json.RawMessage }

func (buildStageRun) TaskID() uint64                                          { return 42 }
func (buildStageRun) StageID() uint64                                         { return 7 }
func (buildStageRun) Attempt() int                                            { return 1 }
func (r buildStageRun) Input() json.RawMessage                                { return r.input }
func (buildStageRun) CancellationRequested(context.Context) bool              { return false }
func (buildStageRun) AppendLog(context.Context, moduleapi.TaskLogEntry) error { return nil }

type promotionRegistryStub struct {
	binding moduleapi.RegistryArtifactCopyBinding
	err     error
	copy    moduleapi.AuthorizedArtifactCopy
}

func (s *promotionRegistryStub) AuthorizeArtifactCopy(_ context.Context, _ uint64, source moduleapi.ArtifactPublicationSource, destination moduleapi.BuildDestination) (moduleapi.AuthorizedArtifactCopy, error) {
	return moduleapi.AuthorizedArtifactCopy{Source: source, Destination: moduleapi.AuthorizedArtifactDestination(destination)}, s.err
}

func (s *promotionRegistryStub) ResolveArtifactCopyBinding(_ context.Context, copy moduleapi.AuthorizedArtifactCopy) (moduleapi.RegistryArtifactCopyBinding, error) {
	s.copy = copy
	return s.binding, s.err
}

type promotionExecutionAdapterStub struct {
	copy func(context.Context, int64, moduleapi.OCIArtifactCopyInput, moduleapi.RegistryArtifactCopyBinding, moduleapi.DockerImageBuildLogSink) (moduleapi.OCIArtifactCopyResult, error)
}

type manifestRegistryStub struct {
	binding moduleapi.RegistryPublicationBinding
}

func (s manifestRegistryStub) ResolvePublicationBinding(context.Context, moduleapi.AuthorizedArtifactDestination) (moduleapi.RegistryPublicationBinding, error) {
	return s.binding, nil
}

type manifestExecutionAdapterStub struct{ calls int }

func (*manifestExecutionAdapterStub) PublishImage(context.Context, int64, moduleapi.DockerImageBuildResult, moduleapi.RegistryPublicationBinding, moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error) {
	return moduleapi.DockerImageBuildResult{}, errors.New("not implemented")
}

func (s *manifestExecutionAdapterStub) PublishManifest(_ context.Context, _ int64, input moduleapi.OCIManifestPublicationInput, _ moduleapi.RegistryPublicationBinding, _ moduleapi.DockerImageBuildLogSink) (moduleapi.OCIManifestPublicationResult, error) {
	s.calls++
	if len(input.PlatformArtifacts) != 2 {
		return moduleapi.OCIManifestPublicationResult{}, errors.New("platform artifacts are incomplete")
	}
	return moduleapi.OCIManifestPublicationResult{Digest: "sha256:" + strings.Repeat("b", 64), MediaType: "application/vnd.oci.image.index.v1+json"}, nil
}

func (*manifestExecutionAdapterStub) CopyOCIArtifact(context.Context, int64, moduleapi.OCIArtifactCopyInput, moduleapi.RegistryArtifactCopyBinding, moduleapi.DockerImageBuildLogSink) (moduleapi.OCIArtifactCopyResult, error) {
	return moduleapi.OCIArtifactCopyResult{}, errors.New("not implemented")
}

func (s promotionExecutionAdapterStub) CopyOCIArtifact(ctx context.Context, targetID int64, input moduleapi.OCIArtifactCopyInput, binding moduleapi.RegistryArtifactCopyBinding, sink moduleapi.DockerImageBuildLogSink) (moduleapi.OCIArtifactCopyResult, error) {
	return s.copy(ctx, targetID, input, binding, sink)
}

func (promotionExecutionAdapterStub) PublishImage(context.Context, int64, moduleapi.DockerImageBuildResult, moduleapi.RegistryPublicationBinding, moduleapi.DockerImageBuildLogSink) (moduleapi.DockerImageBuildResult, error) {
	return moduleapi.DockerImageBuildResult{}, errors.New("not implemented")
}

func (promotionExecutionAdapterStub) PublishManifest(context.Context, int64, moduleapi.OCIManifestPublicationInput, moduleapi.RegistryPublicationBinding, moduleapi.DockerImageBuildLogSink) (moduleapi.OCIManifestPublicationResult, error) {
	return moduleapi.OCIManifestPublicationResult{}, errors.New("not implemented")
}

func TestArtifactPromotionExecutorCopiesFrozenDigestThenSettles(t *testing.T) {
	digest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	source := moduleapi.ArtifactPublicationSource{ArtifactID: "artifact_1", PublicationID: "publication_1", Digest: digest, MediaType: "application/vnd.oci.image.manifest.v1+json", DestinationKind: "oci_registry", ConnectionRef: "registry:source", RepositoryRef: "team/source"}
	destination := moduleapi.AuthorizedArtifactDestination{Kind: "oci_registry", ConnectionRef: "registry:destination", RepositoryRef: "team/destination", Reference: "stable"}
	repository := &recordingBuildRepository{}
	service, err := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildDocker{}, repository)
	if err != nil {
		t.Fatal(err)
	}
	registry := &promotionRegistryStub{binding: moduleapi.RegistryArtifactCopyBinding{SourceEndpoint: "https://source.example", SourceCredentialRef: "ref:source", SourceAuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}, Destination: moduleapi.RegistryPublicationBinding{Destination: destination, Endpoint: "https://destination.example", CredentialRef: "ref:destination", AuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}}}}
	adapter := promotionExecutionAdapterStub{copy: func(_ context.Context, targetID int64, input moduleapi.OCIArtifactCopyInput, _ moduleapi.RegistryArtifactCopyBinding, _ moduleapi.DockerImageBuildLogSink) (moduleapi.OCIArtifactCopyResult, error) {
		if targetID != 4 || input.Source != source || input.Destination != destination {
			t.Fatalf("copy input = %#v target=%d", input, targetID)
		}
		return moduleapi.OCIArtifactCopyResult{Digest: digest, MediaType: source.MediaType, SizeBytes: 19}, nil
	}}
	executor := &artifactPromotionExecutor{service: service, adapter: adapter, registry: registry, cancels: make(map[uint64]context.CancelFunc)}
	payload, _ := json.Marshal(moduleapi.ArtifactPromotionTaskInput{Source: source, Destination: destination, RuntimeTargetID: 4})
	if err := executor.Execute(context.Background(), buildStageRun{input: payload}); err != nil {
		t.Fatal(err)
	}
	if !repository.promotionSettled || repository.promotionInput.Source != source || repository.promotionInput.Destination != destination || repository.promotionResult.Digest != digest || repository.promotionAuth.Mode != moduleapi.RegistryAuthExecutionEphemeral {
		t.Fatalf("promotion settlement = input:%#v result:%#v auth:%#v settled:%t", repository.promotionInput, repository.promotionResult, repository.promotionAuth, repository.promotionSettled)
	}
}

func TestArtifactPromotionExecutorCancellationDoesNotSettle(t *testing.T) {
	digest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	source := moduleapi.ArtifactPublicationSource{ArtifactID: "artifact_1", PublicationID: "publication_1", Digest: digest, MediaType: "application/vnd.oci.image.manifest.v1+json", DestinationKind: "oci_registry", ConnectionRef: "registry:source", RepositoryRef: "team/source"}
	destination := moduleapi.AuthorizedArtifactDestination{Kind: "oci_registry", ConnectionRef: "registry:destination", RepositoryRef: "team/destination", Reference: "stable"}
	repository := &recordingBuildRepository{}
	service, _ := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildDocker{}, repository)
	started := make(chan struct{})
	executor := &artifactPromotionExecutor{service: service, registry: &promotionRegistryStub{binding: moduleapi.RegistryArtifactCopyBinding{Destination: moduleapi.RegistryPublicationBinding{Destination: destination}}}, cancels: make(map[uint64]context.CancelFunc), adapter: promotionExecutionAdapterStub{copy: func(ctx context.Context, _ int64, _ moduleapi.OCIArtifactCopyInput, _ moduleapi.RegistryArtifactCopyBinding, _ moduleapi.DockerImageBuildLogSink) (moduleapi.OCIArtifactCopyResult, error) {
		close(started)
		<-ctx.Done()
		return moduleapi.OCIArtifactCopyResult{}, ctx.Err()
	}}}
	payload, _ := json.Marshal(moduleapi.ArtifactPromotionTaskInput{Source: source, Destination: destination, RuntimeTargetID: 4})
	done := make(chan error, 1)
	go func() { done <- executor.Execute(context.Background(), buildStageRun{input: payload}) }()
	<-started
	if err := executor.Cancel(context.Background(), buildStageRun{input: payload}); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled execution error = %v", err)
	}
	if repository.promotionSettled {
		t.Fatal("cancelled promotion unexpectedly settled")
	}
}

func TestSubmitFreezesAuthorizedBuildSnapshot(t *testing.T) {
	contexts := &recordingBuildContexts{}
	tasks := &recordingBuildTasks{}
	repository := &recordingBuildRepository{}
	service, err := NewService(contexts, tasks, tasks, &recordingBuildDocker{}, repository)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Submit(context.Background(), SubmitRequest{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", BuildArgs: []moduleapi.DockerImageBuildArg{{Name: "MODE", Value: "release"}}, IdempotencyKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	if contexts.calls != 1 || repository.created.TaskID != 42 || repository.created.WorkspaceRoot != "/workspace/app" || len(repository.created.BuildArgs) != 1 {
		t.Fatalf("unexpected frozen snapshot: %#v calls=%d", repository.created, contexts.calls)
	}
	if tasks.beginCalls != 1 || tasks.materializeCalls != 1 || tasks.discardCalls != 0 {
		t.Fatalf("submission lifecycle = begin:%d materialize:%d discard:%d", tasks.beginCalls, tasks.materializeCalls, tasks.discardCalls)
	}
	var input moduleapi.BuildTaskInput
	if err := json.Unmarshal(tasks.input.Input, &input); err != nil || input.BuildID != repository.created.BuildID {
		t.Fatalf("task input must contain build identity: %#v err=%v", input, err)
	}
}

func TestSubmitReturnsPersistedTaskStatusForActivatedSubmission(t *testing.T) {
	taskID := uint64(42)
	tasks := &recordingBuildTasks{
		beginSubmission: &moduleapi.TaskSubmission{ID: "submission_test", TaskID: &taskID, State: moduleapi.TaskSubmissionStateActivated},
		views:           []moduleapi.TaskView{{ID: taskID, Status: moduleapi.TaskStatusRunning}},
	}
	service, err := NewService(&recordingBuildContexts{}, tasks, tasks, &recordingBuildDocker{}, &recordingBuildRepository{})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := service.Submit(context.Background(), SubmitRequest{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", IdempotencyKey: "key"})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.TaskID != taskID || receipt.Status != moduleapi.TaskStatusRunning {
		t.Fatalf("replayed receipt = %#v, want running task %d", receipt, taskID)
	}
	if tasks.materializeCalls != 0 {
		t.Fatalf("materialize calls = %d, want 0 for activated submission", tasks.materializeCalls)
	}
	if len(tasks.batchIDs) != 1 || tasks.batchIDs[0] != taskID {
		t.Fatalf("receipt task lookup ids = %v, want [%d]", tasks.batchIDs, taskID)
	}
}

func TestSubmitRejectsActivatedSubmissionWithoutPersistedTask(t *testing.T) {
	taskID := uint64(42)
	tasks := &recordingBuildTasks{
		beginSubmission: &moduleapi.TaskSubmission{ID: "submission_test", TaskID: &taskID, State: moduleapi.TaskSubmissionStateActivated},
		views:           []moduleapi.TaskView{{ID: 43, Status: moduleapi.TaskStatusReady}},
	}
	service, err := NewService(&recordingBuildContexts{}, tasks, tasks, &recordingBuildDocker{}, &recordingBuildRepository{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Submit(context.Background(), SubmitRequest{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", IdempotencyKey: "key"})
	if !errors.Is(err, buildstore.ErrNotFound) {
		t.Fatalf("Submit error = %v, want build snapshot not found", err)
	}
	if tasks.materializeCalls != 0 {
		t.Fatalf("materialize calls = %d, want 0 for activated submission", tasks.materializeCalls)
	}
}

func TestSubmitReturnsActivatedReceiptLookupError(t *testing.T) {
	taskID := uint64(42)
	lookupErr := errors.New("task lookup failed")
	tasks := &recordingBuildTasks{
		beginSubmission: &moduleapi.TaskSubmission{ID: "submission_test", TaskID: &taskID, State: moduleapi.TaskSubmissionStateActivated},
		batchErr:        lookupErr,
	}
	service, err := NewService(&recordingBuildContexts{}, tasks, tasks, &recordingBuildDocker{}, &recordingBuildRepository{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Submit(context.Background(), SubmitRequest{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", IdempotencyKey: "key"})
	if !errors.Is(err, lookupErr) {
		t.Fatalf("Submit error = %v, want wrapped task lookup error", err)
	}
}

func TestSubmitReturnsErrorWhenAtomicSnapshotMaterializationFails(t *testing.T) {
	tasks := &recordingBuildTasks{}
	repository := &recordingBuildRepository{createErr: errors.New("snapshot write failed")}
	service, err := NewService(&recordingBuildContexts{}, tasks, tasks, &recordingBuildDocker{}, repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Submit(context.Background(), SubmitRequest{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", IdempotencyKey: "key"}); err == nil {
		t.Fatal("Submit error = nil, want snapshot failure")
	}
	if tasks.beginCalls != 1 || tasks.materializeCalls != 1 || tasks.discardCalls != 0 {
		t.Fatalf("submission lifecycle = begin:%d materialize:%d discard:%d", tasks.beginCalls, tasks.materializeCalls, tasks.discardCalls)
	}
}

func TestExecutorUsesBuildSnapshotAndSettlesArtifact(t *testing.T) {
	repository := &recordingBuildRepository{snapshot: buildstore.JobSnapshot{TaskID: 42, BuildID: "build_test", WorkspaceRoot: "/workspace/app", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", BuildArgs: []moduleapi.DockerImageBuildArg{{Name: "MODE", Value: "release"}}}}
	docker := &recordingBuildDocker{}
	raw, _ := json.Marshal(moduleapi.BuildTaskInput{BuildID: "build_test"})
	executor := &dockerfileBuildExecutor{repository: repository, docker: docker, cancels: make(map[uint64]context.CancelFunc)}
	if err := executor.Execute(context.Background(), buildStageRun{input: raw}); err != nil {
		t.Fatal(err)
	}
	if docker.input.WorkspaceRoot != "/workspace/app" || docker.input.ContextPath != "src" || repository.settledID != 42 {
		t.Fatalf("executor did not use and settle snapshot: docker=%#v settled=%d", docker.input, repository.settledID)
	}
}

func TestExecutorSettlesArtifactAfterCallerCancellation(t *testing.T) {
	repository := &recordingBuildRepository{snapshot: buildstore.JobSnapshot{TaskID: 42, BuildID: "build_test", WorkspaceRoot: "/workspace/app", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1"}}
	raw, _ := json.Marshal(moduleapi.BuildTaskInput{BuildID: "build_test"})
	executor := &dockerfileBuildExecutor{repository: repository, docker: &recordingBuildDocker{}, cancels: make(map[uint64]context.CancelFunc)}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := executor.Execute(ctx, buildStageRun{input: raw}); err != nil {
		t.Fatal(err)
	}
	if repository.settleCanceled {
		t.Fatal("artifact settlement inherited caller cancellation")
	}
	if !repository.settleDeadline {
		t.Fatal("artifact settlement has no bounded timeout")
	}
}

func TestPlatformLegRecordsArtifactWithoutSettlingFinalManifest(t *testing.T) {
	repository := &recordingBuildRepository{}
	executor := v2ExecutionPlanExecutor{repository: repository}
	plan := moduleapi.BuildExecutionPlan{ID: "plan_1", Platforms: []string{"linux/amd64", "linux/arm64"}}
	result := moduleapi.DockerImageBuildResult{Digest: "sha256:" + strings.Repeat("a", 64), SizeBytes: 7}
	if err := executor.recordPlatformArtifact(context.Background(), buildStageRun{}, plan, moduleapi.BuildPlanTaskInput{LegID: "platform-1", Platform: "linux/amd64"}, result); err != nil {
		t.Fatalf("record platform artifact: %v", err)
	}
	if len(repository.platformArtifacts) != 1 || repository.manifestSettled || len(repository.manifestInput.PlatformArtifacts) != 0 {
		t.Fatalf("platform leg unexpectedly settled a manifest: %#v", repository)
	}
}

func TestAggregateStagePublishesManifestAfterPlatformArtifacts(t *testing.T) {
	repository := &recordingBuildRepository{platformArtifacts: []moduleapi.PlatformArtifact{{LegID: "platform-1", Platform: "linux/amd64", Digest: "sha256:" + strings.Repeat("a", 64)}, {LegID: "platform-2", Platform: "linux/arm64", Digest: "sha256:" + strings.Repeat("c", 64)}}}
	adapter := &manifestExecutionAdapterStub{}
	executor := v2ExecutionPlanExecutor{repository: repository, registry: manifestRegistryStub{binding: moduleapi.RegistryPublicationBinding{AuthExecution: moduleapi.RegistryAuthExecution{Mode: moduleapi.RegistryAuthExecutionEphemeral}}}, executionAdapter: adapter}
	plan := moduleapi.BuildExecutionPlan{ID: "plan_1", RuntimeTargetID: 4, Platforms: []string{"linux/amd64", "linux/arm64"}}
	if err := executor.publishPlatformManifest(context.Background(), buildStageRun{}, plan); err != nil {
		t.Fatalf("publish platform manifest: %v", err)
	}
	if adapter.calls != 1 || !repository.manifestSettled {
		t.Fatalf("aggregate manifest settlement calls=%d settled=%t", adapter.calls, repository.manifestSettled)
	}
}

func TestSubmitRejectsInvalidInputBeforeTaskSubmission(t *testing.T) {
	invalidRequests := []SubmitRequest{
		{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "../etc", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", IdempotencyKey: "key"},
		{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "/Dockerfile", ImageRepository: "example/app", ImageTag: "v1", IdempotencyKey: "key"},
		{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "v1", BuildArgs: []moduleapi.DockerImageBuildArg{{Name: "MODE"}, {Name: "MODE"}}, IdempotencyKey: "key"},
		{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app:latest", ImageTag: "v1", IdempotencyKey: "key"},
		{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: "example/app", ImageTag: "invalid tag", IdempotencyKey: "key"},
	}

	for _, request := range invalidRequests {
		t.Run(request.ContextPath+request.DockerfilePath+request.ImageRepository+request.ImageTag, func(t *testing.T) {
			tasks := &recordingBuildTasks{}
			service, err := NewService(&recordingBuildContexts{}, tasks, tasks, &recordingBuildDocker{}, &recordingBuildRepository{})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.Submit(context.Background(), request); !errors.Is(err, errInvalidBuildRequest) {
				t.Fatalf("Submit error = %v, want invalid request", err)
			}
			if tasks.beginCalls != 0 || tasks.materializeCalls != 0 || tasks.discardCalls != 0 {
				t.Fatalf(
					"task submission lifecycle = begin:%d materialize:%d discard:%d, want all 0",
					tasks.beginCalls, tasks.materializeCalls, tasks.discardCalls,
				)
			}
		})
	}
}

func TestNormalizeSubmitRequestTrimsDockerReference(t *testing.T) {
	request, err := normalizeSubmitRequest(SubmitRequest{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContextPath: "src", DockerfilePath: "Dockerfile", ImageRepository: " example/app ", ImageTag: " v1 "})
	if err != nil {
		t.Fatal(err)
	}
	if request.ImageRepository != "example/app" || request.ImageTag != "v1" {
		t.Fatalf("normalized reference = %q:%q", request.ImageRepository, request.ImageTag)
	}
}
