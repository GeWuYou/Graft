package contract

const (
	// SecurityGroup 是安全模块 API 的稳定分组路径。
	SecurityGroup = "/security"
	// OverviewCollection 是安全概览集合接口的稳定路径片段。
	OverviewCollection = "/overview"
	// OverviewMenuPath 是安全概览页面的稳定前端路由。
	OverviewMenuPath = "/security/overview"
	// OverviewAPIPath 是安全概览接口对外暴露的完整路径。
	OverviewAPIPath = SecurityGroup + OverviewCollection
)
