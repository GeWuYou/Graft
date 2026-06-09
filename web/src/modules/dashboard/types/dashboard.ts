import type { components, paths } from '@/contracts/openapi/generated/schema';

import { DASHBOARD_API_PATH } from '../contract/paths';

type DashboardSummaryPath = (typeof DASHBOARD_API_PATH)['SUMMARY'];
type GetDashboardSummaryOperation = paths[DashboardSummaryPath]['get'];
type GetDashboardSummaryEnvelope = GetDashboardSummaryOperation['responses'][200]['content']['application/json'];

type DashboardWidgetPath = (typeof DASHBOARD_API_PATH)['WIDGET'];
type GetDashboardWidgetOperation = paths[DashboardWidgetPath]['get'];
type GetDashboardWidgetEnvelope = GetDashboardWidgetOperation['responses'][200]['content']['application/json'];
export type GetDashboardWidgetPathParams = GetDashboardWidgetOperation['parameters']['path'];

export type DashboardSummaryResponse = NonNullable<GetDashboardSummaryEnvelope['data']>;
export type DashboardQuickLink = components['schemas']['dashboard-quick-link'];
export type DashboardWidget = components['schemas']['dashboard-widget'];
export type DashboardWidgetType = components['schemas']['dashboard-widget-type'];
export type DashboardWidgetSize = components['schemas']['dashboard-widget-size'];
export type DashboardWidgetCategory = components['schemas']['dashboard-widget-category'];
export type DashboardWidgetPriority = components['schemas']['dashboard-widget-priority'];
export type DashboardWidgetState = components['schemas']['dashboard-widget-state'];
export type DashboardWidgetAction = components['schemas']['dashboard-widget-action'];
export type DashboardWidgetStatus = components['schemas']['dashboard-widget-status'];
export type DashboardWidgetError = components['schemas']['dashboard-widget-error'];
export type DashboardSystemSummary = components['schemas']['dashboard-system-summary'];
export type DashboardStatGroupPayload = components['schemas']['dashboard-stat-group-payload'];
export type DashboardAlertListPayload = components['schemas']['dashboard-alert-list-payload'];
export type DashboardLinkListPayload = components['schemas']['dashboard-link-list-payload'];
export type DashboardTimelinePayload = components['schemas']['dashboard-timeline-payload'];
export type DashboardHealthPayload = components['schemas']['dashboard-health-payload'];
export type DashboardHealthStatus = components['schemas']['dashboard-health-status'];
export type DashboardWidgetResponse = NonNullable<GetDashboardWidgetEnvelope['data']>;

export type DashboardPayloadByType = {
  'stat-group': DashboardStatGroupPayload;
  'alert-list': DashboardAlertListPayload;
  'link-list': DashboardLinkListPayload;
  timeline: DashboardTimelinePayload;
  health: DashboardHealthPayload;
};
