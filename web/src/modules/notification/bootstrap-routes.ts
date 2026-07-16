import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { NOTIFICATION_BOOTSTRAP_ROUTE } from './contract/bootstrap';
import { NOTIFICATION_ROUTE_PATH } from './contract/paths';

const notificationRouteTitle = localizeRouteTitleKey('menu.notification.title');

/** 通知中心不占用侧边菜单，作为壳层可打开的全局路由保留 Tab 和历史导航语义。 */
export const notificationBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [];

export const notificationGlobalRouteRegistrations: GlobalRouteRegistration[] = [
  {
    path: NOTIFICATION_ROUTE_PATH.LIST,
    routeName: NOTIFICATION_BOOTSTRAP_ROUTE.LIST.routeName,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      hiddenMenu: true,
      keepAlive: true,
      pageKind: 'list',
      semanticTitle: notificationRouteTitle,
      tabGroup: 'notification',
      tabTitle: notificationRouteTitle,
      title: notificationRouteTitle,
      titleKey: 'menu.notification.title',
    },
  },
];
