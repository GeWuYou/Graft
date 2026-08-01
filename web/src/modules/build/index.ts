import type { WebModuleRegistration } from '@/modules/types';

import { buildBootstrapRouteRegistrations, buildGlobalRouteRegistrations } from './bootstrap-routes';
export { buildCreateJobPath } from './contract/navigation';
import { BUILD_PERMISSION_CODE } from './contract/permissions';
export { BUILD_TASK_TYPE } from './contract/task-types';

export const buildModuleRegistration: WebModuleRegistration = {
  moduleId: 'build',
  bootstrapRoutes: buildBootstrapRouteRegistrations,
  globalRoutes: buildGlobalRouteRegistrations,
};

export const buildModulePermissionCodes = BUILD_PERMISSION_CODE;

export default buildModuleRegistration;
