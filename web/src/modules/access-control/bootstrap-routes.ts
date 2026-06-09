// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { BootstrapRouteRegistration } from '@/modules/types';

import { ACCESS_CONTROL_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const accessControlBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...ACCESS_CONTROL_BOOTSTRAP_ROUTE.OVERVIEW,
    loadPage: () => import('./pages/overview/index.vue'),
    meta: {
      domain: 'rbac',
      tabGroup: 'rbac',
      dashboard: true,
      pageKind: 'overview',
      semanticTitle: {
        'zh-CN': '访问控制 - 概览',
        'en-US': 'Access Control - Overview',
      },
      breadcrumbTitle: {
        'zh-CN': '概览',
        'en-US': 'Overview',
      },
      tabTitle: {
        'zh-CN': '访问控制 - 概览',
        'en-US': 'Access Control - Overview',
      },
    },
  },
];
