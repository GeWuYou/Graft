// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { WebModuleRegistration } from '@/modules/types';

import { appLogBootstrapRouteRegistrations } from './bootstrap-routes';
import { APP_LOG_PERMISSION_CODE } from './contract/permissions';

export const appLogModuleRegistration: WebModuleRegistration = {
  moduleId: 'app-log',
  bootstrapRoutes: appLogBootstrapRouteRegistrations,
};

export const appLogModulePermissionCodes = APP_LOG_PERMISSION_CODE;

export default appLogModuleRegistration;
