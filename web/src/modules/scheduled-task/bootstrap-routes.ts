import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { SCHEDULED_TASK_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const listRouteTitle = localizeRouteTitleKey('scheduledTask.route.list.title');
const listBreadcrumbTitle = localizeRouteTitleKey('scheduledTask.route.list.breadcrumb');

export const scheduledTaskBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...SCHEDULED_TASK_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'server',
      pageKind: 'list',
      semanticTitle: listRouteTitle,
      breadcrumbTitle: listBreadcrumbTitle,
      tabTitle: listRouteTitle,
    },
  },
];
