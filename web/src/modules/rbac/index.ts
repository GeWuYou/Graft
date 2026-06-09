// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { WebModuleRegistration } from '@/modules/types';

import { rbacBootstrapRouteRegistrations } from './bootstrap-routes';

export const rbacModuleRegistration: WebModuleRegistration = {
  moduleId: 'rbac',
  bootstrapRoutes: rbacBootstrapRouteRegistrations,
};

export default rbacModuleRegistration;
