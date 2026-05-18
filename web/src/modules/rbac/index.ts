import type { WebModuleRegistration } from '@/modules/types';

import { rbacBootstrapRouteRegistrations } from './bootstrap-routes';

export const rbacModuleRegistration: WebModuleRegistration = {
  moduleId: 'rbac',
  bootstrapRoutes: rbacBootstrapRouteRegistrations,
};

export default rbacModuleRegistration;
