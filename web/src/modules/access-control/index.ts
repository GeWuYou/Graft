// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { WebModuleRegistration } from '@/modules/types';

import { accessControlBootstrapRouteRegistrations } from './bootstrap-routes';

export const accessControlModuleRegistration: WebModuleRegistration = {
  moduleId: 'access-control',
  bootstrapRoutes: accessControlBootstrapRouteRegistrations,
};

export default accessControlModuleRegistration;
