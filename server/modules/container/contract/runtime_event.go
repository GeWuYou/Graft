package contract

// RuntimeEventSeverity 是容器运行时事件的规范严重程度契约。
type RuntimeEventSeverity string

// String 返回用于线序列化的严重程度值。
func (s RuntimeEventSeverity) String() string {
	return string(s)
}

const (
	// RuntimeEventSeverityInfo 表示信息性生命周期或运行时事实。
	RuntimeEventSeverityInfo RuntimeEventSeverity = "info"
	// RuntimeEventSeverityWarning 表示降级或需要关注的生命周期事实。
	RuntimeEventSeverityWarning RuntimeEventSeverity = "warning"
	// RuntimeEventSeverityError 表示严重运行时失败事实。
	RuntimeEventSeverityError RuntimeEventSeverity = "error"
)

// RuntimeEventType 是容器运行时事件类型的规范契约。
type RuntimeEventType string

// String 返回用于线序列化的事件类型值。
func (t RuntimeEventType) String() string {
	return string(t)
}

const (
	// RuntimeEventTypeContainerCreated 表示容器已创建。
	RuntimeEventTypeContainerCreated RuntimeEventType = "container.created"
	// RuntimeEventTypeContainerStarted 表示容器已启动。
	RuntimeEventTypeContainerStarted RuntimeEventType = "container.started"
	// RuntimeEventTypeContainerRestarted 表示容器已重启。
	RuntimeEventTypeContainerRestarted RuntimeEventType = "container.restarted"
	// RuntimeEventTypeContainerStopped 表示容器已停止或退出。
	RuntimeEventTypeContainerStopped RuntimeEventType = "container.stopped"
	// RuntimeEventTypeContainerRemoved 表示容器已移除。
	RuntimeEventTypeContainerRemoved RuntimeEventType = "container.removed"
	// RuntimeEventTypeContainerOOMKilled 表示容器因 OOM 被终止。
	RuntimeEventTypeContainerOOMKilled RuntimeEventType = "container.oom_killed"
	// RuntimeEventTypeContainerHealthStatusChanged 表示健康状态发生变化。
	RuntimeEventTypeContainerHealthStatusChanged RuntimeEventType = "container.health_status_changed"
	// RuntimeEventTypeContainerExecStarted 表示 exec 会话已开始。
	RuntimeEventTypeContainerExecStarted RuntimeEventType = "container.exec_started"
	// RuntimeEventTypeContainerExecFinished 表示 exec 会话已结束。
	RuntimeEventTypeContainerExecFinished RuntimeEventType = "container.exec_finished"
)
