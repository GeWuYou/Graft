package contract

// AnnouncementLevel 标识公告稳定的展示等级契约。
type AnnouncementLevel string

// String 返回规范化的公告等级值。
func (l AnnouncementLevel) String() string {
	return string(l)
}

const (
	// AnnouncementLevelInfo 表示中性的系统信息。
	AnnouncementLevelInfo AnnouncementLevel = "info"
	// AnnouncementLevelWarning 表示需要关注的信息。
	AnnouncementLevelWarning AnnouncementLevel = "warning"
	// AnnouncementLevelSuccess 表示积极或已完成的系统公告。
	AnnouncementLevelSuccess AnnouncementLevel = "success"
	// AnnouncementLevelError 表示高影响或与失败相关的公告。
	AnnouncementLevelError AnnouncementLevel = "error"
)

// ValidAnnouncementLevel 判断 value 是否为已知的公告等级契约。
func ValidAnnouncementLevel(value AnnouncementLevel) bool {
	switch value {
	case AnnouncementLevelInfo, AnnouncementLevelWarning, AnnouncementLevelSuccess, AnnouncementLevelError:
		return true
	default:
		return false
	}
}
