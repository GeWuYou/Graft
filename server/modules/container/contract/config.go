package contract

// ConfigKey 是容器管理系统配置键的稳定契约。
type ConfigKey string

// String 返回规范配置键值。
func (k ConfigKey) String() string {
	return string(k)
}

// EnvironmentPolicy 是容器环境变量展示策略的稳定契约。
type EnvironmentPolicy string

// String 返回规范环境变量展示策略值。
func (p EnvironmentPolicy) String() string {
	return string(p)
}

// OrchestratorActionLevel 是单个编排器来源的危险操作策略契约。
type OrchestratorActionLevel string

// String 返回规范操作级别策略值。
func (l OrchestratorActionLevel) String() string {
	return string(l)
}

const (
	// ContainerRuntimeEnabledConfig 控制是否允许访问已配置的容器运行时。
	ContainerRuntimeEnabledConfig ConfigKey = "ops.container.runtime.enabled"
	// ContainerLogsDefaultTailConfig 保存默认日志尾部行数。
	ContainerLogsDefaultTailConfig ConfigKey = "ops.container.logs.default_tail"
	// ContainerLogsMaxTailConfig 保存日志尾部行数上限。
	ContainerLogsMaxTailConfig ConfigKey = "ops.container.logs.max_tail"
	// ContainerResourceStatsCacheTTLConfig 保存容器资源统计快照的缓存新鲜窗口。
	ContainerResourceStatsCacheTTLConfig ConfigKey = "ops.container.resource_stats.cache_ttl_seconds"
	// ContainerResourceStatsCacheStaleWindowConfig 保存容器资源统计快照的过期可用窗口。
	ContainerResourceStatsCacheStaleWindowConfig ConfigKey = "ops.container.resource_stats.stale_window_seconds"
	// ContainerResourceStatsCollectIntervalConfig 保存实时统计采集器的发布间隔秒数。
	ContainerResourceStatsCollectIntervalConfig ConfigKey = "ops.container.resource_stats.collect_interval_seconds"
	// ContainerDangerousActionsEnabledConfig 控制是否允许高风险容器操作。
	ContainerDangerousActionsEnabledConfig ConfigKey = "ops.container.actions.dangerous_enabled"
	// ContainerComposeActionLevelConfig 保存 Compose 管理容器的危险操作策略。
	ContainerComposeActionLevelConfig ConfigKey = "ops.container.actions.compose_level"
	// ContainerSwarmActionLevelConfig 保存 Swarm 管理容器的危险操作策略。
	ContainerSwarmActionLevelConfig ConfigKey = "ops.container.actions.swarm_level"
	// ContainerKubernetesActionLevelConfig 保存 Kubernetes 管理容器的危险操作策略。
	ContainerKubernetesActionLevelConfig ConfigKey = "ops.container.actions.kubernetes_level"
	// ContainerUnknownActionLevelConfig 保存未分类管理容器的危险操作策略。
	ContainerUnknownActionLevelConfig ConfigKey = "ops.container.actions.unknown_level"
	// ContainerShellEnabledConfig 控制是否允许交互式 Shell 会话。
	ContainerShellEnabledConfig ConfigKey = "ops.container.shell.enabled"
	// ContainerEnvironmentPolicyConfig 控制容器环境变量值展示方式。
	ContainerEnvironmentPolicyConfig ConfigKey = "ops.container.environment.policy"
	// ContainerEnvironmentMaskedCopyEnabledConfig 控制已授权读者能否复制被遮罩敏感环境变量的原始值。
	ContainerEnvironmentMaskedCopyEnabledConfig ConfigKey = "ops.container.environment.masked_copy_enabled"
)

const (
	// ContainerEnvironmentPolicyHidden 隐藏全部环境变量值。
	ContainerEnvironmentPolicyHidden EnvironmentPolicy = "hidden"
	// ContainerEnvironmentPolicyMasked 遮罩敏感环境变量值。
	ContainerEnvironmentPolicyMasked EnvironmentPolicy = "masked"
	// ContainerEnvironmentPolicyPlain 展示环境变量值。
	ContainerEnvironmentPolicyPlain EnvironmentPolicy = "plain"
)

const (
	// ContainerOrchestratorActionLevelReadonly 禁止该来源的危险操作。
	ContainerOrchestratorActionLevelReadonly OrchestratorActionLevel = "readonly"
	// ContainerOrchestratorActionLevelWarn 允许单项危险操作，但禁止批量操作。
	ContainerOrchestratorActionLevelWarn OrchestratorActionLevel = "warn"
	// ContainerOrchestratorActionLevelAllow 同时允许单项和批量危险操作。
	ContainerOrchestratorActionLevelAllow OrchestratorActionLevel = "allow"
)
