import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { PROJECT_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const listRouteTitle = localizeRouteTitleKey('project.route.list.title');
const listBreadcrumbTitle = localizeRouteTitleKey('project.route.list.breadcrumb');
const importRouteTitle = localizeRouteTitleKey('project.route.createImport.title');
const importBreadcrumbTitle = localizeRouteTitleKey('project.route.createImport.breadcrumb');
const createRouteTitle = localizeRouteTitleKey('project.route.create.title');
const createBreadcrumbTitle = localizeRouteTitleKey('project.route.create.breadcrumb');
const createSourceRouteTitle = localizeRouteTitleKey('project.route.createSource.title');
const createSourceBreadcrumbTitle = localizeRouteTitleKey('project.route.createSource.breadcrumb');
const createDiscoveryRouteTitle = localizeRouteTitleKey('project.route.createDiscovery.title');
const createDiscoveryBreadcrumbTitle = localizeRouteTitleKey('project.route.createDiscovery.breadcrumb');
const createBlankRouteTitle = localizeRouteTitleKey('project.route.createBlank.title');
const createBlankBreadcrumbTitle = localizeRouteTitleKey('project.route.createBlank.breadcrumb');
const createTemplateRouteTitle = localizeRouteTitleKey('project.route.createTemplate.title');
const createTemplateBreadcrumbTitle = localizeRouteTitleKey('project.route.createTemplate.breadcrumb');
const detailRouteTitle = localizeRouteTitleKey('project.route.detail.title');
const detailBreadcrumbTitle = localizeRouteTitleKey('project.route.detail.breadcrumb');
const configurationWorkspaceRouteTitle = localizeRouteTitleKey('project.route.configurationWorkspace.title');
const configurationWorkspaceBreadcrumbTitle = localizeRouteTitleKey('project.route.configurationWorkspace.breadcrumb');

export const projectBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...PROJECT_BOOTSTRAP_ROUTE.LIST,
    loadPage: () => import('./pages/list/index.vue'),
    meta: {
      tabGroup: 'application',
      pageKind: 'list',
      semanticTitle: listRouteTitle,
      breadcrumbTitle: listBreadcrumbTitle,
      tabTitle: listRouteTitle,
    },
  },
];

export const projectGlobalRouteRegistrations: GlobalRouteRegistration[] = [
  {
    ...PROJECT_BOOTSTRAP_ROUTE.CREATE_IMPORT,
    navigationParentPath: PROJECT_BOOTSTRAP_ROUTE.LIST.menuPath,
    loadPage: () => import('./pages/import/index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: true,
      pageKind: 'detail',
      pageSurface: 'paged-table',
      semanticTitle: importRouteTitle,
      breadcrumbTitle: importBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'application',
      tabTitle: importRouteTitle,
      title: importRouteTitle,
      titleKey: 'project.route.createImport.title',
    },
  },
  {
    ...PROJECT_BOOTSTRAP_ROUTE.CREATE,
    navigationParentPath: PROJECT_BOOTSTRAP_ROUTE.LIST.menuPath,
    loadPage: () => import('./pages/create/runtime-index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: true,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: createRouteTitle,
      breadcrumbTitle: createBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'application',
      tabTitle: createRouteTitle,
      title: createRouteTitle,
      titleKey: 'project.route.create.title',
    },
  },
  {
    ...PROJECT_BOOTSTRAP_ROUTE.CREATE_SOURCE,
    navigationParentPath: PROJECT_BOOTSTRAP_ROUTE.LIST.menuPath,
    loadPage: () => import('./pages/create/source-index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: true,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: createSourceRouteTitle,
      breadcrumbTitle: createSourceBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'application',
      tabTitle: createSourceRouteTitle,
      title: createSourceRouteTitle,
      titleKey: 'project.route.createSource.title',
    },
  },
  {
    ...PROJECT_BOOTSTRAP_ROUTE.CREATE_DISCOVERY,
    navigationParentPath: PROJECT_BOOTSTRAP_ROUTE.LIST.menuPath,
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
      tabGroup: 'application',
      tabTitle: createDiscoveryRouteTitle,
      title: createDiscoveryRouteTitle,
      titleKey: 'project.route.createDiscovery.title',
    },
  },
  {
    ...PROJECT_BOOTSTRAP_ROUTE.CREATE_BLANK,
    navigationParentPath: PROJECT_BOOTSTRAP_ROUTE.LIST.menuPath,
    loadPage: () => import('./pages/create/index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: true,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: createBlankRouteTitle,
      breadcrumbTitle: createBlankBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'application',
      tabTitle: createBlankRouteTitle,
      title: createBlankRouteTitle,
      titleKey: 'project.route.createBlank.title',
    },
  },
  {
    ...PROJECT_BOOTSTRAP_ROUTE.CREATE_TEMPLATE,
    navigationParentPath: PROJECT_BOOTSTRAP_ROUTE.LIST.menuPath,
    loadPage: () => import('./pages/create/source-create.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: false,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      semanticTitle: createTemplateRouteTitle,
      breadcrumbTitle: createTemplateBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'application',
      tabTitle: createTemplateRouteTitle,
      title: createTemplateRouteTitle,
      titleKey: 'project.route.createTemplate.title',
    },
  },
  {
    ...PROJECT_BOOTSTRAP_ROUTE.DETAIL,
    navigationParentPath: PROJECT_BOOTSTRAP_ROUTE.LIST.menuPath,
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
      tabGroup: 'application',
      tabTitle: detailRouteTitle,
      title: detailRouteTitle,
      titleKey: 'project.route.detail.title',
    },
  },
  {
    ...PROJECT_BOOTSTRAP_ROUTE.CONFIGURATION_WORKSPACE,
    navigationParentPath: PROJECT_BOOTSTRAP_ROUTE.LIST.menuPath,
    loadPage: () => import('./pages/configuration-workspace/index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: true,
      pageKind: 'detail',
      pageSurface: 'editor',
      semanticTitle: configurationWorkspaceRouteTitle,
      breadcrumbTitle: configurationWorkspaceBreadcrumbTitle,
      domainTitle: listRouteTitle,
      tabGroup: 'application',
      tabTitle: configurationWorkspaceRouteTitle,
      title: configurationWorkspaceRouteTitle,
      titleKey: 'project.route.configurationWorkspace.title',
    },
  },
];
