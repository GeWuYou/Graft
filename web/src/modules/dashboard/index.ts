// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { WebModuleRegistration } from '@/modules/types';

import { dashboardBootstrapRouteRegistrations } from './bootstrap-routes';

export const dashboardModuleRegistration: WebModuleRegistration = {
  bootstrapRoutes: dashboardBootstrapRouteRegistrations,
  moduleId: 'dashboard',
};

export default dashboardModuleRegistration;
