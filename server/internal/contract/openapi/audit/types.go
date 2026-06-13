package auditopenapi

// ReadServerInterface is the minimal generated handler contract for guarded audit read routes.
type ReadServerInterface interface {
	GetAuditLogs(params GetAuditLogsParams)
	GetAuditLogDetail(id int64, params GetAuditLogDetailParams)
	GetAuditOverview(params GetAuditOverviewParams)
	GetAuditIncident(params GetAuditIncidentParams)
}
