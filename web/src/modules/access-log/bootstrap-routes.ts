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
      semanticTitle: {
        'zh-CN': '日志中心 - 访问日志',
        'en-US': 'Log Center - Access Logs',
      },
      breadcrumbTitle: {
        'zh-CN': '访问日志',
        'en-US': 'Access Logs',
      },
      tabTitle: {
        'zh-CN': '日志中心 - 访问日志',
        'en-US': 'Log Center - Access Logs',
      },
    },
  },
];
