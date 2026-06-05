package contract

const (
	// ScheduledTasksGroup identifies the scheduled task API route group.
	ScheduledTasksGroup = "/scheduled-tasks"
	// ScheduledTaskCollectionRoute identifies the scheduled task collection route fragment.
	ScheduledTaskCollectionRoute = ""
	// ScheduledTaskDetailRoute identifies the scheduled task detail route fragment.
	ScheduledTaskDetailRoute = "/:taskKey"
	// ScheduledTaskEnableRoute identifies the scheduled task enable route fragment.
	ScheduledTaskEnableRoute = "/:taskKey/enable"
	// ScheduledTaskDisableRoute identifies the scheduled task disable route fragment.
	ScheduledTaskDisableRoute = "/:taskKey/disable"
	// ScheduledTaskRunRoute identifies the manual run route fragment.
	ScheduledTaskRunRoute = "/:taskKey/run"
	// ScheduledTaskRunsRoute identifies the scheduled task run history route fragment.
	ScheduledTaskRunsRoute = "/:taskKey/runs"
	// ScheduledTaskRunDetailRoute identifies one run-history detail route fragment.
	ScheduledTaskRunDetailRoute = "/runs/:runID"
	// ScheduledTaskMenuPath identifies the canonical scheduled task menu path.
	ScheduledTaskMenuPath = "/server/scheduled-tasks"
)
