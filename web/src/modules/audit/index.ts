import type { WebModuleRegistration } from '@/modules/types';

import { auditBootstrapRouteRegistrations } from './bootstrap-routes';
import { AUDIT_PERMISSION_CODE } from './contract/permissions';

export const auditModuleRegistration: WebModuleRegistration = {
  moduleId: 'audit',
  bootstrapRoutes: auditBootstrapRouteRegistrations,
};

// 通过模块注册面暴露权限契约，使权限与路由注册保持同一模块归属并可被静态治理发现。
export const auditModulePermissionCodes = AUDIT_PERMISSION_CODE;

export default auditModuleRegistration;
