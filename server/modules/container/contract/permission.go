// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// PermissionCode identifies a stable container management permission contract.
//
// Canonical owner: server/modules/container/contract.
// Lifecycle: stable values remain authoritative until this package marks a replacement or removal.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// ContainerViewPermission identifies container list access.
	// Lifecycle: stable.
	ContainerViewPermission PermissionCode = "ops.container.view"
	// ContainerDetailPermission identifies container detail access.
	// Lifecycle: stable.
	ContainerDetailPermission PermissionCode = "ops.container.detail"
	// ContainerEnvironmentPermission identifies container environment variable value access.
	// Lifecycle: stable.
	ContainerEnvironmentPermission PermissionCode = "ops.container.environment"
	// ContainerLogsPermission identifies container log access.
	// Lifecycle: stable.
	ContainerLogsPermission PermissionCode = "ops.container.logs"
	// ContainerShellPermission identifies interactive shell session access.
	// Lifecycle: stable.
	ContainerShellPermission PermissionCode = "ops.container.shell"
	// ContainerStartPermission identifies container start access.
	// Lifecycle: stable.
	ContainerStartPermission PermissionCode = "ops.container.start"
	// ContainerStopPermission identifies container stop access.
	// Lifecycle: stable.
	ContainerStopPermission PermissionCode = "ops.container.stop"
	// ContainerRestartPermission identifies container restart access.
	// Lifecycle: stable.
	ContainerRestartPermission PermissionCode = "ops.container.restart"
	// ContainerRemovePermission identifies container remove access.
	// Lifecycle: stable.
	ContainerRemovePermission PermissionCode = "ops.container.remove"
)
