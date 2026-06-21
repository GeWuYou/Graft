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

// OrchestratorActionLevel identifies the stable dangerous-action policy for one orchestrator source.
type OrchestratorActionLevel string

// String returns the canonical action level policy.
func (l OrchestratorActionLevel) String() string {
	return string(l)
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
	// ContainerComposeActionLevelConfig stores the dangerous-action policy for compose-managed containers.
	ContainerComposeActionLevelConfig ConfigKey = "ops.container.actions.compose_level"
	// ContainerSwarmActionLevelConfig stores the dangerous-action policy for swarm-managed containers.
	ContainerSwarmActionLevelConfig ConfigKey = "ops.container.actions.swarm_level"
	// ContainerKubernetesActionLevelConfig stores the dangerous-action policy for kubernetes-managed containers.
	ContainerKubernetesActionLevelConfig ConfigKey = "ops.container.actions.kubernetes_level"
	// ContainerUnknownActionLevelConfig stores the dangerous-action policy for unclassified managed containers.
	ContainerUnknownActionLevelConfig ConfigKey = "ops.container.actions.unknown_level"
	// ContainerShellEnabledConfig enables interactive shell session access.
	ContainerShellEnabledConfig ConfigKey = "ops.container.shell.enabled"
	// ContainerEnvironmentPolicyConfig controls container environment variable value display.
	ContainerEnvironmentPolicyConfig ConfigKey = "ops.container.environment.policy"
	// ContainerEnvironmentMaskedCopyEnabledConfig controls whether masked sensitive environment entries expose copy-only raw values to already-authorized readers.
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

const (
	// ContainerOrchestratorActionLevelReadonly disables dangerous actions for the source.
	ContainerOrchestratorActionLevelReadonly OrchestratorActionLevel = "readonly"
	// ContainerOrchestratorActionLevelWarn allows single-item dangerous actions but blocks batch actions.
	ContainerOrchestratorActionLevelWarn OrchestratorActionLevel = "warn"
	// ContainerOrchestratorActionLevelAllow allows both single and batch dangerous actions.
	ContainerOrchestratorActionLevelAllow OrchestratorActionLevel = "allow"
)
