import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { ACCESS_CONTROL_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const overviewRouteTitle = localizeRouteTitleKey('accessControl.route.overview.title');
const overviewBreadcrumbTitle = localizeRouteTitleKey('accessControl.route.overview.breadcrumb');

export const accessControlBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...ACCESS_CONTROL_BOOTSTRAP_ROUTE.OVERVIEW,
    loadPage: () => import('./pages/overview/index.vue'),
    meta: {
      domain: 'rbac',
      tabGroup: 'rbac',
      dashboard: true,
      pageKind: 'overview',
      semanticTitle: overviewRouteTitle,
      breadcrumbTitle: overviewBreadcrumbTitle,
      tabTitle: overviewRouteTitle,
    },
  },
];
