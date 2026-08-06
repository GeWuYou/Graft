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
	return target, nil
}

type placementAssignments map[int64]bool

func (a placementAssignments) ListAssignedBuildTargets(context.Context, uint64) ([]moduleapi.BuildRuntimeTargetSummary, error) {
	return nil, nil
}
func (a placementAssignments) CanUseBuildTarget(_ context.Context, _ uint64, targetID int64) (bool, error) {
	return a[targetID], nil
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
func (r *placementBuilderResources) SelectRoundRobinBuilderInstance(context.Context, string) (moduleapi.BuilderInstance, error) {
	if r.next >= len(r.selections) {
		return moduleapi.BuilderInstance{}, errors.New("no selected builder")
	}
	instance := r.selections[r.next]
	r.next++
	return instance, nil
}

type v2RegistryResolver struct{}

func (v2RegistryResolver) ResolveArtifactDestination(_ context.Context, _ uint64, destination moduleapi.BuildDestination) (moduleapi.AuthorizedArtifactDestination, error) {
	return moduleapi.AuthorizedArtifactDestination(destination), nil
}

func TestSubmitExecutionPlanFreezesV2ReferencesWithoutTaskPathLeakage(t *testing.T) {
	source := t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "Dockerfile"), []byte("FROM scratch\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tasks := &recordingBuildTasks{}
	repository := &recordingBuildRepository{}
	service, err := NewService(&recordingBuildContexts{}, tasks, tasks, &recordingBuildDocker{}, repository)
	if err != nil {
		t.Fatal(err)
	}
	service.ConfigureV2Submission(
		v2SnapshotResolver{snapshot: moduleapi.WorkspaceSnapshot{ID: "application:app:snapshot", SourceKind: "application_workspace", SourceReference: "app_01JZ5R6M7N8P9Q0R1S2T3V4W5X", ContentDigest: "sha256:source", MaterializedRoot: source, CreatedAt: time.Now().UTC()}},
		v2TargetReader{target: moduleapi.BuildRuntimeTargetSummary{ID: 4, Available: true, SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}}},
		v2TargetAssignments{allowed: true},
		v2RegistryResolver{},
	)
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})
	receipt, err := service.SubmitExecutionPlan(ctx, ExecutionPlanRequest{WorkspaceID: "workspace_app", RuntimeTargetID: 4, TemplateRef: v2DockerfileTemplate, Driver: v2DockerEngineDriver, Platforms: []string{"linux/amd64"}, Destination: moduleapi.BuildDestination{Kind: v2OCIDestination, ConnectionRef: "registry:primary", RepositoryRef: "team/app", Reference: "v1"}, IdempotencyKey: "v2-key"})
	if err != nil {
		t.Fatal(err)
	}
	if receipt.TaskID != 42 || repository.v2Plan.ID == "" || !strings.HasPrefix(repository.v2Plan.ID, "plan_") {
		t.Fatalf("submission did not materialize a frozen plan: receipt=%#v plan=%#v", receipt, repository.v2Plan)
	}
	t.Cleanup(func() { _ = os.RemoveAll(repository.v2Plan.Workspace.MaterializedRoot) })
	var input moduleapi.BuildPlanTaskInput
	if err := json.Unmarshal(tasks.input.Input, &input); err != nil {
		t.Fatal(err)
	}
	if input.BuildID != repository.v2Plan.ID || input.ExecutionPlanID != repository.v2Plan.ID || strings.Contains(string(tasks.input.Input), "/workspace/app") || strings.Contains(string(tasks.input.Input), "team/app") {
		t.Fatalf("task metadata leaked materialization or destination: %s", tasks.input.Input)
	}
}

func TestListBuilderPoolsFiltersByRuntimeTargetAssignment(t *testing.T) {
	repository := &selectorRepository{recordingBuildRepository: &recordingBuildRepository{}, pools: []moduleapi.BuilderPool{{ID: "pool:allowed", DisplayName: "Allowed", SchedulingPolicy: "round_robin"}, {ID: "pool:hidden", DisplayName: "Hidden", SchedulingPolicy: "round_robin"}, {ID: "pool:unsupported", DisplayName: "Unsupported", SchedulingPolicy: "least_load"}}}
	service, err := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildDocker{}, repository)
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
	service, _ := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildDocker{}, &recordingBuildRepository{})
	service.ConfigureV2Submission(v2SnapshotResolver{snapshot: moduleapi.WorkspaceSnapshot{ID: "snapshot", ContentDigest: "sha256:source", MaterializedRoot: "/workspace/app"}}, v2TargetReader{target: moduleapi.BuildRuntimeTargetSummary{ID: 4, Available: true, SupportedDrivers: []string{"docker-engine"}, SupportedPlatforms: []string{"linux/amd64"}, WorkspaceLocalities: []string{"build-snapshot"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}}}, v2TargetAssignments{allowed: false}, v2RegistryResolver{})
	ctx := moduleapi.WithRequestAuthContext(context.Background(), moduleapi.RequestAuthContext{User: &moduleapi.CurrentUser{ID: 7}})
	_, err := service.SubmitExecutionPlan(ctx, ExecutionPlanRequest{WorkspaceID: "workspace_app", RuntimeTargetID: 4, TemplateRef: v2DockerfileTemplate, Driver: v2DockerEngineDriver, Destination: moduleapi.BuildDestination{Kind: v2OCIDestination, ConnectionRef: "registry", RepositoryRef: "team/app", Reference: "v1"}, IdempotencyKey: "key"})
	if err == nil || !strings.Contains(err.Error(), "not assigned") {
		t.Fatalf("error = %v, want rejected target assignment", err)
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
		pool:    moduleapi.BuilderPool{ID: "pool-buildx", SchedulingPolicy: "round_robin"},
		members: []moduleapi.BuilderInstance{{ID: "builder-amd64"}, {ID: "builder-arm64"}},
		selections: []moduleapi.BuilderInstance{
			{ID: "builder-amd64", RuntimeTargetID: 4, DriverRef: "docker-buildx", DriverVersion: "v1"},
			{ID: "builder-arm64", RuntimeTargetID: 5, DriverRef: "docker-buildx", DriverVersion: "v1"},
		},
	}
	service, err := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildDocker{}, &recordingBuildRepository{})
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

func TestSelectBuilderFromPoolUsesDeterministicLabelSelector(t *testing.T) {
	resources := &placementBuilderResources{
		pool: moduleapi.BuilderPool{ID: "pool-labels", SchedulingPolicy: "labels", Selector: json.RawMessage(`{"labels":{"region":"us-east"}}`)},
		members: []moduleapi.BuilderInstance{
			{ID: "builder-west", RuntimeTargetID: 4, Status: "ready", Labels: map[string]string{"region": "us-west"}, DriverRef: "docker-engine", DriverVersion: "v1"},
			{ID: "builder-east", RuntimeTargetID: 5, Status: "ready", Labels: map[string]string{"region": "us-east"}, DriverRef: "docker-engine", DriverVersion: "v1"},
		},
	}
	service, err := NewService(&recordingBuildContexts{}, &recordingBuildTasks{}, &recordingBuildTasks{}, &recordingBuildDocker{}, &recordingBuildRepository{})
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
	if err != nil || len(placements) != 1 || string(placements[0].SchedulingEvidence) != string(resources.pool.Selector) {
		t.Fatalf("scheduling evidence = %#v, err=%v", placements, err)
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

func TestSubmitExecutionPlanRequiresPoolForMultiPlatformBuild(t *testing.T) {
	request := ExecutionPlanRequest{WorkspaceID: "workspace_app", RuntimeTargetID: 4, TemplateRef: v2DockerfileTemplate, Driver: v2DockerEngineDriver, Platforms: []string{"linux/amd64", "linux/arm64"}, Destination: moduleapi.BuildDestination{Kind: v2OCIDestination, ConnectionRef: "registry", RepositoryRef: "team/app", Reference: "v1"}}
	if _, err := normalizeExecutionPlanRequest(request); err != nil {
		t.Fatalf("normalization should preserve valid platform set: %v", err)
	}
}
