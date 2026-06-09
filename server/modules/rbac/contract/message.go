// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// MenuMessageKey identifies a stable rbac module menu title message key.
type MenuMessageKey string

// String returns the canonical menu message key value.
func (k MenuMessageKey) String() string {
	return string(k)
}

const (
	// AccessControlMenuTitle identifies the localized title for the access-control root menu.
	AccessControlMenuTitle MenuMessageKey = "menu.access_control.title"
	// RoleListMenuTitle identifies the localized title for the role list menu.
	RoleListMenuTitle MenuMessageKey = "menu.access_control.roles.title"
	// PermissionListMenuTitle identifies the localized title for the permission list menu.
	PermissionListMenuTitle MenuMessageKey = "menu.access_control.permissions.title"
	// AccessControlOverviewMenuTitle identifies the localized title for the access-control overview menu.
	AccessControlOverviewMenuTitle MenuMessageKey = "menu.access_control.overview.title"
)
