import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { ACCESS_LOG_API_PATH, buildAccessLogSavedViewApiPath } from '../contract/paths';
import type {
  AccessLogDetailResponse,
  AccessLogListResponse,
  AccessLogQuery,
  AccessLogSavedView,
  AccessLogSavedViewRequest,
} from '../types/access-log';

type AccessLogListPath = (typeof ACCESS_LOG_API_PATH)['LIST'];
type GetAccessLogsOperation = paths[AccessLogListPath]['get'];
type GetAccessLogsResponse = GetAccessLogsOperation['responses'][200]['content']['application/json'];
type GetAccessLogsResponseData = NonNullable<GetAccessLogsResponse['data']>;

type AccessLogDetailPath = (typeof ACCESS_LOG_API_PATH)['DETAIL'];
type GetAccessLogDetailOperation = paths[AccessLogDetailPath]['get'];
type GetAccessLogDetailResponse = GetAccessLogDetailOperation['responses'][200]['content']['application/json'];
type GetAccessLogDetailResponseData = NonNullable<GetAccessLogDetailResponse['data']>;

type AccessLogSavedViewsPath = (typeof ACCESS_LOG_API_PATH)['SAVED_VIEWS'];
type GetAccessLogSavedViewsOperation = paths[AccessLogSavedViewsPath]['get'];
type GetAccessLogSavedViewsResponse = GetAccessLogSavedViewsOperation['responses'][200]['content']['application/json'];
type GetAccessLogSavedViewsResponseData = NonNullable<GetAccessLogSavedViewsResponse['data']>;
type PostAccessLogSavedViewOperation = paths[AccessLogSavedViewsPath]['post'];
type PostAccessLogSavedViewResponse = PostAccessLogSavedViewOperation['responses'][201]['content']['application/json'];
type PostAccessLogSavedViewResponseData = NonNullable<PostAccessLogSavedViewResponse['data']>;

type AccessLogSavedViewPath = (typeof ACCESS_LOG_API_PATH)['SAVED_VIEW'];
type PutAccessLogSavedViewOperation = paths[AccessLogSavedViewPath]['put'];
type PutAccessLogSavedViewResponse = PutAccessLogSavedViewOperation['responses'][200]['content']['application/json'];
type PutAccessLogSavedViewResponseData = NonNullable<PutAccessLogSavedViewResponse['data']>;

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

export async function getAccessLogSavedViews(): Promise<AccessLogSavedView[]> {
  const data = await request.get<GetAccessLogSavedViewsResponseData>({ url: ACCESS_LOG_API_PATH.SAVED_VIEWS });
  return data.items;
}

export function postAccessLogSavedView(payload: AccessLogSavedViewRequest) {
  return request.post<PostAccessLogSavedViewResponseData>({
    url: ACCESS_LOG_API_PATH.SAVED_VIEWS,
    data: payload,
  }) as Promise<AccessLogSavedView>;
}

export function putAccessLogSavedView(viewId: number, payload: AccessLogSavedViewRequest) {
  return request.put<PutAccessLogSavedViewResponseData>({
    url: buildAccessLogSavedViewApiPath(viewId),
    data: payload,
  }) as Promise<AccessLogSavedView>;
}

export function deleteAccessLogSavedView(viewId: number) {
  return request.delete({ url: buildAccessLogSavedViewApiPath(viewId) });
}
