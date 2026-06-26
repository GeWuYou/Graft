package contract

const (
	// ContainerListStatsTopic is the realtime topic for list-level container stats snapshots.
	ContainerListStatsTopic = "container.stats.list"
	// ContainerStatsTopicPrefix is the realtime topic prefix for per-container stats snapshots.
	ContainerStatsTopicPrefix = "container.stats:"
	// ContainerLogsTopicPrefix is the realtime topic prefix for per-container incremental log events.
	ContainerLogsTopicPrefix = "container.logs:"
	// ContainerDashboardSummaryTopic is the realtime topic for dashboard summary snapshots.
	ContainerDashboardSummaryTopic = "container.dashboard.summary"
)
