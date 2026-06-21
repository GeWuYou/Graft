import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { APP_LOG_API_PATH } from '../contract/paths';
import type { AppLogBatchDeleteRequest, AppLogDetailResponse, AppLogListResponse, AppLogQuery } from '../types/app-log';

type AppLogListPath = (typeof APP_LOG_API_PATH)['LIST'];
type GetAppLogsOperation = paths[AppLogListPath]['get'];
type GetAppLogsResponse = GetAppLogsOperation['responses'][200]['content']['application/json'];
type GetAppLogsResponseData = NonNullable<GetAppLogsResponse['data']>;

type AppLogDetailPath = (typeof APP_LOG_API_PATH)['DETAIL'];
type GetAppLogDetailOperation = paths[AppLogDetailPath]['get'];
type GetAppLogDetailResponse = GetAppLogDetailOperation['responses'][200]['content']['application/json'];
type GetAppLogDetailResponseData = NonNullable<GetAppLogDetailResponse['data']>;
type DeleteAppLogOperation = paths[AppLogDetailPath]['delete'];
type DeleteAppLogResponse = DeleteAppLogOperation['responses'][200]['content']['application/json'];
type DeleteAppLogResponseData = NonNullable<DeleteAppLogResponse['data']>;
type AppLogBatchDeletePath = (typeof APP_LOG_API_PATH)['BATCH_DELETE'];
type PostAppLogBatchDeleteOperation = paths[AppLogBatchDeletePath]['post'];
type PostAppLogBatchDeleteResponse = PostAppLogBatchDeleteOperation['responses'][200]['content']['application/json'];
type PostAppLogBatchDeleteResponseData = NonNullable<PostAppLogBatchDeleteResponse['data']>;

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

export function deleteAppLog(id: number) {
  return request.delete<DeleteAppLogResponseData>({
    url: APP_LOG_API_PATH.DETAIL.replace('{id}', String(id)),
  });
}

export function deleteAppLogs(payload: AppLogBatchDeleteRequest) {
  return request.post<PostAppLogBatchDeleteResponseData>({
    url: APP_LOG_API_PATH.BATCH_DELETE,
    data: payload,
  });
}
