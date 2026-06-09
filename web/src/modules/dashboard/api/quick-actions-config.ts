// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { components } from '@/contracts/openapi/generated/schema';
import { SYSTEM_CONFIG_API_PATH } from '@/modules/system-config/contract/paths';
import { request } from '@/utils/request';

export type DashboardSystemConfigItem = components['schemas']['system-config-item'];
type DashboardSystemConfigListResponse = components['schemas']['system-config-list-response'];

export function getDashboardSystemConfigs() {
  return request.get<DashboardSystemConfigListResponse>({
    url: SYSTEM_CONFIG_API_PATH.LIST,
  }) as Promise<DashboardSystemConfigListResponse>;
}
