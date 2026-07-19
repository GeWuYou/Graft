import { CONTAINER_PERMISSION_CODE } from '@/contracts/generated/modules/container';
import type { WebModuleRegistration } from '@/modules/types';

import { getContainerDashboardSummary } from './api/dashboard-summary';
import { containerBootstrapRouteRegistrations, containerGlobalRouteRegistrations } from './bootstrap-routes';

export const containerModuleRegistration: WebModuleRegistration = {
  moduleId: 'container',
  bootstrapRoutes: containerBootstrapRouteRegistrations,
  globalRoutes: containerGlobalRouteRegistrations,
};

export const containerModulePermissionCodes = CONTAINER_PERMISSION_CODE;
export const containerModuleFacades = {
  getContainerDashboardSummary,
};

export default containerModuleRegistration;
