import type { WebModuleRegistration } from '@/modules/types';

import { monitorBootstrapRouteRegistrations } from './bootstrap-routes';

export const monitorModuleRegistration: WebModuleRegistration = {
  moduleId: 'monitor',
  bootstrapRoutes: monitorBootstrapRouteRegistrations,
};

export default monitorModuleRegistration;
