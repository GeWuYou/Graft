// Package route 定义平台级 HTTP 路由契约，供 core runtime 挂载模块路由及模块逻辑组合完整路径。
package route

const (
	// APIRoot 是所有模块 HTTP API 的统一根路径。
	APIRoot = "/api"
)

// APIPath 将模块拥有的路由片段组合为完整 API 路径。
func APIPath(fragment string) string {
	return APIRoot + fragment
}
