package contract

const (
	// ContainerListStatsTopic 是容器列表级资源统计快照的实时 topic。
	ContainerListStatsTopic = "container.stats.list"
	// ContainerStatsTopicPrefix 是单容器资源统计快照的实时 topic 前缀。
	ContainerStatsTopicPrefix = "container.stats:"
	// ContainerEventsTopicPrefix 是单容器运行时事件流的实时 topic 前缀。
	ContainerEventsTopicPrefix = "container.events:"
	// ContainerLogsTopicPrefix 是单容器增量日志事件的实时 topic 前缀。
	ContainerLogsTopicPrefix = "container.logs:"
	// ContainerDashboardSummaryTopic 是容器仪表盘摘要快照的实时 topic。
	ContainerDashboardSummaryTopic = "container.dashboard.summary"
)
