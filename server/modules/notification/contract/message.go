// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// MenuMessageKey identifies a stable notification module menu title message key.
type MenuMessageKey string

// String returns the canonical menu message key value.
func (k MenuMessageKey) String() string {
	return string(k)
}

const (
	// NotificationMenuTitle identifies the localized title for the notification center menu.
	NotificationMenuTitle MenuMessageKey = "menu.notification.title"
)
