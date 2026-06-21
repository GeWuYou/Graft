import type { WebModuleRegistration } from '@/modules/types';

import { systemConfigBootstrapRouteRegistrations } from './bootstrap-routes';
import { SYSTEM_CONFIG_PERMISSION_CODE } from './contract/permissions';

export const systemConfigModuleRegistration: WebModuleRegistration = {
  moduleId: 'system-config',
  bootstrapRoutes: systemConfigBootstrapRouteRegistrations,
};

export const systemConfigModulePermissionCodes = SYSTEM_CONFIG_PERMISSION_CODE;

export default systemConfigModuleRegistration;
