package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	composeapi "github.com/docker/compose/v2/pkg/api"
	dockerfilters "github.com/docker/docker/api/types/filters"
	dockerimage "github.com/docker/docker/api/types/image"
)

type recordingComposeService struct {
	composeapi.Compose
	upOptions      composeapi.UpOptions
	stopOptions    composeapi.StopOptions
	restartOptions composeapi.RestartOptions
	pullProject    *composetypes.Project
}

func TestUpdateControllerMaterialIsStrictlyBoundToLeaseIntent(t *testing.T) {
	lease := executionLease{Input: json.RawMessage(`{"operation_id":"update-42"}`)}
	valid := updateControllerMaterial{ControllerReference: "ghcr.io/gewuyou/graft-compose-runner@sha256:" + strings.Repeat("a", 64), InputBase64: "encoded", ComposeRoot: "/opt/graft", DockerSocket: "/var/run/docker.sock", StateVolume: "graft-update-state", OperationID: "update-42"}
	if !validUpdateControllerMaterial(valid, lease) {
		t.Fatal("valid update controller material rejected")
	}
	valid.OperationID = "update-43"
	if validUpdateControllerMaterial(valid, lease) {
		t.Fatal("material accepted for a different operation")
	}
}

func TestUpdateControllerContainerConfigUsesOnlyFixedMountsAndNoNetwork(t *testing.T) {
	material := updateControllerMaterial{ControllerReference: "ghcr.io/gewuyou/graft-compose-runner@sha256:" + strings.Repeat("a", 64), InputBase64: "encoded", ComposeRoot: "/opt/graft", StateVolume: "graft-update-state", OperationID: "update-42"}
	configuration, host := updateControllerContainerConfig(material)
	if configuration.Image != material.ControllerReference || configuration.User != "0:0" || len(configuration.Env) != 1 || host.NetworkMode != "none" || !host.ReadonlyRootfs {
		t.Fatalf("unexpected controller config: %#v %#v", configuration, host)
	}
	if len(host.Binds) != 3 || host.CapDrop[0] != "ALL" || len(host.CapAdd) != 1 || host.CapAdd[0] != "CHOWN" || len(host.SecurityOpt) != 1 || host.SecurityOpt[0] != "no-new-privileges:true" {
		t.Fatalf("unexpected controller host config: %#v", host)
	}
}

func (s *recordingComposeService) Up(_ context.Context, _ *composetypes.Project, options composeapi.UpOptions) error {
	s.upOptions = options
	return nil
}

func (s *recordingComposeService) Stop(_ context.Context, _ string, options composeapi.StopOptions) error {
	s.stopOptions = options
	return nil
}

func (s *recordingComposeService) Restart(_ context.Context, _ string, options composeapi.RestartOptions) error {
	s.restartOptions = options
	return nil
}

func (s *recordingComposeService) Pull(_ context.Context, project *composetypes.Project, _ composeapi.PullOptions) error {
	s.pullProject = project
	return nil
}

type recordingImagePruner struct {
	filters dockerfilters.Args
	calls   int
}

func composeRequest(service composeapi.Compose, pruner dockerImagePruner, project *composetypes.Project, managed []string, action string, policy applicationComposePolicy) composeDispatchRequest {
	return composeDispatchRequest{runtime: &composeRuntime{project: project, service: service, pruner: pruner}, managedServices: managed, action: action, policy: policy}
}

func (p *recordingImagePruner) ImagesPrune(_ context.Context, filters dockerfilters.Args) (dockerimage.PruneReport, error) {
	p.calls++
	p.filters = filters
	return dockerimage.PruneReport{}, nil
}

func TestComposeUpHonorsDomainWaitPolicy(t *testing.T) {
	service := &recordingComposeService{}
	project := &composetypes.Project{Name: "demo"}
	policy := applicationComposePolicy{WaitAfterUp: true, WaitTimeoutSeconds: 17}
	if err := dispatchComposeOperation(context.Background(), composeRequest(service, nil, project, nil, "up", policy)); err != nil {
		t.Fatal(err)
	}
	if !service.upOptions.Start.Wait || service.upOptions.Start.WaitTimeout != 17*time.Second {
		t.Fatalf("start options=%#v", service.upOptions.Start)
	}
}

func TestComposeUpLeavesWaitTimeoutUnsetWhenWaitDisabled(t *testing.T) {
	service := &recordingComposeService{}
	project := &composetypes.Project{Name: "demo"}
	policy := applicationComposePolicy{WaitAfterUp: false, WaitTimeoutSeconds: 17}
	if err := dispatchComposeOperation(context.Background(), composeRequest(service, nil, project, nil, "up", policy)); err != nil {
		t.Fatal(err)
	}
	if service.upOptions.Start.Wait || service.upOptions.Start.WaitTimeout != 0 {
		t.Fatalf("start options=%#v", service.upOptions.Start)
	}
}

func TestComposeOperationsHonorManagedServiceBoundary(t *testing.T) {
	service := &recordingComposeService{}
	project := &composetypes.Project{
		Name: "demo",
		Services: composetypes.Services{
			"api":    {Name: "api"},
			"worker": {Name: "worker"},
		},
	}
	managed := []string{"api"}
	runComposeOperation(t, service, project, managed, "up")
	assertManagedService(t, "up create", service.upOptions.Create.Services)
	assertManagedService(t, "up start", service.upOptions.Start.Services)
	runComposeOperation(t, service, project, managed, "stop")
	assertManagedService(t, "stop", service.stopOptions.Services)
	runComposeOperation(t, service, project, managed, "restart")
	assertManagedService(t, "restart", service.restartOptions.Services)
	runComposeOperation(t, service, project, managed, "pull")
	assertManagedPullProject(t, service.pullProject)
}

func runComposeOperation(t *testing.T, service composeapi.Compose, project *composetypes.Project, managed []string, action string) {
	t.Helper()
	if err := dispatchComposeOperation(context.Background(), composeRequest(service, nil, project, managed, action, applicationComposePolicy{})); err != nil {
		t.Fatal(err)
	}
}

func assertManagedService(t *testing.T, operation string, services []string) {
	t.Helper()
	if len(services) != 1 || services[0] != "api" {
		t.Fatalf("%s services=%#v", operation, services)
	}
}

func assertManagedPullProject(t *testing.T, project *composetypes.Project) {
	t.Helper()
	if project == nil || len(project.Services) != 1 {
		t.Fatalf("pull services=%#v", project)
	}
	if _, ok := project.Services["api"]; !ok {
		t.Fatalf("pull services=%#v", project.Services)
	}
}

func TestComposeOperationsUseAllServicesWhenManagedSetEmpty(t *testing.T) {
	service := &recordingComposeService{}
	project := &composetypes.Project{Name: "demo", Services: composetypes.Services{"api": {Name: "api"}, "worker": {Name: "worker"}}}
	if err := dispatchComposeOperation(context.Background(), composeRequest(service, nil, project, nil, "pull", applicationComposePolicy{})); err != nil {
		t.Fatal(err)
	}
	if service.pullProject != project || len(service.pullProject.Services) != 2 {
		t.Fatalf("pull project=%#v", service.pullProject)
	}
}

func TestComposeImagePruneUsesDaemonDanglingPrune(t *testing.T) {
	pruner := &recordingImagePruner{}
	project := &composetypes.Project{Name: "demo"}
	if err := dispatchComposeOperation(context.Background(), composeRequest(&recordingComposeService{}, pruner, project, nil, "image-prune", applicationComposePolicy{})); err != nil {
		t.Fatal(err)
	}
	values := pruner.filters.Get("dangling")
	if pruner.calls != 1 || len(values) != 1 || values[0] != "true" {
		t.Fatalf("calls=%d filters=%v", pruner.calls, pruner.filters)
	}
}

func TestComposeDefinitionUsesEnvFileWithoutYieldingApplicationAuthority(t *testing.T) {
	workspace := t.TempDir()
	composeFile := filepath.Join(workspace, "compose.yaml")
	envFile := filepath.Join(workspace, ".env")
	if err := os.WriteFile(composeFile, []byte("services:\n  app:\n    image: ${APP_IMAGE}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envFile, []byte("APP_IMAGE=busybox:latest\nCOMPOSE_PROJECT_NAME=forbidden\nCOMPOSE_FILE=missing.yaml\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	material := applicationComposeMaterial{WorkspacePath: workspace, ProjectName: "authoritative", ComposeFiles: []string{composeFile}, EnvFiles: []string{envFile}}
	if !validComposeMaterial(material) {
		t.Fatal("valid workspace material rejected")
	}
	project, err := loadComposeDefinition(context.Background(), material)
	if err != nil {
		t.Fatal(err)
	}
	service, err := project.GetService("app")
	if err != nil {
		t.Fatal(err)
	}
	if project.Name != "authoritative" || service.Image != "busybox:latest" {
		t.Fatalf("project=%q image=%q", project.Name, service.Image)
	}
}

func TestApplicationComposePolicyRejectsCrossActionFields(t *testing.T) {
	tests := []struct {
		action string
		policy applicationComposePolicy
	}{
		{action: "up", policy: applicationComposePolicy{SnapshotDigest: "digest", WaitTimeoutSeconds: 30}},
		{action: "up", policy: applicationComposePolicy{SnapshotDigest: "digest", WaitAfterUp: true, WaitTimeoutSeconds: 3601}},
		{action: "down", policy: applicationComposePolicy{SnapshotDigest: "digest", RemoveOrphans: true}},
		{action: "stop", policy: applicationComposePolicy{SnapshotDigest: "digest", ForceRecreate: true}},
		{action: "pull", policy: applicationComposePolicy{SnapshotDigest: "digest", RemoveNamedVolumes: true}},
	}
	for _, test := range tests {
		if validApplicationComposePolicy(test.action, test.policy) {
			t.Fatalf("action=%q policy=%#v", test.action, test.policy)
		}
	}
	if !validApplicationComposePolicy("up", applicationComposePolicy{SnapshotDigest: "digest", WaitAfterUp: true, WaitTimeoutSeconds: 60, BuildBeforeUp: true}) {
		t.Fatal("valid up policy rejected")
	}
	if !validApplicationComposePolicy("down", applicationComposePolicy{SnapshotDigest: "digest", RemoveNamedVolumes: true}) {
		t.Fatal("valid down policy rejected")
	}
}
