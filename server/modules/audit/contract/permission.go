package contract

// PermissionCode 标识审计模块稳定的权限契约。
type PermissionCode string

// String 返回用于权限校验和 HTTP 契约的权限码。
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// AuditReadPermission 表示审计日志数据读取权限。
	AuditReadPermission PermissionCode = "audit.read"
	// AuditManagePermission 表示审计可见性策略管理权限。
	AuditManagePermission PermissionCode = "audit.manage"
	// AuditDeletePermission 表示审计日志手工删除权限。
	AuditDeletePermission PermissionCode = "audit.delete"

	// AuditRead 是审计模块消费者使用的规范读取权限。
	AuditRead PermissionCode = AuditReadPermission
	// AuditManage 是审计可见性策略管理使用的规范权限。
	AuditManage PermissionCode = AuditManagePermission
	// AuditDelete 是审计日志删除使用的规范权限。
	AuditDelete PermissionCode = AuditDeletePermission
)
