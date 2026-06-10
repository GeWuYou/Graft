// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';

import { NOTIFICATION_BOOTSTRAP_ROUTE } from './contract/bootstrap';
import { NOTIFICATION_ROUTE_PATH } from './contract/paths';

const notificationRouteTitle = {
  'en-US': 'Notification Center',
  'zh-CN': '通知中心',
};

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
