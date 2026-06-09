// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// MessageKey identifies a stable system configuration message key.
type MessageKey string

// String returns the canonical message key value.
func (k MessageKey) String() string {
	return string(k)
}

const (
	// SystemConfigMenuTitle identifies the system configuration menu title.
	SystemConfigMenuTitle MessageKey = "menu.server.system_config.title"
	// SystemConfigNotFound identifies the not-found error message.
	SystemConfigNotFound MessageKey = "system_config.not_found"
	// SystemConfigInvalidRequest identifies the invalid-request error message.
	SystemConfigInvalidRequest MessageKey = "system_config.invalid_request"
)
