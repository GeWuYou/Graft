package contract

const (
	// ContainerListStatsTopic is the realtime topic for list-level container stats snapshots.
	ContainerListStatsTopic = "container.stats.list"
	// ContainerStatsTopicPrefix is the realtime topic prefix for per-container stats snapshots.
	ContainerStatsTopicPrefix = "container.stats:"
	// ContainerDashboardSummaryTopic is the realtime topic for dashboard summary snapshots.
	ContainerDashboardSummaryTopic = "container.dashboard.summary"
)
