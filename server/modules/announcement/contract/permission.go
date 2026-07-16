package contract

// PermissionCode 标识公告模块稳定的权限契约。
//
// 权限值由本包作为唯一权威维护，直到本包明确标记替换或移除。
type PermissionCode string

// String 返回接口传输使用的权限编码。
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// AnnouncementReadPermission 允许读取管理端公告。
	AnnouncementReadPermission PermissionCode = "announcement.read"
	// AnnouncementCreatePermission 允许创建管理端公告。
	AnnouncementCreatePermission PermissionCode = "announcement.create"
	// AnnouncementUpdatePermission 允许更新管理端公告。
	AnnouncementUpdatePermission PermissionCode = "announcement.update"
	// AnnouncementPublishPermission 允许发布和归档管理端公告。
	AnnouncementPublishPermission PermissionCode = "announcement.publish"
	// AnnouncementDeletePermission 允许软删除管理端公告。
	AnnouncementDeletePermission PermissionCode = "announcement.delete"
)
