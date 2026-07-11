import type { WebModuleRegistration } from '@/modules/types';

import { taskBootstrapRouteRegistrations, taskGlobalRouteRegistrations } from './bootstrap-routes';

export type { TaskReceipt } from './types/task';

const taskModuleRegistration: WebModuleRegistration = {
  moduleId: 'task',
  bootstrapRoutes: taskBootstrapRouteRegistrations,
  globalRoutes: taskGlobalRouteRegistrations,
};

export default taskModuleRegistration;
