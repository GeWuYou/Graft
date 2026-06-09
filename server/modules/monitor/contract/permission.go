// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// PermissionCode identifies a stable monitor module permission contract.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// ServerStatusReadPermission identifies read access to server status data.
	ServerStatusReadPermission PermissionCode = "monitor.server-status.read"

	// ServerStatusRead is the canonical permission used by monitor module consumers.
	ServerStatusRead PermissionCode = ServerStatusReadPermission
)
