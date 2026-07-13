import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { APP_LOG_API_PATH, buildAppLogSavedViewApiPath } from '../contract/paths';
import type {
  AppLogBatchDeleteRequest,
  AppLogDetailResponse,
  AppLogListResponse,
  AppLogQuery,
  AppLogSavedView,
  AppLogSavedViewRequest,
} from '../types/app-log';

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

type AppLogSavedViewsPath = (typeof APP_LOG_API_PATH)['SAVED_VIEWS'];
type GetAppLogSavedViewsOperation = paths[AppLogSavedViewsPath]['get'];
type GetAppLogSavedViewsResponse = GetAppLogSavedViewsOperation['responses'][200]['content']['application/json'];
type GetAppLogSavedViewsResponseData = NonNullable<GetAppLogSavedViewsResponse['data']>;
type PostAppLogSavedViewOperation = paths[AppLogSavedViewsPath]['post'];
type PostAppLogSavedViewResponse = PostAppLogSavedViewOperation['responses'][201]['content']['application/json'];
type PostAppLogSavedViewResponseData = NonNullable<PostAppLogSavedViewResponse['data']>;

type AppLogSavedViewPath = (typeof APP_LOG_API_PATH)['SAVED_VIEW'];
type PutAppLogSavedViewOperation = paths[AppLogSavedViewPath]['put'];
type PutAppLogSavedViewResponse = PutAppLogSavedViewOperation['responses'][200]['content']['application/json'];
type PutAppLogSavedViewResponseData = NonNullable<PutAppLogSavedViewResponse['data']>;

/**
 * 根据查询条件获取应用日志列表。
 *
 * @param query - 应用日志查询条件
 * @returns 应用日志列表响应
 */
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

/**
 * 批量删除应用日志。
 *
 * @param payload - 包含待删除日志标识的批量删除请求参数
 */
export function deleteAppLogs(payload: AppLogBatchDeleteRequest) {
  return request.post<PostAppLogBatchDeleteResponseData>({
    url: APP_LOG_API_PATH.BATCH_DELETE,
    data: payload,
  });
}

/**
 * 获取应用日志的已保存视图列表。
 *
 * @returns 已保存的应用日志视图数组
 */
export async function getAppLogSavedViews(): Promise<AppLogSavedView[]> {
  const data = await request.get<GetAppLogSavedViewsResponseData>({ url: APP_LOG_API_PATH.SAVED_VIEWS });
  return data.items;
}

/**
 * 创建一个 App Log 保存视图。
 *
 * @param payload - 保存视图的请求数据
 * @returns 创建后的保存视图
 */
export function postAppLogSavedView(payload: AppLogSavedViewRequest) {
  return request.post<PostAppLogSavedViewResponseData>({
    url: APP_LOG_API_PATH.SAVED_VIEWS,
    data: payload,
  }) as Promise<AppLogSavedView>;
}

/**
 * 更新指定的 App Log 保存视图。
 *
 * @param viewId - 要更新的保存视图 ID
 * @param payload - 保存视图的更新内容
 * @returns 更新后的保存视图
 */
export function putAppLogSavedView(viewId: number, payload: AppLogSavedViewRequest) {
  return request.put<PutAppLogSavedViewResponseData>({
    url: buildAppLogSavedViewApiPath(viewId),
    data: payload,
  }) as Promise<AppLogSavedView>;
}

/**
 * 删除指定的应用日志保存视图。
 *
 * @param viewId - 要删除的保存视图标识
 */
export function deleteAppLogSavedView(viewId: number) {
  return request.delete({ url: buildAppLogSavedViewApiPath(viewId) });
}
