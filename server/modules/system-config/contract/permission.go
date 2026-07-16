package contract

// PermissionCode 标识系统配置模块稳定的权限契约。
type PermissionCode string

// String 返回接口传输使用的权限编码。
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// SystemConfigReadPermission 允许读取系统配置定义和有效值。
	SystemConfigReadPermission PermissionCode = "system-config.read"
	// SystemConfigWritePermission 允许写入用户覆盖值。
	SystemConfigWritePermission PermissionCode = "system-config.write"
)
