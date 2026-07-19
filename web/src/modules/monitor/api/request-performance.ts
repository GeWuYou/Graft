import { OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type { MonitorTrendRange } from '../contract/trend';
import type { RequestPerformanceResponse } from '../types/request-performance';

type RequestPerformancePath = typeof OPENAPI_RUNTIME_PATH.getMonitorRequestPerformance;
type RequestPerformanceOperation = paths[RequestPerformancePath]['get'];
type RequestPerformanceQuery = NonNullable<RequestPerformanceOperation['parameters']['query']>;
type RequestPerformanceEnvelope = RequestPerformanceOperation['responses'][200]['content']['application/json'];
type RequestPerformanceData = NonNullable<RequestPerformanceEnvelope['data']>;

export function getRequestPerformance(range: MonitorTrendRange) {
  const params: RequestPerformanceQuery = { range };
  return request.get<RequestPerformanceData>({
    url: OPENAPI_RUNTIME_PATH.getMonitorRequestPerformance,
    params,
  }) as Promise<RequestPerformanceResponse>;
}
