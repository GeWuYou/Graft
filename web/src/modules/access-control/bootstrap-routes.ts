import type { BootstrapRouteRegistration } from '@/modules/types';

import { ACCESS_CONTROL_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const accessControlBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...ACCESS_CONTROL_BOOTSTRAP_ROUTE.OVERVIEW,
    loadPage: () => import('./pages/overview/index.vue'),
  },
];
