package contract

// PermissionCode identifies a stable scheduled task module permission contract.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// ScheduledTaskReadPermission identifies read access to scheduled task runtime data.
	ScheduledTaskReadPermission PermissionCode = "scheduled-task.read"
	// ScheduledTaskCreatePermission identifies create access for user HTTP scheduled tasks.
	ScheduledTaskCreatePermission PermissionCode = "scheduled-task.create"
	// ScheduledTaskUpdatePermission identifies update access for scheduled task definitions.
	ScheduledTaskUpdatePermission PermissionCode = "scheduled-task.update"
	// ScheduledTaskDeletePermission identifies delete access for user HTTP scheduled tasks.
	ScheduledTaskDeletePermission PermissionCode = "scheduled-task.delete"
	// ScheduledTaskRunPermission identifies manual run access for scheduled task runtime jobs.
	ScheduledTaskRunPermission PermissionCode = "scheduled-task.run"
	// ScheduledTaskEnablePermission identifies enable/disable access for scheduled task lifecycle state.
	ScheduledTaskEnablePermission PermissionCode = "scheduled-task.enable"
)
