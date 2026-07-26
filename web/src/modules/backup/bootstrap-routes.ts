import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { BACKUP_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const routeTitle = localizeRouteTitleKey('backup.route.list.title');
const breadcrumbTitle = localizeRouteTitleKey('backup.route.list.breadcrumb');

export const backupBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...BACKUP_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'platform',
      pageKind: 'list',
      pageSurface: 'form-detail',
      semanticTitle: routeTitle,
      breadcrumbTitle,
      tabTitle: routeTitle,
    },
  },
];
