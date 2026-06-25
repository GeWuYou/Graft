package contract

// AuditAction identifies a stable container audit action contract.
type AuditAction string

// String returns the canonical audit action value.
func (a AuditAction) String() string {
	return string(a)
}

const (
	// ContainerAuditActionStart identifies single-item start auditing.
	ContainerAuditActionStart AuditAction = "ops.container.action.start"
	// ContainerAuditActionStop identifies single-item stop auditing.
	ContainerAuditActionStop AuditAction = "ops.container.action.stop"
	// ContainerAuditActionRestart identifies single-item restart auditing.
	ContainerAuditActionRestart AuditAction = "ops.container.action.restart"
	// ContainerAuditActionRemove identifies single-item remove auditing.
	ContainerAuditActionRemove AuditAction = "ops.container.action.remove"
	// ContainerAuditActionBatchStart identifies batch start summary auditing.
	ContainerAuditActionBatchStart AuditAction = "ops.container.action.batch.start"
	// ContainerAuditActionBatchStop identifies batch stop summary auditing.
	ContainerAuditActionBatchStop AuditAction = "ops.container.action.batch.stop"
	// ContainerAuditActionBatchRestart identifies batch restart summary auditing.
	ContainerAuditActionBatchRestart AuditAction = "ops.container.action.batch.restart"
	// ContainerAuditActionBatchRemove identifies batch remove summary auditing.
	ContainerAuditActionBatchRemove AuditAction = "ops.container.action.batch.remove"
	// ContainerAuditActionShellSessionRequested identifies shell session request auditing.
	ContainerAuditActionShellSessionRequested AuditAction = "ops.container.shell.session.requested"
	// ContainerAuditActionShellTicketIssued identifies shell ticket issue auditing.
	ContainerAuditActionShellTicketIssued AuditAction = "ops.container.shell.ticket.issued"
	// ContainerAuditActionShellTicketRejected identifies shell ticket rejection auditing.
	ContainerAuditActionShellTicketRejected AuditAction = "ops.container.shell.ticket.rejected"
	// ContainerAuditActionShellSessionStarted identifies shell session start auditing.
	ContainerAuditActionShellSessionStarted AuditAction = "ops.container.shell.session.started"
	// ContainerAuditActionShellSessionClosed identifies shell session close auditing.
	ContainerAuditActionShellSessionClosed AuditAction = "ops.container.shell.session.closed"
	// ContainerAuditActionShellSessionFailed identifies shell session failure auditing.
	ContainerAuditActionShellSessionFailed AuditAction = "ops.container.shell.session.failed"
)
