import type { WebModuleRegistration } from '@/modules/types';

import { dashboardBootstrapRouteRegistrations } from './bootstrap-routes';

export const dashboardModuleRegistration: WebModuleRegistration = {
  bootstrapRoutes: dashboardBootstrapRouteRegistrations,
  moduleId: 'dashboard',
};

export default dashboardModuleRegistration;
