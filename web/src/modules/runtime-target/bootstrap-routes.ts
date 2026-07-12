import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { RUNTIME_TARGET_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const runtimeTargetBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...RUNTIME_TARGET_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      pageKind: 'list',
      semanticTitle: localizeRouteTitleKey('runtimeTarget.route.list.title'),
      breadcrumbTitle: localizeRouteTitleKey('runtimeTarget.route.list.title'),
      tabGroup: 'infrastructure',
    },
  },
];
