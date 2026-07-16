package contract

const (
	// SystemConfigGroup 是系统配置管理接口的路由组。
	SystemConfigGroup = "/system-configs"
	// SystemConfigCollectionRoute 是配置集合接口的路径片段。
	SystemConfigCollectionRoute = ""
	// SystemConfigDetailRoute 是配置详情接口的路径片段。
	SystemConfigDetailRoute = "/:key"
	// SystemConfigResetRoute 是删除用户覆盖并恢复默认值的路径片段。
	SystemConfigResetRoute = "/:key/reset"
	// SystemConfigMenuPath 是系统配置管理页面使用的前端菜单路径。
	SystemConfigMenuPath = "/platform/system-config"
)
