import type { RouteLocationRaw } from 'vue-router';

import { MONITOR_ROUTE_PATH } from './paths';
import type { MonitorTrendRange } from './trend';

const MONITOR_ORIGIN_QUERY_KEY = {
  VIEW: 'monitorView',
  TREND_RANGE: 'monitorTrendRange',
  ANOMALY_KEY: 'monitorAnomalyKey',
  SCOPE_REF: 'monitorScopeRef',
} as const;

export type MonitorOriginView = 'overview' | 'runtime' | 'dependencies';

export type MonitorOriginContext = Partial<{
  view: MonitorOriginView;
  trendRange: MonitorTrendRange;
  anomalyKey: string;
  scopeRef: string;
}>;

function trimQueryValue(value: unknown) {
  return typeof value === 'string' ? value.trim() : '';
}

function normalizeView(value: string): MonitorOriginView | '' {
  switch (value) {
    case 'overview':
    case 'runtime':
    case 'dependencies':
      return value;
    default:
      return '';
  }
}

export function normalizeMonitorOriginContext(context: MonitorOriginContext): MonitorOriginContext {
  const view = normalizeView(trimQueryValue(context.view));
  const trendRange = trimQueryValue(context.trendRange);
  const anomalyKey = trimQueryValue(context.anomalyKey);
  const scopeRef = trimQueryValue(context.scopeRef);

  return {
    ...(view ? { view } : {}),
    ...(trendRange ? { trendRange: trendRange as MonitorTrendRange } : {}),
    ...(anomalyKey ? { anomalyKey } : {}),
    ...(scopeRef ? { scopeRef } : {}),
  };
}

export function buildMonitorOriginQuery(context: MonitorOriginContext) {
  const normalized = normalizeMonitorOriginContext(context);

  return Object.fromEntries(
    [
      [MONITOR_ORIGIN_QUERY_KEY.VIEW, normalized.view],
      [MONITOR_ORIGIN_QUERY_KEY.TREND_RANGE, normalized.trendRange],
      [MONITOR_ORIGIN_QUERY_KEY.ANOMALY_KEY, normalized.anomalyKey],
      [MONITOR_ORIGIN_QUERY_KEY.SCOPE_REF, normalized.scopeRef],
    ].filter(([, value]) => value !== undefined && value !== ''),
  ) as Record<string, string>;
}

export function parseMonitorOriginQuery(query: Record<string, unknown>): MonitorOriginContext | null {
  const view = normalizeView(trimQueryValue(query[MONITOR_ORIGIN_QUERY_KEY.VIEW]));
  const context = normalizeMonitorOriginContext({
    ...(view ? { view } : {}),
    trendRange: trimQueryValue(query[MONITOR_ORIGIN_QUERY_KEY.TREND_RANGE]) as MonitorTrendRange,
    anomalyKey: trimQueryValue(query[MONITOR_ORIGIN_QUERY_KEY.ANOMALY_KEY]),
    scopeRef: trimQueryValue(query[MONITOR_ORIGIN_QUERY_KEY.SCOPE_REF]),
  });

  return Object.keys(context).length > 0 ? context : null;
}

export function buildMonitorLocationFromOrigin(context: MonitorOriginContext): RouteLocationRaw {
  const normalized = normalizeMonitorOriginContext(context);

  const path =
    normalized.view === 'runtime'
      ? MONITOR_ROUTE_PATH.SERVER_RUNTIME
      : normalized.view === 'dependencies'
        ? MONITOR_ROUTE_PATH.SERVER_DEPENDENCIES
        : MONITOR_ROUTE_PATH.SERVER_OVERVIEW;

  return {
    path,
    query: buildMonitorOriginQuery(normalized),
  };
}
