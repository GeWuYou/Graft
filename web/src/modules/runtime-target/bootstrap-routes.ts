import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { RUNTIME_TARGET_BOOTSTRAP_ROUTE } from './contract/bootstrap';

/** 列表是导航入口，详情作为菜单外全局路由挂在同一父路径下。 */
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

export const runtimeTargetGlobalRouteRegistrations: GlobalRouteRegistration[] = [
  {
    ...RUNTIME_TARGET_BOOTSTRAP_ROUTE.DETAIL,
    navigationParentPath: RUNTIME_TARGET_BOOTSTRAP_ROUTE.LIST.menuPath,
    loadPage: () => import('./pages/detail/index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: false,
      pageKind: 'detail',
      pageSurface: 'overview-dashboard',
      semanticTitle: localizeRouteTitleKey('runtimeTarget.route.detail.title'),
      breadcrumbTitle: localizeRouteTitleKey('runtimeTarget.route.detail.title'),
      domainTitle: localizeRouteTitleKey('runtimeTarget.route.list.title'),
      tabGroup: 'infrastructure',
      tabTitle: localizeRouteTitleKey('runtimeTarget.route.detail.title'),
      title: localizeRouteTitleKey('runtimeTarget.route.detail.title'),
      titleKey: 'runtimeTarget.route.detail.title',
    },
  },
];
