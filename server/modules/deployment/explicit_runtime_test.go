package deployment

import (
	"context"
	"reflect"
	"testing"

	"graft/server/internal/moduleapi"
)

func explicitRuntimeWithFacts(facts moduleapi.DockerContainerFacts) moduleapi.DeploymentRuntime {
	return NewRuntime(func(key string) (string, bool) {
		values := map[string]string{deploymentRuntimeEnv: "compose", deploymentComposeRootEnv: "/opt/graft"}
		value, exists := values[key]
		return value, exists
	}, dockerFactsStub{facts: facts})
}

func TestExplicitRootRetainsMatchingComposeProjectAndAllOverrides(t *testing.T) {
	files := []string{"/opt/graft/compose.yml", "/opt/graft/compose.build.yml", "/opt/graft/compose.release.yml"}
	runtime := explicitRuntimeWithFacts(deploymentFacts(map[string]string{
		composeWorkingDirLabel: "/opt/graft", composeProjectLabel: "custom-official-project",
		composeConfigFilesLabel: "/opt/graft/compose.yml,/opt/graft/compose.build.yml,/opt/graft/compose.release.yml",
	}))
	snapshot, err := runtime.Freeze(t.Context(), moduleapi.DeploymentFreezeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	candidate := snapshot.Candidate()
	if candidate.Root() != "/opt/graft" || candidate.ProjectName() != "custom-official-project" || !reflect.DeepEqual(candidate.ConfigFiles(), files) {
		t.Fatalf("candidate root=%q project=%q files=%v", candidate.Root(), candidate.ProjectName(), candidate.ConfigFiles())
	}
	if snapshot.Context().ComposeRootSource() != "explicit_config" || snapshot.Context().IsComposeConfirmationRequired() || snapshot.Fingerprint() == "" {
		t.Fatalf("snapshot=%#v", snapshot)
	}
}

func TestExplicitRootProvidesCanonicalFilesWithoutBorrowingAnotherProject(t *testing.T) {
	runtime := explicitRuntimeWithFacts(deploymentFacts(map[string]string{
		composeWorkingDirLabel: "/different/root", composeConfigFilesLabel: "/different/root/compose.yaml",
		composeProjectLabel: "unrelated-project",
	}))
	snapshot, err := runtime.Freeze(t.Context(), moduleapi.DeploymentFreezeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	candidate := snapshot.Candidate()
	if candidate.Root() != "/opt/graft" || candidate.ProjectName() != "" || !reflect.DeepEqual(candidate.ConfigFiles(), []string{"/opt/graft/compose.yml"}) {
		t.Fatalf("unrelated facts redirected explicit candidate: %#v", candidate)
	}
}

func TestExplicitMatchingRootRejectsIncompleteOrUnsafeOverrideGraphs(t *testing.T) {
	for _, files := range []string{
		"/opt/graft/compose.yml,/outside/override.yml",
		"/opt/graft/compose.yml,relative.yml",
		"/opt/graft/compose.yml,/opt/graft/link/../../outside.yml",
		"/opt/graft/compose.yml,/opt/graft",
		"/opt/graft/compose.yml,",
	} {
		t.Run(files, func(t *testing.T) {
			runtime := explicitRuntimeWithFacts(deploymentFacts(map[string]string{
				composeWorkingDirLabel: "/opt/graft", composeConfigFilesLabel: files,
			}))
			current := runtime.Current(context.Background())
			if current.IsAvailable() || current.Diagnostics()[0].Code != "configured_compose_files_invalid" {
				t.Fatalf("unsafe graph remained available: %#v", current)
			}
			if _, err := runtime.Freeze(t.Context(), moduleapi.DeploymentFreezeRequest{}); err == nil {
				t.Fatal("unsafe explicit composition was frozen")
			}
		})
	}
}

func TestEmptyExplicitRootStillFailsWithoutDiscoveryFallback(t *testing.T) {
	runtime := NewRuntime(func(key string) (string, bool) {
		if key == deploymentRuntimeEnv {
			return "compose", true
		}
		return "", key == deploymentComposeRootEnv
	}, dockerFactsStub{facts: deploymentFacts(map[string]string{composeWorkingDirLabel: "/opt/graft"})})
	current := runtime.Current(t.Context())
	if current.IsAvailable() || current.Diagnostics()[0].Code != "configured_compose_root_invalid" {
		t.Fatalf("empty explicit root did not fail closed: %#v", current)
	}
}

func TestExplicitFreezeReadsCurrentOverrideGraph(t *testing.T) {
	provider := &changingDockerFactsStub{facts: []moduleapi.DockerContainerFacts{
		deploymentFacts(map[string]string{composeWorkingDirLabel: "/opt/graft", composeConfigFilesLabel: "/opt/graft/compose.yml"}),
		deploymentFacts(map[string]string{composeWorkingDirLabel: "/opt/graft", composeConfigFilesLabel: "/opt/graft/compose.yml,/opt/graft/current.yml"}),
	}}
	runtime := NewRuntime(func(key string) (string, bool) {
		return "/opt/graft", key == deploymentComposeRootEnv
	}, provider)
	before := runtime.Current(t.Context()).ComposeCandidates()[0]
	snapshot, err := runtime.Freeze(t.Context(), moduleapi.DeploymentFreezeRequest{})
	if err != nil || snapshot.Candidate().Key() == before.Key() || len(snapshot.Candidate().ConfigFiles()) != 2 {
		t.Fatalf("snapshot=%#v err=%v", snapshot, err)
	}
}
