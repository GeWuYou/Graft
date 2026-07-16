import type { BootstrapRouteRegistration } from '@/modules/types';
import { USER_ROUTE_PATH } from '@/modules/user/contract/paths';

/** 用户管理页通过模块注册面接入动态菜单，权限指令仍由页面根据用户能力控制操作项。 */
export const userBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    menuPath: USER_ROUTE_PATH.LIST,
    routeName: 'UserList',
    loadPage: () => import('./pages/index.vue'),
    meta: {
      pageKind: 'list',
    },
  },
];
