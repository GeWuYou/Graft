// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package announcementopenapi

//go:generate go tool oapi-codegen --include-operation-ids getAnnouncements,postAnnouncements,getAnnouncement,putAnnouncement,postAnnouncementPublish,postAnnouncementArchive,deleteAnnouncement,getMyAnnouncements,postMyAnnouncementRead,postMyAnnouncementsReadAll,getMyAnnouncementsUnreadCount --generate types --package announcementopenapi -o zz_generated.announcement.go ../../../../../openapi/openapi.yaml
