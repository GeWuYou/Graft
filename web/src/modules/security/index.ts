import type { WebModuleRegistration } from '@/modules/types';

import { securityBootstrapRouteRegistrations } from './bootstrap-routes';
import { SECURITY_PERMISSION_CODE } from './contract/permissions';

export const securityModuleRegistration: WebModuleRegistration = {
  moduleId: 'security',
  bootstrapRoutes: securityBootstrapRouteRegistrations,
};

export const securityModulePermissionCodes = SECURITY_PERMISSION_CODE;

export default securityModuleRegistration;
