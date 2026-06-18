// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// MenuMessageKey identifies a stable audit module menu title message key.
type MenuMessageKey string

// TargetLabelMessageKey identifies a stable localized label for one built-in audit target type.
type TargetLabelMessageKey string

// String returns the canonical menu message key value.
func (k MenuMessageKey) String() string {
	return string(k)
}

// String returns the canonical target-label message key value.
func (k TargetLabelMessageKey) String() string {
	return string(k)
}

const (
	// AuditRootMenuTitle identifies the localized title for the audit root menu.
	AuditRootMenuTitle MenuMessageKey = "menu.audit.title"
	// AuditOverviewMenuTitle identifies the localized title for the audit overview menu.
	AuditOverviewMenuTitle MenuMessageKey = "menu.audit.overview.title"
	// AuditLogMenuTitle identifies the localized title for the audit-log menu.
	AuditLogMenuTitle MenuMessageKey = "menu.audit.logs.title"

	// AuditTargetLabelUser identifies the localized label for built-in user targets.
	AuditTargetLabelUser TargetLabelMessageKey = "audit.target.user"
	// AuditTargetLabelRole identifies the localized label for built-in role targets.
	AuditTargetLabelRole TargetLabelMessageKey = "audit.target.role"
	// AuditTargetLabelPermission identifies the localized label for built-in permission targets.
	AuditTargetLabelPermission TargetLabelMessageKey = "audit.target.permission"
	// AuditTargetLabelAudit identifies the localized label for built-in audit targets.
	AuditTargetLabelAudit TargetLabelMessageKey = "audit.target.audit"
	// AuditTargetLabelServerStatus identifies the localized label for built-in server-status targets.
	AuditTargetLabelServerStatus TargetLabelMessageKey = "audit.target.serverStatus"
	// AuditTargetLabelAuth identifies the localized label for built-in authentication targets.
	AuditTargetLabelAuth TargetLabelMessageKey = "audit.target.auth"
)
