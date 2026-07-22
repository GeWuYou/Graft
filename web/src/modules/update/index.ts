import type { WebModuleRegistration } from '@/modules/types';

import { updateBootstrapRouteRegistrations } from './bootstrap-routes';
import UpdateNotification from './components/UpdateNotification.vue';
import UpdateProvider from './components/UpdateProvider.vue';
import UpdateVersionEntry from './components/UpdateVersionEntry.vue';
import { UPDATE_PERMISSION_CODE } from './contract/permissions';

export const updateModuleRegistration: WebModuleRegistration = {
  moduleId: 'platform-update',
  bootstrapRoutes: updateBootstrapRouteRegistrations,
};

export const updateModulePermissionCodes = UPDATE_PERMISSION_CODE;
export const updateNotification = UpdateNotification;
export const updateProvider = UpdateProvider;
export const updateVersionEntry = UpdateVersionEntry;

export default updateModuleRegistration;
