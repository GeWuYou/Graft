import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { components, paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

type ListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getRuntimeTargets]['get'];
type DetailOperation = paths[typeof OPENAPI_RUNTIME_PATH.getRuntimeTarget]['get'];
type RefreshOperation = paths[typeof OPENAPI_RUNTIME_PATH.postRuntimeTargetRefresh]['post'];
type DiscoverLocalOperation = paths[typeof OPENAPI_RUNTIME_PATH.postRuntimeTargetsDiscoverLocalDocker]['post'];
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
    url: OPENAPI_RUNTIME_PATH.getRuntimeTargets,
    params,
  });
}
export async function discoverLocalDocker(): Promise<RuntimeTargetDiscoverLocal | null> {
  return request.post<RuntimeTargetDiscoverLocal | null>({
    url: OPENAPI_RUNTIME_PATH.postRuntimeTargetsDiscoverLocalDocker,
  });
}

export async function getRuntimeTarget(id: number): Promise<RuntimeTargetDetail> {
  return request.get<RuntimeTargetDetail>({ url: buildOpenApiRuntimePath('getRuntimeTarget', { id }) });
}

export async function refreshRuntimeTarget(id: number): Promise<RuntimeTargetRefresh> {
  return request.post<RuntimeTargetRefresh>({ url: buildOpenApiRuntimePath('postRuntimeTargetRefresh', { id }) });
}
