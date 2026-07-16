package contract

const (
	// ApplicationListSummaryTopic 是应用列表摘要更新的规范实时 topic。
	ApplicationListSummaryTopic = "application.list.summary"
	// ApplicationRuntimeTopicPrefix 是按公开 Application ID 区分的应用运行时快照 topic 前缀。
	ApplicationRuntimeTopicPrefix = "application.runtime:"
	// ApplicationLifecycleConfigTopicPrefix 是按公开 Application ID 区分的应用生命周期配置快照 topic 前缀。
	ApplicationLifecycleConfigTopicPrefix = "application.lifecycle-config:"
	// ApplicationLogsTopicPrefix 是按公开 Application ID 区分的应用聚合日志 topic 前缀。
	ApplicationLogsTopicPrefix = "application.logs:"
)
