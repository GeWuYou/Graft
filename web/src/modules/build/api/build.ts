import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type { BuildJobCreateRequest, BuildJobDetail, BuildJobListResponse } from '../types/build';

type ListOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildJobs]['get'];
type DetailOperation = paths[typeof OPENAPI_RUNTIME_PATH.getBuildJob]['get'];
type CreateOperation = paths[typeof OPENAPI_RUNTIME_PATH.postBuildJob]['post'];

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
