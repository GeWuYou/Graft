import type { WebModuleRegistration } from '@/modules/types';

import { applicationBootstrapRouteRegistrations, applicationGlobalRouteRegistrations } from './bootstrap-routes';

export const applicationModuleRegistration: WebModuleRegistration = {
  moduleId: 'project',
  bootstrapRoutes: applicationBootstrapRouteRegistrations,
  globalRoutes: applicationGlobalRouteRegistrations,
};

export default applicationModuleRegistration;
