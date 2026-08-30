import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type {
  BuildArtifactListResponse,
  BuildArtifactPromotionCreateRequest,
  BuildArtifactPublicationListResponse,
  BuildInputSnapshot,
  BuildJobCreateRequest,
  BuildJobDetail,
  BuildJobListResponse,
  BuildWorkspaceListResponse,
} from '../types/build';

type ListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildJobs]['get'];
type DetailOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildJob]['get'];
type CreateOperation = paths[typeof OPENAPI_RUNTIME_PATH.postBuildJob]['post'];
type InputSnapshotUploadOperation = paths[typeof OPENAPI_RUNTIME_PATH.postBuildInputSnapshot]['post'];
type InputSnapshotListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildInputSnapshots]['get'];
type TargetListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildRuntimeTargets]['get'];
type PoolListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildBuilderPools]['get'];
type ArtifactListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildArtifacts]['get'];
type ArtifactPublicationListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildArtifactPublications]['get'];
type PromotionOperation = paths[typeof OPENAPI_RUNTIME_PATH.postBuildArtifactPromotion]['post'];
type RegistryDestinationOperation = paths[typeof OPENAPI_RUNTIME_PATH.getRegistryAvailableDestinations]['get'];
type WorkspaceListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildWorkspaces]['get'];

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

export function uploadBuildInputSnapshot(file: File) {
  type ResponseData = NonNullable<
    InputSnapshotUploadOperation['responses'][201]['content']['application/json']['data']
  >;
  const data = new FormData();
  data.append('archive', file);
  return request.post<ResponseData>({
    url: OPENAPI_RUNTIME_PATH.postBuildInputSnapshot,
    data,
  }) as Promise<BuildInputSnapshot>;
}

export function getBuildInputSnapshots(query?: InputSnapshotListOperation['parameters']['query']) {
  return request.get<NonNullable<InputSnapshotListOperation['responses'][200]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.getBuildInputSnapshots,
    params: query,
  });
}

export function getBuildWorkspaces(query?: WorkspaceListOperation['parameters']['query']) {
  return request.get<NonNullable<WorkspaceListOperation['responses'][200]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.getBuildWorkspaces,
    params: query,
  }) as Promise<BuildWorkspaceListResponse>;
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

export function getBuildArtifactPublications(artifactId: string) {
  type ResponseData = NonNullable<
    ArtifactPublicationListOperation['responses'][200]['content']['application/json']['data']
  >;
  return request.get<ResponseData>({
    url: buildOpenApiRuntimePath('getBuildArtifactPublications', { artifactId }),
  }) as Promise<BuildArtifactPublicationListResponse>;
}

/** 创建制品 Promotion 请求；调用方必须提供幂等键，以避免重试造成重复提交。 */
export function createArtifactPromotion(payload: BuildArtifactPromotionCreateRequest, idempotencyKey: string) {
  return request.post<NonNullable<PromotionOperation['responses'][202]['content']['application/json']['data']>>({
    url: OPENAPI_RUNTIME_PATH.postBuildArtifactPromotion,
    data: payload,
    headers: { 'Idempotency-Key': idempotencyKey },
  });
}
