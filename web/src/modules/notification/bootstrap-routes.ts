// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { BootstrapRouteRegistration } from '@/modules/types';

import { NOTIFICATION_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const notificationBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...NOTIFICATION_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'notification',
      pageKind: 'list',
      semanticTitle: {
        'zh-CN': '通知中心',
        'en-US': 'Notification Center',
      },
      breadcrumbTitle: {
        'zh-CN': '通知中心',
        'en-US': 'Notification Center',
      },
      tabTitle: {
        'zh-CN': '通知中心',
        'en-US': 'Notification Center',
      },
    },
  },
];
