import type { WebModuleRegistration } from '@/modules/types';

import { registryBootstrapRouteRegistrations } from './bootstrap-routes';

export default {
  moduleId: 'registry',
  bootstrapRoutes: registryBootstrapRouteRegistrations,
} satisfies WebModuleRegistration;
