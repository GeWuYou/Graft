package accesslogopenapi

//go:generate go tool oapi-codegen --include-operation-ids getAccessLogs,getAccessLogSavedViews,postAccessLogSavedView,putAccessLogSavedView,deleteAccessLogSavedView,getAccessLogDetail --generate types --package accesslogopenapi -o zz_generated.accesslog.go ../../../../../openapi/openapi.yaml
