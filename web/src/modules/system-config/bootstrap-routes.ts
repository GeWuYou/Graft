import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { SYSTEM_CONFIG_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const listRouteTitle = localizeRouteTitleKey('systemConfig.route.list.title');
const listBreadcrumbTitle = localizeRouteTitleKey('systemConfig.route.list.breadcrumb');

export const systemConfigBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...SYSTEM_CONFIG_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'server',
      pageKind: 'list',
      pageSurface: 'form-detail',
      semanticTitle: listRouteTitle,
      breadcrumbTitle: listBreadcrumbTitle,
      tabTitle: listRouteTitle,
    },
  },
];
