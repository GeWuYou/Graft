package contract

const (
	// ScheduledTasksGroup identifies the scheduled task API route group.
	ScheduledTasksGroup = "/scheduled-tasks"
	// ScheduledTaskCollectionRoute identifies the scheduled task collection route fragment.
	ScheduledTaskCollectionRoute = ""
	// ScheduledTaskDetailRoute identifies the scheduled task detail route fragment.
	ScheduledTaskDetailRoute = "/:taskKey"
	// ScheduledTaskRunsRoute identifies the scheduled task run history route fragment.
	ScheduledTaskRunsRoute = "/:taskKey/runs"
	// ScheduledTaskRunRoute identifies the manual run route fragment.
	ScheduledTaskRunRoute = "/:taskKey/run"
	// ScheduledTaskMenuPath identifies the canonical scheduled task menu path.
	ScheduledTaskMenuPath = "/server/scheduled-tasks"
)
