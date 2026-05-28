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
        'zh-CN': '访问控制 · 角色管理',
        'en-US': 'Access Roles',
      },
      tabTitle: {
        'zh-CN': '访问控制 · 角色',
        'en-US': 'Access Roles',
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
        'zh-CN': '访问控制 · 权限管理',
        'en-US': 'Access Permissions',
      },
      tabTitle: {
        'zh-CN': '访问控制 · 权限',
        'en-US': 'Access Permissions',
      },
    },
  },
];
