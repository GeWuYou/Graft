package contract

// PermissionCode 标识安全模块稳定的权限契约。
type PermissionCode string

// String 返回接口传输使用的权限编码。
func (c PermissionCode) String() string { return string(c) }

const (
	// OverviewReadPermission 允许读取聚合后的安全概览。
	OverviewReadPermission PermissionCode = "security.overview.read"
)
