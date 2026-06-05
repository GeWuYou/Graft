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
	// ScheduledTaskRunPermission identifies manual run access for scheduled task runtime jobs.
	ScheduledTaskRunPermission PermissionCode = "scheduled-task.run"
)
