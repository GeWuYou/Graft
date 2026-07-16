import type { WebModuleRegistration } from '@/modules/types';

import { accessLogBootstrapRouteRegistrations } from './bootstrap-routes';
import { ACCESS_LOG_PERMISSION_CODE } from './contract/permissions';

/** 访问日志模块只向壳层暴露动态路由和权限码，页面实现仍由模块内部持有。 */
export const accessLogModuleRegistration: WebModuleRegistration = {
  moduleId: 'access-log',
  bootstrapRoutes: accessLogBootstrapRouteRegistrations,
};

export const accessLogModulePermissionCodes = ACCESS_LOG_PERMISSION_CODE;

export default accessLogModuleRegistration;
