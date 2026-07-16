import type { WebModuleRegistration } from '@/modules/types';

import { appLogBootstrapRouteRegistrations } from './bootstrap-routes';
import { APP_LOG_PERMISSION_CODE } from './contract/permissions';

/** 应用日志模块通过注册面接入壳层，删除和查询能力仍受模块权限与 API 边界约束。 */
export const appLogModuleRegistration: WebModuleRegistration = {
  moduleId: 'app-log',
  bootstrapRoutes: appLogBootstrapRouteRegistrations,
};

export const appLogModulePermissionCodes = APP_LOG_PERMISSION_CODE;

export default appLogModuleRegistration;
