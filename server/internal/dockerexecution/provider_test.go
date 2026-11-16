package dockerexecution

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	composeapi "github.com/docker/compose/v5/pkg/api"
	mobyclient "github.com/moby/moby/client"
)

type recordingComposeService struct {
	composeapi.Compose
	upOptions      composeapi.UpOptions
	upProject      *composetypes.Project
	stopOptions    composeapi.StopOptions
	restartOptions composeapi.RestartOptions
	pullProject    *composetypes.Project
}

func TestUpdateControllerMaterialIsStrictlyBoundToLeaseIntent(t *testing.T) {
	lease := Request{Input: json.RawMessage(`{"operation_id":"update-42","source_version":"1.0.0","target_version":"1.1.0"}`)}
	valid := updateControllerMaterial{ControllerReference: "ghcr.io/gewuyou/graft-compose-runner@sha256:" + strings.Repeat("a", 64), InputBase64: "encoded", ComposeRoot: "/opt/graft", DockerSocket: "/var/run/docker.sock", StateVolume: "graft-update-state", OperationID: "update-42"}
	if !validUpdateControllerMaterial(valid, lease) {
		t.Fatal("valid update controller material rejected")
	}
	valid.OperationID = "update-43"
	if validUpdateControllerMaterial(valid, lease) {
		t.Fatal("material accepted for a different operation")
	}
}

func TestUpdateControllerMaterialRejectsIncompleteOrUnknownIntent(t *testing.T) {
	material := updateControllerMaterial{ControllerReference: "ghcr.io/gewuyou/graft-compose-runner@sha256:" + strings.Repeat("a", 64), InputBase64: "encoded", ComposeRoot: "/opt/graft", DockerSocket: "/var/run/docker.sock", StateVolume: "graft-update-state", OperationID: "update-42"}
	for _, input := range []string{
		`{"operation_id":"update-42"}`,
		`{"operation_id":"update-42","source_version":" ","target_version":"1.1.0"}`,
		`{"operation_id":"update-42","source_version":"1.0.0","target_version":" "}`,
		`{"operation_id":"update-42","source_version":"1.0.0","target_version":"1.0.0"}`,
		`{"operation_id":"update-42","source_version":"1.0.0","target_version":"1.1.0","unexpected":true}`,
	} {
		if validUpdateControllerMaterial(material, Request{Input: json.RawMessage(input)}) {
			t.Fatalf("invalid controller intent accepted: %s", input)
		}
	}
}

func TestRecoveryControllerMaterialMatchesFrozenIntentVersions(t *testing.T) {
	lease := Request{Input: json.RawMessage(`{"operation_id":"update-42","source_version":"1.0.0","target_version":"1.1.0"}`)}
	for _, testCase := range []struct {
		name    string
		source  string
		target  string
		allowed bool
	}{
		{name: "same frozen versions", source: "1.0.0", target: "1.1.0", allowed: true},
		{name: "different source", source: "0.9.0", target: "1.1.0"},
		{name: "different target", source: "1.0.0", target: "1.2.0"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			encoded := base64.RawStdEncoding.EncodeToString([]byte(`{"operation_id":"update-42","runner_id":"runner-recovery-42","source_version":"` + testCase.source + `","target_version":"` + testCase.target + `","deployment_strategy":"beta_tracking"}`))
			material := updateControllerMaterial{ControllerReference: "ghcr.io/gewuyou/graft-compose-runner@sha256:" + strings.Repeat("a", 64), RecoveryStateBase64: encoded, RecoveryClaimID: "recovery-42", Recovery: true, StateVolume: "graft-update-state", OperationID: "update-42"}
			if validUpdateControllerMaterial(material, lease) != testCase.allowed {
				t.Fatalf("recovery material version binding did not match allowed=%v", testCase.allowed)
			}
		})
	}
}

func TestUpdateControllerContainerConfigUsesOnlyFixedMountsAndNoNetwork(t *testing.T) {
	material := updateControllerMaterial{ControllerReference: "ghcr.io/gewuyou/graft-compose-runner@sha256:" + strings.Repeat("a", 64), InputBase64: "encoded", ComposeRoot: "/opt/graft", StateVolume: "graft-update-state", OperationID: "update-42"}
	configuration, host := updateControllerContainerConfig(material)
	if configuration.Image != material.ControllerReference || configuration.User != "0:0" || len(configuration.Env) != 1 || host.NetworkMode != "none" || !host.ReadonlyRootfs {
		t.Fatalf("unexpected controller Environment: %#v %#v", configuration, host)
	}
	if len(host.Binds) != 3 || host.CapDrop[0] != "ALL" || !slices.Equal(host.CapAdd, []string{"CHOWN", "DAC_OVERRIDE"}) || len(host.SecurityOpt) != 1 || host.SecurityOpt[0] != "no-new-privileges:true" || host.Privileged {
		t.Fatalf("unexpected controller host Environment: %#v", host)
	}
}

func TestUpdateControllerAdvertisesRunnerReceiptProtocol(t *testing.T) {
	for _, recovery := range []bool{false, true} {
		configuration, _ := updateControllerContainerConfig(updateControllerMaterial{Recovery: recovery})
		if configuration.Labels["io.graft.update.protocol"] != "compose-runner/v2" {
			t.Fatalf("controller receipt label = %q, recovery=%v", configuration.Labels["io.graft.update.protocol"], recovery)
		}
	}
}

func TestRecoveryControllerMaterialUsesStateVolumeOnlyAndFixedIsolation(t *testing.T) {
	encoded := base64.RawStdEncoding.EncodeToString([]byte(`{"operation_id":"update-42","runner_id":"runner-recovery-42","source_version":"1.0.0","target_version":"1.1.0","deployment_strategy":"beta_tracking"}`))
	material := updateControllerMaterial{ControllerReference: "ghcr.io/gewuyou/graft-compose-runner@sha256:" + strings.Repeat("a", 64), RecoveryStateBase64: encoded, RecoveryClaimID: "recovery-42", Recovery: true, StateVolume: "graft-update-state", OperationID: "update-42"}
	configuration, host := updateControllerContainerConfig(material)
	if configuration.User != "0:0" || !slices.Equal(host.CapDrop, []string{"ALL"}) || !slices.Equal(host.CapAdd, []string{"CHOWN", "DAC_OVERRIDE"}) || host.Privileged {
		t.Fatalf("unexpected recovery controller filesystem authority: %#v %#v", configuration, host)
	}
	if configuration.Env[0] != "GRAFT_UPDATE_RUNNER_RECOVERY_STATE_B64="+encoded || host.NetworkMode != "none" || !host.ReadonlyRootfs {
		t.Fatalf("unexpected recovery isolation: %#v %#v", configuration, host)
	}
	if len(host.Binds) != 1 || host.Binds[0] != "graft-update-state:/var/lib/graft/update-state:rw" {
		t.Fatalf("recovery must mount only state volume: %#v", host.Binds)
	}
	if configuration.Labels["io.graft.update.recovery"] != "true" || configuration.Labels["io.graft.update.operation"] != material.OperationID {
		t.Fatalf("recovery labels = %#v", configuration.Labels)
	}
	if strings.Contains(strings.Join(host.Binds, "\n"), "docker.sock") {
		t.Fatal("recovery container must not mount Docker socket")
	}
}

func (s *recordingComposeService) Up(_ context.Context, project *composetypes.Project, options composeapi.UpOptions) error {
	s.upProject = project
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
	filters mobyclient.Filters
	calls   int
}

func composeRequest(service composeapi.Compose, pruner dockerImagePruner, project *composetypes.Project, managed []string, action string, policy applicationComposePolicy) composeDispatchRequest {
	return composeDispatchRequest{runtime: &composeRuntime{project: project, service: service, pruner: pruner}, managedServices: managed, action: action, policy: policy}
}

func (p *recordingImagePruner) ImagePrune(_ context.Context, options mobyclient.ImagePruneOptions) (mobyclient.ImagePruneResult, error) {
	p.calls++
	p.filters = options.Filters
	return mobyclient.ImagePruneResult{}, nil
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
	values := pruner.filters["dangling"]
	if pruner.calls != 1 || len(values) != 1 || !values["true"] {
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
