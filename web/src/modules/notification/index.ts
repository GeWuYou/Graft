// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { WebModuleRegistration } from '@/modules/types';

import { notificationBootstrapRouteRegistrations } from './bootstrap-routes';
import NotificationBellPanel from './components/NotificationBellPanel.vue';
import { NOTIFICATION_PERMISSION_CODE } from './contract/permissions';

export const notificationModuleRegistration: WebModuleRegistration = {
  moduleId: 'notification',
  bootstrapRoutes: notificationBootstrapRouteRegistrations,
};

export const notificationModulePermissionCodes = NOTIFICATION_PERMISSION_CODE;
export { NotificationBellPanel };

export default notificationModuleRegistration;
