import type { components, paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { RUNTIME_TARGET_API_PATH } from '../contract/paths';

type ListOperation = paths[(typeof RUNTIME_TARGET_API_PATH)['LIST']]['get'];
type DiscoverLocalOperation = paths[(typeof RUNTIME_TARGET_API_PATH)['DISCOVER_LOCAL']]['post'];
export type RuntimeTarget = NonNullable<
  ListOperation['responses'][200]['content']['application/json']['data']
>['items'][number];
type RuntimeTargetList = NonNullable<ListOperation['responses'][200]['content']['application/json']['data']>;
type RuntimeTargetDiscoverLocal = NonNullable<
  DiscoverLocalOperation['responses'][200]['content']['application/json']['data']
>;
export type RuntimeTargetMetric = components['schemas']['runtime-target-usage-metric'];
export type RuntimeTargetPage = RuntimeTargetList;

/** Returns the bounded target collection used by existing selector consumers. */
export async function listRuntimeTargets(): Promise<RuntimeTarget[]> {
  const page = await listRuntimeTargetPage({ limit: 100, offset: 0 });
  return page.items;
}

export async function listRuntimeTargetPage(params: { limit: number; offset: number }): Promise<RuntimeTargetPage> {
  return request.get<RuntimeTargetPage>({
    url: RUNTIME_TARGET_API_PATH.LIST,
    params,
  });
}
/**
 * 探测当前服务器的 Local Docker；服务端负责幂等创建或恢复系统管理目标。
 */
export async function discoverLocalDocker(): Promise<RuntimeTargetDiscoverLocal | null> {
  return request.post<RuntimeTargetDiscoverLocal | null>({ url: RUNTIME_TARGET_API_PATH.DISCOVER_LOCAL });
}
