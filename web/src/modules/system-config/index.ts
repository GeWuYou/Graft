// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { WebModuleRegistration } from '@/modules/types';

import { systemConfigBootstrapRouteRegistrations } from './bootstrap-routes';
import { SYSTEM_CONFIG_PERMISSION_CODE } from './contract/permissions';

export const systemConfigModuleRegistration: WebModuleRegistration = {
  moduleId: 'system-config',
  bootstrapRoutes: systemConfigBootstrapRouteRegistrations,
};

export const systemConfigModulePermissionCodes = SYSTEM_CONFIG_PERMISSION_CODE;

export default systemConfigModuleRegistration;
