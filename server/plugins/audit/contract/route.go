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

	// AuditMenuPath identifies the audit page path exposed through bootstrap menus.
	AuditMenuPath = "/audit/logs"
)
