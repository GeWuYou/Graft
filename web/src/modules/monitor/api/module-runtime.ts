// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { buildModuleRuntimeDetailApiPath, MONITOR_API_PATH } from '../contract/paths';
import type { ModuleRuntimeItem, ModuleRuntimeSnapshot } from '../types/module-runtime';

type ModuleRuntimePath = (typeof MONITOR_API_PATH)['MODULE_RUNTIME'];
type GetModuleRuntimeOperation = paths[ModuleRuntimePath]['get'];
type GetModuleRuntimeEnvelope = GetModuleRuntimeOperation['responses'][200]['content']['application/json'];
type GetModuleRuntimeData = NonNullable<GetModuleRuntimeEnvelope['data']>;

type ModuleRuntimeDetailPath = (typeof MONITOR_API_PATH)['MODULE_RUNTIME_DETAIL'];
type GetModuleRuntimeDetailOperation = paths[ModuleRuntimeDetailPath]['get'];
type GetModuleRuntimeDetailEnvelope = GetModuleRuntimeDetailOperation['responses'][200]['content']['application/json'];
type GetModuleRuntimeDetailData = NonNullable<GetModuleRuntimeDetailEnvelope['data']>;
type GetModuleRuntimeDetailParams = GetModuleRuntimeDetailOperation['parameters']['path'];

export function getModuleRuntimeSnapshot() {
  return request.get<GetModuleRuntimeData>({
    url: MONITOR_API_PATH.MODULE_RUNTIME,
  }) as Promise<ModuleRuntimeSnapshot>;
}

export function getModuleRuntimeDetail(moduleKey: GetModuleRuntimeDetailParams['module_key']) {
  return request.get<GetModuleRuntimeDetailData>({
    url: buildModuleRuntimeDetailApiPath(moduleKey),
  }) as Promise<ModuleRuntimeItem>;
}
