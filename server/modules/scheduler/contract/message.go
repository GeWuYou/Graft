package contract

// MessageKey 标识 scheduler 模块稳定的消息键。
type MessageKey string

// String 返回 canonical 消息键值。
func (k MessageKey) String() string {
	return string(k)
}

const (
	// ScheduledTaskMenuTitle 标识定时任务菜单的本地化标题。
	ScheduledTaskMenuTitle MessageKey = "menu.scheduled_task.title"
	// ScheduledTaskNotFound 标识定时任务不存在的错误消息。
	ScheduledTaskNotFound MessageKey = "scheduled_task.not_found"
	// ScheduledTaskAlreadyRunning 标识重复手动运行的错误消息。
	ScheduledTaskAlreadyRunning MessageKey = "scheduled_task.already_running"
	// ScheduledTaskInvalidRequest 标识调度管理输入无效的错误消息。
	ScheduledTaskInvalidRequest MessageKey = "scheduled_task.invalid_request"
	// ScheduledTaskRunFailedNotificationTitle 标识调度失败通知的标题消息。
	ScheduledTaskRunFailedNotificationTitle MessageKey = "scheduledTask.notification.runFailed.title"
	// ScheduledTaskRunFailedNotificationMessage 标识调度失败通知的正文消息。
	ScheduledTaskRunFailedNotificationMessage MessageKey = "scheduledTask.notification.runFailed.message"
	// ScheduledTaskRunSucceededNotificationTitle 标识手动运行成功通知的标题消息。
	ScheduledTaskRunSucceededNotificationTitle MessageKey = "scheduledTask.notification.runSucceeded.title"
	// ScheduledTaskRunSucceededNotificationMessage 标识手动运行成功通知的正文消息。
	ScheduledTaskRunSucceededNotificationMessage MessageKey = "scheduledTask.notification.runSucceeded.message"
)
