import type { BootstrapRouteRegistration } from '@/modules/types';

import { AUDIT_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const auditBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...AUDIT_BOOTSTRAP_ROUTE.OVERVIEW,
    loadPage: () => import('./pages/overview/index.vue'),
  },
  {
    ...AUDIT_BOOTSTRAP_ROUTE.LOG_LIST,
    loadPage: () => import('./pages/logs/index.vue'),
  },
];
