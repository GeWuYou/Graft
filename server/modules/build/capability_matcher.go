package build

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"graft/server/internal/moduleapi"
)

// staticCapabilityMatcher performs the Phase 2 required capability negotiation.
// Dynamic telemetry and placement policy are intentionally outside this boundary.
type staticCapabilityMatcher struct{}

var _ moduleapi.CapabilityMatcher = staticCapabilityMatcher{}

//nolint:cyclop,gocognit,gocyclo // Capability identity, delivery and feature-mode verdicts must remain one pure admission boundary.
func (staticCapabilityMatcher) MatchBuildCapability(requirement moduleapi.BuildCapabilityRequirement, capability moduleapi.BuildExecutionCapability) (moduleapi.NegotiatedCapability, error) {
	if strings.TrimSpace(requirement.DriverRef) == "" || strings.TrimSpace(capability.ProviderCapabilityProfile) == "" || strings.TrimSpace(capability.ProviderCapabilityVersion) == "" {
		return moduleapi.NegotiatedCapability{}, errors.New("build capability identity is incomplete")
	}
	if !containsBuildRef(capability.SupportedDrivers, requirement.DriverRef) {
		return moduleapi.NegotiatedCapability{}, fmt.Errorf("driver %q is unsupported", requirement.DriverRef)
	}
	for _, platform := range requirement.Platforms {
		if !slices.Contains(capability.SupportedPlatforms, platform) {
			return moduleapi.NegotiatedCapability{}, fmt.Errorf("platform %q is unsupported", platform)
		}
	}
	delivery := ""
	for _, requested := range requirement.SnapshotDeliveryModes {
		if slices.Contains(capability.SnapshotDeliveryModes, requested) {
			delivery = requested
			break
		}
	}
	if delivery == "" {
		return moduleapi.NegotiatedCapability{}, errors.New("snapshot delivery mode is unsupported")
	}
	unsatisfied := make([]string, 0)
	satisfied := make([]string, 0, len(requirement.RequiredFeatures))
	preferredMisses := map[string]string{}
	optionalOmissions := map[string]string{}
	featureRequirements := append([]moduleapi.BuildCapabilityFeatureRequirement(nil), requirement.FeatureRequirements...)
	for _, feature := range requirement.RequiredFeatures {
		featureRequirements = append(featureRequirements, moduleapi.BuildCapabilityFeatureRequirement{Feature: feature, Mode: moduleapi.BuildCapabilityFeatureRequired})
	}
	for _, requested := range featureRequirements {
		feature, mode := strings.TrimSpace(requested.Feature), strings.TrimSpace(requested.Mode)
		if feature == "" || (mode != moduleapi.BuildCapabilityFeatureRequired && mode != moduleapi.BuildCapabilityFeaturePreferred && mode != moduleapi.BuildCapabilityFeatureOptional) {
			return moduleapi.NegotiatedCapability{}, errors.New("build capability feature requirement is invalid")
		}
		if slices.Contains(capability.Features, feature) {
			satisfied = append(satisfied, feature)
		} else if mode == moduleapi.BuildCapabilityFeatureRequired {
			unsatisfied = append(unsatisfied, feature)
		} else if mode == moduleapi.BuildCapabilityFeaturePreferred {
			preferredMisses[feature] = "provider_feature_unavailable"
		} else {
			optionalOmissions[feature] = "provider_feature_unavailable"
		}
	}
	if len(unsatisfied) > 0 {
		return moduleapi.NegotiatedCapability{ProviderCapabilityProfile: capability.ProviderCapabilityProfile, ProviderCapabilityVersion: capability.ProviderCapabilityVersion, DriverRef: requirement.DriverRef, SnapshotDeliveryMode: delivery, SatisfiedFeatures: satisfied, UnsatisfiedFeatures: unsatisfied, PreferredMissReasons: preferredMisses, OptionalOmissionReasons: optionalOmissions}, errors.New("required build capability is unsupported")
	}
	return moduleapi.NegotiatedCapability{ProviderCapabilityProfile: capability.ProviderCapabilityProfile, ProviderCapabilityVersion: capability.ProviderCapabilityVersion, DriverRef: requirement.DriverRef, SnapshotDeliveryMode: delivery, SatisfiedFeatures: satisfied, PreferredMissReasons: preferredMisses, OptionalOmissionReasons: optionalOmissions}, nil
}
