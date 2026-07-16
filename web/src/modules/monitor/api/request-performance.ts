import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { MONITOR_API_PATH } from '../contract/paths';
import type { MonitorTrendRange } from '../contract/trend';
import type { RequestPerformanceResponse } from '../types/request-performance';

type RequestPerformancePath = (typeof MONITOR_API_PATH)['REQUEST_PERFORMANCE'];
type RequestPerformanceOperation = paths[RequestPerformancePath]['get'];
type RequestPerformanceQuery = NonNullable<RequestPerformanceOperation['parameters']['query']>;
type RequestPerformanceEnvelope = RequestPerformanceOperation['responses'][200]['content']['application/json'];
type RequestPerformanceData = NonNullable<RequestPerformanceEnvelope['data']>;

export function getRequestPerformance(range: MonitorTrendRange) {
  const params: RequestPerformanceQuery = { range };
  return request.get<RequestPerformanceData>({
    url: MONITOR_API_PATH.REQUEST_PERFORMANCE,
    params,
  }) as Promise<RequestPerformanceResponse>;
}
