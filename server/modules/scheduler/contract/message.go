package contract

// MessageKey identifies a stable scheduled task module message key.
type MessageKey string

// String returns the canonical message key value.
func (k MessageKey) String() string {
	return string(k)
}

const (
	// ScheduledTaskMenuTitle identifies the localized title for the scheduled task menu.
	ScheduledTaskMenuTitle MessageKey = "menu.server.scheduled_tasks.title"
	// ScheduledTaskNotFound identifies missing scheduled task failures.
	ScheduledTaskNotFound MessageKey = "scheduled_task.not_found"
	// ScheduledTaskAlreadyRunning identifies duplicate manual run failures.
	ScheduledTaskAlreadyRunning MessageKey = "scheduled_task.already_running"
	// ScheduledTaskInvalidRequest identifies invalid scheduler management input.
	ScheduledTaskInvalidRequest MessageKey = "scheduled_task.invalid_request"
)
