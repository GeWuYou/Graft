package contract

import "testing"

func TestComposeDeploymentAdapterResolvesRuntimeModes(t *testing.T) {
	t.Parallel()
	adapter := ComposeDeploymentAdapter{}

	mode, err := adapter.ResolveExecutionMode(map[string]bool{"compose_execution": true})
	if err != nil || mode != DeploymentExecutionModeCompose {
		t.Fatalf("expected compose execution mode, got mode=%q err=%v", mode, err)
	}
	mode, err = adapter.ResolveExecutionMode(map[string]bool{"docker_stack_deploy": true})
	if err != nil || mode != DeploymentExecutionModeDockerStack {
		t.Fatalf("expected Docker Stack execution mode, got mode=%q err=%v", mode, err)
	}
	if _, err = adapter.ResolveExecutionMode(map[string]bool{}); err == nil {
		t.Fatal("expected an incompatible runtime target to be rejected")
	}
}

func TestDeploymentAdapterRegistryRejectsDuplicateKinds(t *testing.T) {
	t.Parallel()
	if _, err := NewDeploymentAdapterRegistry(ComposeDeploymentAdapter{}, ComposeDeploymentAdapter{}); err == nil {
		t.Fatal("expected duplicate deployment adapter registration to fail")
	}
}
