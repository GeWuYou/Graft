import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { SECURITY_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const routeTitle = localizeRouteTitleKey('security.route.overview.title');
const breadcrumbTitle = localizeRouteTitleKey('security.route.overview.breadcrumb');

export const securityBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...SECURITY_BOOTSTRAP_ROUTE.OVERVIEW,
    loadPage: () => import('./pages/overview/index.vue'),
    meta: {
      domain: 'security',
      tabGroup: 'security-overview',
      dashboard: true,
      pageKind: 'overview',
      semanticTitle: routeTitle,
      breadcrumbTitle,
      tabTitle: routeTitle,
    },
  },
];
