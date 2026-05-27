import type { BootstrapRouteRegistration } from '@/modules/types';

import { AUDIT_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const auditBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...AUDIT_BOOTSTRAP_ROUTE.LOG_LIST,
    loadPage: () => import('./pages/index.vue'),
  },
];
