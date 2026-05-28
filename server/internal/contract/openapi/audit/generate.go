package auditopenapi

//go:generate go tool oapi-codegen --include-operation-ids getAuditLogs,getAuditOverview --generate types --package auditopenapi -o zz_generated.audit.go ../../../../../openapi/openapi.yaml
