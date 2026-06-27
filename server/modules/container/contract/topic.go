package contract

const (
	// ContainerListStatsTopic is the realtime topic for list-level container stats snapshots.
	ContainerListStatsTopic = "container.stats.list"
	// ContainerStatsTopicPrefix is the realtime topic prefix for per-container stats snapshots.
	ContainerStatsTopicPrefix = "container.stats:"
	// ContainerEventsTopicPrefix is the realtime topic prefix for per-container runtime event streams.
	ContainerEventsTopicPrefix = "container.events:"
	// ContainerLogsTopicPrefix is the realtime topic prefix for per-container incremental log events.
	ContainerLogsTopicPrefix = "container.logs:"
	// ContainerDashboardSummaryTopic is the realtime topic for dashboard summary snapshots.
	ContainerDashboardSummaryTopic = "container.dashboard.summary"
)
