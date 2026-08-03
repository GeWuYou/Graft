import type { WebModuleRegistration } from '@/modules/types';

import { networkBootstrapRouteRegistrations } from './bootstrap-routes';

export const networkModuleRegistration: WebModuleRegistration = {
  moduleId: 'network',
  bootstrapRoutes: networkBootstrapRouteRegistrations,
};

export default networkModuleRegistration;
