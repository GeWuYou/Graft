// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import {
  buildSystemConfigDetailApiPath,
  buildSystemConfigResetApiPath,
  SYSTEM_CONFIG_API_PATH,
} from '../contract/paths';
import type { SystemConfigItem, SystemConfigListResponse, UpdateSystemConfigRequest } from '../types/system-config';

type SystemConfigListPath = (typeof SYSTEM_CONFIG_API_PATH)['LIST'];
type GetSystemConfigsOperation = paths[SystemConfigListPath]['get'];
type GetSystemConfigsEnvelope = GetSystemConfigsOperation['responses'][200]['content']['application/json'];
type GetSystemConfigsData = NonNullable<GetSystemConfigsEnvelope['data']>;

type SystemConfigDetailPath = (typeof SYSTEM_CONFIG_API_PATH)['DETAIL'];
type GetSystemConfigOperation = paths[SystemConfigDetailPath]['get'];
type GetSystemConfigEnvelope = GetSystemConfigOperation['responses'][200]['content']['application/json'];
type GetSystemConfigData = NonNullable<GetSystemConfigEnvelope['data']>;
type GetSystemConfigPathParams = GetSystemConfigOperation['parameters']['path'];

type PutSystemConfigOperation = paths[SystemConfigDetailPath]['put'];
type PutSystemConfigEnvelope = PutSystemConfigOperation['responses'][200]['content']['application/json'];
type PutSystemConfigData = NonNullable<PutSystemConfigEnvelope['data']>;
type PutSystemConfigPathParams = PutSystemConfigOperation['parameters']['path'];
type PutSystemConfigBody = PutSystemConfigOperation['requestBody']['content']['application/json'];

type SystemConfigResetPath = (typeof SYSTEM_CONFIG_API_PATH)['RESET'];
type PostSystemConfigResetOperation = paths[SystemConfigResetPath]['post'];
type PostSystemConfigResetEnvelope = PostSystemConfigResetOperation['responses'][200]['content']['application/json'];
type PostSystemConfigResetData = NonNullable<PostSystemConfigResetEnvelope['data']>;
type PostSystemConfigResetPathParams = PostSystemConfigResetOperation['parameters']['path'];

export function getSystemConfigs() {
  return request.get<GetSystemConfigsData>({
    url: SYSTEM_CONFIG_API_PATH.LIST,
  }) as Promise<SystemConfigListResponse>;
}

export function getSystemConfig(key: GetSystemConfigPathParams['key']) {
  return request.get<GetSystemConfigData>({
    url: buildSystemConfigDetailApiPath(key),
  }) as Promise<SystemConfigItem>;
}

export function updateSystemConfig(key: PutSystemConfigPathParams['key'], payload: UpdateSystemConfigRequest) {
  return request.put<PutSystemConfigData>({
    url: buildSystemConfigDetailApiPath(key),
    data: payload as PutSystemConfigBody,
  }) as Promise<SystemConfigItem>;
}

export function resetSystemConfig(key: PostSystemConfigResetPathParams['key']) {
  return request.post<PostSystemConfigResetData>({
    url: buildSystemConfigResetApiPath(key),
  }) as Promise<SystemConfigItem>;
}
