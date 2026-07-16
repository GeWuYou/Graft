package contract

// PermissionCode 是容器管理的稳定权限契约。
//
// 规范 owner：server/modules/container/contract。
// 生命周期：除非本包明确标记替代或移除，稳定值始终是权限匹配的权威值。
type PermissionCode string

// String 返回用于线序列化的权限码。
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// ContainerViewPermission 表示容器列表访问权限，值保持稳定。
	ContainerViewPermission PermissionCode = "ops.container.view"
	// ContainerDetailPermission 表示容器详情访问权限，值保持稳定。
	ContainerDetailPermission PermissionCode = "ops.container.detail"
	// ContainerEventsPermission 表示容器运行时事件访问权限，值保持稳定。
	ContainerEventsPermission PermissionCode = "ops.container.events"
	// ContainerEnvironmentPermission 表示容器环境变量值访问权限，值保持稳定。
	ContainerEnvironmentPermission PermissionCode = "ops.container.environment"
	// ContainerLogsPermission 表示容器日志访问权限，值保持稳定。
	ContainerLogsPermission PermissionCode = "ops.container.logs"
	// ContainerShellPermission 表示交互式 Shell 会话访问权限，值保持稳定。
	ContainerShellPermission PermissionCode = "ops.container.shell"
	// ContainerStartPermission 表示容器启动权限，值保持稳定。
	ContainerStartPermission PermissionCode = "ops.container.start"
	// ContainerStopPermission 表示容器停止权限，值保持稳定。
	ContainerStopPermission PermissionCode = "ops.container.stop"
	// ContainerRestartPermission 表示容器重启权限，值保持稳定。
	ContainerRestartPermission PermissionCode = "ops.container.restart"
	// ContainerRemovePermission 表示容器移除权限，值保持稳定。
	ContainerRemovePermission PermissionCode = "ops.container.remove"
)
