import type { components, paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { RUNTIME_TARGET_API_PATH, runtimeTargetDetailApiPath, runtimeTargetRefreshApiPath } from '../contract/paths';

type ListOperation = paths[(typeof RUNTIME_TARGET_API_PATH)['LIST']]['get'];
type DetailOperation = paths[(typeof RUNTIME_TARGET_API_PATH)['DETAIL']]['get'];
type RefreshOperation = paths[(typeof RUNTIME_TARGET_API_PATH)['REFRESH']]['post'];
type DiscoverLocalOperation = paths[(typeof RUNTIME_TARGET_API_PATH)['DISCOVER_LOCAL_DOCKER']]['post'];
export type RuntimeTarget = NonNullable<
  ListOperation['responses'][200]['content']['application/json']['data']
>['items'][number];
type RuntimeTargetList = NonNullable<ListOperation['responses'][200]['content']['application/json']['data']>;
export type RuntimeTargetDetail = NonNullable<DetailOperation['responses'][200]['content']['application/json']['data']>;
type RuntimeTargetRefresh = NonNullable<RefreshOperation['responses'][200]['content']['application/json']['data']>;
type RuntimeTargetDiscoverLocal = NonNullable<
  DiscoverLocalOperation['responses'][200]['content']['application/json']['data']
>;
export type RuntimeTargetUsageMetric = components['schemas']['runtime-target-usage-metric'];
export type RuntimeTargetPage = RuntimeTargetList;

const runtimeTargetSelectorPageLimit = 100;

export async function listRuntimeTargets(): Promise<RuntimeTarget[]> {
  const targets: RuntimeTarget[] = [];
  for (let offset = 0; ; offset += runtimeTargetSelectorPageLimit) {
    const page = await listRuntimeTargetPage({ limit: runtimeTargetSelectorPageLimit, offset });
    targets.push(...page.items);
    if (targets.length >= page.total || page.items.length === 0) {
      return targets;
    }
  }
}

export async function listRuntimeTargetPage(params: { limit: number; offset: number }): Promise<RuntimeTargetPage> {
  return request.get<RuntimeTargetPage>({
    url: RUNTIME_TARGET_API_PATH.LIST,
    params,
  });
}
export async function discoverLocalDocker(): Promise<RuntimeTargetDiscoverLocal | null> {
  return request.post<RuntimeTargetDiscoverLocal | null>({ url: RUNTIME_TARGET_API_PATH.DISCOVER_LOCAL_DOCKER });
}

export async function getRuntimeTarget(id: number): Promise<RuntimeTargetDetail> {
  return request.get<RuntimeTargetDetail>({ url: runtimeTargetDetailApiPath(id) });
}

export async function refreshRuntimeTarget(id: number): Promise<RuntimeTargetRefresh> {
  return request.post<RuntimeTargetRefresh>({ url: runtimeTargetRefreshApiPath(id) });
}
