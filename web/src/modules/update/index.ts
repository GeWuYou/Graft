import type { WebModuleRegistration } from '@/modules/types';

import { updateBootstrapRouteRegistrations } from './bootstrap-routes';
import UpdateVersionEntry from './components/UpdateVersionEntry.vue';
import { UPDATE_PERMISSION_CODE } from './contract/permissions';

export const updateModuleRegistration: WebModuleRegistration = {
  moduleId: 'platform-update',
  bootstrapRoutes: updateBootstrapRouteRegistrations,
};

export const updateModulePermissionCodes = UPDATE_PERMISSION_CODE;
export const updateVersionEntry = UpdateVersionEntry;

export default updateModuleRegistration;
