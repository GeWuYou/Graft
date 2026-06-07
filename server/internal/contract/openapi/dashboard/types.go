package dashboardopenapi

// ServerInterface is the minimal generated handler contract for dashboard routes.
type ServerInterface interface {
	GetDashboardSummary(params GetDashboardSummaryParams)
	GetDashboardWidget(params GetDashboardWidgetParams)
}
