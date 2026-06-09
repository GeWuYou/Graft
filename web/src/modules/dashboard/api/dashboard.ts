// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { buildDashboardWidgetApiPath, DASHBOARD_API_PATH } from '../contract/paths';
import type {
  DashboardSummaryResponse,
  DashboardWidgetResponse,
  GetDashboardWidgetPathParams,
} from '../types/dashboard';

type DashboardSummaryPath = (typeof DASHBOARD_API_PATH)['SUMMARY'];
type GetDashboardSummaryOperation = paths[DashboardSummaryPath]['get'];
type GetDashboardSummaryEnvelope = GetDashboardSummaryOperation['responses'][200]['content']['application/json'];
type GetDashboardSummaryData = NonNullable<GetDashboardSummaryEnvelope['data']>;

type DashboardWidgetPath = (typeof DASHBOARD_API_PATH)['WIDGET'];
type GetDashboardWidgetOperation = paths[DashboardWidgetPath]['get'];
type GetDashboardWidgetEnvelope = GetDashboardWidgetOperation['responses'][200]['content']['application/json'];
type GetDashboardWidgetData = NonNullable<GetDashboardWidgetEnvelope['data']>;

export function getDashboardSummary() {
  return request.get<GetDashboardSummaryData>({
    url: DASHBOARD_API_PATH.SUMMARY,
  }) as Promise<DashboardSummaryResponse>;
}

export function getDashboardWidget(widgetId: GetDashboardWidgetPathParams['widget_id']) {
  return request.get<GetDashboardWidgetData>({
    url: buildDashboardWidgetApiPath(widgetId),
  }) as Promise<DashboardWidgetResponse>;
}
