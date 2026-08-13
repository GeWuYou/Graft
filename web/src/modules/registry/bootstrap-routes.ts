import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { REGISTRY_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const registryBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...REGISTRY_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      pageKind: 'list',
      pageSurface: 'paged-table',
      semanticTitle: localizeRouteTitleKey('menu.registries.title'),
      breadcrumbTitle: localizeRouteTitleKey('menu.registries.title'),
      tabGroup: 'infrastructure',
    },
  },
];

export const registryGlobalRouteRegistrations: GlobalRouteRegistration[] = [
  {
    ...REGISTRY_BOOTSTRAP_ROUTE.DETAIL,
    navigationParentPath: REGISTRY_BOOTSTRAP_ROUTE.LIST.menuPath,
    loadPage: () => import('./pages/detail/index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: false,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: localizeRouteTitleKey('registry.route.detail.title'),
      breadcrumbTitle: localizeRouteTitleKey('registry.route.detail.title'),
      domainTitle: localizeRouteTitleKey('menu.registries.title'),
      tabGroup: 'infrastructure',
      tabTitle: localizeRouteTitleKey('registry.route.detail.title'),
      title: localizeRouteTitleKey('registry.route.detail.title'),
      titleKey: 'registry.route.detail.title',
    },
  },
];
