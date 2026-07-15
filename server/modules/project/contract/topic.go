package contract

const (
	// ProjectListSummaryTopic is the canonical realtime topic for project list summary updates.
	ProjectListSummaryTopic = "project.list.summary"
	// ProjectRuntimeTopicPrefix is the realtime topic prefix for project runtime snapshots keyed by public Application ID.
	ProjectRuntimeTopicPrefix = "project.runtime:"
	// ProjectLifecycleConfigTopicPrefix is the realtime topic prefix for lifecycle configuration snapshots keyed by public Application ID.
	ProjectLifecycleConfigTopicPrefix = "project.lifecycle-config:"
	// ProjectLogsTopicPrefix is the realtime topic prefix for project-owned aggregated logs keyed by public Application ID.
	ProjectLogsTopicPrefix = "project.logs:"
)
