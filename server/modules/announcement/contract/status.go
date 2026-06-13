// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// AnnouncementStatus identifies a stable announcement lifecycle status contract.
type AnnouncementStatus string

// String returns the canonical status value.
func (s AnnouncementStatus) String() string {
	return string(s)
}

const (
	// AnnouncementStatusDraft indicates an unpublished management draft.
	AnnouncementStatusDraft AnnouncementStatus = "draft"
	// AnnouncementStatusPublished indicates an announcement eligible for user visibility when time rules match.
	AnnouncementStatusPublished AnnouncementStatus = "published"
	// AnnouncementStatusArchived indicates a management-retained announcement hidden from user listings.
	AnnouncementStatusArchived AnnouncementStatus = "archived"
)

// ValidAnnouncementStatus reports whether value is a known announcement status contract.
func ValidAnnouncementStatus(value AnnouncementStatus) bool {
	switch value {
	case AnnouncementStatusDraft, AnnouncementStatusPublished, AnnouncementStatusArchived:
		return true
	default:
		return false
	}
}
