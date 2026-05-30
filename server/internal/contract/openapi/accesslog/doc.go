// Package accesslogopenapi 提供 access-log OpenAPI 读接口对应的最小生成契约包装。
package accesslogopenapi

// ReadServerInterface is the minimal generated handler contract for access-log read routes.
type ReadServerInterface interface {
	GetAccessLogs(params GetAccessLogsParams)
	GetAccessLogDetail(id int64, params GetAccessLogDetailParams)
}
