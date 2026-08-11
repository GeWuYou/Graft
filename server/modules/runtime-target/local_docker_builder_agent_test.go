package runtimetarget

import (
	"errors"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestShouldReusePendingLocalDockerBuilderGenerationRejectsTrustBundleRotation(t *testing.T) {
	binding := moduleapi.RuntimeTargetAgentBinding{
		Generation:         3,
		Status:             moduleapi.RuntimeTargetAgentStatusPending,
		TrustBundleVersion: "bundle-v1",
	}
	currentBundle := moduleapi.TrustBundleReference{Version: "bundle-v2", ExpiresAt: time.Now().Add(time.Hour)}

	if shouldReusePendingLocalDockerBuilderGeneration(binding, nil, currentBundle) {
		t.Fatal("pending binding with rotated trust bundle must enter generation rotation")
	}
	if shouldReusePendingLocalDockerBuilderGeneration(binding, errors.New("read failed"), moduleapi.TrustBundleReference{Version: "bundle-v1"}) {
		t.Fatal("failed binding lookup must not reuse a pending generation")
	}
	if !shouldReusePendingLocalDockerBuilderGeneration(binding, nil, moduleapi.TrustBundleReference{Version: "bundle-v1"}) {
		t.Fatal("matching pending binding should reuse its generation")
	}
}
