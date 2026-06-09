// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { WebModuleRegistration } from '@/modules/types';

import { scheduledTaskBootstrapRouteRegistrations } from './bootstrap-routes';

export const scheduledTaskModuleRegistration: WebModuleRegistration = {
  moduleId: 'scheduled-task',
  bootstrapRoutes: scheduledTaskBootstrapRouteRegistrations,
};

export default scheduledTaskModuleRegistration;
