import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { APP_LOG_API_PATH } from '../contract/paths';
import type { AppLogDetailResponse, AppLogListResponse, AppLogQuery } from '../types/app-log';

type AppLogListPath = (typeof APP_LOG_API_PATH)['LIST'];
type GetAppLogsOperation = paths[AppLogListPath]['get'];
type GetAppLogsResponse = GetAppLogsOperation['responses'][200]['content']['application/json'];
type GetAppLogsResponseData = NonNullable<GetAppLogsResponse['data']>;

type AppLogDetailPath = (typeof APP_LOG_API_PATH)['DETAIL'];
type GetAppLogDetailOperation = paths[AppLogDetailPath]['get'];
type GetAppLogDetailResponse = GetAppLogDetailOperation['responses'][200]['content']['application/json'];
type GetAppLogDetailResponseData = NonNullable<GetAppLogDetailResponse['data']>;

export function getAppLogs(query: AppLogQuery) {
  return request.get<GetAppLogsResponseData>({
    url: APP_LOG_API_PATH.LIST,
    params: query,
  }) as Promise<AppLogListResponse>;
}

export function getAppLogDetail(id: number) {
  return request.get<GetAppLogDetailResponseData>({
    url: APP_LOG_API_PATH.DETAIL.replace('{id}', String(id)),
  }) as Promise<AppLogDetailResponse>;
}
