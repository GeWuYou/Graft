import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  BuildArtifactListResponse,
  BuildArtifactPromotionCreateRequest,
  BuildJobCreateRequest,
  BuildJobDetail,
  BuildJobListResponse,
  BuildWorkspace,
  BuildWorkspaceCreateRequest,
} from '../types/build';

type ListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildJobs]['get'];
type DetailOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildJob]['get'];
type CreateOperation = paths[typeof OPENAPI_RUNTIME_PATH.postBuildJob]['post'];
type WorkspaceListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildWorkspaces]['get'];
type WorkspaceCreateOperation = paths[typeof OPENAPI_RUNTIME_PATH.postBuildWorkspace]['post'];
type TargetListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildRuntimeTargets]['get'];
type PoolListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildBuilderPools]['get'];
type ArtifactListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildArtifacts]['get'];
type PromotionOperation = paths[typeof OPENAPI_RUNTIME_PATH.postBuildArtifactPromotion]['post'];
type RegistryDestinationOperation = paths[typeof OPENAPI_RUNTIME_PATH.getRegistryAvailableDestinations]['get'];

export function getBuildJobs(query?: ListOperation['parameters']['query']) {
  return request.get<NonNullable<ListOperation['responses'][200]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.getBuildJobs,
    params: query,
  }) as Promise<BuildJobListResponse>;
}

export function getBuildJob(buildId: DetailOperation['parameters']['path']['buildId']) {
  return request.get<NonNullable<DetailOperation['responses'][200]['content']['application/json']['data']>>({
    url: buildOpenApiRuntimePath('getBuildJob', { buildId }),
  }) as Promise<BuildJobDetail>;
}

export function createBuildJob(payload: BuildJobCreateRequest, idempotencyKey: string) {
  return request.post<NonNullable<CreateOperation['responses'][202]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.postBuildJob,
    data: payload,
    headers: { 'Idempotency-Key': idempotencyKey },
  });
}

export function getBuildWorkspaces(query?: WorkspaceListOperation['parameters']['query']) {
  return request.get<NonNullable<WorkspaceListOperation['responses'][200]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.getBuildWorkspaces,
    params: query,
  });
}

export function createBuildWorkspace(payload: BuildWorkspaceCreateRequest) {
  type ResponseData = NonNullable<WorkspaceCreateOperation['responses'][201]['content']['application/json']['data']>;
  return request.post<ResponseData>({
    url: OPENAPI_RUNTIME_PATH.postBuildWorkspace,
    data: payload,
  }) as Promise<BuildWorkspace>;
}

export function getBuildRuntimeTargets() {
  return request.get<NonNullable<TargetListOperation['responses'][200]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.getBuildRuntimeTargets,
  });
}

export function getBuildBuilderPools() {
  return request.get<NonNullable<PoolListOperation['responses'][200]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.getBuildBuilderPools,
  });
}

export function getBuildRegistryDestinations() {
  return request.get<
    NonNullable<RegistryDestinationOperation['responses'][200]['content']['application/json']['data']>
  >({
    url: OPENAPI_RUNTIME_PATH.getRegistryAvailableDestinations,
  });
}

export function getBuildArtifacts(query?: ArtifactListOperation['parameters']['query']) {
  return request.get<NonNullable<ArtifactListOperation['responses'][200]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.getBuildArtifacts,
    params: query,
  }) as Promise<BuildArtifactListResponse>;
}

/** @public Promotion 写入契约先由 API 客户端提供，Publication 发现与页面工作流另行接入。 */
export function createArtifactPromotion(payload: BuildArtifactPromotionCreateRequest, idempotencyKey: string) {
  return request.post<NonNullable<PromotionOperation['responses'][202]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.postBuildArtifactPromotion,
    data: payload,
    headers: { 'Idempotency-Key': idempotencyKey },
  });
}
