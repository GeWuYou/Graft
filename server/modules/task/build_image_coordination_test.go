package task

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/dockerexecution"
	"graft/server/internal/moduleapi"
)

func TestConcurrentRegistryBuildsShareCanonicalLocalImageClaim(t *testing.T) {
	runtime, _ := localRuntimeForTest(t)
	first := localSubmitInput("registry-one", dockerexecution.ImageResourceKey("team/api:latest"))
	first.Plan.Stages[0].ExternalExecution.ResourceKeys = append(first.Plan.Stages[0].ExternalExecution.ResourceKeys, "registry:one:team/api:latest")
	second := localSubmitInput("registry-two", dockerexecution.ImageResourceKey("docker.io/team/api:latest"))
	second.Plan.Stages[0].ExternalExecution.ResourceKeys = append(second.Plan.Stages[0].ExternalExecution.ResourceKeys, "registry:two:team/api:latest")
	if _, err := runtime.Submit(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Submit(context.Background(), second); !errors.Is(err, moduleapi.ErrTaskOwnerBusy) {
		t.Fatalf("different registries admitted overlapping local build tags: %v", err)
	}
}
