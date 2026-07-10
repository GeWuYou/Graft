import type { WebModuleRegistration } from '@/modules/types';

import { taskBootstrapRouteRegistrations, taskGlobalRouteRegistrations } from './bootstrap-routes';

export { default as TaskDetailDrawer } from './components/TaskDetailDrawer.vue';
export { default as TaskHistoryTable } from './components/TaskHistoryTable.vue';
export type { TaskReceipt } from './types/task';

const taskModuleRegistration: WebModuleRegistration = {
  moduleId: 'task',
  bootstrapRoutes: taskBootstrapRouteRegistrations,
  globalRoutes: taskGlobalRouteRegistrations,
};

export default taskModuleRegistration;
