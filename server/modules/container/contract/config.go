// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// ConfigKey identifies a stable container management system configuration key.
type ConfigKey string

// String returns the canonical config key.
func (k ConfigKey) String() string {
	return string(k)
}

// EnvironmentPolicy identifies the stable container environment variable display policy.
type EnvironmentPolicy string

// String returns the canonical environment display policy.
func (p EnvironmentPolicy) String() string {
	return string(p)
}

const (
	// ContainerRuntimeEnabledConfig enables access to the configured container runtime.
	ContainerRuntimeEnabledConfig ConfigKey = "ops.container.runtime.enabled"
	// ContainerRuntimeConfig selects the runtime adapter.
	ContainerRuntimeConfig ConfigKey = "ops.container.runtime"
	// ContainerDockerEndpointConfig stores the first local runtime endpoint.
	ContainerDockerEndpointConfig ConfigKey = "ops.container.docker.endpoint"
	// ContainerLogsDefaultTailConfig stores the default log tail size.
	ContainerLogsDefaultTailConfig ConfigKey = "ops.container.logs.default_tail"
	// ContainerLogsMaxTailConfig stores the maximum log tail size.
	ContainerLogsMaxTailConfig ConfigKey = "ops.container.logs.max_tail"
	// ContainerDangerousActionsEnabledConfig enables high-risk container actions.
	ContainerDangerousActionsEnabledConfig ConfigKey = "ops.container.actions.dangerous_enabled"
	// ContainerEnvironmentPolicyConfig controls container environment variable value display.
	ContainerEnvironmentPolicyConfig ConfigKey = "ops.container.environment.policy"
	// ContainerEnvironmentMaskedCopyEnabledConfig controls whether masked environment values expose copy-only raw values.
	ContainerEnvironmentMaskedCopyEnabledConfig ConfigKey = "ops.container.environment.masked_copy_enabled"
)

const (
	// ContainerEnvironmentPolicyHidden hides all environment variable values.
	ContainerEnvironmentPolicyHidden EnvironmentPolicy = "hidden"
	// ContainerEnvironmentPolicyMasked masks sensitive environment variable values.
	ContainerEnvironmentPolicyMasked EnvironmentPolicy = "masked"
	// ContainerEnvironmentPolicyPlain shows environment variable values.
	ContainerEnvironmentPolicyPlain EnvironmentPolicy = "plain"
)
