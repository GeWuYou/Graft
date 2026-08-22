package project

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
	projectcontract "graft/server/modules/project/contract"
	projectstore "graft/server/modules/project/store"
)

type executionTestRepository struct {
	projectstore.Repository
	aggregate projectstore.ApplicationAggregate
}

func (r executionTestRepository) Get(context.Context, uint64) (projectstore.ApplicationAggregate, error) {
	return r.aggregate, nil
}

func (r executionTestRepository) GetByApplicationID(_ context.Context, applicationID string) (projectstore.ApplicationAggregate, error) {
	if applicationID != r.aggregate.Application.ApplicationID {
		return projectstore.ApplicationAggregate{}, projectstore.ErrApplicationNotFound
	}
	return r.aggregate, nil
}

func TestLifecycleTaskPlanUsesExternalProviderNeutralLease(t *testing.T) {
	aggregate := applicationExecutionAggregate()
	plan, err := lifecycleTaskPlan(aggregate, generated.ApplicationActionResponseActionApplicationActionUp, applicationExecutionTarget(), 9, nil)
	if err != nil {
		t.Fatalf("build lifecycle plan: %v", err)
	}
	if len(plan.Stages) != 1 {
		t.Fatalf("expected one stage, got %#v", plan.Stages)
	}
	assertApplicationUpExternalLease(t, plan.Stages[0])
}

func assertApplicationUpExternalLease(t *testing.T, stage moduleapi.StagePlan) {
	t.Helper()
	if stage.ExecutorType != "application.compose.up" || stage.ExternalExecution == nil {
		t.Fatalf("expected external compose stage, got %#v", stage)
	}
	expectation := stage.ExternalExecution
	bindings := []struct {
		name string
		got  any
		want any
	}{
		{name: "runtime target", got: expectation.RuntimeTargetID, want: int64(7)},
		{name: "provider", got: expectation.ProviderID, want: "docker"},
		{name: "capability", got: expectation.Capability, want: composeExecutionCapability},
		{name: "capability version", got: expectation.CapabilityVersion, want: composeExecutionCapabilityVersion},
		{name: "protocol", got: expectation.Protocol, want: composeExecutionProtocol},
		{name: "operation", got: expectation.OperationID, want: "application.compose.up.v1"},
	}
	for _, binding := range bindings {
		if !reflect.DeepEqual(binding.got, binding.want) {
			t.Errorf("unexpected %s binding: got %#v, want %#v", binding.name, binding.got, binding.want)
		}
	}
	assertProviderNeutralApplicationStageInput(t, stage.Input)
}

func assertProviderNeutralApplicationStageInput(t *testing.T, raw json.RawMessage) {
	t.Helper()
	var input map[string]any
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatalf("decode stage input: %v", err)
	}
	if !reflect.DeepEqual(sortedExecutionInputKeys(input), []string{"application_id", "policy"}) {
		t.Fatalf("stage input must contain only application identity and policy: %#v", input)
	}
	policy, ok := input["policy"].(map[string]any)
	if !ok || !reflect.DeepEqual(sortedExecutionInputKeys(policy), []string{"remove_orphans", "snapshot_digest"}) {
		t.Fatalf("up policy must contain only typed operation fields: %#v", input["policy"])
	}
	encoded := string(raw)
	for _, forbidden := range []string{"workspace_path", "compose_files", "argv", "command", "endpoint", "credential", "/srv/demo"} {
		if strings.Contains(encoded, forbidden) {
			t.Fatalf("stage input leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestRedeployAndDestroyPlansKeepDomainStageOrder(t *testing.T) {
	aggregate := applicationExecutionAggregate()
	aggregate.Application.LifecycleConfig.DownBeforeRedeploy = true
	aggregate.Application.LifecycleConfig.PullBeforeRedeploy = true
	aggregate.Application.LifecycleConfig.PruneImagesAfterRedeploy = true
	redeploy, err := lifecycleTaskPlan(aggregate, generated.ApplicationActionResponseActionApplicationActionRedeploy, applicationExecutionTarget(), 9, nil)
	if err != nil {
		t.Fatalf("build redeploy plan: %v", err)
	}
	if got := stageKeys(redeploy); !reflect.DeepEqual(got, []string{"down", "pull", "up", "image-prune"}) {
		t.Fatalf("unexpected redeploy stages: %#v", got)
	}
	for index, operationID := range []string{"application.compose.down.v1", "application.compose.pull.v1", "application.compose.up.v1", "application.compose.image-prune.v1"} {
		if redeploy.Stages[index].ExternalExecution == nil || redeploy.Stages[index].ExternalExecution.OperationID != operationID {
			t.Fatalf("unexpected operation identity at stage %d: %#v", index, redeploy.Stages[index].ExternalExecution)
		}
	}
	destroyRequest := &DestroyRequest{RemoveNamedVolumes: true, ImagePrune: true, DeleteWorkspacePath: true, AutoUnregister: true}
	destroy, err := lifecycleTaskPlan(aggregate, generated.ApplicationActionResponseActionApplicationActionDestroy, applicationExecutionTarget(), 9, destroyRequest)
	if err != nil {
		t.Fatalf("build destroy plan: %v", err)
	}
	if got := stageKeys(destroy); !reflect.DeepEqual(got, []string{"down", "image-prune", "cleanup"}) {
		t.Fatalf("unexpected destroy stages: %#v", got)
	}
	if destroy.Stages[2].ExternalExecution != nil || destroy.Stages[2].ExecutorType != moduleapi.StageExecutorType(destroyCleanupStageType) {
		t.Fatalf("cleanup must remain a local domain stage: %#v", destroy.Stages[2])
	}
	for _, stage := range destroy.Stages[:2] {
		encoded := string(stage.Input)
		for _, forbidden := range []string{"delete_workspace_path", "auto_unregister", "actor_id"} {
			if strings.Contains(encoded, forbidden) {
				t.Fatalf("external destroy policy leaked cleanup field %q: %s", forbidden, encoded)
			}
		}
	}
}

func TestComposeMaterialResolverReturnsTransientMaterialAndRejectsDrift(t *testing.T) {
	aggregate := applicationExecutionAggregate()
	service, err := NewService(executionTestRepository{aggregate: aggregate})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	resolver := composeExecutionMaterialResolver{typeName: "application.compose.up", service: service}
	input, err := json.Marshal(composeStageInput{ApplicationID: aggregate.Application.ApplicationID, Policy: composeExecutionPolicy{SnapshotDigest: aggregate.Snapshot.ConfigHash, RemoveOrphans: true}})
	if err != nil {
		t.Fatalf("encode input: %v", err)
	}
	material, err := resolver.ResolveExternalExecutionMaterial(context.Background(), moduleapi.ExternalExecutionMaterialRequest{ExecutorType: resolver.Type(), Input: input})
	if err != nil {
		t.Fatalf("resolve material: %v", err)
	}
	if material.Protocol != composeMaterialProtocol {
		t.Fatalf("unexpected material protocol %q", material.Protocol)
	}
	var payload composeExecutionMaterial
	if err := json.Unmarshal(material.Payload, &payload); err != nil {
		t.Fatalf("decode material: %v", err)
	}
	if payload.WorkspacePath != "/srv/demo" || payload.ProjectName != "demo" || !reflect.DeepEqual(payload.ComposeFiles, []string{"/srv/demo/compose.yaml"}) || !reflect.DeepEqual(payload.EnvFiles, []string{"/srv/demo/.env"}) || !reflect.DeepEqual(payload.Profiles, []string{"production"}) {
		t.Fatalf("unexpected transient material: %#v", payload)
	}
	unknownInput := []byte(`{"application_id":"` + aggregate.Application.ApplicationID + `","policy":{"snapshot_digest":"cfg-demo"},"command":"forbidden"}`)
	if _, err := resolver.ResolveExternalExecutionMaterial(context.Background(), moduleapi.ExternalExecutionMaterialRequest{ExecutorType: resolver.Type(), Input: unknownInput}); err == nil {
		t.Fatal("unknown provider command field must fail closed")
	}
	drifted, _ := json.Marshal(composeStageInput{ApplicationID: aggregate.Application.ApplicationID, Policy: composeExecutionPolicy{SnapshotDigest: "other"}})
	if _, err := resolver.ResolveExternalExecutionMaterial(context.Background(), moduleapi.ExternalExecutionMaterialRequest{ExecutorType: resolver.Type(), Input: drifted}); err == nil {
		t.Fatal("snapshot mismatch must fail closed")
	}
	outside := applicationExecutionAggregate()
	outside.Files[1].AbsolutePath = "/srv/shared/.env"
	outsideService, err := NewService(executionTestRepository{aggregate: outside})
	if err != nil {
		t.Fatalf("new outside-path service: %v", err)
	}
	outsideResolver := composeExecutionMaterialResolver{typeName: resolver.Type(), service: outsideService}
	if _, err := outsideResolver.ResolveExternalExecutionMaterial(context.Background(), moduleapi.ExternalExecutionMaterialRequest{ExecutorType: outsideResolver.Type(), Input: input}); err == nil {
		t.Fatal("env file outside workspace must fail closed")
	}
}

func applicationExecutionAggregate() projectstore.ApplicationAggregate {
	targetID := uint64(7)
	return projectstore.ApplicationAggregate{
		Application: projectstore.Application{
			ApplicationRecordID: 1, ApplicationID: "app_01ARZ3NDEKTSV4RRFFQ69G5FAV", RuntimeTargetID: &targetID,
			WorkspacePath: "/srv/demo", ComposeProjectName: "demo", LifecycleReviewStatus: projectcontract.LifecycleReviewStatusConfirmed.String(),
			LifecycleConfig: projectstore.LifecycleConfig{Profiles: []string{"production"}, RemoveOrphans: true, WaitTimeoutSeconds: 120},
		},
		Files: []projectstore.ApplicationFile{
			{Kind: projectcontract.FileKindCompose.String(), AbsolutePath: "/srv/demo/compose.yaml", OrderIndex: 0},
			{Kind: projectcontract.FileKindEnv.String(), AbsolutePath: "/srv/demo/.env", OrderIndex: 1},
		},
		Snapshot: &projectstore.Snapshot{ConfigHash: "cfg-demo", DeclaredServiceCount: 1},
	}
}

func applicationExecutionTarget() moduleapi.ComposeRuntimeTargetSummary {
	return moduleapi.ComposeRuntimeTargetSummary{ID: 7, Provider: "docker", Capabilities: []string{composeExecutionCapability}, Available: true}
}

func stageKeys(plan moduleapi.TaskPlan) []string {
	keys := make([]string, 0, len(plan.Stages))
	for _, stage := range plan.Stages {
		keys = append(keys, stage.Key)
	}
	return keys
}

func sortedExecutionInputKeys(input map[string]any) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	if len(keys) == 2 && keys[0] > keys[1] {
		keys[0], keys[1] = keys[1], keys[0]
	}
	return keys
}
