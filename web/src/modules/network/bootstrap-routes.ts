import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { NETWORK_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const routeTitle = localizeRouteTitleKey('network.route.outbound.title');
const breadcrumbTitle = localizeRouteTitleKey('network.route.outbound.breadcrumb');

export const networkBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...NETWORK_BOOTSTRAP_ROUTE.OUTBOUND,
    loadPage: () => import('./pages/outbound/index.vue'),
    meta: {
      tabGroup: 'platform',
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: routeTitle,
      breadcrumbTitle,
      tabTitle: routeTitle,
    },
  },
  {
    ...NETWORK_BOOTSTRAP_ROUTE.CONNECTIVITY,
    loadPage: () => import('./pages/connectivity/index.vue'),
    meta: {
      tabGroup: 'platform',
      pageKind: 'list',
      pageSurface: 'overview-dashboard',
      semanticTitle: routeTitle,
      breadcrumbTitle,
      tabTitle: routeTitle,
    },
  },
  {
    ...NETWORK_BOOTSTRAP_ROUTE.CONNECTIVITY_DIAGNOSTICS,
    loadPage: () => import('./pages/connectivity/diagnostics.vue'),
    meta: {
      tabGroup: 'platform',
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: routeTitle,
      breadcrumbTitle,
      tabTitle: routeTitle,
      hidden: true,
    },
  },
];
