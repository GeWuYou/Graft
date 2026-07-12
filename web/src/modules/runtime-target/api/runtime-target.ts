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

export async function listRuntimeTargets(): Promise<RuntimeTarget[]> {
  const data = await request.get<RuntimeTargetList>({ url: RUNTIME_TARGET_API_PATH.LIST });
  return data.items ?? [];
}
export async function getRuntimeTarget(id: number): Promise<RuntimeTargetDetail | null> {
  return request.get<RuntimeTargetDetail>({ url: runtimeTargetDetailApiPath(id) });
}
export async function refreshRuntimeTarget(id: number): Promise<RuntimeTargetRefresh | null> {
  return request.post<RuntimeTargetRefresh>({ url: runtimeTargetRefreshApiPath(id) });
}
