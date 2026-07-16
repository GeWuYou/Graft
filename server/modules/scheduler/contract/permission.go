package contract

// PermissionCode 标识 scheduler 模块稳定的权限契约。
type PermissionCode string

// String 返回权限码的 wire-format 字符串。
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// ScheduledTaskReadPermission identifies read access to scheduled task runtime data.
	ScheduledTaskReadPermission PermissionCode = "scheduled-task.read"
	// ScheduledTaskCreatePermission identifies create access for user scheduled task instances.
	ScheduledTaskCreatePermission PermissionCode = "scheduled-task.create"
	// ScheduledTaskUpdatePermission identifies update access for scheduled task definitions.
	ScheduledTaskUpdatePermission PermissionCode = "scheduled-task.update"
	// ScheduledTaskDeletePermission identifies delete access for user scheduled task instances.
	ScheduledTaskDeletePermission PermissionCode = "scheduled-task.delete"
	// ScheduledTaskRunPermission identifies manual run access for scheduled task runtime jobs.
	ScheduledTaskRunPermission PermissionCode = "scheduled-task.run"
	// ScheduledTaskEnablePermission identifies enable/disable access for scheduled task lifecycle state.
	ScheduledTaskEnablePermission PermissionCode = "scheduled-task.enable"
)
