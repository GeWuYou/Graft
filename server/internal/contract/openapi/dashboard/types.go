package dashboardopenapi

// ServerInterface 定义 dashboard 路由对外暴露的最小 OpenAPI handler 契约。
type ServerInterface interface {
	// GetDashboardSummary 对应 dashboard 聚合摘要查询操作，由路由实现绑定具体响应写入。
	GetDashboardSummary(params GetDashboardSummaryParams)
	// GetDashboardWidget 对应单个 dashboard widget 查询操作，由路由实现绑定具体响应写入。
	GetDashboardWidget(params GetDashboardWidgetParams)
}
