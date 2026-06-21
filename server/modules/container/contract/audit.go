package contract

// AuditAction identifies a stable container audit action contract.
type AuditAction string

// String returns the canonical audit action value.
func (a AuditAction) String() string {
	return string(a)
}

const (
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
