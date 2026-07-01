import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { PROJECT_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const listRouteTitle = localizeRouteTitleKey('project.route.list.title');
const listBreadcrumbTitle = localizeRouteTitleKey('project.route.list.breadcrumb');
const importRouteTitle = localizeRouteTitleKey('project.route.import.title');
const importBreadcrumbTitle = localizeRouteTitleKey('project.route.import.breadcrumb');
const createRouteTitle = localizeRouteTitleKey('project.route.create.title');
const createBreadcrumbTitle = localizeRouteTitleKey('project.route.create.breadcrumb');
const createDiscoveryRouteTitle = localizeRouteTitleKey('project.route.createDiscovery.title');
const createDiscoveryBreadcrumbTitle = localizeRouteTitleKey('project.route.createDiscovery.breadcrumb');
const createManagedRouteTitle = localizeRouteTitleKey('project.route.createManaged.title');
const createManagedBreadcrumbTitle = localizeRouteTitleKey('project.route.createManaged.breadcrumb');
const createGitRouteTitle = localizeRouteTitleKey('project.route.createGit.title');
const createGitBreadcrumbTitle = localizeRouteTitleKey('project.route.createGit.breadcrumb');
const createTemplateRouteTitle = localizeRouteTitleKey('project.route.createTemplate.title');
const createTemplateBreadcrumbTitle = localizeRouteTitleKey('project.route.createTemplate.breadcrumb');
const createRemoteHostRouteTitle = localizeRouteTitleKey('project.route.createRemoteHost.title');
const createRemoteHostBreadcrumbTitle = localizeRouteTitleKey('project.route.createRemoteHost.breadcrumb');
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
    ...PROJECT_BOOTSTRAP_ROUTE.IMPORT,
    loadPage: () => import('./pages/import/index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: true,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: importRouteTitle,
      breadcrumbTitle: importBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'ops',
      tabTitle: importRouteTitle,
      title: importRouteTitle,
      titleKey: 'project.route.import.title',
    },
  },
  {
    ...PROJECT_BOOTSTRAP_ROUTE.CREATE,
    loadPage: () => import('./pages/create/source-index.vue'),
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
    ...PROJECT_BOOTSTRAP_ROUTE.CREATE_DISCOVERY,
    loadPage: () => import('./pages/create/discovery-index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: true,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: createDiscoveryRouteTitle,
      breadcrumbTitle: createDiscoveryBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'ops',
      tabTitle: createDiscoveryRouteTitle,
      title: createDiscoveryRouteTitle,
      titleKey: 'project.route.createDiscovery.title',
    },
  },
  {
    ...PROJECT_BOOTSTRAP_ROUTE.CREATE_MANAGED,
    loadPage: () => import('./pages/create/index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: true,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: createManagedRouteTitle,
      breadcrumbTitle: createManagedBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'ops',
      tabTitle: createManagedRouteTitle,
      title: createManagedRouteTitle,
      titleKey: 'project.route.createManaged.title',
    },
  },
  {
    ...PROJECT_BOOTSTRAP_ROUTE.CREATE_GIT,
    loadPage: () => import('./pages/create/planned-index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: false,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: createGitRouteTitle,
      breadcrumbTitle: createGitBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'ops',
      tabTitle: createGitRouteTitle,
      title: createGitRouteTitle,
      titleKey: 'project.route.createGit.title',
    },
  },
  {
    ...PROJECT_BOOTSTRAP_ROUTE.CREATE_TEMPLATE,
    loadPage: () => import('./pages/create/planned-index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: false,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: createTemplateRouteTitle,
      breadcrumbTitle: createTemplateBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'ops',
      tabTitle: createTemplateRouteTitle,
      title: createTemplateRouteTitle,
      titleKey: 'project.route.createTemplate.title',
    },
  },
  {
    ...PROJECT_BOOTSTRAP_ROUTE.CREATE_REMOTE_HOST,
    loadPage: () => import('./pages/create/planned-index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: false,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: createRemoteHostRouteTitle,
      breadcrumbTitle: createRemoteHostBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'ops',
      tabTitle: createRemoteHostRouteTitle,
      title: createRemoteHostRouteTitle,
      titleKey: 'project.route.createRemoteHost.title',
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
