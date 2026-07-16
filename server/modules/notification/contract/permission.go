package contract

// PermissionCode 标识通知模块稳定的权限契约。
//
// 权限值由本包作为唯一权威维护，直到本包明确标记替换或移除。
type PermissionCode string

// String 返回接口传输使用的权限编码。
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// NotificationViewPermission 允许读取当前用户通知和未读数量。
	NotificationViewPermission PermissionCode = "notification.view"
	// NotificationReadPermission 允许修改当前用户通知的已读和删除状态。
	NotificationReadPermission PermissionCode = "notification.read"
	// NotificationManagePermission 预留给未来的全局通知管理能力。
	NotificationManagePermission PermissionCode = "notification.manage"
)
