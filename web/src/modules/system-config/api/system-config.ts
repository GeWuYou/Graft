import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type { SystemConfigItem, SystemConfigListResponse, UpdateSystemConfigRequest } from '../types/system-config';

type SystemConfigListPath = typeof OPENAPI_RUNTIME_PATH.getSystemConfigs;
type GetSystemConfigsOperation = paths[SystemConfigListPath]['get'];
type GetSystemConfigsEnvelope = GetSystemConfigsOperation['responses'][200]['content']['application/json'];
type GetSystemConfigsData = NonNullable<GetSystemConfigsEnvelope['data']>;

type SystemConfigDetailPath = typeof OPENAPI_RUNTIME_PATH.getSystemConfig;
type GetSystemConfigOperation = paths[SystemConfigDetailPath]['get'];
type GetSystemConfigEnvelope = GetSystemConfigOperation['responses'][200]['content']['application/json'];
type GetSystemConfigData = NonNullable<GetSystemConfigEnvelope['data']>;
type GetSystemConfigPathParams = GetSystemConfigOperation['parameters']['path'];

type PutSystemConfigOperation = paths[SystemConfigDetailPath]['put'];
type PutSystemConfigEnvelope = PutSystemConfigOperation['responses'][200]['content']['application/json'];
type PutSystemConfigData = NonNullable<PutSystemConfigEnvelope['data']>;
type PutSystemConfigPathParams = PutSystemConfigOperation['parameters']['path'];
type PutSystemConfigBody = PutSystemConfigOperation['requestBody']['content']['application/json'];

type SystemConfigResetPath = typeof OPENAPI_RUNTIME_PATH.postSystemConfigReset;
type PostSystemConfigResetOperation = paths[SystemConfigResetPath]['post'];
type PostSystemConfigResetEnvelope = PostSystemConfigResetOperation['responses'][200]['content']['application/json'];
type PostSystemConfigResetData = NonNullable<PostSystemConfigResetEnvelope['data']>;
type PostSystemConfigResetPathParams = PostSystemConfigResetOperation['parameters']['path'];

export function getSystemConfigs() {
  return request.get<GetSystemConfigsData>({
    url: OPENAPI_RUNTIME_PATH.getSystemConfigs,
  }) as Promise<SystemConfigListResponse>;
}

export function getSystemConfig(key: GetSystemConfigPathParams['key']) {
  return request.get<GetSystemConfigData>({
    url: buildOpenApiRuntimePath('getSystemConfig', { key }),
  }) as Promise<SystemConfigItem>;
}

export function updateSystemConfig(key: PutSystemConfigPathParams['key'], payload: UpdateSystemConfigRequest) {
  return request.put<PutSystemConfigData>({
    url: buildOpenApiRuntimePath('putSystemConfig', { key }),
    data: payload as PutSystemConfigBody,
  }) as Promise<SystemConfigItem>;
}

export function resetSystemConfig(key: PostSystemConfigResetPathParams['key']) {
  return request.post<PostSystemConfigResetData>({
    url: buildOpenApiRuntimePath('postSystemConfigReset', { key }),
  }) as Promise<SystemConfigItem>;
}
