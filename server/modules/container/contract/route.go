// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

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
