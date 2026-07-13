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

/**
 * 获取供现有选择器使用的运行时目标集合。
 *
 * @returns 最多包含 100 个运行时目标的数组
 */
export async function listRuntimeTargets(): Promise<RuntimeTarget[]> {
  const page = await listRuntimeTargetPage({ limit: 100, offset: 0 });
  return page.items;
}

/**
 * 获取分页的运行时目标列表。
 *
 * @param params - 分页参数，包括每页数量和偏移量
 * @returns 运行时目标分页数据
 */
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
