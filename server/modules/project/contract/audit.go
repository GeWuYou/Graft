package contract

// AuditAction identifies a stable project audit action contract.
type AuditAction string

// String returns the canonical audit action value.
func (a AuditAction) String() string {
	return string(a)
}

const (
	// ProjectAuditActionUp identifies single-item compose up auditing.
	ProjectAuditActionUp AuditAction = "ops.project.action.up"
	// ProjectAuditActionStop identifies single-item compose stop auditing.
	ProjectAuditActionStop AuditAction = "ops.project.action.stop"
	// ProjectAuditActionRestart identifies single-item compose restart auditing.
	ProjectAuditActionRestart AuditAction = "ops.project.action.restart"
	// ProjectAuditActionRedeploy identifies single-item compose redeploy auditing.
	ProjectAuditActionRedeploy AuditAction = "ops.project.action.redeploy"
	// ProjectAuditActionUpdateDeploy identifies single-item compose update-deploy auditing.
	ProjectAuditActionUpdateDeploy AuditAction = "ops.project.action.updateDeploy"
	// ProjectAuditActionUnregister identifies single-item unregister auditing.
	ProjectAuditActionUnregister AuditAction = "ops.project.action.unregister"
	// ProjectAuditActionDestroy identifies single-item destroy auditing.
	ProjectAuditActionDestroy AuditAction = "ops.project.action.destroy"
	// ProjectAuditActionBatchStart identifies batch start summary auditing.
	ProjectAuditActionBatchStart AuditAction = "ops.project.action.batch.start"
	// ProjectAuditActionBatchStop identifies batch stop summary auditing.
	ProjectAuditActionBatchStop AuditAction = "ops.project.action.batch.stop"
	// ProjectAuditActionBatchRestart identifies batch restart summary auditing.
	ProjectAuditActionBatchRestart AuditAction = "ops.project.action.batch.restart"
	// ProjectAuditActionBatchRedeploy identifies batch redeploy summary auditing.
	ProjectAuditActionBatchRedeploy AuditAction = "ops.project.action.batch.redeploy"
	// ProjectAuditActionBatchUpdateDeploy identifies batch update-deploy summary auditing.
	ProjectAuditActionBatchUpdateDeploy AuditAction = "ops.project.action.batch.updateDeploy"
	// ProjectAuditActionBatchUnregister identifies batch unregister summary auditing.
	ProjectAuditActionBatchUnregister AuditAction = "ops.project.action.batch.unregister"
	// ProjectAuditActionBatchDestroy identifies batch destroy summary auditing.
	ProjectAuditActionBatchDestroy AuditAction = "ops.project.action.batch.destroy"
)
