package contract

// AnnouncementDeliveryMode identifies how a published announcement should be surfaced to users.
type AnnouncementDeliveryMode string

// String returns the canonical delivery mode value.
func (m AnnouncementDeliveryMode) String() string {
	return string(m)
}

const (
	// AnnouncementDeliveryModeSilent shows the announcement only in Announcement Center.
	AnnouncementDeliveryModeSilent AnnouncementDeliveryMode = "silent"
	// AnnouncementDeliveryModePopup also prompts unread target users with an in-app dialog.
	AnnouncementDeliveryModePopup AnnouncementDeliveryMode = "popup"
)

// ValidAnnouncementDeliveryMode reports whether value is a known announcement delivery mode contract.
func ValidAnnouncementDeliveryMode(value AnnouncementDeliveryMode) bool {
	switch value {
	case AnnouncementDeliveryModeSilent, AnnouncementDeliveryModePopup:
		return true
	default:
		return false
	}
}
