// Package contract 定义 Build 模块对外稳定契约。
package contract

// BuildReadPermission 等权限码是 Build 模块注册菜单和后续服务鉴权共享的稳定契约。
const (
	BuildReadPermission   = "build.read"
	BuildCreatePermission = "build.create"
	BuildCancelPermission = "build.cancel"
	BuildRetryPermission  = "build.retry"
)
