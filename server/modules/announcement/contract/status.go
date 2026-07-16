package contract

// AnnouncementStatus 标识公告稳定的生命周期状态契约。
type AnnouncementStatus string

// String 返回规范化的公告状态值。
func (s AnnouncementStatus) String() string {
	return string(s)
}

const (
	// AnnouncementStatusDraft 表示尚未发布的管理端草稿。
	AnnouncementStatusDraft AnnouncementStatus = "draft"
	// AnnouncementStatusPublished 表示在时间条件满足时可对用户可见的公告。
	AnnouncementStatusPublished AnnouncementStatus = "published"
	// AnnouncementStatusArchived 表示管理端保留但从用户列表隐藏的公告。
	AnnouncementStatusArchived AnnouncementStatus = "archived"
)

// ValidAnnouncementStatus 判断 value 是否为已知的公告状态契约。
func ValidAnnouncementStatus(value AnnouncementStatus) bool {
	switch value {
	case AnnouncementStatusDraft, AnnouncementStatusPublished, AnnouncementStatusArchived:
		return true
	default:
		return false
	}
}
