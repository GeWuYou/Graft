import type { WebModuleRegistration } from '@/modules/types';

import { accessControlBootstrapRouteRegistrations } from './bootstrap-routes';

export const accessControlModuleRegistration: WebModuleRegistration = {
  moduleId: 'access-control',
  bootstrapRoutes: accessControlBootstrapRouteRegistrations,
};

export default accessControlModuleRegistration;
