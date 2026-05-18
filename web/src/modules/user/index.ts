import type { WebModuleRegistration } from '@/modules/types';

import { userBootstrapRouteRegistrations } from './bootstrap-routes';

export const userModuleRegistration: WebModuleRegistration = {
  moduleId: 'user',
  bootstrapRoutes: userBootstrapRouteRegistrations,
};

export default userModuleRegistration;
