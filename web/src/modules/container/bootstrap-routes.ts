// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { CONTAINER_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const listRouteTitle = localizeRouteTitleKey('container.route.list.title');
const listBreadcrumbTitle = localizeRouteTitleKey('container.route.list.breadcrumb');

export const containerBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...CONTAINER_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'ops',
      pageKind: 'list',
      semanticTitle: listRouteTitle,
      breadcrumbTitle: listBreadcrumbTitle,
      tabTitle: listRouteTitle,
    },
  },
];
