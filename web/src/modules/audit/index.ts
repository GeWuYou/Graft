// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { WebModuleRegistration } from '@/modules/types';

import { auditBootstrapRouteRegistrations } from './bootstrap-routes';
import { AUDIT_PERMISSION_CODE } from './contract/permissions';

export const auditModuleRegistration: WebModuleRegistration = {
  moduleId: 'audit',
  bootstrapRoutes: auditBootstrapRouteRegistrations,
};

// Expose the module-owned permission contract through the module boundary so it stays
// discoverable to static governance checks alongside the route registration surface.
export const auditModulePermissionCodes = AUDIT_PERMISSION_CODE;

export default auditModuleRegistration;
