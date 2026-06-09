// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { BootstrapRouteRegistration } from '@/modules/types';

import { SYSTEM_CONFIG_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const systemConfigBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...SYSTEM_CONFIG_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'server',
      pageKind: 'list',
      semanticTitle: {
        'zh-CN': '服务管理 - 系统配置',
        'en-US': 'Service Management - System Configuration',
      },
      breadcrumbTitle: {
        'zh-CN': '系统配置',
        'en-US': 'System Configuration',
      },
      tabTitle: {
        'zh-CN': '服务管理 - 系统配置',
        'en-US': 'Service Management - System Configuration',
      },
    },
  },
];
