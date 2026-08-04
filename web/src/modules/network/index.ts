import type { WebModuleRegistration } from '@/modules/types';

import { networkBootstrapRouteRegistrations, networkGlobalRouteRegistrations } from './bootstrap-routes';

export const networkModuleRegistration: WebModuleRegistration = {
  moduleId: 'network',
  bootstrapRoutes: networkBootstrapRouteRegistrations,
  globalRoutes: networkGlobalRouteRegistrations,
};

export default networkModuleRegistration;
