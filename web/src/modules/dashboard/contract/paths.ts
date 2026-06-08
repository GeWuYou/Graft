export const DASHBOARD_API_PATH = {
  SUMMARY: '/api/dashboard/summary',
  WIDGET: '/api/dashboard/widgets/{widget_id}',
} as const;

export function buildDashboardWidgetApiPath(widgetId: string) {
  return DASHBOARD_API_PATH.WIDGET.replace('{widget_id}', encodeURIComponent(widgetId));
}
