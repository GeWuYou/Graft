package contract

// PermissionCode 标识 platform-update 模块稳定的权限码。
type PermissionCode string

// String 返回权限的 wire-format 表示。
func (c PermissionCode) String() string { return string(c) }

const (
	// UpdateReadPermission 允许读取当前版本、可用发布和安装能力。
	UpdateReadPermission PermissionCode = "platform-update.read"
	// UpdateCheckPermission 允许主动刷新上游发布目录。
	UpdateCheckPermission PermissionCode = "platform-update.check"
	// UpdateManagePermission 为后续人工确认升级和恢复操作预留管理边界。
	UpdateManagePermission PermissionCode = "platform-update.manage"
)
