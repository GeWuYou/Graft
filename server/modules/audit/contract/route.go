// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// JoinRoute combines a route group path with a route fragment.
func JoinRoute(group, fragment string) string {
	return group + fragment
}

const (
	// AuditGroup identifies the audit route group.
	AuditGroup = "/audit"

	// AuditIncidentParam identifies the canonical audit incident route parameter name.
	AuditIncidentParam = "event_id"

	// AuditCollection identifies the audit-log collection route fragment.
	AuditCollection = "/logs"

	// AuditOverviewCollection identifies the audit overview route fragment.
	AuditOverviewCollection = "/overview"

	// AuditIncidentItem identifies the audit incident route fragment.
	AuditIncidentItem = "/incidents/:" + AuditIncidentParam

	// AuditMenuPath identifies the canonical audit root menu path.
	AuditMenuPath = AuditGroup

	// AuditOverviewMenuPath identifies the canonical audit overview menu path.
	AuditOverviewMenuPath = AuditGroup + "/overview"

	// AuditLogsMenuPath identifies the canonical audit logs menu path.
	AuditLogsMenuPath = AuditGroup + AuditCollection

	// AuditOverviewAPIPath identifies the canonical audit overview API path.
	AuditOverviewAPIPath = AuditGroup + AuditOverviewCollection

	// AuditIncidentAPIPath identifies the canonical audit incident API path template.
	AuditIncidentAPIPath = AuditGroup + "/incidents/{" + AuditIncidentParam + "}"
)
