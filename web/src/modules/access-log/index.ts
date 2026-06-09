// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { WebModuleRegistration } from '@/modules/types';

import { accessLogBootstrapRouteRegistrations } from './bootstrap-routes';
import { ACCESS_LOG_PERMISSION_CODE } from './contract/permissions';

export const accessLogModuleRegistration: WebModuleRegistration = {
  moduleId: 'access-log',
  bootstrapRoutes: accessLogBootstrapRouteRegistrations,
};

export const accessLogModulePermissionCodes = ACCESS_LOG_PERMISSION_CODE;

export default accessLogModuleRegistration;
