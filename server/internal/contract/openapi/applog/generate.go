package applogopenapi

//go:generate go tool oapi-codegen --include-operation-ids getAppLogs,getAppLogDetail,deleteAppLog,postAppLogBatchDelete --generate types --package applogopenapi -o zz_generated.applog.go ../../../../../openapi/openapi.yaml
