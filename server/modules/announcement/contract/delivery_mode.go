package contract

// AnnouncementDeliveryMode 标识已发布公告向用户呈现的方式。
type AnnouncementDeliveryMode string

// String 返回规范化的公告投递方式值。
func (m AnnouncementDeliveryMode) String() string {
	return string(m)
}

const (
	// AnnouncementDeliveryModeSilent 仅在公告中心展示公告。
	AnnouncementDeliveryModeSilent AnnouncementDeliveryMode = "silent"
	// AnnouncementDeliveryModePopup 除公告中心外，还向目标用户弹出未读提示。
	AnnouncementDeliveryModePopup AnnouncementDeliveryMode = "popup"
)

// ValidAnnouncementDeliveryMode 判断 value 是否为已知的公告投递方式契约。
func ValidAnnouncementDeliveryMode(value AnnouncementDeliveryMode) bool {
	switch value {
	case AnnouncementDeliveryModeSilent, AnnouncementDeliveryModePopup:
		return true
	default:
		return false
	}
}
