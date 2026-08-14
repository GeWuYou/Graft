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
type RuntimeTargetAssignmentCandidatesOperation =
  paths[typeof OPENAPI_RUNTIME_PATH.getRuntimeTargetAssignmentCandidates]['get'];
type RuntimeTargetAssignmentsOperation = paths[typeof OPENAPI_RUNTIME_PATH.getRuntimeTargetAssignments]['get'];
type RuntimeTargetAssignmentsData = NonNullable<
  RuntimeTargetAssignmentsOperation['responses'][200]['content']['application/json']['data']
>;
type RuntimeTargetAssignmentsReplaceOperation = paths[typeof OPENAPI_RUNTIME_PATH.putRuntimeTargetAssignments]['put'];
type RuntimeTargetAssignmentsBatchOperation =
  paths[typeof OPENAPI_RUNTIME_PATH.postRuntimeTargetAssignmentsBatch]['post'];
export type RuntimeTargetUsageMetric = components['schemas']['runtime-target-usage-metric'];
export type RuntimeTargetPage = RuntimeTargetList;
export type RuntimeTargetAssignment = components['schemas']['runtime-target-user-assignment'];
export type RuntimeTargetAssignmentCandidate = components['schemas']['runtime-target-assignment-candidate'];
const runtimeTargetAssignmentRevisions = new Map<number, number>();
type RuntimeTargetSavedViewsOperation = paths[typeof OPENAPI_RUNTIME_PATH.getRuntimeTargetSavedViews]['get'];
type RuntimeTargetSavedViewsData = NonNullable<
  RuntimeTargetSavedViewsOperation['responses'][200]['content']['application/json']['data']
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

export function postRuntimeTargetSavedView(input: RuntimeTargetSavedViewInput): Promise<RuntimeTargetSavedView> {
  return request.post<RuntimeTargetSavedView>({
    url: OPENAPI_RUNTIME_PATH.postRuntimeTargetSavedView,
    data: toRuntimeTargetSavedViewRequest(input),
  });
}

export function putRuntimeTargetSavedView(
  viewId: number,
  input: RuntimeTargetSavedViewInput,
): Promise<RuntimeTargetSavedView> {
  return request.put<RuntimeTargetSavedView>({
    url: buildOpenApiRuntimePath('putRuntimeTargetSavedView', { viewId }),
    data: toRuntimeTargetSavedViewRequest(input),
  });
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

export async function getRuntimeTargetAssignments(id: number): Promise<RuntimeTargetAssignmentsData> {
  const response = await request.get<RuntimeTargetAssignmentsData>({
    url: buildOpenApiRuntimePath('getRuntimeTargetAssignments', { id }),
  });
  runtimeTargetAssignmentRevisions.set(id, response.revision);
  return response;
}

export async function getRuntimeTargetAssignmentCandidates(
  id: number,
  params: NonNullable<RuntimeTargetAssignmentCandidatesOperation['parameters']['query']> = {},
): Promise<
  NonNullable<RuntimeTargetAssignmentCandidatesOperation['responses'][200]['content']['application/json']['data']>
> {
  return request.get<
    NonNullable<RuntimeTargetAssignmentCandidatesOperation['responses'][200]['content']['application/json']['data']>
  >({
    url: buildOpenApiRuntimePath('getRuntimeTargetAssignmentCandidates', { id }),
    params,
  });
}

export async function replaceRuntimeTargetAssignments(
  id: number,
  userIds: number[],
  revision = runtimeTargetAssignmentRevisions.get(id) ?? 1,
) {
  type Response = NonNullable<
    RuntimeTargetAssignmentsReplaceOperation['responses'][200]['content']['application/json']['data']
  >;
  type Request = NonNullable<RuntimeTargetAssignmentsReplaceOperation['requestBody']>['content']['application/json'];
  const response = await request.put<Response>({
    url: buildOpenApiRuntimePath('putRuntimeTargetAssignments', { id }),
    data: { user_ids: userIds, revision } satisfies Request,
  });
  runtimeTargetAssignmentRevisions.set(id, response.revision);
  return response;
}

export async function applyRuntimeTargetAssignmentBatch(
  targetIds: number[],
  userIds: number[],
  action: 'grant' | 'revoke',
) {
  type Request = NonNullable<RuntimeTargetAssignmentsBatchOperation['requestBody']>['content']['application/json'];
  type Response = NonNullable<
    RuntimeTargetAssignmentsBatchOperation['responses'][200]['content']['application/json']['data']
  >;
  return request.post<Response>({
    url: OPENAPI_RUNTIME_PATH.postRuntimeTargetAssignmentsBatch,
    data: { target_ids: targetIds, user_ids: userIds, action } satisfies Request,
  });
}

export async function getRuntimeTargetAssignmentsForTargets(targetIds: number[]): Promise<Map<number, Set<number>>> {
  const entries = await Promise.all(
    targetIds.map(async (targetId) => {
      const assignments = await getRuntimeTargetAssignments(targetId);
      return [targetId, new Set(assignments.items.map((item) => item.user_id))] as const;
    }),
  );
  return new Map(entries);
}
