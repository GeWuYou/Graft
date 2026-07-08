package contract

const (
	// ProjectListSummaryTopic is the canonical realtime topic for project list summary updates.
	ProjectListSummaryTopic = "project.list.summary"
	// ProjectDetailTopicPrefix is the realtime topic prefix for project detail live snapshots.
	ProjectDetailTopicPrefix = "project.detail:"
	// ProjectLogsTopicPrefix is the realtime topic prefix for project-owned aggregated logs.
	ProjectLogsTopicPrefix = "project.logs:"
)
