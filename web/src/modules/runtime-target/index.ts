import type { WebModuleRegistration } from '@/modules/types';

import { runtimeTargetBootstrapRouteRegistrations, runtimeTargetGlobalRouteRegistrations } from './bootstrap-routes';
export default {
  moduleId: 'runtime-target',
  bootstrapRoutes: runtimeTargetBootstrapRouteRegistrations,
  globalRoutes: runtimeTargetGlobalRouteRegistrations,
} satisfies WebModuleRegistration;
