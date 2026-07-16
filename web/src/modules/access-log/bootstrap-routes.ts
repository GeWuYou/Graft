import type { BootstrapRouteRegistration } from '@/modules/types';

import { ACCESS_LOG_BOOTSTRAP_ROUTE } from './contract/bootstrap';

/** 将访问日志列表接入 bootstrap 驱动的动态路由，不在壳层维护页面白名单。 */
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
