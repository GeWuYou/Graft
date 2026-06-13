// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

// AnnouncementLevel identifies a stable announcement presentation level contract.
type AnnouncementLevel string

// String returns the canonical level value.
func (l AnnouncementLevel) String() string {
	return string(l)
}

const (
	// AnnouncementLevelInfo indicates neutral platform information.
	AnnouncementLevelInfo AnnouncementLevel = "info"
	// AnnouncementLevelWarning indicates information that needs attention.
	AnnouncementLevelWarning AnnouncementLevel = "warning"
	// AnnouncementLevelSuccess indicates a positive or completed platform announcement.
	AnnouncementLevelSuccess AnnouncementLevel = "success"
	// AnnouncementLevelError indicates a high-impact or failure-related announcement.
	AnnouncementLevelError AnnouncementLevel = "error"
)

// ValidAnnouncementLevel reports whether value is a known announcement level contract.
func ValidAnnouncementLevel(value AnnouncementLevel) bool {
	switch value {
	case AnnouncementLevelInfo, AnnouncementLevelWarning, AnnouncementLevelSuccess, AnnouncementLevelError:
		return true
	default:
		return false
	}
}
