// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package contract

const (
	// AnnouncementGroup identifies the announcement management API route group.
	AnnouncementGroup = "/announcements"
	// AnnouncementCollectionRoute identifies the management collection route fragment.
	AnnouncementCollectionRoute = ""
	// AnnouncementDetailRoute identifies one management announcement route fragment.
	AnnouncementDetailRoute = "/:id"
	// AnnouncementPublishRoute identifies the management publish action route fragment.
	AnnouncementPublishRoute = "/:id/publish"
	// AnnouncementArchiveRoute identifies the management archive action route fragment.
	AnnouncementArchiveRoute = "/:id/archive"

	// MyAnnouncementGroup identifies current-user announcement API routes.
	MyAnnouncementGroup = "/my/announcements"
	// MyAnnouncementCollectionRoute identifies the current-user announcement collection route fragment.
	MyAnnouncementCollectionRoute = ""
	// MyAnnouncementReadRoute identifies the current-user mark-read route fragment.
	MyAnnouncementReadRoute = "/:id/read"
	// MyAnnouncementReadAllRoute identifies the current-user mark-all-read route fragment.
	MyAnnouncementReadAllRoute = "/read-all"
	// MyAnnouncementUnreadCountRoute identifies the current-user unread-count route fragment.
	MyAnnouncementUnreadCountRoute = "/unread-count"

	// AnnouncementMenuPath identifies the canonical announcement management menu path.
	AnnouncementMenuPath = "/server/announcements"
)
