// Package httpheader 定义服务端运行时共享的稳定 HTTP 请求头契约。
package httpheader

// Name 标识稳定的 HTTP 请求头契约名称。
type Name string

// String 返回线上协议使用的请求头名称。
func (n Name) String() string {
	return string(n)
}

const (
	// AcceptLanguage 携带标准 HTTP 请求的语言偏好。
	AcceptLanguage Name = "Accept-Language"

	// Authorization 携带调用方的认证方案和令牌。
	Authorization Name = "Authorization"

	// Locale 携带平台定义的显式语言区域覆盖值。
	Locale Name = "X-Graft-Locale"

	// RequestID 携带稳定请求标识，并在响应封套中回显。
	RequestID Name = "X-Request-Id"

	// TraceID 携带兼容旧调用方的上游追踪标识回退值。
	TraceID Name = "X-Trace-Id"
)
