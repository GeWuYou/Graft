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
type RuntimeTargetListQuery = NonNullable<ListOperation['parameters']['query']>;
export type RuntimeTargetDetail = NonNullable<DetailOperation['responses'][200]['content']['application/json']['data']>;
type RuntimeTargetRefresh = NonNullable<RefreshOperation['responses'][200]['content']['application/json']['data']>;
type RuntimeTargetDiscoverLocal = NonNullable<
  DiscoverLocalOperation['responses'][200]['content']['application/json']['data']
>;
export type RuntimeTargetUsageMetric = components['schemas']['runtime-target-usage-metric'];
export type RuntimeTargetPage = RuntimeTargetList;
type RuntimeTargetSavedViewsOperation = paths[typeof OPENAPI_RUNTIME_PATH.getRuntimeTargetSavedViews]['get'];
type RuntimeTargetSavedViewsData = NonNullable<
  RuntimeTargetSavedViewsOperation['responses'][200]['content']['application/json']['data']
>;
type RuntimeTargetCreateSavedViewOperation = paths[typeof OPENAPI_RUNTIME_PATH.postRuntimeTargetSavedView]['post'];
type RuntimeTargetCreateSavedViewData = NonNullable<
  RuntimeTargetCreateSavedViewOperation['responses'][201]['content']['application/json']['data']
>;
type RuntimeTargetUpdateSavedViewOperation = paths[typeof OPENAPI_RUNTIME_PATH.putRuntimeTargetSavedView]['put'];
type RuntimeTargetUpdateSavedViewData = NonNullable<
  RuntimeTargetUpdateSavedViewOperation['responses'][200]['content']['application/json']['data']
>;
export type RuntimeTargetSavedView = components['schemas']['saved-view'];
type RuntimeTargetSavedViewRequest = components['schemas']['saved-view-request'];
export type RuntimeTargetSavedViewInput = {
  name: string;
  pageSize: number;
  queryState: RuntimeTargetSavedViewRequest['query_state'];
  visibleColumns: RuntimeTargetSavedViewRequest['visible_columns'];
  isDefault: boolean;
};

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

export async function listRuntimeTargetPage(params: RuntimeTargetListQuery): Promise<RuntimeTargetPage> {
  return request.get<RuntimeTargetPage>({
    url: OPENAPI_RUNTIME_PATH.getRuntimeTargets,
    params,
  });
}

export async function getRuntimeTargetSavedViews(): Promise<RuntimeTargetSavedView[]> {
  const data = await request.get<RuntimeTargetSavedViewsData>({ url: OPENAPI_RUNTIME_PATH.getRuntimeTargetSavedViews });
  return data.items;
}

export function postRuntimeTargetSavedView(input: RuntimeTargetSavedViewInput) {
  return request.post<RuntimeTargetCreateSavedViewData>({
    url: OPENAPI_RUNTIME_PATH.postRuntimeTargetSavedView,
    data: toRuntimeTargetSavedViewRequest(input),
  }) as Promise<RuntimeTargetSavedView>;
}

export function putRuntimeTargetSavedView(viewId: number, input: RuntimeTargetSavedViewInput) {
  return request.put<RuntimeTargetUpdateSavedViewData>({
    url: buildOpenApiRuntimePath('putRuntimeTargetSavedView', { viewId }),
    data: toRuntimeTargetSavedViewRequest(input),
  }) as Promise<RuntimeTargetSavedView>;
}

function toRuntimeTargetSavedViewRequest(input: RuntimeTargetSavedViewInput): RuntimeTargetSavedViewRequest {
  return {
    name: input.name,
    page_size: input.pageSize,
    query_state: input.queryState,
    visible_columns: input.visibleColumns,
    is_default: input.isDefault,
  };
}

export function deleteRuntimeTargetSavedView(viewId: number) {
  return request.delete({ url: buildOpenApiRuntimePath('deleteRuntimeTargetSavedView', { viewId }) });
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
