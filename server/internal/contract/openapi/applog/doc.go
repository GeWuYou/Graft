// Package applogopenapi provides minimal generated bindings for logger-owned App Log routes.
package applogopenapi

// ServerInterface is the minimal generated handler contract for app-log routes.
type ServerInterface interface {
	GetAppLogs(params GetAppLogsParams)
	GetAppLogDetail(id int64, params GetAppLogDetailParams)
	DeleteAppLog(id int64, params DeleteAppLogParams)
	PostAppLogBatchDelete(params PostAppLogBatchDeleteParams, body PostAppLogBatchDeleteJSONRequestBody)
}
