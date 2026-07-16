package contract

// AuditAction 标识项目模块稳定的审计动作契约。
type AuditAction string

// String 返回规范的审计动作值，供审计事件和本地化查找使用。
func (a AuditAction) String() string {
	return string(a)
}

const (
	// ProjectAuditActionUp 标识单项目 Compose 启动审计动作。
	ProjectAuditActionUp AuditAction = "ops.project.action.up"
	// ProjectAuditActionStop 标识单项目 Compose 停止审计动作。
	ProjectAuditActionStop AuditAction = "ops.project.action.stop"
	// ProjectAuditActionRestart 标识单项目 Compose 重启审计动作。
	ProjectAuditActionRestart AuditAction = "ops.project.action.restart"
	// ProjectAuditActionRedeploy 标识单项目 Compose 重部署审计动作。
	ProjectAuditActionRedeploy AuditAction = "ops.project.action.redeploy"
	// ProjectAuditActionUnregister 标识单项目注销审计动作。
	ProjectAuditActionUnregister AuditAction = "ops.project.action.unregister"
	// ProjectAuditActionDestroy 标识单项目销毁审计动作。
	ProjectAuditActionDestroy AuditAction = "ops.project.action.destroy"
	// ProjectAuditActionBatchStart 标识批量启动汇总审计动作。
	ProjectAuditActionBatchStart AuditAction = "ops.project.action.batch.start"
	// ProjectAuditActionBatchStop 标识批量停止汇总审计动作。
	ProjectAuditActionBatchStop AuditAction = "ops.project.action.batch.stop"
	// ProjectAuditActionBatchRestart 标识批量重启汇总审计动作。
	ProjectAuditActionBatchRestart AuditAction = "ops.project.action.batch.restart"
	// ProjectAuditActionBatchRedeploy 标识批量重部署汇总审计动作。
	ProjectAuditActionBatchRedeploy AuditAction = "ops.project.action.batch.redeploy"
	// ProjectAuditActionBatchUnregister 标识批量注销汇总审计动作。
	ProjectAuditActionBatchUnregister AuditAction = "ops.project.action.batch.unregister"
	// ProjectAuditActionBatchDestroy 标识批量销毁汇总审计动作。
	ProjectAuditActionBatchDestroy AuditAction = "ops.project.action.batch.destroy"
)
