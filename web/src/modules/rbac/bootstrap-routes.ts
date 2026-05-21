import type { BootstrapRouteRegistration } from '@/modules/types';

import { RBAC_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const rbacBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...RBAC_BOOTSTRAP_ROUTE.ROLE_LIST,
    loadPage: () => import('./pages/index.vue'),
  },
];
