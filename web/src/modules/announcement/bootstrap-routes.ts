import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { ANNOUNCEMENT_BOOTSTRAP_ROUTE } from './contract/bootstrap';
import { ANNOUNCEMENT_ROUTE_PATH } from './contract/paths';

const managementRouteTitle = localizeRouteTitleKey('announcement.route.management.title');
const managementBreadcrumbTitle = localizeRouteTitleKey('announcement.route.management.breadcrumb');
const userRouteTitle = localizeRouteTitleKey('announcement.route.user.title');

/** 公告管理页进入平台菜单；用户公告中心作为菜单外全局路由挂到管理页的导航层级下。 */
export const announcementBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...ANNOUNCEMENT_BOOTSTRAP_ROUTE.MANAGEMENT,
    loadPage: () => import('./pages/management/index.vue'),
    meta: {
      pageKind: 'list',
      semanticTitle: managementRouteTitle,
      breadcrumbTitle: managementBreadcrumbTitle,
      tabGroup: 'platform',
    },
  },
];

export const announcementGlobalRouteRegistrations: GlobalRouteRegistration[] = [
  {
    path: ANNOUNCEMENT_ROUTE_PATH.USER_LIST,
    routeName: ANNOUNCEMENT_BOOTSTRAP_ROUTE.USER_LIST.routeName,
    navigationParentPath: ANNOUNCEMENT_BOOTSTRAP_ROUTE.MANAGEMENT.menuPath,
    loadPage: () => import('./pages/user-list/index.vue'),
    meta: {
      hiddenMenu: true,
      keepAlive: true,
      pageKind: 'list',
      semanticTitle: userRouteTitle,
      tabGroup: 'announcement',
      tabTitle: userRouteTitle,
      title: userRouteTitle,
      titleKey: 'announcement.route.user.title',
    },
  },
];
