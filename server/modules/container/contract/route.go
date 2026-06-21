package contract

const (
	// ContainerAPIGroup is the API route group for container management.
	ContainerAPIGroup = "/ops/containers"
	// ContainerCollectionRoute is the collection route fragment.
	ContainerCollectionRoute = ""
	// ContainerDetailRoute is the detail route fragment.
	ContainerDetailRoute = "/:id"
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
	// ContainerMenuRootPath is the web menu root path for operations.
	ContainerMenuRootPath = "/ops"
	// ContainerMenuPath is the web menu path for container management.
	ContainerMenuPath = "/ops/containers"
)
