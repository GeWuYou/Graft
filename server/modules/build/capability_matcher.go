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

//nolint:cyclop // 静态协商必须在一个纯函数边界内完成所有 fail-closed 要求校验。
func (staticCapabilityMatcher) MatchBuildCapability(requirement moduleapi.BuildCapabilityRequirement, capability moduleapi.BuildExecutionCapability) (moduleapi.NegotiatedCapability, error) {
	if strings.TrimSpace(requirement.DriverRef) == "" || strings.TrimSpace(capability.ProviderCapabilityVersion) == "" {
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
	for _, feature := range requirement.RequiredFeatures {
		if slices.Contains(capability.Features, feature) {
			satisfied = append(satisfied, feature)
		} else {
			unsatisfied = append(unsatisfied, feature)
		}
	}
	if len(unsatisfied) > 0 {
		return moduleapi.NegotiatedCapability{ProviderCapabilityVersion: capability.ProviderCapabilityVersion, DriverRef: requirement.DriverRef, SnapshotDeliveryMode: delivery, SatisfiedFeatures: satisfied, UnsatisfiedFeatures: unsatisfied}, errors.New("required build capability is unsupported")
	}
	return moduleapi.NegotiatedCapability{ProviderCapabilityVersion: capability.ProviderCapabilityVersion, DriverRef: requirement.DriverRef, SnapshotDeliveryMode: delivery, SatisfiedFeatures: satisfied}, nil
}
