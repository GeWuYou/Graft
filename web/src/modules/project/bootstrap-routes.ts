import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { PROJECT_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const listRouteTitle = localizeRouteTitleKey('project.route.list.title');
const listBreadcrumbTitle = localizeRouteTitleKey('project.route.list.breadcrumb');
const createRouteTitle = localizeRouteTitleKey('project.route.create.title');
const createBreadcrumbTitle = localizeRouteTitleKey('project.route.create.breadcrumb');
const detailRouteTitle = localizeRouteTitleKey('project.route.detail.title');
const detailBreadcrumbTitle = localizeRouteTitleKey('project.route.detail.breadcrumb');

export const projectBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...PROJECT_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'ops',
      pageKind: 'list',
      pageSurface: 'form-detail',
      semanticTitle: listRouteTitle,
      breadcrumbTitle: listBreadcrumbTitle,
      tabTitle: listRouteTitle,
    },
  },
];

export const projectGlobalRouteRegistrations: GlobalRouteRegistration[] = [
  {
    ...PROJECT_BOOTSTRAP_ROUTE.CREATE,
    loadPage: () => import('./pages/create/index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: true,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: createRouteTitle,
      breadcrumbTitle: createBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'ops',
      tabTitle: createRouteTitle,
      title: createRouteTitle,
      titleKey: 'project.route.create.title',
    },
  },
  {
    ...PROJECT_BOOTSTRAP_ROUTE.DETAIL,
    loadPage: () => import('./pages/detail/index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: true,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: detailRouteTitle,
      breadcrumbTitle: detailBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'ops',
      tabTitle: detailRouteTitle,
      title: detailRouteTitle,
      titleKey: 'project.route.detail.title',
    },
  },
];
