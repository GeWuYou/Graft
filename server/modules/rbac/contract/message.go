// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// MessageKey identifies a stable rbac module message key.
type MessageKey string

// String returns the canonical menu message key value.
func (k MessageKey) String() string {
	return string(k)
}

const (
	// AccessControlMenuTitle identifies the localized title for the access-control root menu.
	AccessControlMenuTitle MessageKey = "menu.access_control.title"
	// RoleListMenuTitle identifies the localized title for the role list menu.
	RoleListMenuTitle MessageKey = "menu.access_control.roles.title"
	// PermissionListMenuTitle identifies the localized title for the permission list menu.
	PermissionListMenuTitle MessageKey = "menu.access_control.permissions.title"
	// AccessControlOverviewMenuTitle identifies the localized title for the access-control overview menu.
	AccessControlOverviewMenuTitle MessageKey = "menu.access_control.overview.title"
	// AuditRolePermissionsAdded identifies role-permission append audit messages.
	AuditRolePermissionsAdded MessageKey = "rbac.audit.rolePermissionsAdded"
	// AuditRolePermissionsRemoved identifies role-permission removal audit messages.
	AuditRolePermissionsRemoved MessageKey = "rbac.audit.rolePermissionsRemoved"
)
