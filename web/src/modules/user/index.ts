// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { WebModuleRegistration } from '@/modules/types';

import { userBootstrapRouteRegistrations } from './bootstrap-routes';

export const userModuleRegistration: WebModuleRegistration = {
  moduleId: 'user',
  bootstrapRoutes: userBootstrapRouteRegistrations,
};

export default userModuleRegistration;
