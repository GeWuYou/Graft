package contract

const (
	// ProjectListSummaryTopic is the canonical realtime topic for project list summary updates.
	ProjectListSummaryTopic = "project.list.summary"
	// ProjectRuntimeTopicPrefix is the realtime topic prefix for project runtime snapshots.
	ProjectRuntimeTopicPrefix = "project.runtime:"
	// ProjectLifecycleConfigTopicPrefix is the realtime topic prefix for lifecycle configuration snapshots.
	ProjectLifecycleConfigTopicPrefix = "project.lifecycle-config:"
	// ProjectLogsTopicPrefix is the realtime topic prefix for project-owned aggregated logs.
	ProjectLogsTopicPrefix = "project.logs:"
)
