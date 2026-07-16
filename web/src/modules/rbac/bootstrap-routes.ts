import type { BootstrapRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { RBAC_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const roleListRouteTitle = localizeRouteTitleKey('rbac.route.roleList.title');
const roleListBreadcrumbTitle = localizeRouteTitleKey('rbac.route.roleList.breadcrumb');
const permissionListRouteTitle = localizeRouteTitleKey('rbac.route.permissionList.title');
const permissionListBreadcrumbTitle = localizeRouteTitleKey('rbac.route.permissionList.breadcrumb');

/** RBAC 页面通过模块注册面接入动态菜单，路由元数据只声明壳层所需的标题和页面类型。 */
export const rbacBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...RBAC_BOOTSTRAP_ROUTE.ROLE_LIST,
    loadPage: () => import('./pages/index.vue'),
    meta: {
      domain: 'rbac',
      tabGroup: 'rbac',
      pageKind: 'list',
      semanticTitle: roleListRouteTitle,
      breadcrumbTitle: roleListBreadcrumbTitle,
      tabTitle: roleListRouteTitle,
    },
  },
  {
    ...RBAC_BOOTSTRAP_ROUTE.PERMISSION_LIST,
    loadPage: () => import('./pages/permissions/index.vue'),
    meta: {
      domain: 'rbac',
      tabGroup: 'rbac',
      pageKind: 'list',
      semanticTitle: permissionListRouteTitle,
      breadcrumbTitle: permissionListBreadcrumbTitle,
      tabTitle: permissionListRouteTitle,
    },
  },
];
