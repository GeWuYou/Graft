package contract

// PermissionCode 标识 scheduler 模块稳定的权限契约。
type PermissionCode string

// String 返回权限码的 wire-format 字符串。
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// ScheduledTaskReadPermission 标识读取定时任务运行数据的权限。
	ScheduledTaskReadPermission PermissionCode = "scheduled-task.read"
	// ScheduledTaskCreatePermission 标识创建用户定时任务实例的权限。
	ScheduledTaskCreatePermission PermissionCode = "scheduled-task.create"
	// ScheduledTaskUpdatePermission 标识更新定时任务定义的权限。
	ScheduledTaskUpdatePermission PermissionCode = "scheduled-task.update"
	// ScheduledTaskDeletePermission 标识删除用户定时任务实例的权限。
	ScheduledTaskDeletePermission PermissionCode = "scheduled-task.delete"
	// ScheduledTaskRunPermission 标识手动运行定时任务作业的权限。
	ScheduledTaskRunPermission PermissionCode = "scheduled-task.run"
	// ScheduledTaskEnablePermission 标识变更定时任务启停状态的权限。
	ScheduledTaskEnablePermission PermissionCode = "scheduled-task.enable"
)
