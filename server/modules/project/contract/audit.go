package contract

// AuditAction 标识 Application 模块稳定的审计动作契约。
type AuditAction string

// String 返回规范的审计动作值，供审计事件和本地化查找使用。
func (a AuditAction) String() string {
	return string(a)
}

const (
	// ApplicationAuditActionUp 标识单应用 Compose 启动审计动作。
	ApplicationAuditActionUp AuditAction = "ops.application.action.up"
	// ApplicationAuditActionStop 标识单应用 Compose 停止审计动作。
	ApplicationAuditActionStop AuditAction = "ops.application.action.stop"
	// ApplicationAuditActionRestart 标识单应用 Compose 重启审计动作。
	ApplicationAuditActionRestart AuditAction = "ops.application.action.restart"
	// ApplicationAuditActionRedeploy 标识单应用 Compose 重部署审计动作。
	ApplicationAuditActionRedeploy AuditAction = "ops.application.action.redeploy"
	// ApplicationAuditActionUnregister 标识单应用注销审计动作。
	ApplicationAuditActionUnregister AuditAction = "ops.application.action.unregister"
	// ApplicationAuditActionDestroy 标识单应用销毁审计动作。
	ApplicationAuditActionDestroy AuditAction = "ops.application.action.destroy"
	// ApplicationAuditActionBatchStart 标识批量启动汇总审计动作。
	ApplicationAuditActionBatchStart AuditAction = "ops.application.action.batch.start"
	// ApplicationAuditActionBatchStop 标识批量停止汇总审计动作。
	ApplicationAuditActionBatchStop AuditAction = "ops.application.action.batch.stop"
	// ApplicationAuditActionBatchRestart 标识批量重启汇总审计动作。
	ApplicationAuditActionBatchRestart AuditAction = "ops.application.action.batch.restart"
	// ApplicationAuditActionBatchRedeploy 标识批量重部署汇总审计动作。
	ApplicationAuditActionBatchRedeploy AuditAction = "ops.application.action.batch.redeploy"
	// ApplicationAuditActionBatchUnregister 标识批量注销汇总审计动作。
	ApplicationAuditActionBatchUnregister AuditAction = "ops.application.action.batch.unregister"
	// ApplicationAuditActionBatchDestroy 标识批量销毁汇总审计动作。
	ApplicationAuditActionBatchDestroy AuditAction = "ops.application.action.batch.destroy"
)
