// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { ANNOUNCEMENT_BOOTSTRAP_ROUTE } from './contract/bootstrap';
import { ANNOUNCEMENT_ROUTE_PATH } from './contract/paths';

const managementRouteTitle = localizeRouteTitleKey('announcement.route.management.title');
const managementBreadcrumbTitle = localizeRouteTitleKey('announcement.route.management.breadcrumb');
const userRouteTitle = localizeRouteTitleKey('announcement.route.user.title');

export const announcementBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...ANNOUNCEMENT_BOOTSTRAP_ROUTE.MANAGEMENT,
    loadPage: () => import('./pages/management/index.vue'),
    meta: {
      pageKind: 'list',
      semanticTitle: managementRouteTitle,
      breadcrumbTitle: managementBreadcrumbTitle,
      tabGroup: 'server',
    },
  },
];

export const announcementGlobalRouteRegistrations: GlobalRouteRegistration[] = [
  {
    path: ANNOUNCEMENT_ROUTE_PATH.USER_LIST,
    routeName: ANNOUNCEMENT_BOOTSTRAP_ROUTE.USER_LIST.routeName,
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
