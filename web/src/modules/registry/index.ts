import type { WebModuleRegistration } from '@/modules/types';

import { registryBootstrapRouteRegistrations, registryGlobalRouteRegistrations } from './bootstrap-routes';

export default {
  moduleId: 'registry',
  bootstrapRoutes: registryBootstrapRouteRegistrations,
  globalRoutes: registryGlobalRouteRegistrations,
} satisfies WebModuleRegistration;
