// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// PermissionCode identifies a stable notification module permission contract.
//
// Canonical owner: server/modules/notification/contract.
// Lifecycle: stable values remain authoritative until this package marks a replacement or removal.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// NotificationViewPermission identifies read access to current-user notifications and unread count.
	// Lifecycle: stable.
	NotificationViewPermission PermissionCode = "notification.view"
	// NotificationReadPermission identifies access to mutate current-user read/delete delivery state.
	// Lifecycle: stable.
	NotificationReadPermission PermissionCode = "notification.read"
	// NotificationManagePermission is reserved for future global notification management.
	// Lifecycle: experimental.
	NotificationManagePermission PermissionCode = "notification.manage"
)
