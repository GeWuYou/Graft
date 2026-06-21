package contract

// PermissionCode identifies a stable announcement module permission contract.
//
// Canonical owner: server/modules/announcement/contract.
// Lifecycle: stable values remain authoritative until this package marks a replacement or removal.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// AnnouncementReadPermission identifies management-side read access to announcements.
	// Lifecycle: stable.
	AnnouncementReadPermission PermissionCode = "announcement.read"
	// AnnouncementCreatePermission identifies management-side announcement creation access.
	// Lifecycle: stable.
	AnnouncementCreatePermission PermissionCode = "announcement.create"
	// AnnouncementUpdatePermission identifies management-side announcement update access.
	// Lifecycle: stable.
	AnnouncementUpdatePermission PermissionCode = "announcement.update"
	// AnnouncementPublishPermission identifies management-side publish and archive access.
	// Lifecycle: stable.
	AnnouncementPublishPermission PermissionCode = "announcement.publish"
	// AnnouncementDeletePermission identifies management-side soft-delete access.
	// Lifecycle: stable.
	AnnouncementDeletePermission PermissionCode = "announcement.delete"
)
