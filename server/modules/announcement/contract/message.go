package contract

// MessageKey 标识公告模块稳定的消息 key。
type MessageKey string

// String 返回规范化的消息 key。
func (k MessageKey) String() string {
	return string(k)
}

const (
	// AnnouncementMenuTitle 标识公告管理菜单标题。
	AnnouncementMenuTitle MessageKey = "menu.announcement.title"
	// AnnouncementPublishedDeleteForbidden 标识已发布公告直接删除时的冲突提示。
	AnnouncementPublishedDeleteForbidden MessageKey = "announcement.published_delete_forbidden"
)
