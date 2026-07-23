import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { UPDATE_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const routeTitle = localizeRouteTitleKey('update.route.center.title');
const breadcrumbTitle = localizeRouteTitleKey('update.route.center.breadcrumb');

export const updateBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...UPDATE_BOOTSTRAP_ROUTE.CENTER,
    loadPage: () => import('./pages/center/index.vue'),
    meta: {
      tabGroup: 'platform-update',
      semanticTitle: routeTitle,
      breadcrumbTitle,
      tabTitle: routeTitle,
    },
  },
];
