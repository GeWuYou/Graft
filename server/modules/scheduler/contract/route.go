package contract

const (
	// ScheduledTasksGroup 标识定时任务 API 的路由组。
	ScheduledTasksGroup = "/scheduled-tasks"
	// ScheduledTaskCollectionRoute 标识定时任务集合路由片段。
	ScheduledTaskCollectionRoute = ""
	// ScheduledTaskJobDefinitionsRoute 标识可创建作业定义集合的路由片段。
	ScheduledTaskJobDefinitionsRoute = "/job-definitions"
	// ScheduledTaskJobDefinitionDetailRoute 标识单个作业定义的路由片段。
	ScheduledTaskJobDefinitionDetailRoute = "/job-definitions/:jobKey"
	// ScheduledTaskDetailRoute 标识定时任务详情路由片段。
	ScheduledTaskDetailRoute = "/:taskKey"
	// ScheduledTaskEnableRoute 标识启用定时任务的路由片段。
	ScheduledTaskEnableRoute = "/:taskKey/enable"
	// ScheduledTaskDisableRoute 标识停用定时任务的路由片段。
	ScheduledTaskDisableRoute = "/:taskKey/disable"
	// ScheduledTaskRunRoute 标识手动运行定时任务的路由片段。
	ScheduledTaskRunRoute = "/:taskKey/run"
	// ScheduledTaskActionRoute 标识后端定义的单个任务操作路由片段。
	ScheduledTaskActionRoute = "/:taskKey/actions/:actionKey"
	// ScheduledTaskRunsRoute 标识定时任务运行历史的路由片段。
	ScheduledTaskRunsRoute = "/:taskKey/runs"
	// ScheduledTaskRunDetailRoute 标识单条运行历史详情的路由片段。
	ScheduledTaskRunDetailRoute = "/runs/:runID"
	// ScheduledTaskMenuPath 标识定时任务菜单的规范路径。
	ScheduledTaskMenuPath = "/platform/scheduled-tasks"
	// ScheduledTaskSavedViewsRoute 标识定时任务列表保存视图集合路由。
	ScheduledTaskSavedViewsRoute = "/saved-views"
	// ScheduledTaskSavedViewRoute 标识定时任务列表单个保存视图路由。
	ScheduledTaskSavedViewRoute = "/saved-views/:viewId"
)
