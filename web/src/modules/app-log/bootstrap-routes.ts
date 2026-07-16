import type { BootstrapRouteRegistration } from '@/modules/types';

import { APP_LOG_BOOTSTRAP_ROUTE } from './contract/bootstrap';

/** 将应用日志列表声明为 bootstrap 动态路由，页面组件不由路由壳层直接引用。 */
export const appLogBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...APP_LOG_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'app-log',
      pageKind: 'list',
      titleKey: 'menu.appLog.title',
    },
  },
];
