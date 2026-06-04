package applogopenapi

//go:generate go tool oapi-codegen --include-operation-ids getAppLogs,getAppLogDetail --generate types --package applogopenapi -o zz_generated.applog.go ../../../../../openapi/openapi.yaml
