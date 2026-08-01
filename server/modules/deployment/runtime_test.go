package deployment

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/moduleapi"
)

type dockerFactsStub struct {
	facts moduleapi.DockerContainerFacts
	err   error
}

type changingDockerFactsStub struct {
	facts []moduleapi.DockerContainerFacts
}

func (s *changingDockerFactsStub) CurrentContainer(context.Context) (moduleapi.DockerContainerFacts, error) {
	facts := s.facts[0]
	if len(s.facts) > 1 {
		s.facts = s.facts[1:]
	}
	return facts, nil
}

func (s dockerFactsStub) CurrentContainer(context.Context) (moduleapi.DockerContainerFacts, error) {
	return s.facts, s.err
}

func TestRuntimeExplicitRootWinsOverDockerFacts(t *testing.T) {
	runtime := NewRuntime(func(key string) (string, bool) {
		return map[string]string{deploymentRuntimeEnv: "compose", deploymentComposeRootEnv: "/opt/graft"}[key], key == deploymentRuntimeEnv || key == deploymentComposeRootEnv
	}, dockerFactsStub{facts: deploymentFacts(map[string]string{composeWorkingDirLabel: "/other"})})
	current := runtime.Current(context.Background())
	if !current.IsAvailable() || current.ComposeRootSource() != "explicit_config" || current.ComposeCandidates()[0].Root() != "/opt/graft" {
		t.Fatalf("explicit declaration did not win: %#v", current)
	}
}

func TestRuntimeInvalidExplicitRootFailsClosed(t *testing.T) {
	runtime := NewRuntime(func(key string) (string, bool) {
		return map[string]string{deploymentRuntimeEnv: "compose", deploymentComposeRootEnv: "relative"}[key], true
	}, dockerFactsStub{})
	if current := runtime.Current(context.Background()); current.IsAvailable() || current.Diagnostics()[0].Code != "configured_compose_root_invalid" || current.Diagnostics()[0].MessageKey != "deployment.diagnostics.configured_compose_root_invalid" {
		t.Fatalf("invalid explicit root did not fail closed: %#v", current)
	}
}

func TestRuntimeDiscoversUniqueComposeLabelsAndFreezesSnapshot(t *testing.T) {
	runtime := NewRuntime(func(key string) (string, bool) { return "compose", key == deploymentRuntimeEnv }, dockerFactsStub{facts: deploymentFacts(map[string]string{
		composeWorkingDirLabel: "/srv/graft", composeConfigFilesLabel: "/srv/graft/compose.yml,/srv/graft/compose.override.yml", composeProjectLabel: "graft",
	})})
	current := runtime.Current(context.Background())
	if !current.IsAvailable() || current.IsComposeConfirmationRequired() || len(current.ComposeCandidates()) != 1 {
		t.Fatalf("expected unique high confidence candidate: %#v", current)
	}
	snapshot, err := runtime.Freeze(context.Background(), moduleapi.DeploymentFreezeRequest{})
	if err != nil || snapshot.Candidate().Root() != "/srv/graft" || snapshot.Fingerprint() == "" {
		t.Fatalf("freeze unique candidate: snapshot=%#v err=%v", snapshot, err)
	}
}

func TestRuntimeRequiresSelectionForAmbiguousBindCandidates(t *testing.T) {
	runtime := NewRuntime(func(key string) (string, bool) { return "compose", key == deploymentRuntimeEnv }, dockerFactsStub{facts: moduleapi.DockerContainerFacts{Mounts: []moduleapi.DockerMountFact{{Type: dockerVolumeMountType, Destination: runnerStateRoot}, {Type: "bind", Source: "/one"}, {Type: "bind", Source: "/two"}}}})
	current := runtime.Current(context.Background())
	if !current.IsAvailable() || !current.IsComposeConfirmationRequired() {
		t.Fatalf("expected confirmation-required candidates: %#v", current)
	}
	if _, err := runtime.Freeze(context.Background(), moduleapi.DeploymentFreezeRequest{}); err == nil {
		t.Fatal("freeze without candidate selection succeeded")
	}
	if _, err := runtime.Freeze(context.Background(), moduleapi.DeploymentFreezeRequest{CandidateKey: current.ComposeCandidates()[0].Key()}); err != nil {
		t.Fatalf("freeze selected candidate: %v", err)
	}
}

func TestRuntimeReportsDockerFactsFailure(t *testing.T) {
	runtime := NewRuntime(func(key string) (string, bool) { return "compose", key == deploymentRuntimeEnv }, dockerFactsStub{err: errors.New("socket unavailable")})
	current := runtime.Current(context.Background())
	if current.IsAvailable() || current.Diagnostics()[0].Code != "docker_facts_unavailable" || current.Diagnostics()[0].MessageKey != "deployment.diagnostics.docker_facts_unavailable" {
		t.Fatalf("unexpected unavailable context: %#v", current)
	}
}

func TestRuntimeFreezeUsesFreshDockerFacts(t *testing.T) {
	provider := &changingDockerFactsStub{facts: []moduleapi.DockerContainerFacts{
		deploymentFacts(map[string]string{composeWorkingDirLabel: "/before"}),
		deploymentFacts(map[string]string{composeWorkingDirLabel: "/after"}),
	}}
	runtime := NewRuntime(func(key string) (string, bool) { return "compose", key == deploymentRuntimeEnv }, provider)
	if current := runtime.Current(context.Background()); current.ComposeCandidates()[0].Root() != "/before" {
		t.Fatalf("unexpected initial context: %#v", current)
	}
	snapshot, err := runtime.Freeze(context.Background(), moduleapi.DeploymentFreezeRequest{})
	if err != nil || snapshot.Candidate().Root() != "/after" {
		t.Fatalf("freeze did not re-read current facts: snapshot=%#v err=%v", snapshot, err)
	}
}

func TestRuntimeDefaultsToComposeWhenRuntimeIsUnsetOrEmpty(t *testing.T) {
	lookups := map[string]func(string) (string, bool){
		"unset": nil,
		"empty": func(key string) (string, bool) {
			return "", key == deploymentRuntimeEnv
		},
	}
	for name, lookup := range lookups {
		t.Run(name, func(t *testing.T) {
			runtime := NewRuntime(lookup, dockerFactsStub{})
			if current := runtime.Current(context.Background()); current.Mode() != "compose" || current.IsAvailable() || current.Diagnostics()[0].Code != "runner_state_volume_missing" {
				t.Fatalf("runtime did not default to Compose discovery: %#v", current)
			}
		})
	}
}

func TestRuntimeRequiresRunnerStateVolumeForExplicitRoot(t *testing.T) {
	runtime := NewRuntime(func(key string) (string, bool) {
		return map[string]string{deploymentRuntimeEnv: "compose", deploymentComposeRootEnv: "/opt/graft"}[key], key == deploymentRuntimeEnv || key == deploymentComposeRootEnv
	}, dockerFactsStub{})
	current := runtime.Current(context.Background())
	if current.IsAvailable() || current.Diagnostics()[0].Code != "runner_state_volume_missing" || current.Diagnostics()[0].MessageKey != "deployment.diagnostics.runner_state_volume_missing" {
		t.Fatalf("explicit root without runner state volume did not fail closed: %#v", current)
	}
}

func TestRuntimeRequiresRunnerStateVolumeForDiscoveredRoot(t *testing.T) {
	runtime := NewRuntime(func(key string) (string, bool) { return "compose", key == deploymentRuntimeEnv }, dockerFactsStub{facts: moduleapi.DockerContainerFacts{Labels: map[string]string{composeWorkingDirLabel: "/srv/graft"}}})
	current := runtime.Current(context.Background())
	if current.IsAvailable() || current.Diagnostics()[0].Code != "runner_state_volume_missing" {
		t.Fatalf("discovered root without runner state volume did not fail closed: %#v", current)
	}
}

func deploymentFacts(labels map[string]string) moduleapi.DockerContainerFacts {
	return moduleapi.DockerContainerFacts{
		Labels: labels,
		Mounts: []moduleapi.DockerMountFact{{Type: dockerVolumeMountType, Destination: runnerStateRoot}},
	}
}

func TestRuntimePreservesDeclaredUnsupportedRuntimeInDiagnosticContext(t *testing.T) {
	runtime := NewRuntime(func(key string) (string, bool) { return "binary", key == deploymentRuntimeEnv }, dockerFactsStub{})
	if current := runtime.Current(context.Background()); current.Mode() != "binary" || current.IsAvailable() || current.Diagnostics()[0].Code != "deployment_mode_unsupported" || current.Diagnostics()[0].MessageKey != "deployment.diagnostics.mode_unsupported" {
		t.Fatalf("binary runtime was not represented as unavailable context: %#v", current)
	}
}
