import type { WebModuleRegistration } from '@/modules/types';

import { runtimeTargetBootstrapRouteRegistrations, runtimeTargetGlobalRouteRegistrations } from './bootstrap-routes';

// 运行时目标同时拥有菜单列表和菜单外详情路由，二者都通过模块注册面交给壳层装配。
export default {
  moduleId: 'runtime-target',
  bootstrapRoutes: runtimeTargetBootstrapRouteRegistrations,
  globalRoutes: runtimeTargetGlobalRouteRegistrations,
} satisfies WebModuleRegistration;
