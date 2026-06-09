// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { BootstrapRouteRegistration } from '@/modules/types';
import { USER_ROUTE_PATH } from '@/modules/user/contract/paths';

export const userBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    menuPath: USER_ROUTE_PATH.LIST,
    routeName: 'UserList',
    loadPage: () => import('./pages/index.vue'),
  },
];
