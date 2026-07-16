package contract

// PermissionCode 标识审计模块稳定的权限契约。
type PermissionCode string

// String 返回用于权限校验和 HTTP 契约的权限码。
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// AuditReadPermission identifies read access to audit-log data.
	AuditReadPermission PermissionCode = "audit.read"
	// AuditManagePermission identifies access to audit visibility policy management.
	AuditManagePermission PermissionCode = "audit.manage"

	// AuditRead is the canonical permission used by audit module consumers.
	AuditRead PermissionCode = AuditReadPermission
	// AuditManage is the canonical permission used by audit visibility policy management.
	AuditManage PermissionCode = AuditManagePermission
)
