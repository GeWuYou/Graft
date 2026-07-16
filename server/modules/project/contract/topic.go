package contract

const (
	// ProjectListSummaryTopic 是项目列表摘要更新的规范实时 topic。
	ProjectListSummaryTopic = "project.list.summary"
	// ProjectRuntimeTopicPrefix 是按公开 Application ID 区分的项目运行时快照 topic 前缀。
	ProjectRuntimeTopicPrefix = "project.runtime:"
	// ProjectLifecycleConfigTopicPrefix 是按公开 Application ID 区分的生命周期配置快照 topic 前缀。
	ProjectLifecycleConfigTopicPrefix = "project.lifecycle-config:"
	// ProjectLogsTopicPrefix 是按公开 Application ID 区分的项目聚合日志 topic 前缀。
	ProjectLogsTopicPrefix = "project.logs:"
)
