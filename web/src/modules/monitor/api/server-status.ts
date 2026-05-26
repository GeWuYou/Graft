import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { MONITOR_API_PATH } from '../contract/paths';
import type { MonitorTrendRange } from '../contract/trend';
import type { ServerStatusResponse } from '../types/server-status';

type MonitorServerStatusPath = (typeof MONITOR_API_PATH)['SERVER_STATUS'];
type GetMonitorServerStatusOperation = paths[MonitorServerStatusPath]['get'];
type GetMonitorServerStatusQuery = NonNullable<GetMonitorServerStatusOperation['parameters']['query']>;
type GetMonitorServerStatusEnvelope = GetMonitorServerStatusOperation['responses'][200]['content']['application/json'];
type GetMonitorServerStatusData = NonNullable<GetMonitorServerStatusEnvelope['data']>;

export function getServerStatus(trendRange: MonitorTrendRange) {
  const params: GetMonitorServerStatusQuery = {
    trend_range: trendRange,
  };

  return request.get<GetMonitorServerStatusData>({
    url: MONITOR_API_PATH.SERVER_STATUS,
    params,
  }) as Promise<ServerStatusResponse>;
}
