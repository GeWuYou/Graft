package build

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

type v2SnapshotResolver struct{ snapshot moduleapi.WorkspaceSnapshot }

func (r v2SnapshotResolver) FreezeApplicationWorkspaceSnapshot(context.Context, string) (moduleapi.WorkspaceSnapshot, moduleapi.ApplicationBuildContext, error) {
	return r.snapshot, moduleapi.ApplicationBuildContext{ApplicationID: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ApplicationRecordID: 9, DisplayName: "app", WorkspaceRoot: r.snapshot.MaterializedRoot, RuntimeTargetID: 4, RuntimeTargetName: "Local Docker", RuntimeProvider: "docker"}, nil
}

type v2TargetReader struct {
	target moduleapi.BuildRuntimeTargetSummary
}

func (r v2TargetReader) ReadBuildTarget(context.Context, int64) (moduleapi.BuildRuntimeTargetSummary, error) {
	return r.target, nil
}

type v2TargetAssignments struct {
	allowed bool
	targets []moduleapi.BuildRuntimeTargetSummary
}

func (r v2TargetAssignments) ListAssignedBuildTargets(context.Context, uint64) ([]moduleapi.BuildRuntimeTargetSummary, error) {
	return append([]moduleapi.BuildRuntimeTargetSummary(nil), r.targets...), nil
}
func (r v2TargetAssignments) CanUseBuildTarget(context.Context, uint64, int64) (bool, error) {
	return r.allowed, nil
}

type placementTargetReader map[int64]moduleapi.BuildRuntimeTargetSummary

func (r placementTargetReader) ReadBuildTarget(_ context.Context, targetID int64) (moduleapi.BuildRuntimeTargetSummary, error) {
	target, ok := r[targetID]
	if !ok {
		return moduleapi.BuildRuntimeTargetSummary{}, errors.New("target not found")
	}
	if target.ProviderCapabilityVersion == "" {
		target.ProviderCapabilityVersion = "docker/v1"
	}
	if target.ProviderCapabilityProfile == "" {
		target.ProviderCapabilityProfile = "buildkit"
	}
	if target.BuildFeatures == nil {
		target.BuildFeatures = []string{"registry-login"}
	}
	return target, nil
}

type placementAssignments map[int64]bool

func (a placementAssignments) ListAssignedBuildTargets(context.Context, uint64) ([]moduleapi.BuildRuntimeTargetSummary, error) {
	return nil, nil
}
func (a placementAssignments) CanUseBuildTarget(_ context.Context, _ uint64, targetID int64) (bool, error) {
	return a[targetID], nil
}

type builderTelemetryReaderStub struct {
	snapshots map[int64]moduleapi.BuilderTelemetrySnapshot
	admitted  map[int64]bool
}

func (r builderTelemetryReaderStub) ListBuilderTelemetry(_ context.Context, targetIDs []int64) ([]moduleapi.BuilderTelemetrySnapshot, error) {
	results := make([]moduleapi.BuilderTelemetrySnapshot, 0, len(targetIDs))
	for _, targetID := range targetIDs {
		if snapshot, ok := r.snapshots[targetID]; ok {
			results = append(results, snapshot)
		}
	}
	return results, nil
}

func (r builderTelemetryReaderStub) ConformBuilderTelemetry(_ context.Context, targetIDs []int64) (bool, error) {
	for _, targetID := range targetIDs {
		if !r.admitted[targetID] {
			return false, nil
		}
	}
	return len(targetIDs) > 0, nil
}

type placementBuilderResources struct {
	pool       moduleapi.BuilderPool
	members    []moduleapi.BuilderInstance
	selections []moduleapi.BuilderInstance
	next       int
}

type selectorBuilderResources struct {
	placementBuilderResources
	membersByPool map[string][]moduleapi.BuilderInstance
}

func (r *selectorBuilderResources) ListBuilderPoolMembers(_ context.Context, poolID string) ([]moduleapi.BuilderInstance, error) {
	return append([]moduleapi.BuilderInstance(nil), r.membersByPool[poolID]...), nil
}

type selectorRepository struct {
	*recordingBuildRepository
	pools []moduleapi.BuilderPool
}

func (r *selectorRepository) ListBuilderPools(context.Context) ([]moduleapi.BuilderPool, error) {
	return append([]moduleapi.BuilderPool(nil), r.pools...), nil
}

func (*placementBuilderResources) CreateBuilderProfile(context.Context, moduleapi.BuilderProfile, uint64) error {
	return errors.New("not implemented")
}
func (*placementBuilderResources) CreateBuilderInstance(context.Context, moduleapi.BuilderInstance, uint64) error {
	return errors.New("not implemented")
}
func (*placementBuilderResources) CreateBuilderPool(context.Context, moduleapi.BuilderPool, uint64) error {
	return errors.New("not implemented")
}
func (*placementBuilderResources) ReplaceBuilderPoolMembers(context.Context, string, []moduleapi.BuilderPoolMember, uint64) error {
	return errors.New("not implemented")
}
func (r *placementBuilderResources) GetBuilderPool(context.Context, string) (moduleapi.BuilderPool, error) {
	return r.pool, nil
}
func (r *placementBuilderResources) ListBuilderPoolMembers(context.Context, string) ([]moduleapi.BuilderInstance, error) {
	return append([]moduleapi.BuilderInstance(nil), r.members...), nil
}
func (r *placementBuilderResources) SelectRoundRobinBuilderInstance(context.Context, string) (moduleapi.BuilderPoolSelection, error) {
	if r.next >= len(r.selections) {
		return moduleapi.BuilderPoolSelection{}, errors.New("no selected builder")
	}
	instance := r.selections[r.next]
	cursor := int64(r.next)
	r.next++
	return moduleapi.BuilderPoolSelection{Instance: instance, Cursor: &cursor}, nil
}

type v2RegistryResolver struct{}

func (v2RegistryResolver) ResolveArtifactDestination(_ context.Context, _ uint64, destination moduleapi.BuildDestination) (moduleapi.AuthorizedArtifactDestination, error) {
	return moduleapi.AuthorizedArtifactDestination(destination), nil
}

func TestSubmitArtifactPromotionFreezesAuthorizedDigestSourceForTaskRuntime(t *testing.T) {
	digest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	source := moduleapi.ArtifactPublicationSource{ArtifactID: "artifact_1", PublicationID: "publication_1", Digest: digest, MediaType: "application/vnd.oci.image.manifest.v1+json", DestinationKind: "oci_registry", ConnectionRef: "registry:source", RepositoryRef: "team/source"}
	repository := &recordingBuildRepository{publicationSources: []moduleapi.ArtifactPublicationSource{source}}
	tasks := &recordingBuildTasks{}
	service, err := NewService(&recordingBuildContexts{}, tasks, tasks, repository)
	if err != nil {
		t.Fatal(err)
	}
	registry := &promotionRegistryStub{}
	service.ConfigureArtifactPromotion(tasks, registry, v2TargetAssignments{allowed: true})
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})
	receipt, err := service.SubmitArtifactPromotion(ctx, ArtifactPromotionRequest{ArtifactID: source.ArtifactID, PublicationID: source.PublicationID, Destination: moduleapi.BuildDestination{Kind: "oci_registry", ConnectionRef: "registry:destination", RepositoryRef: "team/destination", Reference: "stable"}, RuntimeTargetID: 4, IdempotencyKey: "promote-once"})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.TaskID != 42 || tasks.submitCalls != 1 || tasks.input.IdempotencyKey != "promote-once" || tasks.input.Type != artifactPromotionTaskType || tasks.input.Plan.Stages[0].RecoveryPolicy != moduleapi.StageRecoveryManualReconcile {
		t.Fatalf("promotion task = %#v receipt=%#v", tasks.input, receipt)
	}
	var input moduleapi.ArtifactPromotionTaskInput
	if err := json.Unmarshal(tasks.input.Input, &input); err != nil {
		t.Fatal(err)
	}
	if input.Source != source || input.RuntimeTargetID != 4 || input.Destination.ConnectionRef != "registry:destination" || strings.Contains(string(tasks.input.Input), "endpoint") || strings.Contains(string(tasks.input.Input), "credential") {
		t.Fatalf("frozen promotion input = %s", tasks.input.Input)
	}
}

func TestSubmitExecutionPlanFreezesV2ReferencesWithoutTaskPathLeakage(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tasks := &recordingBuildTasks{}
	repository := &recordingBuildRepository{}
	service, err := NewService(&recordingBuildContexts{}, tasks, tasks, repository)
	if err != nil {
		t.Fatal(err)
	}
	service.ConfigureV2Submission(
		v2SnapshotResolver{snapshot: moduleapi.WorkspaceSnapshot{ID: "application:app:snapshot", SourceKind: "application_workspace", SourceReference: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContentDigest: "sha256:source", MaterializedRoot: source, CreatedAt: time.Now().UTC()}},
		v2TargetReader{target: moduleapi.BuildRuntimeTargetSummary{ID: 4, Available: true, ProviderCapabilityProfile: "oci-build", ProviderCapabilityVersion: "docker/v1", SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}, BuildFeatures: []string{"registry-login"}}},
		v2TargetAssignments{allowed: true},
		v2RegistryResolver{},
	)
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})
	receipt, err := service.SubmitExecutionPlan(ctx, ExecutionPlanRequest{WorkspaceID: "workspace_app", RuntimeTargetID: 4, TemplateRef: v2DockerfileTemplate, Driver: v2DockerEngineDriver, Platforms: []string{"linux/amd64"}, Destination: moduleapi.BuildDestination{Kind: v2OCIDestination, ConnectionRef: "registry:primary", RepositoryRef: "team/app", Reference: "v1"}, IdempotencyKey: "v2-key"})
	if err != nil {
		t.Fatal(err)
	}
	assertFrozenV2Submission(t, receipt, repository.v2Plan)
	if repository.v2Plan.Workspace.MaterializedRoot != "" || repository.v2Plan.Workspace.MaterializationRef == "" {
		t.Fatalf("persisted snapshot must retain only opaque materialization reference: %#v", repository.v2Plan.Workspace)
	}
	t.Cleanup(func() {
		_ = releaseMaterialization(context.Background(), repository.v2Plan.Workspace.MaterializationRef)
	})
	var input moduleapi.BuildPlanTaskInput
	if err := json.Unmarshal(tasks.input.Input, &input); err != nil {
		t.Fatal(err)
	}
	assertFrozenV2TaskInput(t, input, repository.v2Plan.ID, tasks.input.Input)
	placement, found := repository.v2Plan.PlacementForPlatform("linux/amd64")
	assertFrozenV2Placement(t, found, placement.SchedulingEvidence)
}

func assertFrozenV2Submission(t *testing.T, receipt moduleapi.TaskReceipt, plan moduleapi.BuildExecutionPlan) {
	t.Helper()
	if receipt.TaskID != 42 || plan.ID == "" || !strings.HasPrefix(plan.ID, "plan_") {
		t.Fatalf("submission did not materialize a frozen plan: receipt=%#v plan=%#v", receipt, plan)
	}
	if plan.CachePolicy != "disabled" || plan.SecurityPolicy != "default" {
		t.Fatalf("resolved policies were not frozen: %#v", plan)
	}
}

func assertFrozenV2TaskInput(t *testing.T, input moduleapi.BuildPlanTaskInput, planID string, raw []byte) {
	t.Helper()
	if input.BuildID != planID || input.ExecutionPlanID != planID || strings.Contains(string(raw), "/workspace/app") || strings.Contains(string(raw), "team/app") {
		t.Fatalf("task metadata leaked materialization or destination: %s", raw)
	}
}

func assertFrozenV2Placement(t *testing.T, found bool, evidence json.RawMessage) {
	t.Helper()
	if !found || !strings.Contains(string(evidence), `"capability_negotiation"`) || !strings.Contains(string(evidence), `"ProviderCapabilityVersion":"docker/v1"`) {
		t.Fatalf("capability negotiation was not frozen: %#v", evidence)
	}
}

func TestListBuilderPoolsFiltersByRuntimeTargetAssignmentAndPhaseFourPolicyGate(t *testing.T) {
	repository := &selectorRepository{recordingBuildRepository: &recordingBuildRepository{}, pools: []moduleapi.BuilderPool{
		{ID: "pool:allowed", DisplayName: "Allowed", SchedulingPolicy: "round_robin"},
		{ID: "pool:hidden", DisplayName: "Hidden", SchedulingPolicy: "round_robin"},
		{ID: "pool:least-load", DisplayName: "Least Load", SchedulingPolicy: "least_load"},
		{ID: "pool:capacity", DisplayName: "Capacity", SchedulingPolicy: "capacity"},
		{ID: "pool:affinity", DisplayName: "Affinity", SchedulingPolicy: "affinity"},
	}}
	service, err := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, repository)
	if err != nil {
		t.Fatal(err)
	}
	service.builderResources = &selectorBuilderResources{membersByPool: map[string][]moduleapi.BuilderInstance{
		"pool:allowed": {{RuntimeTargetID: 4}},
		"pool:hidden":  {{RuntimeTargetID: 8}},
	}}
	service.buildAssignments = placementAssignments{4: true, 8: false}

	items, err := service.ListBuilderPools(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "pool:allowed" {
		t.Fatalf("unexpected visible pools: %#v", items)
	}
}

func TestSubmitExecutionPlanRejectsUnassignedOrIncompatibleTarget(t *testing.T) {
	service, _ := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildRepository{})
	service.ConfigureV2Submission(v2SnapshotResolver{snapshot: moduleapi.WorkspaceSnapshot{ID: "snapshot", ContentDigest: "sha256:source", MaterializedRoot: "/workspace/app"}}, v2TargetReader{target: moduleapi.BuildRuntimeTargetSummary{ID: 4, Available: true, SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}}}, v2TargetAssignments{allowed: false}, v2RegistryResolver{})
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})
	_, err := service.SubmitExecutionPlan(ctx, ExecutionPlanRequest{WorkspaceID: "workspace_app", RuntimeTargetID: 4, TemplateRef: v2DockerfileTemplate, Driver: v2DockerEngineDriver, Destination: moduleapi.BuildDestination{Kind: v2OCIDestination, ConnectionRef: "registry", RepositoryRef: "team/app", Reference: "v1"}, IdempotencyKey: "key"})
	if err == nil || !strings.Contains(err.Error(), "not assigned") {
		t.Fatalf("error = %v, want rejected target assignment", err)
	}
}

func TestSubmitExecutionPlanRejectsMultiPlatformRequestWithoutBuildxDriver(t *testing.T) {
	service, _ := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildRepository{})
	service.ConfigureV2Submission(
		v2SnapshotResolver{snapshot: moduleapi.WorkspaceSnapshot{ID: "snapshot", ContentDigest: "sha256:source", MaterializedRoot: "/workspace/app"}},
		v2TargetReader{target: moduleapi.BuildRuntimeTargetSummary{ID: 4, Available: true, SupportedDrivers: []string{"docker-engine", "docker-buildx"}, SupportedPlatforms: []string{"linux/amd64", "linux/arm64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}}},
		v2TargetAssignments{allowed: true},
		v2RegistryResolver{},
	)
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})
	_, err := service.SubmitExecutionPlan(ctx, ExecutionPlanRequest{WorkspaceID: "workspace_app", BuilderPoolID: "pool:default", TemplateRef: v2DockerfileTemplate, Driver: v2DockerEngineDriver, Platforms: []string{"linux/amd64", "linux/arm64"}, Destination: moduleapi.BuildDestination{Kind: v2OCIDestination, ConnectionRef: "registry", RepositoryRef: "team/app", Reference: "v1"}, IdempotencyKey: "multi-platform-engine"})
	if err == nil || !strings.Contains(err.Error(), "docker-buildx") {
		t.Fatalf("error = %v, want docker-buildx rejection", err)
	}
}

func TestFreezeExecutionPlanRecordsPoolAndInstanceSelection(t *testing.T) {
	snapshot := moduleapi.WorkspaceSnapshot{ID: "snapshot", ContentDigest: "sha256:source", MaterializedRoot: "/managed/snapshot"}
	request := ExecutionPlanRequest{WorkspaceID: "workspace_app", BuilderPoolID: "pool:default", RuntimeTargetID: 4, BuilderPlacements: []moduleapi.BuilderPlacement{{Platform: "linux/amd64", BuilderInstanceID: "instance:docker-a", RuntimeTargetID: 4, SchedulingPolicy: "round_robin"}}, TemplateRef: v2DockerfileTemplate, Driver: v2DockerEngineDriver, Platforms: []string{"linux/amd64"}, Destination: moduleapi.BuildDestination{Kind: v2OCIDestination, ConnectionRef: "registry", RepositoryRef: "team/app", Reference: "v1"}}
	plan, err := freezeExecutionPlan(snapshot, request, "instance:docker-a")
	if err != nil {
		t.Fatal(err)
	}
	if plan.BuilderPoolID != "pool:default" || plan.BuilderInstanceID != "instance:docker-a" || plan.RuntimeTargetID != 4 {
		t.Fatalf("pool selection was not frozen: %#v", plan)
	}
	if plan.Digest == "" {
		t.Fatal("expected a digest for the pool-bound execution plan")
	}
	if placement, ok := plan.PlacementForPlatform("linux/amd64"); !ok || placement.BuilderInstanceID != "instance:docker-a" || placement.SchedulingPolicy != "round_robin" {
		t.Fatalf("placement was not frozen: %#v ok=%t", placement, ok)
	}
}

func TestFreezeExecutionPlanDigestIncludesPlatformPlacements(t *testing.T) {
	snapshot := moduleapi.WorkspaceSnapshot{ID: "snapshot", ContentDigest: "sha256:source", MaterializedRoot: "/managed/snapshot"}
	base := ExecutionPlanRequest{WorkspaceID: "workspace_app", BuilderPoolID: "pool:default", RuntimeTargetID: 4, TemplateRef: v2DockerfileTemplate, Driver: "docker-buildx@v1", Platforms: []string{"linux/amd64", "linux/arm64"}, Destination: moduleapi.BuildDestination{Kind: v2OCIDestination, ConnectionRef: "registry", RepositoryRef: "team/app", Reference: "v1"}}
	base.BuilderPlacements = []moduleapi.BuilderPlacement{{Platform: "linux/amd64", BuilderInstanceID: "builder-a", RuntimeTargetID: 4, SchedulingPolicy: "round_robin"}, {Platform: "linux/arm64", BuilderInstanceID: "builder-b", RuntimeTargetID: 5, SchedulingPolicy: "round_robin"}}
	first, err := freezeExecutionPlan(snapshot, base, "builder-a")
	if err != nil {
		t.Fatal(err)
	}
	base.BuilderPlacements[1].RuntimeTargetID = 6
	second, err := freezeExecutionPlan(snapshot, base, "builder-a")
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest == second.Digest {
		t.Fatal("placement change must produce a distinct execution plan digest")
	}
}

func TestSelectBuilderPlacementsFromPoolFreezesDifferentTargetsPerPlatform(t *testing.T) {
	resources := &placementBuilderResources{
		pool: moduleapi.BuilderPool{ID: "pool-buildx", SchedulingPolicy: "round_robin"},
		members: []moduleapi.BuilderInstance{
			{ID: "builder-amd64", RuntimeTargetID: 4, Status: "ready", DriverRef: "docker-buildx", DriverVersion: "v1"},
			{ID: "builder-arm64", RuntimeTargetID: 5, Status: "ready", DriverRef: "docker-buildx", DriverVersion: "v1"},
		},
		selections: []moduleapi.BuilderInstance{
			{ID: "builder-amd64", RuntimeTargetID: 4, DriverRef: "docker-buildx", DriverVersion: "v1"},
			{ID: "builder-arm64", RuntimeTargetID: 5, DriverRef: "docker-buildx", DriverVersion: "v1"},
		},
	}
	service, err := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildRepository{})
	if err != nil {
		t.Fatal(err)
	}
	service.builderResources = resources
	service.buildTargets = placementTargetReader{
		4: {ID: 4, Available: true, SupportedDrivers: []string{"docker-buildx"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}},
		5: {ID: 5, Available: true, SupportedDrivers: []string{"docker-buildx"}, SupportedPlatforms: []string{"linux/arm64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}},
	}
	service.buildAssignments = placementAssignments{4: true, 5: true}
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})
	placements, err := service.SelectBuilderPlacementsFromPool(ctx, "pool-buildx", "docker-buildx@v1", []string{"linux/amd64", "linux/arm64"})
	if err != nil {
		t.Fatal(err)
	}
	if len(placements) != 2 || placements[0].RuntimeTargetID != 4 || placements[1].RuntimeTargetID != 5 || placements[0].BuilderInstanceID != "builder-amd64" || placements[1].BuilderInstanceID != "builder-arm64" {
		t.Fatalf("placements = %#v", placements)
	}
}

//nolint:cyclop,gocyclo // This integration seam keeps candidate admission, telemetry and frozen evidence in one scenario.
func TestSelectBuilderPlacementsFromPoolUsesConformantTelemetryAndFreezesDynamicEvidence(t *testing.T) {
	now := time.Now().UTC()
	resources := &placementBuilderResources{pool: moduleapi.BuilderPool{ID: "pool-least-load", SchedulingPolicy: "least_load"}, members: []moduleapi.BuilderInstance{
		{ID: "builder-small", RuntimeTargetID: 4, Status: "ready", DriverRef: "docker-engine", DriverVersion: "v1"},
		{ID: "builder-large", RuntimeTargetID: 5, Status: "ready", DriverRef: "docker-engine", DriverVersion: "v1"},
		{ID: "builder-rejected", RuntimeTargetID: 6, Status: "ready", DriverRef: "docker-engine", DriverVersion: "v1"},
	}}
	service, err := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildRepository{})
	if err != nil {
		t.Fatal(err)
	}
	service.builderResources = resources
	service.buildTargets = placementTargetReader{
		4: {ID: 4, Available: true, ProviderCapabilityProfile: "buildkit", ProviderCapabilityVersion: "v1", SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}},
		5: {ID: 5, Available: true, ProviderCapabilityProfile: "buildkit", ProviderCapabilityVersion: "v2", SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}},
		6: {ID: 6, Available: true, ProviderCapabilityProfile: "buildkit", ProviderCapabilityVersion: "v1", SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}},
	}
	service.buildAssignments = placementAssignments{4: true, 5: true, 6: true}
	service.ConfigureBuilderTelemetry(builderTelemetryReaderStub{admitted: map[int64]bool{4: true, 5: true, 6: true}, snapshots: map[int64]moduleapi.BuilderTelemetrySnapshot{
		4: {TargetID: 4, BuilderScope: "builder:small", ProviderID: "agent", CapabilityProfile: "buildkit", CapabilityVersion: "v1", Available: true, Running: 1, Queued: 0, AllocatableSlots: 1, ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute), SourceRef: "report:small", Provenance: "agent", Integrity: "sha256:small"},
		5: {TargetID: 5, BuilderScope: "builder:large", ProviderID: "agent", CapabilityProfile: "buildkit", CapabilityVersion: "v2", Available: true, Running: 2, Queued: 0, AllocatableSlots: 4, ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute), SourceRef: "report:large", Provenance: "agent", Integrity: "sha256:large"},
		6: {TargetID: 6, BuilderScope: "builder:zero", ProviderID: "agent", CapabilityProfile: "buildkit", CapabilityVersion: "v1", Available: true, Running: 0, Queued: 0, AllocatableSlots: 0, ObservedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Minute), SourceRef: "report:zero", Provenance: "agent", Integrity: "sha256:zero"},
	}})
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})
	placements, err := service.SelectBuilderPlacementsFromPool(ctx, "pool-least-load", v2DockerEngineDriver, []string{"linux/amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if len(placements) != 1 || placements[0].BuilderInstanceID != "builder-small" || placements[0].RuntimeTargetID != 4 {
		t.Fatalf("dynamic placement = %#v", placements)
	}
	var evidence dynamicPlacementEvidence
	if err := json.Unmarshal(placements[0].SchedulingEvidence, &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.PolicyID != "build.pool.least_load" || evidence.PolicyVersion != "v1" || evidence.SelectedInstanceID != "builder-small" || evidence.Telemetry.SourceRef != "report:small" || evidence.Telemetry.CapabilityVersion != "v1" || evidence.Telemetry.Integrity != "sha256:small" || evidence.CandidateFingerprint == "" || evidence.ReservationSlotBudget != 1 || evidence.ReservationObservedAt.IsZero() || evidence.CapabilityRequirementFingerprint == "" || evidence.CapabilityProfile != "buildkit" || evidence.CapabilityVersion != "v1" || evidence.CapabilityNegotiation.SnapshotDeliveryMode == "" {
		t.Fatalf("frozen dynamic evidence = %#v", evidence)
	}
}

func TestSelectBuilderPlacementsFromPoolRejectsDynamicPolicyWithoutConformantTelemetry(t *testing.T) {
	resources := &placementBuilderResources{pool: moduleapi.BuilderPool{ID: "pool-load", SchedulingPolicy: "least_load"}, members: []moduleapi.BuilderInstance{{ID: "builder-a", RuntimeTargetID: 4, Status: "ready", DriverRef: "docker-engine", DriverVersion: "v1"}}}
	service, err := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildRepository{})
	if err != nil {
		t.Fatal(err)
	}
	service.builderResources = resources
	service.buildTargets = placementTargetReader{4: {ID: 4, Available: true, SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}}}
	service.buildAssignments = placementAssignments{4: true}
	service.ConfigureBuilderTelemetry(builderTelemetryReaderStub{admitted: map[int64]bool{4: false}})
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})
	if _, err := service.SelectBuilderPlacementsFromPool(ctx, "pool-load", v2DockerEngineDriver, []string{"linux/amd64"}); err == nil || !strings.Contains(err.Error(), "dynamically conformant") {
		t.Fatalf("dynamic placement error = %v", err)
	}
}

func TestSelectBuilderFromPoolUsesDeterministicLabelSelector(t *testing.T) {
	resources := &placementBuilderResources{
		pool: moduleapi.BuilderPool{ID: "pool-labels", SchedulingPolicy: "manual", Selector: json.RawMessage(`{"instance_id":"builder-east","labels":{"region":"us-east"}}`)},
		members: []moduleapi.BuilderInstance{
			{ID: "builder-west", RuntimeTargetID: 4, Status: "ready", Labels: map[string]string{"region": "us-west"}, DriverRef: "docker-engine", DriverVersion: "v1"},
			{ID: "builder-east", RuntimeTargetID: 5, Status: "ready", Labels: map[string]string{"region": "us-east"}, DriverRef: "docker-engine", DriverVersion: "v1"},
		},
	}
	service, err := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildRepository{})
	if err != nil {
		t.Fatal(err)
	}
	service.builderResources = resources
	service.buildTargets = placementTargetReader{4: {ID: 4, Available: true, SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}}, 5: {ID: 5, Available: true, SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}}}
	service.buildAssignments = placementAssignments{5: true}
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})
	instance, err := service.SelectBuilderFromPool(ctx, "pool-labels", v2DockerEngineDriver, []string{"linux/amd64"})
	if err != nil {
		t.Fatal(err)
	}
	if instance.ID != "builder-east" {
		t.Fatalf("selected instance = %#v", instance)
	}
	placements, err := service.SelectBuilderPlacementsFromPool(ctx, "pool-labels", v2DockerEngineDriver, []string{"linux/amd64"})
	if err != nil || len(placements) != 1 || !strings.Contains(string(placements[0].SchedulingEvidence), `"labels":{"region":"us-east"}`) {
		t.Fatalf("scheduling evidence = %#v, err=%v", placements, err)
	}
}

func TestSelectBuilderFromPoolRandomIsDeterministicAndFreezesSeed(t *testing.T) {
	resources := &placementBuilderResources{
		pool: moduleapi.BuilderPool{ID: "pool-random", SchedulingPolicy: "random"},
		members: []moduleapi.BuilderInstance{
			{ID: "builder-a", RuntimeTargetID: 4, Status: "ready", DriverRef: "docker-engine", DriverVersion: "v1"},
			{ID: "builder-b", RuntimeTargetID: 5, Status: "ready", DriverRef: "docker-engine", DriverVersion: "v1"},
		},
	}
	service, err := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildRepository{})
	if err != nil {
		t.Fatal(err)
	}
	service.builderResources = resources
	service.buildTargets = placementTargetReader{4: {ID: 4, Available: true, SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}}, 5: {ID: 5, Available: true, SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}}}
	service.buildAssignments = placementAssignments{4: true, 5: true}
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})
	first, err := service.SelectBuilderPlacementsFromPool(ctx, "pool-random", v2DockerEngineDriver, []string{"linux/amd64"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.SelectBuilderPlacementsFromPool(ctx, "pool-random", v2DockerEngineDriver, []string{"linux/amd64"})
	if err != nil || len(first) != 1 || len(second) != 1 || first[0].BuilderInstanceID != second[0].BuilderInstanceID || !strings.Contains(string(first[0].SchedulingEvidence), `"seed":"`) {
		t.Fatalf("random placement is not deterministic/evidenced: first=%#v second=%#v err=%v", first, second, err)
	}
}

func TestNormalizeExecutionPlanRequestRequiresExactlyOneBuilderSelector(t *testing.T) {
	base := ExecutionPlanRequest{WorkspaceID: "workspace_app", TemplateRef: v2DockerfileTemplate, Driver: v2DockerEngineDriver, Destination: moduleapi.BuildDestination{Kind: v2OCIDestination, ConnectionRef: "registry", RepositoryRef: "team/app", Reference: "v1"}}
	if _, err := normalizeExecutionPlanRequest(base); err == nil {
		t.Fatal("expected missing builder selector to be rejected")
	}
	base.RuntimeTargetID, base.BuilderPoolID = 4, "pool:default"
	if _, err := normalizeExecutionPlanRequest(base); err == nil {
		t.Fatal("expected ambiguous builder selectors to be rejected")
	}
}

func TestNormalizeExecutionPlanRequestFreezesOnlySupportedResolvedPolicies(t *testing.T) {
	base := ExecutionPlanRequest{WorkspaceID: "workspace_app", RuntimeTargetID: 4, TemplateRef: v2DockerfileTemplate, Driver: v2DockerEngineDriver, Destination: moduleapi.BuildDestination{Kind: v2OCIDestination, ConnectionRef: "registry", RepositoryRef: "team/app", Reference: "v1"}}
	normalized, err := normalizeExecutionPlanRequest(base)
	if err != nil || normalized.CachePolicy != "disabled" || normalized.SecurityPolicy != "default" {
		t.Fatalf("default resolved policies = %#v, err=%v", normalized, err)
	}
	base.CachePolicy = "registry-import"
	if _, err := normalizeExecutionPlanRequest(base); err == nil {
		t.Fatal("expected unsupported cache policy to be rejected")
	}
}

func TestFreezeExecutionPlanDigestIncludesResolvedPolicies(t *testing.T) {
	base := ExecutionPlanRequest{WorkspaceID: "workspace_app", RuntimeTargetID: 4, BuilderPlacements: []moduleapi.BuilderPlacement{{Platform: "linux/amd64", BuilderInstanceID: "runtime-target:4", RuntimeTargetID: 4, SchedulingPolicy: "manual"}}, TemplateRef: v2DockerfileTemplate, Driver: v2DockerEngineDriver, CachePolicy: "disabled", SecurityPolicy: "default", Platforms: []string{"linux/amd64"}, Destination: moduleapi.BuildDestination{Kind: v2OCIDestination, ConnectionRef: "registry", RepositoryRef: "team/app", Reference: "v1"}}
	snapshot := moduleapi.WorkspaceSnapshot{ID: "snapshot_1", ContentDigest: "sha256:source", MaterializedRoot: "/managed/snapshot"}
	first, err := freezeExecutionPlan(snapshot, base, "runtime-target:4")
	if err != nil {
		t.Fatal(err)
	}
	changed := base
	changed.SecurityPolicy = "provenance-required"
	second, err := freezeExecutionPlan(snapshot, changed, "runtime-target:4")
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest == second.Digest {
		t.Fatal("expected resolved security policy to change plan digest")
	}
}

func TestSubmitExecutionPlanRequiresPoolForMultiPlatformBuild(t *testing.T) {
	request := ExecutionPlanRequest{WorkspaceID: "workspace_app", RuntimeTargetID: 4, TemplateRef: v2DockerfileTemplate, Driver: v2DockerEngineDriver, Platforms: []string{"linux/amd64", "linux/arm64"}, Destination: moduleapi.BuildDestination{Kind: v2OCIDestination, ConnectionRef: "registry", RepositoryRef: "team/app", Reference: "v1"}}
	if _, err := normalizeExecutionPlanRequest(request); err != nil {
		t.Fatalf("normalization should preserve valid platform set: %v", err)
	}
}
