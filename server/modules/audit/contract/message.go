// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// MenuMessageKey identifies a stable audit module menu title message key.
type MenuMessageKey string

// String returns the canonical menu message key value.
func (k MenuMessageKey) String() string {
	return string(k)
}

const (
	// AuditRootMenuTitle identifies the localized title for the audit root menu.
	AuditRootMenuTitle MenuMessageKey = "menu.audit.title"
	// AuditOverviewMenuTitle identifies the localized title for the audit overview menu.
	AuditOverviewMenuTitle MenuMessageKey = "menu.audit.overview.title"
	// AuditLogMenuTitle identifies the localized title for the audit-log menu.
	AuditLogMenuTitle MenuMessageKey = "menu.audit.logs.title"
)
