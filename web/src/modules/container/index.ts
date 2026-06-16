// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { WebModuleRegistration } from '@/modules/types';

import { containerBootstrapRouteRegistrations, containerGlobalRouteRegistrations } from './bootstrap-routes';
import { CONTAINER_PERMISSION_CODE } from './contract/permissions';

export const containerModuleRegistration: WebModuleRegistration = {
  moduleId: 'container',
  bootstrapRoutes: containerBootstrapRouteRegistrations,
  globalRoutes: containerGlobalRouteRegistrations,
};

export const containerModulePermissionCodes = CONTAINER_PERMISSION_CODE;

export default containerModuleRegistration;
