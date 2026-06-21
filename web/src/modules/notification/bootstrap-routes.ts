import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { NOTIFICATION_BOOTSTRAP_ROUTE } from './contract/bootstrap';
import { NOTIFICATION_ROUTE_PATH } from './contract/paths';

const notificationRouteTitle = localizeRouteTitleKey('menu.notification.title');

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
