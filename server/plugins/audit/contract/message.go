package contract

// MenuMessageKey identifies a stable audit-plugin menu title message key.
type MenuMessageKey string

// String returns the canonical menu message key value.
func (k MenuMessageKey) String() string {
	return string(k)
}

const (
	// AuditLogMenuTitle identifies the localized title for the audit-log menu.
	AuditLogMenuTitle MenuMessageKey = "menu.audit.logs.title"
)
