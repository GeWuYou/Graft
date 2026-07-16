package contract

// PermissionCode 标识 monitor 模块稳定的权限契约。
type PermissionCode string

// String 返回线路传输格式的权限码。
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// ServerStatusReadPermission 标识服务器状态数据的读取权限。
	ServerStatusReadPermission PermissionCode = "monitor.server-status.read"

	// ServerStatusRead 是 monitor 模块消费者使用的规范权限码。
	ServerStatusRead PermissionCode = ServerStatusReadPermission
)
