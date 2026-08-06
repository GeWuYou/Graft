import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  BuildArtifactListResponse,
  BuildJobCreateRequest,
  BuildJobDetail,
  BuildJobListResponse,
} from '../types/build';

type ListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildJobs]['get'];
type DetailOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildJob]['get'];
type CreateOperation = paths[typeof OPENAPI_RUNTIME_PATH.postBuildJob]['post'];
type WorkspaceListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildWorkspaces]['get'];
type TargetListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildRuntimeTargets]['get'];
type PoolListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildBuilderPools]['get'];
type ArtifactListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildArtifacts]['get'];

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

export function getBuildWorkspaces() {
  return request.get<NonNullable<WorkspaceListOperation['responses'][200]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.getBuildWorkspaces,
  });
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

export function getBuildArtifacts(query?: ArtifactListOperation['parameters']['query']) {
  return request.get<NonNullable<ArtifactListOperation['responses'][200]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.getBuildArtifacts,
    params: query,
  }) as Promise<BuildArtifactListResponse>;
}
