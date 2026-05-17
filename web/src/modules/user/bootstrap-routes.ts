import type { BootstrapRouteRegistration } from '@/modules/types';

export const userBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    menuPath: '/users',
    routeName: 'UserList',
    loadPage: () => import('./pages/index.vue'),
  },
];
