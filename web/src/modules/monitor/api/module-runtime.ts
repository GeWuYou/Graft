import { buildOpenApiRuntimePath, OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type { ModuleRuntimeItem, ModuleRuntimeSnapshot } from '../types/module-runtime';

type ModuleRuntimePath = typeof OPENAPI_RUNTIME_PATH.getModulesRuntime;
type GetModuleRuntimeOperation = paths[ModuleRuntimePath]['get'];
type GetModuleRuntimeEnvelope = GetModuleRuntimeOperation['responses'][200]['content']['application/json'];
type GetModuleRuntimeData = NonNullable<GetModuleRuntimeEnvelope['data']>;

type ModuleRuntimeDetailPath = typeof OPENAPI_RUNTIME_PATH.getModulesRuntimeModule;
type GetModuleRuntimeDetailOperation = paths[ModuleRuntimeDetailPath]['get'];
type GetModuleRuntimeDetailEnvelope = GetModuleRuntimeDetailOperation['responses'][200]['content']['application/json'];
type GetModuleRuntimeDetailData = NonNullable<GetModuleRuntimeDetailEnvelope['data']>;
type GetModuleRuntimeDetailParams = GetModuleRuntimeDetailOperation['parameters']['path'];

export function getModuleRuntimeSnapshot() {
  return request.get<GetModuleRuntimeData>({
    url: OPENAPI_RUNTIME_PATH.getModulesRuntime,
  }) as Promise<ModuleRuntimeSnapshot>;
}

export function getModuleRuntimeDetail(moduleKey: GetModuleRuntimeDetailParams['module_key']) {
  return request.get<GetModuleRuntimeDetailData>({
    url: buildOpenApiRuntimePath('getModulesRuntimeModule', { module_key: moduleKey }),
  }) as Promise<ModuleRuntimeItem>;
}
