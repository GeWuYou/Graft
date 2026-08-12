import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { REGISTRY_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const registryBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...REGISTRY_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      pageKind: 'list',
      pageSurface: 'form-detail',
      semanticTitle: localizeRouteTitleKey('registry.list.title'),
      breadcrumbTitle: localizeRouteTitleKey('registry.list.title'),
      tabGroup: 'infrastructure',
    },
  },
];
