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

/**
 * 获取访问日志列表。
 *
 * @param query - 访问日志查询条件
 * @returns 访问日志列表及其分页信息
 */
export function getAccessLogs(query: AccessLogQuery) {
  return request.get<GetAccessLogsResponseData>({
    url: ACCESS_LOG_API_PATH.LIST,
    params: query,
  }) as Promise<AccessLogListResponse>;
}

/**
 * 获取指定访问日志的详细信息。
 *
 * @param id - 访问日志的标识符
 * @returns 指定访问日志的详细信息
 */
export function getAccessLogDetail(id: number) {
  return request.get<GetAccessLogDetailResponseData>({
    url: ACCESS_LOG_API_PATH.DETAIL.replace('{id}', String(id)),
  }) as Promise<AccessLogDetailResponse>;
}

/**
 * 获取访问日志的已保存视图列表。
 *
 * @returns 已保存的访问日志视图数组
 */
export async function getAccessLogSavedViews(): Promise<AccessLogSavedView[]> {
  const data = await request.get<GetAccessLogSavedViewsResponseData>({ url: ACCESS_LOG_API_PATH.SAVED_VIEWS });
  return data.items;
}

/**
 * 创建访问日志保存视图。
 *
 * @param payload - 保存视图的请求数据
 * @returns 创建后的访问日志保存视图
 */
export function postAccessLogSavedView(payload: AccessLogSavedViewRequest) {
  return request.post<PostAccessLogSavedViewResponseData>({
    url: ACCESS_LOG_API_PATH.SAVED_VIEWS,
    data: payload,
  }) as Promise<AccessLogSavedView>;
}

/**
 * 更新指定的访问日志保存视图。
 *
 * @param viewId - 要更新的保存视图 ID
 * @param payload - 保存视图的更新内容
 * @returns 更新后的访问日志保存视图
 */
export function putAccessLogSavedView(viewId: number, payload: AccessLogSavedViewRequest) {
  return request.put<PutAccessLogSavedViewResponseData>({
    url: buildAccessLogSavedViewApiPath(viewId),
    data: payload,
  }) as Promise<AccessLogSavedView>;
}

/**
 * 删除指定的访问日志保存视图。
 *
 * @param viewId - 要删除的保存视图 ID
 * @returns 删除请求的响应结果
 */
export function deleteAccessLogSavedView(viewId: number) {
  return request.delete({ url: buildAccessLogSavedViewApiPath(viewId) });
}
