import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { RUNTIME_TARGET_API_PATH, runtimeTargetDetailApiPath, runtimeTargetRefreshApiPath } from '../contract/paths';

type ListOperation = paths[(typeof RUNTIME_TARGET_API_PATH)['LIST']]['get'];
type DetailOperation = paths[(typeof RUNTIME_TARGET_API_PATH)['DETAIL']]['get'];
type RefreshOperation = paths[(typeof RUNTIME_TARGET_API_PATH)['REFRESH']]['post'];
export type RuntimeTarget = NonNullable<
  ListOperation['responses'][200]['content']['application/json']['data']
>['items'][number];
type RuntimeTargetList = NonNullable<ListOperation['responses'][200]['content']['application/json']['data']>;
type RuntimeTargetDetail = NonNullable<DetailOperation['responses'][200]['content']['application/json']['data']>;
type RuntimeTargetRefresh = NonNullable<RefreshOperation['responses'][200]['content']['application/json']['data']>;

/**
 * 获取运行时目标列表。
 *
 * @returns 运行时目标数组；响应中没有列表项时返回空数组。
 */
export async function listRuntimeTargets(): Promise<RuntimeTarget[]> {
  const data = await request.get<RuntimeTargetList>({ url: RUNTIME_TARGET_API_PATH.LIST });
  return data.items ?? [];
}
/**
 * 获取指定 ID 的运行时目标详情。
 *
 * @param id - 运行时目标的唯一标识
 * @returns 运行时目标详情；无结果时为 `null`
 */
export async function getRuntimeTarget(id: number): Promise<RuntimeTargetDetail | null> {
  return request.get<RuntimeTargetDetail>({ url: runtimeTargetDetailApiPath(id) });
}
/**
 * 刷新指定的运行时目标。
 *
 * @param id - 运行时目标的标识符
 * @returns 刷新后的运行时目标信息，或 `null`
 */
export async function refreshRuntimeTarget(id: number): Promise<RuntimeTargetRefresh | null> {
  return request.post<RuntimeTargetRefresh>({ url: runtimeTargetRefreshApiPath(id) });
}
