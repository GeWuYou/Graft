// Package contract 定义 task 模块向跨模块消费者公开的稳定契约。
package contract

// RealtimeTopic 标识 task 模块公开给订阅消费者的稳定主题片段。
type RealtimeTopic string

// RealtimeEvent 标识 task 模块实时通知的稳定事件类型。
type RealtimeEvent string

// String 返回实时主题的 wire-format 字符串。
func (value RealtimeTopic) String() string { return string(value) }

// String 返回实时事件的 wire-format 字符串。
func (value RealtimeEvent) String() string { return string(value) }

const (
	// TaskRealtimeTopicPrefix 是按任务 ID 区分的实时订阅主题前缀。
	TaskRealtimeTopicPrefix RealtimeTopic = "task:"

	// TaskRealtimeEventCreated 表示任务已持久化创建。
	TaskRealtimeEventCreated RealtimeEvent = "task.created"
	// TaskRealtimeEventCancelled 表示未运行任务已取消。
	TaskRealtimeEventCancelled RealtimeEvent = "task.cancelled"
	// TaskRealtimeEventCancelRequested 表示运行中任务已收到取消请求。
	TaskRealtimeEventCancelRequested RealtimeEvent = "task.cancel_requested"
	// TaskRealtimeEventRetryRequested 表示失败任务阶段已请求重试。
	TaskRealtimeEventRetryRequested RealtimeEvent = "task.retry_requested"
	// TaskRealtimeEventStageStarted 表示一个任务阶段已被 worker 领取。
	TaskRealtimeEventStageStarted RealtimeEvent = "task.stage_started"
	// TaskRealtimeEventStageCompleted 表示任务阶段已成功完成。
	TaskRealtimeEventStageCompleted RealtimeEvent = "task.stage_completed"
	// TaskRealtimeEventStageFailed 表示任务阶段执行失败。
	TaskRealtimeEventStageFailed RealtimeEvent = "task.stage_failed"
	// TaskRealtimeEventLogAppended 表示任务日志已持久化并可重放。
	TaskRealtimeEventLogAppended RealtimeEvent = "task.log_appended"
)
