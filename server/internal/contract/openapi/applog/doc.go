// Package applogopenapi provides minimal generated bindings for logger-owned App Log read routes.
package applogopenapi

// ReadServerInterface is the minimal generated handler contract for app-log read routes.
type ReadServerInterface interface {
	GetAppLogs(params GetAppLogsParams)
	GetAppLogDetail(id int64, params GetAppLogDetailParams)
}
