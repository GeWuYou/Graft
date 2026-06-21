import type { BootstrapRouteRegistration } from '@/modules/types';

import { ACCESS_LOG_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const accessLogBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...ACCESS_LOG_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'access-log',
      pageKind: 'list',
      titleKey: 'menu.accessLog.title',
    },
  },
];
