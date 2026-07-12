import type { WebModuleRegistration } from '@/modules/types';

import { runtimeTargetBootstrapRouteRegistrations } from './bootstrap-routes';
export default {
  moduleId: 'runtime-target',
  bootstrapRoutes: runtimeTargetBootstrapRouteRegistrations,
} satisfies WebModuleRegistration;
