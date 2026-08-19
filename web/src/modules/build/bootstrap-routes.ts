import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { BUILD_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const jobsTitle = localizeRouteTitleKey('build.jobs.title');
const artifactsTitle = localizeRouteTitleKey('build.artifacts.title');
const workspacesTitle = localizeRouteTitleKey('build.workspaces.title');
const createTitle = localizeRouteTitleKey('build.jobs.create.title');
const createWorkspaceTitle = localizeRouteTitleKey('build.workspaces.create.title');

export const buildBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...BUILD_BOOTSTRAP_ROUTE.JOBS,
    loadPage: () => import('./pages/jobs/index.vue'),
    meta: {
      tabGroup: 'build-jobs',
      pageKind: 'list',
      pageSurface: 'paged-table',
      semanticTitle: jobsTitle,
      breadcrumbTitle: jobsTitle,
      tabTitle: jobsTitle,
    },
  },
  {
    ...BUILD_BOOTSTRAP_ROUTE.ARTIFACTS,
    loadPage: () => import('./pages/artifacts/index.vue'),
    meta: {
      tabGroup: 'build-artifacts',
      pageKind: 'list',
      pageSurface: 'paged-table',
      semanticTitle: artifactsTitle,
      breadcrumbTitle: artifactsTitle,
      tabTitle: artifactsTitle,
    },
  },
  {
    ...BUILD_BOOTSTRAP_ROUTE.WORKSPACES,
    loadPage: () => import('./pages/workspaces/index.vue'),
    meta: {
      tabGroup: 'build-workspaces',
      pageKind: 'list',
      pageSurface: 'paged-table',
      semanticTitle: workspacesTitle,
      breadcrumbTitle: workspacesTitle,
      tabTitle: workspacesTitle,
    },
  },
];

export const buildGlobalRouteRegistrations: GlobalRouteRegistration[] = [
  {
    ...BUILD_BOOTSTRAP_ROUTE.CREATE,
    navigationParentPath: BUILD_BOOTSTRAP_ROUTE.JOBS.menuPath,
    loadPage: () => import('./pages/create/index.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: false,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      tabGroup: 'build-jobs',
      semanticTitle: createTitle,
      breadcrumbTitle: createTitle,
      tabTitle: createTitle,
      title: createTitle,
      titleKey: 'build.jobs.create.title',
    },
  },
  {
    ...BUILD_BOOTSTRAP_ROUTE.CREATE_WORKSPACE,
    navigationParentPath: BUILD_BOOTSTRAP_ROUTE.WORKSPACES.menuPath,
    loadPage: () => import('./pages/workspaces/create.vue'),
    meta: {
      hidden: false,
      hiddenMenu: true,
      keepAlive: false,
      pageKind: 'detail',
      pageSurface: 'form-detail',
      tabGroup: 'build-workspaces',
      semanticTitle: createWorkspaceTitle,
      breadcrumbTitle: createWorkspaceTitle,
      tabTitle: createWorkspaceTitle,
      title: createWorkspaceTitle,
      titleKey: 'build.workspaces.create.title',
    },
  },
];
