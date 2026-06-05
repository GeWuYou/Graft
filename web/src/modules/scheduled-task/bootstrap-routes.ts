import type { BootstrapRouteRegistration } from '@/modules/types';

import { SCHEDULED_TASK_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const scheduledTaskBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...SCHEDULED_TASK_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'server',
      pageKind: 'list',
      semanticTitle: {
        'zh-CN': '服务管理 - 定时任务',
        'en-US': 'Service Management - Scheduled Tasks',
      },
      breadcrumbTitle: {
        'zh-CN': '定时任务',
        'en-US': 'Scheduled Tasks',
      },
      tabTitle: {
        'zh-CN': '服务管理 - 定时任务',
        'en-US': 'Service Management - Scheduled Tasks',
      },
    },
  },
];
