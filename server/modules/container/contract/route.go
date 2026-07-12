package contract

const (
	// ContainerAPIGroup is the API route group for container management.
	ContainerAPIGroup = "/ops/containers"
	// ContainerCollectionRoute is the collection route fragment.
	ContainerCollectionRoute = ""
	// ContainerDashboardSummaryRoute is the dashboard summary route fragment.
	ContainerDashboardSummaryRoute = "/dashboard-summary"
	// ContainerDetailRoute is the detail route fragment.
	ContainerDetailRoute = "/:id"
	// ContainerEventsRoute is the runtime events route fragment.
	ContainerEventsRoute = "/:id/events"
	// ContainerLogsRoute is the log route fragment.
	ContainerLogsRoute = "/:id/logs"
	// ContainerShellSessionsRoute is the shell session issue route fragment.
	ContainerShellSessionsRoute = "/:id/shell/sessions"
	// ContainerShellWebSocketRoute is the shell websocket route fragment.
	ContainerShellWebSocketRoute = "/:id/shell/ws"
	// ContainerMountUsageRoute is the mount usage route fragment.
	ContainerMountUsageRoute = "/:id/mounts/usage"
	// ContainerMountUsageRefreshRoute is the mount usage refresh route fragment.
	ContainerMountUsageRefreshRoute = "/:id/mounts/:mountId/usage/refresh"
	// ContainerStartRoute is the start action route fragment.
	ContainerStartRoute = "/:id/start"
	// ContainerStopRoute is the stop action route fragment.
	ContainerStopRoute = "/:id/stop"
	// ContainerRestartRoute is the restart action route fragment.
	ContainerRestartRoute = "/:id/restart"
	// ContainerRemoveRoute is the remove action route fragment.
	ContainerRemoveRoute = "/:id/remove"
	// ContainerBatchActionsRoute is the batch action route fragment.
	ContainerBatchActionsRoute = "/batch-actions"
	// DockerAPIGroup is the read-only Docker-native resource API route group.
	DockerAPIGroup = "/ops/docker"
	// DockerImagesRoute is the Docker image collection route fragment.
	DockerImagesRoute = "/images"
	// DockerImageRoute is the Docker image detail route fragment.
	DockerImageRoute = "/images/:id"
	// DockerNetworksRoute is the Docker network collection route fragment.
	DockerNetworksRoute = "/networks"
	// DockerNetworkRoute is the Docker network detail route fragment.
	DockerNetworkRoute = "/networks/:id"
	// DockerVolumesRoute is the Docker volume collection route fragment.
	DockerVolumesRoute = "/volumes"
	// DockerVolumeRoute is the Docker volume detail route fragment.
	DockerVolumeRoute = "/volumes/:id"
	// DockerSystemRoute is the Docker system information route fragment.
	DockerSystemRoute = "/system"
	// ContainerMenuRootPath is the web menu root path for operations.
	ContainerMenuRootPath = "/containers"
	// ContainerMenuPath is the web menu path for container management.
	ContainerMenuPath = "/containers"
)
