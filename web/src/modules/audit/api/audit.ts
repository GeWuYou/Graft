import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  AuditIncidentResponse,
  AuditLogDetailResponse,
  AuditLogListResponse,
  AuditLogQuery,
  AuditSavedView,
  AuditSavedViewRequest,
  AuditVisibilityDefaultResponse,
  AuditVisibilityDefaultUpdateRequest,
  AuditVisibilityOverrideResponse,
  AuditVisibilityOverrideUpsertRequest,
  AuditVisibilityPolicyResponse,
} from '../types/audit';

type AuditLogsPath = typeof OPENAPI_RUNTIME_PATH.getAuditLogs;
type GetAuditLogsOperation = paths[AuditLogsPath]['get'];
type GetAuditLogsResponse = GetAuditLogsOperation['responses'][200]['content']['application/json'];
type GetAuditLogsResponseData = NonNullable<GetAuditLogsResponse['data']>;

type AuditLogDetailPath = typeof OPENAPI_RUNTIME_PATH.getAuditLogDetail;
type GetAuditLogDetailOperation = paths[AuditLogDetailPath]['get'];
type GetAuditLogDetailResponse = GetAuditLogDetailOperation['responses'][200]['content']['application/json'];
type GetAuditLogDetailResponseData = NonNullable<GetAuditLogDetailResponse['data']>;

type AuditSavedViewsPath = typeof OPENAPI_RUNTIME_PATH.getAuditLogSavedViews;
type GetAuditSavedViewsOperation = paths[AuditSavedViewsPath]['get'];
type GetAuditSavedViewsResponse = GetAuditSavedViewsOperation['responses'][200]['content']['application/json'];
type GetAuditSavedViewsResponseData = NonNullable<GetAuditSavedViewsResponse['data']>;
type PostAuditSavedViewOperation = paths[AuditSavedViewsPath]['post'];
type PostAuditSavedViewResponse = PostAuditSavedViewOperation['responses'][201]['content']['application/json'];
type PostAuditSavedViewResponseData = NonNullable<PostAuditSavedViewResponse['data']>;

type AuditSavedViewPath = typeof OPENAPI_RUNTIME_PATH.putAuditLogSavedView;
type PutAuditSavedViewOperation = paths[AuditSavedViewPath]['put'];
type PutAuditSavedViewResponse = PutAuditSavedViewOperation['responses'][200]['content']['application/json'];
type PutAuditSavedViewResponseData = NonNullable<PutAuditSavedViewResponse['data']>;

type AuditIncidentPath = typeof OPENAPI_RUNTIME_PATH.getAuditIncident;
type GetAuditIncidentOperation = paths[AuditIncidentPath]['get'];
type GetAuditIncidentResponse = GetAuditIncidentOperation['responses'][200]['content']['application/json'];
type GetAuditIncidentResponseData = NonNullable<GetAuditIncidentResponse['data']>;

export function getAuditLogs(query: AuditLogQuery) {
  return request.get<GetAuditLogsResponseData>({
    url: OPENAPI_RUNTIME_PATH.getAuditLogs,
    params: query,
  }) as Promise<AuditLogListResponse>;
}

/**
 * 获取指定审计日志的详细信息。
 *
 * @param id - 审计日志的标识
 * @returns 审计日志详情
 */
export function getAuditLogDetail(id: number) {
  return request.get<GetAuditLogDetailResponseData>({
    url: buildOpenApiRuntimePath('getAuditLogDetail', { id }),
  }) as Promise<AuditLogDetailResponse>;
}

/**
 * 获取审计保存视图列表。
 *
 * @returns 审计保存视图数组
 */
export async function getAuditSavedViews(): Promise<AuditSavedView[]> {
  const data = await request.get<GetAuditSavedViewsResponseData>({ url: OPENAPI_RUNTIME_PATH.getAuditLogSavedViews });
  return data.items;
}

/**
 * 创建审计保存视图。
 *
 * @param payload - 保存视图的请求数据
 * @returns 创建的审计保存视图
 */
export function postAuditSavedView(payload: AuditSavedViewRequest) {
  return request.post<PostAuditSavedViewResponseData>({
    url: OPENAPI_RUNTIME_PATH.postAuditLogSavedView,
    data: payload,
  }) as Promise<AuditSavedView>;
}

/**
 * 更新指定的审计保存视图。
 *
 * @param viewId - 要更新的保存视图标识
 * @param payload - 保存视图的更新内容
 * @returns 更新后的审计保存视图
 */
export function putAuditSavedView(viewId: number, payload: AuditSavedViewRequest) {
  return request.put<PutAuditSavedViewResponseData>({
    url: buildOpenApiRuntimePath('putAuditLogSavedView', { viewId }),
    data: payload,
  }) as Promise<AuditSavedView>;
}

/**
 * 删除指定的审计保存视图。
 *
 * @param viewId - 要删除的保存视图 ID
 */
export function deleteAuditSavedView(viewId: number) {
  return request.delete({ url: buildOpenApiRuntimePath('deleteAuditLogSavedView', { viewId }) });
}

/**
 * 获取指定审计事件的事故详情。
 *
 * @param eventId - 审计事件 ID
 * @returns 事故详情数据
 */
export function getAuditIncident(eventId: number) {
  return request.get<GetAuditIncidentResponseData>({
    url: buildOpenApiRuntimePath('getAuditIncident', { event_id: eventId }),
  }) as Promise<AuditIncidentResponse>;
}

/**
 * 获取审计可见性策略。
 *
 * @returns 审计可见性策略配置。
 */
export function getAuditVisibilityPolicy() {
  return request.get<AuditVisibilityPolicyResponse>({
    url: OPENAPI_RUNTIME_PATH.getAuditVisibilityPolicy,
  });
}

/**
 * 更新审计可见性默认策略。
 *
 * @param payload - 默认可见性设置的更新内容
 * @returns 更新后的默认可见性策略
 */
export function updateAuditVisibilityDefault(payload: AuditVisibilityDefaultUpdateRequest) {
  return request.put<AuditVisibilityDefaultResponse>({
    url: OPENAPI_RUNTIME_PATH.putAuditVisibilityPolicy,
    data: payload,
  });
}

/**
 * 新增或更新审计可见性覆盖规则。
 *
 * @param payload - 要提交的覆盖规则内容
 * @returns 可见性覆盖规则的更新结果
 */
export function upsertAuditVisibilityOverride(payload: AuditVisibilityOverrideUpsertRequest) {
  return request.put<AuditVisibilityOverrideResponse>({
    url: OPENAPI_RUNTIME_PATH.putAuditVisibilityOverride,
    data: payload,
  });
}

/**
 * 删除审计可见性覆盖规则。
 *
 * @param source - 覆盖规则所属来源
 * @param actionKey - 覆盖规则对应的动作键
 * @returns 空对象
 */
export function deleteAuditVisibilityOverride(source: string, actionKey: string) {
  return request.delete<Record<string, never>>({
    url: OPENAPI_RUNTIME_PATH.deleteAuditVisibilityOverride,
    params: {
      source,
      action_key: actionKey,
    },
  });
}
