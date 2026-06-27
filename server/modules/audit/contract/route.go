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

	// AuditLogParam identifies the canonical audit log route parameter name.
	AuditLogParam = "id"

	// AuditCollection identifies the audit-log collection route fragment.
	AuditCollection = "/logs"

	// AuditItem identifies the audit-log detail route fragment.
	AuditItem = AuditCollection + "/:" + AuditLogParam

	// AuditOverviewCollection identifies the audit overview route fragment.
	AuditOverviewCollection = "/overview"
	// AuditVisibilityPolicyCollection identifies the audit visibility policy route fragment.
	AuditVisibilityPolicyCollection = "/policies/visibility"
	// AuditVisibilityOverrideCollection identifies the audit visibility override route fragment.
	AuditVisibilityOverrideCollection = AuditVisibilityPolicyCollection + "/overrides"

	// AuditIncidentItem identifies the audit incident route fragment.
	AuditIncidentItem = "/incidents/:" + AuditIncidentParam

	// AuditMenuPath identifies the canonical audit root menu path.
	AuditMenuPath = AuditGroup

	// AuditOverviewMenuPath identifies the canonical audit overview menu path.
	AuditOverviewMenuPath = AuditGroup + "/overview"

	// AuditLogsMenuPath identifies the canonical audit logs menu path.
	AuditLogsMenuPath = AuditGroup + AuditCollection

	// AuditLogDetailAPIPath identifies the canonical audit log detail API path template.
	AuditLogDetailAPIPath = AuditGroup + AuditCollection + "/{" + AuditLogParam + "}"

	// AuditOverviewAPIPath identifies the canonical audit overview API path.
	AuditOverviewAPIPath = AuditGroup + AuditOverviewCollection
	// AuditVisibilityPolicyAPIPath identifies the canonical audit visibility policy API path.
	AuditVisibilityPolicyAPIPath = AuditGroup + AuditVisibilityPolicyCollection
	// AuditVisibilityOverrideAPIPath identifies the canonical audit visibility override API path.
	AuditVisibilityOverrideAPIPath = AuditGroup + AuditVisibilityOverrideCollection

	// AuditIncidentAPIPath identifies the canonical audit incident API path template.
	AuditIncidentAPIPath = AuditGroup + "/incidents/{" + AuditIncidentParam + "}"
)
