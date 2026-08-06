package moduleapi_test

import (
	"context"
	"testing"

	"graft/server/internal/moduleapi"
)

type providerDriverCapabilityStub struct {
	request moduleapi.ProviderDriverExecutionRequest
}

func (s *providerDriverCapabilityStub) ExecuteProviderDriver(_ context.Context, request moduleapi.ProviderDriverExecutionRequest, _ moduleapi.BuildDriverLogSink) (moduleapi.ProviderDriverExecutionResult, error) {
	s.request = request
	return moduleapi.ProviderDriverExecutionResult{}, nil
}

func TestProviderDriverExecutionContractRemainsProviderNeutral(t *testing.T) {
	stub := &providerDriverCapabilityStub{}
	var capability moduleapi.TargetBoundProviderDriverExecutionCapability = stub
	request := moduleapi.ProviderDriverExecutionRequest{
		TargetID:      7,
		DriverRef:     "buildkit@v1",
		Platform:      "linux/arm64",
		SnapshotID:    "snapshot-1",
		ContentDigest: "sha256:source",
		DeliveryProof: moduleapi.WorkspaceSnapshotDeliveryResult{
			TargetID:      7,
			SnapshotID:    "snapshot-1",
			ContentDigest: "sha256:source",
			DeliveryMode:  moduleapi.SnapshotDeliveryModeProviderTransfer,
		},
	}
	result, err := capability.ExecuteProviderDriver(context.Background(), request, nil)
	if err != nil {
		t.Fatalf("ExecuteProviderDriver() error = %v", err)
	}
	if result.TargetID != 0 || result.ArtifactDigest != "" {
		t.Fatalf("stub result unexpectedly carried execution facts: %+v", result)
	}
	if stub.request.TargetID != request.TargetID || stub.request.DriverRef != request.DriverRef || stub.request.SnapshotID != request.SnapshotID || stub.request.DeliveryProof.DeliveryMode != moduleapi.SnapshotDeliveryModeProviderTransfer {
		t.Fatalf("provider-neutral request was not preserved: %+v", stub.request)
	}
}
