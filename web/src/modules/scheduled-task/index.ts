import type { WebModuleRegistration } from '@/modules/types';

import { scheduledTaskBootstrapRouteRegistrations } from './bootstrap-routes';

export const scheduledTaskModuleRegistration: WebModuleRegistration = {
  moduleId: 'scheduled-task',
  bootstrapRoutes: scheduledTaskBootstrapRouteRegistrations,
};

export default scheduledTaskModuleRegistration;
