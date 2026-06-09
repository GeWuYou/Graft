// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { BootstrapRouteRegistration } from '@/modules/types';

import { RBAC_BOOTSTRAP_ROUTE } from './contract/bootstrap';

export const rbacBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...RBAC_BOOTSTRAP_ROUTE.ROLE_LIST,
    loadPage: () => import('./pages/index.vue'),
    meta: {
      domain: 'rbac',
      tabGroup: 'rbac',
      pageKind: 'list',
      semanticTitle: {
        'zh-CN': '访问控制 - 角色管理',
        'en-US': 'Access Control - Role Management',
      },
      breadcrumbTitle: {
        'zh-CN': '角色管理',
        'en-US': 'Role Management',
      },
      tabTitle: {
        'zh-CN': '访问控制 - 角色管理',
        'en-US': 'Access Control - Role Management',
      },
    },
  },
  {
    ...RBAC_BOOTSTRAP_ROUTE.PERMISSION_LIST,
    loadPage: () => import('./pages/permissions/index.vue'),
    meta: {
      domain: 'rbac',
      tabGroup: 'rbac',
      pageKind: 'list',
      semanticTitle: {
        'zh-CN': '访问控制 - 权限管理',
        'en-US': 'Access Control - Permission Management',
      },
      breadcrumbTitle: {
        'zh-CN': '权限管理',
        'en-US': 'Permission Management',
      },
      tabTitle: {
        'zh-CN': '访问控制 - 权限管理',
        'en-US': 'Access Control - Permission Management',
      },
    },
  },
];
