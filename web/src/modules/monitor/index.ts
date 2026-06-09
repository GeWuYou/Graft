// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { WebModuleRegistration } from '@/modules/types';

import { monitorBootstrapRouteRegistrations } from './bootstrap-routes';

export const monitorModuleRegistration: WebModuleRegistration = {
  moduleId: 'monitor',
  bootstrapRoutes: monitorBootstrapRouteRegistrations,
};

export default monitorModuleRegistration;
