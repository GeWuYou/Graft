import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { ACCESS_LOG_API_PATH } from '../contract/paths';
import type { AccessLogDetailResponse, AccessLogListResponse, AccessLogQuery } from '../types/access-log';

type AccessLogListPath = (typeof ACCESS_LOG_API_PATH)['LIST'];
type GetAccessLogsOperation = paths[AccessLogListPath]['get'];
type GetAccessLogsResponse = GetAccessLogsOperation['responses'][200]['content']['application/json'];
type GetAccessLogsResponseData = NonNullable<GetAccessLogsResponse['data']>;

type AccessLogDetailPath = (typeof ACCESS_LOG_API_PATH)['DETAIL'];
type GetAccessLogDetailOperation = paths[AccessLogDetailPath]['get'];
type GetAccessLogDetailResponse = GetAccessLogDetailOperation['responses'][200]['content']['application/json'];
type GetAccessLogDetailResponseData = NonNullable<GetAccessLogDetailResponse['data']>;

export function getAccessLogs(query: AccessLogQuery) {
  return request.get<GetAccessLogsResponseData>({
    url: ACCESS_LOG_API_PATH.LIST,
    params: query,
  }) as Promise<AccessLogListResponse>;
}

export function getAccessLogDetail(id: number) {
  return request.get<GetAccessLogDetailResponseData>({
    url: ACCESS_LOG_API_PATH.DETAIL.replace('{id}', String(id)),
  }) as Promise<AccessLogDetailResponse>;
}
