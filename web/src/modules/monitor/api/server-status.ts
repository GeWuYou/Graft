// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { paths } from '@/contracts/openapi/generated/schema';
import { createLogger } from '@/utils/logger';
import { request } from '@/utils/request';

import { MONITOR_API_PATH } from '../contract/paths';
import type { MonitorTrendRange } from '../contract/trend';
import type { ServerStatusResponse } from '../types/server-status';

type MonitorServerStatusPath = (typeof MONITOR_API_PATH)['SERVER_STATUS'];
type GetMonitorServerStatusOperation = paths[MonitorServerStatusPath]['get'];
type GetMonitorServerStatusQuery = NonNullable<GetMonitorServerStatusOperation['parameters']['query']>;
type GetMonitorServerStatusEnvelope = GetMonitorServerStatusOperation['responses'][200]['content']['application/json'];
type GetMonitorServerStatusData = NonNullable<GetMonitorServerStatusEnvelope['data']>;

const logger = createLogger('monitor.server-status.api');

function normalizeCpuPercent(
  rawValue: number,
  point: ServerStatusResponse['trend']['points'][number],
  pointIndex: number,
  trendRange: MonitorTrendRange,
) {
  if (!Number.isFinite(rawValue)) {
    return rawValue;
  }

  const normalizedValue = Math.min(100, Math.max(0, rawValue));
  if (normalizedValue !== rawValue) {
    logger.warn('monitor server status trend cpu_percent out of range', {
      rawValue,
      normalizedValue,
      pointIndex,
      observedAt: point.observed_at,
      trendRange,
    });
  }

  return normalizedValue;
}

function normalizeServerStatusResponse(response: ServerStatusResponse, trendRange: MonitorTrendRange) {
  const trend = response.trend as ServerStatusResponse['trend'] | undefined;
  if (!Array.isArray(trend?.points)) {
    return response;
  }

  let changed = false;
  const points = trend.points.map((point, pointIndex) => {
    const cpuPercent = normalizeCpuPercent(point.cpu_percent, point, pointIndex, trendRange);
    if (cpuPercent === point.cpu_percent) {
      return point;
    }

    changed = true;
    return {
      ...point,
      cpu_percent: cpuPercent,
    };
  });

  if (!changed) {
    return response;
  }

  return {
    ...response,
    trend: {
      ...trend,
      points,
    },
  };
}

export async function getServerStatus(trendRange: MonitorTrendRange) {
  const params: GetMonitorServerStatusQuery = {
    trend_range: trendRange,
  };

  const response = (await request.get<GetMonitorServerStatusData>({
    url: MONITOR_API_PATH.SERVER_STATUS,
    params,
  })) as ServerStatusResponse;

  return normalizeServerStatusResponse(response, trendRange);
}
