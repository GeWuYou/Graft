import type { WebModuleRegistration } from '@/modules/types';

import { notificationBootstrapRouteRegistrations, notificationGlobalRouteRegistrations } from './bootstrap-routes';
import NotificationBellPanel from './components/NotificationBellPanel.vue';
import { NOTIFICATION_PERMISSION_CODE } from './contract/permissions';

export const notificationModuleRegistration: WebModuleRegistration = {
  moduleId: 'notification',
  bootstrapRoutes: notificationBootstrapRouteRegistrations,
  globalRoutes: notificationGlobalRouteRegistrations,
};

export const notificationModulePermissionCodes = NOTIFICATION_PERMISSION_CODE;
export const notificationHeaderWidget = NotificationBellPanel;

export default notificationModuleRegistration;
