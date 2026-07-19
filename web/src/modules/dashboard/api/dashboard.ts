import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  DashboardSummaryResponse,
  DashboardWidgetResponse,
  GetDashboardWidgetPathParams,
} from '../types/dashboard';

type DashboardSummaryPath = typeof OPENAPI_RUNTIME_PATH.getDashboardSummary;
type GetDashboardSummaryOperation = paths[DashboardSummaryPath]['get'];
type GetDashboardSummaryEnvelope = GetDashboardSummaryOperation['responses'][200]['content']['application/json'];
type GetDashboardSummaryData = NonNullable<GetDashboardSummaryEnvelope['data']>;

type DashboardWidgetPath = typeof OPENAPI_RUNTIME_PATH.getDashboardWidget;
type GetDashboardWidgetOperation = paths[DashboardWidgetPath]['get'];
type GetDashboardWidgetEnvelope = GetDashboardWidgetOperation['responses'][200]['content']['application/json'];
type GetDashboardWidgetData = NonNullable<GetDashboardWidgetEnvelope['data']>;

/** 仪表盘 API 只返回服务端摘要和 widget 数据，页面不得复制成另一套契约。 */
export function getDashboardSummary() {
  return request.get<GetDashboardSummaryData>({
    url: OPENAPI_RUNTIME_PATH.getDashboardSummary,
  }) as Promise<DashboardSummaryResponse>;
}

export function getDashboardWidget(widgetId: GetDashboardWidgetPathParams['widget_id']) {
  return request.get<GetDashboardWidgetData>({
    url: buildOpenApiRuntimePath('getDashboardWidget', { widget_id: widgetId }),
  }) as Promise<DashboardWidgetResponse>;
}
