package contract

// AuditAction 是容器模块的稳定审计动作契约。
type AuditAction string

// String 返回规范的审计动作值。
func (a AuditAction) String() string {
	return string(a)
}

const (
	// ContainerAuditActionStart 表示单容器启动审计。
	ContainerAuditActionStart AuditAction = "ops.container.action.start"
	// ContainerAuditActionStop 表示单容器停止审计。
	ContainerAuditActionStop AuditAction = "ops.container.action.stop"
	// ContainerAuditActionRestart 表示单容器重启审计。
	ContainerAuditActionRestart AuditAction = "ops.container.action.restart"
	// ContainerAuditActionRemove 表示单容器移除审计。
	ContainerAuditActionRemove AuditAction = "ops.container.action.remove"
	// DockerImageAuditActionPull 表示 Docker 镜像拉取审计。
	DockerImageAuditActionPull AuditAction = "ops.container.image.pull"
	// DockerImageAuditActionTag 表示 Docker 镜像标签审计。
	DockerImageAuditActionTag AuditAction = "ops.container.image.tag"
	// DockerImageAuditActionRemove 表示 Docker 镜像删除审计。
	DockerImageAuditActionRemove AuditAction = "ops.container.image.remove"
	// DockerImageAuditActionBatchRemove 表示 Docker 镜像批量删除汇总审计。
	DockerImageAuditActionBatchRemove AuditAction = "ops.container.image.remove.batch"
	// ContainerAuditActionBatchStart 表示批量启动汇总审计。
	ContainerAuditActionBatchStart AuditAction = "ops.container.action.batch.start"
	// ContainerAuditActionBatchStop 表示批量停止汇总审计。
	ContainerAuditActionBatchStop AuditAction = "ops.container.action.batch.stop"
	// ContainerAuditActionBatchRestart 表示批量重启汇总审计。
	ContainerAuditActionBatchRestart AuditAction = "ops.container.action.batch.restart"
	// ContainerAuditActionBatchRemove 表示批量移除汇总审计。
	ContainerAuditActionBatchRemove AuditAction = "ops.container.action.batch.remove"
	// ContainerAuditActionShellSessionRequested 表示 Shell 会话请求审计。
	ContainerAuditActionShellSessionRequested AuditAction = "ops.container.shell.session.requested"
	// ContainerAuditActionShellTicketIssued 表示 Shell 票据签发审计。
	ContainerAuditActionShellTicketIssued AuditAction = "ops.container.shell.ticket.issued"
	// ContainerAuditActionShellTicketRejected 表示 Shell 票据拒绝审计。
	ContainerAuditActionShellTicketRejected AuditAction = "ops.container.shell.ticket.rejected"
	// ContainerAuditActionShellSessionStarted 表示 Shell 会话开始审计。
	ContainerAuditActionShellSessionStarted AuditAction = "ops.container.shell.session.started"
	// ContainerAuditActionShellSessionClosed 表示 Shell 会话关闭审计。
	ContainerAuditActionShellSessionClosed AuditAction = "ops.container.shell.session.closed"
	// ContainerAuditActionShellSessionFailed 表示 Shell 会话失败审计。
	ContainerAuditActionShellSessionFailed AuditAction = "ops.container.shell.session.failed"
)
