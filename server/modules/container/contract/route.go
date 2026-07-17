package contract

const (
	// ContainerAPIGroup 是容器管理 API 的根路由组，供模块路由注册与前端请求共同使用。
	ContainerAPIGroup = "/ops/containers"
	// ContainerCollectionRoute 是容器集合路由片段，与 API 根路由组拼接后表示容器列表资源。
	ContainerCollectionRoute = ""
	// ContainerDashboardSummaryRoute 是容器仪表盘摘要路由片段。
	ContainerDashboardSummaryRoute = "/dashboard-summary"
	// ContainerDetailRoute 是容器详情路由片段。
	ContainerDetailRoute = "/:id"
	// ContainerEventsRoute 是容器运行时事件路由片段。
	ContainerEventsRoute = "/:id/events"
	// ContainerLogsRoute 是容器日志路由片段。
	ContainerLogsRoute = "/:id/logs"
	// ContainerShellSessionsRoute 是容器 Shell 会话签发路由片段。
	ContainerShellSessionsRoute = "/:id/shell/sessions"
	// ContainerShellWebSocketRoute 是容器 Shell WebSocket 连接路由片段。
	ContainerShellWebSocketRoute = "/:id/shell/ws"
	// ContainerMountUsageRoute 是容器挂载使用量路由片段。
	ContainerMountUsageRoute = "/:id/mounts/usage"
	// ContainerMountUsageRefreshRoute 是刷新容器挂载使用量的路由片段。
	ContainerMountUsageRefreshRoute = "/:id/mounts/:mountId/usage/refresh"
	// ContainerStartRoute 是启动容器动作的路由片段。
	ContainerStartRoute = "/:id/start"
	// ContainerStopRoute 是停止容器动作的路由片段。
	ContainerStopRoute = "/:id/stop"
	// ContainerRestartRoute 是重启容器动作的路由片段。
	ContainerRestartRoute = "/:id/restart"
	// ContainerRemoveRoute 是移除容器动作的路由片段。
	ContainerRemoveRoute = "/:id/remove"
	// ContainerBatchActionsRoute 是容器批量动作的路由片段。
	ContainerBatchActionsRoute = "/batch-actions"
	// DockerAPIGroup 是 Docker 原生资源 API 的根路由组。
	DockerAPIGroup = "/ops/docker"
	// DockerImagesRoute 是 Docker 镜像集合路由片段。
	DockerImagesRoute = "/images"
	// DockerImageRoute 是 Docker 镜像详情路由片段。
	DockerImageRoute = "/images/:id"
	// DockerNetworksRoute 是 Docker 网络集合路由片段。
	DockerNetworksRoute = "/networks"
	// DockerNetworkRoute 是 Docker 网络详情路由片段。
	DockerNetworkRoute = "/networks/:id"
	// DockerVolumesRoute 是 Docker 卷集合路由片段。
	DockerVolumesRoute = "/volumes"
	// DockerVolumeRoute 是 Docker 卷详情路由片段。
	DockerVolumeRoute = "/volumes/:id"
	// DockerSystemRoute 是 Docker 系统信息路由片段。
	DockerSystemRoute = "/system"
	// ContainerMenuRootPath 是运维侧容器菜单的根路径。
	ContainerMenuRootPath = "/infrastructure/docker/containers"
	// ContainerMenuPath 是容器管理菜单路径，与前端路由和后端菜单注册保持一致。
	ContainerMenuPath = "/infrastructure/docker/containers"
	// DockerNetworkMenuPath 是 Docker 网络管理菜单路径。
	DockerNetworkMenuPath = "/infrastructure/docker/networks"
)
