package contract

const (
	// AnnouncementGroup 标识公告管理 API 的路由组。
	AnnouncementGroup = "/announcements"
	// AnnouncementCollectionRoute 标识公告管理集合路由片段。
	AnnouncementCollectionRoute = ""
	// AnnouncementSavedViewsRoute 标识公告管理列表私有保存视图集合路由片段。
	AnnouncementSavedViewsRoute = "/saved-views"
	// AnnouncementSavedViewRoute 标识单个公告管理列表私有保存视图路由片段。
	AnnouncementSavedViewRoute = "/saved-views/:viewId"
	// AnnouncementDetailRoute 标识单条公告管理路由片段。
	AnnouncementDetailRoute = "/:id"
	// AnnouncementPublishRoute 标识公告发布操作的路由片段。
	AnnouncementPublishRoute = "/:id/publish"
	// AnnouncementArchiveRoute 标识公告归档操作的路由片段。
	AnnouncementArchiveRoute = "/:id/archive"

	// MyAnnouncementGroup 标识当前用户公告 API 的路由组。
	MyAnnouncementGroup = "/my/announcements"
	// MyAnnouncementCollectionRoute 标识当前用户公告集合路由片段。
	MyAnnouncementCollectionRoute = ""
	// MyAnnouncementReadRoute 标识当前用户标记已读的路由片段。
	MyAnnouncementReadRoute = "/:id/read"
	// MyAnnouncementReadAllRoute 标识当前用户全部标记已读的路由片段。
	MyAnnouncementReadAllRoute = "/read-all"
	// MyAnnouncementUnreadCountRoute 标识当前用户未读数量的路由片段。
	MyAnnouncementUnreadCountRoute = "/unread-count"

	// AnnouncementMenuPath 标识公告管理菜单的规范路径。
	AnnouncementMenuPath = "/platform/announcements"
	// MyAnnouncementMenuPath 标识当前用户公告列表的规范前端路径。
	MyAnnouncementMenuPath = "/my-announcements"
)
