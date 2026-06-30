import type { WebModuleRegistration } from '@/modules/types';

import { projectBootstrapRouteRegistrations, projectGlobalRouteRegistrations } from './bootstrap-routes';

export const projectModuleRegistration: WebModuleRegistration = {
  moduleId: 'project',
  bootstrapRoutes: projectBootstrapRouteRegistrations,
  globalRoutes: projectGlobalRouteRegistrations,
};

export default projectModuleRegistration;
