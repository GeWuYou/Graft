import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { NETWORK_BOOTSTRAP_ROUTE, NETWORK_GLOBAL_ROUTE } from './contract/bootstrap';

const outboundRouteTitle = localizeRouteTitleKey('network.outbound.title');
const connectivityRouteTitle = localizeRouteTitleKey('network.outbound.connectivity.title');
const diagnosticsRouteTitle = localizeRouteTitleKey('network.outbound.connectivity.diagnosticsTitle');

export const networkBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...NETWORK_BOOTSTRAP_ROUTE.CONNECTIVITY,
    loadPage: () => import('./pages/connectivity/index.vue'),
    meta: {
      tabGroup: 'platform',
      pageKind: 'list',
      pageSurface: 'overview-dashboard',
      semanticTitle: connectivityRouteTitle,
      breadcrumbTitle: connectivityRouteTitle,
      tabTitle: connectivityRouteTitle,
    },
  },
];

export const networkGlobalRouteRegistrations: GlobalRouteRegistration[] = [
  {
    ...NETWORK_GLOBAL_ROUTE.OUTBOUND,
    navigationParentPath: NETWORK_BOOTSTRAP_ROUTE.CONNECTIVITY.menuPath,
    loadPage: () => import('./pages/outbound/index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: outboundRouteTitle,
      breadcrumbTitle: outboundRouteTitle,
      tabGroup: 'platform',
      tabTitle: outboundRouteTitle,
    },
  },
  {
    ...NETWORK_GLOBAL_ROUTE.CONNECTIVITY_DIAGNOSTICS,
    navigationParentPath: NETWORK_BOOTSTRAP_ROUTE.CONNECTIVITY.menuPath,
    loadPage: () => import('./pages/connectivity/diagnostics.vue'),
    meta: {
      tabGroup: 'platform',
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: diagnosticsRouteTitle,
      breadcrumbTitle: diagnosticsRouteTitle,
      tabTitle: diagnosticsRouteTitle,
      hidden: false,
      hiddenMenu: true,
    },
  },
];
