package build

import (
	"testing"

	"graft/server/internal/moduleapi"
)

func TestStaticCapabilityMatcherNegotiatesRequiredBuildIntent(t *testing.T) {
	result, err := (staticCapabilityMatcher{}).MatchBuildCapability(
		moduleapi.BuildCapabilityRequirement{DriverRef: "docker-engine@v1", Platforms: []string{"linux/amd64"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}, RequiredFeatures: []string{"registry-login"}},
		moduleapi.BuildExecutionCapability{ProviderCapabilityProfile: "oci-build", ProviderCapabilityVersion: "docker/v1", SupportedDrivers: []string{"docker-engine@v1"}, SupportedPlatforms: []string{"linux/amd64"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}, Features: []string{"registry-login"}},
	)
	if err != nil || result.ProviderCapabilityVersion != "docker/v1" || result.SnapshotDeliveryMode != moduleapi.SnapshotDeliveryModeTargetLocal {
		t.Fatalf("negotiation = %#v, err = %v", result, err)
	}
}

func TestStaticCapabilityMatcherRejectsMissingRequiredFeature(t *testing.T) {
	_, err := (staticCapabilityMatcher{}).MatchBuildCapability(moduleapi.BuildCapabilityRequirement{DriverRef: "docker-engine@v1", Platforms: []string{"linux/amd64"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}, RequiredFeatures: []string{"sbom"}}, moduleapi.BuildExecutionCapability{ProviderCapabilityProfile: "oci-build", ProviderCapabilityVersion: "docker/v1", SupportedDrivers: []string{"docker-engine@v1"}, SupportedPlatforms: []string{"linux/amd64"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}})
	if err == nil {
		t.Fatal("expected missing required feature to fail closed")
	}
}

func TestStaticCapabilityMatcherFreezesPreferredAndOptionalOmissions(t *testing.T) {
	result, err := (staticCapabilityMatcher{}).MatchBuildCapability(
		moduleapi.BuildCapabilityRequirement{DriverRef: "docker-engine@v1", Platforms: []string{"linux/amd64"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}, FeatureRequirements: []moduleapi.BuildCapabilityFeatureRequirement{{Feature: "registry-login", Mode: moduleapi.BuildCapabilityFeatureRequired}, {Feature: "sbom", Mode: moduleapi.BuildCapabilityFeaturePreferred}, {Feature: "provenance", Mode: moduleapi.BuildCapabilityFeatureOptional}}},
		moduleapi.BuildExecutionCapability{ProviderCapabilityProfile: "oci-build", ProviderCapabilityVersion: "docker/v1", SupportedDrivers: []string{"docker-engine@v1"}, SupportedPlatforms: []string{"linux/amd64"}, SnapshotDeliveryModes: []string{moduleapi.SnapshotDeliveryModeTargetLocal}, Features: []string{"registry-login"}},
	)
	if err != nil || result.PreferredMissReasons["sbom"] == "" || result.OptionalOmissionReasons["provenance"] == "" {
		t.Fatalf("negotiation = %#v, err = %v", result, err)
	}
}

func TestCapabilityRequirementFingerprintIncludesFrozenIntent(t *testing.T) {
	base := buildCapabilityRequirement("docker-engine@v1", "linux/amd64")
	changed := base
	changed.SecurityPolicy = "provenance-required"
	if fingerprintBuildCapabilityRequirement(base) == fingerprintBuildCapabilityRequirement(changed) {
		t.Fatal("expected security policy change to alter capability requirement fingerprint")
	}
	ordered := base
	ordered.Platforms = []string{"linux/amd64", "linux/arm64"}
	permuted := ordered
	permuted.Platforms = []string{"linux/arm64", "linux/amd64"}
	if fingerprintBuildCapabilityRequirement(ordered) != fingerprintBuildCapabilityRequirement(permuted) {
		t.Fatal("expected canonical platform ordering to produce a stable fingerprint")
	}
}
