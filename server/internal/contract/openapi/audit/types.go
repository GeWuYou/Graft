package auditopenapi

// ReadServerInterface is the minimal generated handler contract for guarded audit read routes.
type ReadServerInterface interface {
	GetAuditLogs(params GetAuditLogsParams)
	GetAuditOverview(params GetAuditOverviewParams)
	GetAuditIncident(params GetAuditIncidentParams)
}
