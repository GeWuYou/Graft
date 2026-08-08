package build

import (
	"testing"

	"graft/server/internal/moduleapi"
)

func TestStaticCapabilityMatcherNegotiatesRequiredBuildIntent(t *testing.T) {
	result, err := (staticCapabilityMatcher{}).MatchBuildCapability(
		moduleapi.BuildCapabilityRequirement{DriverRef: "docker-engine@v1", Platforms: []string{"linux/amd64"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}, RequiredFeatures: []string{"registry-login"}},
		moduleapi.BuildExecutionCapability{ProviderCapabilityVersion: "docker/v1", SupportedDrivers: []string{"docker-engine@v1"}, SupportedPlatforms: []string{"linux/amd64"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}, Features: []string{"registry-login"}},
	)
	if err != nil || result.ProviderCapabilityVersion != "docker/v1" || result.SnapshotDeliveryMode != moduleapi.SnapshotDeliveryModeTargetLocal {
		t.Fatalf("negotiation = %#v, err = %v", result, err)
	}
}

func TestStaticCapabilityMatcherRejectsMissingRequiredFeature(t *testing.T) {
	_, err := (staticCapabilityMatcher{}).MatchBuildCapability(moduleapi.BuildCapabilityRequirement{DriverRef: "docker-engine@v1", Platforms: []string{"linux/amd64"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}, RequiredFeatures: []string{"sbom"}}, moduleapi.BuildExecutionCapability{ProviderCapabilityVersion: "docker/v1", SupportedDrivers: []string{"docker-engine@v1"}, SupportedPlatforms: []string{"linux/amd64"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}})
	if err == nil {
		t.Fatal("expected missing required feature to fail closed")
	}
}
