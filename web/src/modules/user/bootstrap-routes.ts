import type { BootstrapRouteRegistration } from '@/modules/types';
import { USER_ROUTE_PATH } from '@/modules/user/contract/paths';

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
