// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { BootstrapRouteRegistration } from '@/modules/types';

import { APP_LOG_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const appLogBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...APP_LOG_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'app-log',
      pageKind: 'list',
      titleKey: 'menu.appLog.title',
    },
  },
];
