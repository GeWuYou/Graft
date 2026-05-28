package contract

// JoinRoute combines a route group path with a route fragment.
func JoinRoute(group, fragment string) string {
	return group + fragment
}

const (
	// AuditGroup identifies the audit route group.
	AuditGroup = "/audit"

	// AuditCollection identifies the audit-log collection route fragment.
	AuditCollection = "/logs"

	// AuditMenuPath identifies the canonical audit root menu path.
	AuditMenuPath = AuditGroup

	// AuditOverviewMenuPath identifies the canonical audit overview menu path.
	AuditOverviewMenuPath = AuditGroup + "/overview"

	// AuditLogsMenuPath identifies the canonical audit logs menu path.
	AuditLogsMenuPath = AuditGroup + AuditCollection
)
