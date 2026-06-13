package auditopenapi

//go:generate go tool oapi-codegen --include-operation-ids getAuditLogs,getAuditLogDetail,getAuditOverview,getAuditIncident --generate types --package auditopenapi -o zz_generated.audit.go ../../../../../openapi/openapi.yaml
