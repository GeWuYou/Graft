package audit

import (
	"context"

	auditstore "graft/server/plugins/audit/store"
)

type stubAuditRepository struct{}

func (stubAuditRepository) CreateAuditLog(context.Context, auditstore.CreateAuditLogInput) (auditstore.AuditLog, error) {
	return auditstore.AuditLog{}, nil
}

func (stubAuditRepository) ListAuditLogs(context.Context, auditstore.ListAuditLogsQuery) (auditstore.ListAuditLogsResult, error) {
	return auditstore.ListAuditLogsResult{}, nil
}

func (stubAuditRepository) ReadAuditOverview(context.Context, auditstore.OverviewWindow) (auditstore.AuditOverview, error) {
	return auditstore.AuditOverview{}, nil
}
