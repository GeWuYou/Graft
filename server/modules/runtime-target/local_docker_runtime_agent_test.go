package runtimetarget

import (
	"errors"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestShouldReusePendingLocalDockerRuntimeGenerationRejectsTrustBundleRotation(t *testing.T) {
	binding := moduleapi.RuntimeTargetAgentBinding{
		Generation:         3,
		Status:             moduleapi.RuntimeTargetAgentStatusPending,
		TrustBundleVersion: "bundle-v1",
		CapabilityVersion:  "docker/v1",
		Capabilities:       append([]string(nil), localDockerRuntimeAgentCapabilities...),
	}
	currentBundle := moduleapi.TrustBundleReference{Version: "bundle-v2", ExpiresAt: time.Now().Add(time.Hour)}

	if shouldReusePendingLocalDockerRuntimeGeneration(binding, nil, currentBundle) {
		t.Fatal("pending binding with rotated trust bundle must enter generation rotation")
	}
	if shouldReusePendingLocalDockerRuntimeGeneration(binding, errors.New("read failed"), moduleapi.TrustBundleReference{Version: "bundle-v1"}) {
		t.Fatal("failed binding lookup must not reuse a pending generation")
	}
	if !shouldReusePendingLocalDockerRuntimeGeneration(binding, nil, moduleapi.TrustBundleReference{Version: "bundle-v1"}) {
		t.Fatal("matching pending binding should reuse its generation")
	}
}
