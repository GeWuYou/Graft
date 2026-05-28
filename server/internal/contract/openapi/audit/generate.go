package auditopenapi

//go:generate go tool oapi-codegen --include-operation-ids getAuditLogs --generate types --package auditopenapi -o zz_generated.audit.go ../../../../../openapi/openapi.yaml
