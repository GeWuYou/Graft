package auditopenapi

// ReadServerInterface is the minimal generated handler contract for guarded audit read routes.
type ReadServerInterface interface {
	GetAuditLogs(params GetAuditLogsParams)
	GetAuditLogSavedViews(params GetAuditLogSavedViewsParams)
	PostAuditLogSavedView(params PostAuditLogSavedViewParams, body PostAuditLogSavedViewJSONRequestBody)
	PutAuditLogSavedView(viewID int64, params PutAuditLogSavedViewParams, body PutAuditLogSavedViewJSONRequestBody)
	DeleteAuditLogSavedView(viewID int64, params DeleteAuditLogSavedViewParams)
	GetAuditLogDetail(id int64, params GetAuditLogDetailParams)
	GetAuditIncident(params GetAuditIncidentParams)
}
