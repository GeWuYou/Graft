package contract

// MessageKey identifies a stable announcement message key.
type MessageKey string

// String returns the canonical message key value.
func (k MessageKey) String() string {
	return string(k)
}

const (
	// AnnouncementMenuTitle identifies the announcement management menu title.
	AnnouncementMenuTitle MessageKey = "menu.server.announcements.title"
	// AnnouncementPublishedDeleteForbidden identifies published announcement delete conflicts.
	AnnouncementPublishedDeleteForbidden MessageKey = "announcement.published_delete_forbidden"
)
