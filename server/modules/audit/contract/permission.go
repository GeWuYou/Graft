// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// PermissionCode identifies a stable audit module permission contract.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// AuditReadPermission identifies read access to audit-log data.
	AuditReadPermission PermissionCode = "audit.read"

	// AuditRead is the canonical permission used by audit module consumers.
	AuditRead PermissionCode = AuditReadPermission
)
