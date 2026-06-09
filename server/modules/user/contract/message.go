// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// MenuMessageKey identifies a stable user module menu title message key.
type MenuMessageKey string

// String returns the canonical menu message key value.
func (k MenuMessageKey) String() string {
	return string(k)
}

const (
	// UserListMenuTitle identifies the localized title for the user list menu.
	UserListMenuTitle MenuMessageKey = "menu.access_control.users.title"
)
