package accesslogopenapi

//go:generate go tool oapi-codegen --include-operation-ids getAccessLogs,getAccessLogDetail --generate types --package accesslogopenapi -o zz_generated.accesslog.go ../../../../../openapi/openapi.yaml
