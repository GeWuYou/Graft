// Package contract 定义 Registry 模块对外稳定的导航和权限标识。
package contract

const (
	// ModuleID 是 Registry 模块的稳定标识。
	ModuleID = "registry"
	// MenuTitle 是 Infrastructure 导航标题的本地化键。
	MenuTitle = "menu.registries.title"
	// MenuPath 是 Registry 管理页面路由。
	MenuPath = "/infrastructure/registries"
	// MenuIcon 是镜像仓库共享资源的菜单图标语义键。
	MenuIcon = "image-registry"
	// MenuOrder 确保镜像仓库共享资源显示在运行目标之前。
	MenuOrder = 30
	// ReadPermission 允许读取 Registry 管理信息。
	ReadPermission = "registry.read"
	// CreatePermission 允许创建 Registry 连接和 Repository。
	CreatePermission = "registry.create"
	// UpdatePermission 允许修改 Registry 连接和 Repository。
	UpdatePermission = "registry.update"
	// DeletePermission 允许软删除 Registry 资源。
	DeletePermission = "registry.delete"
	// VerifyPermission 允许触发受控连接验证。
	VerifyPermission = "registry.verify"
	// AssignmentManagePermission 允许管理 Repository 用户授权。
	AssignmentManagePermission = "registry.assignment.manage"
)
