import type { BootstrapRouteRegistration, GlobalRouteRegistration } from '@/modules/types';
import { localizeRouteTitleKey } from '@/utils/route/title';

import { BUILD_BOOTSTRAP_ROUTE } from './contract/bootstrap';

const jobsTitle = localizeRouteTitleKey('build.jobs.title');
const createTitle = localizeRouteTitleKey('build.jobs.create.title');

export const buildBootstrapRouteRegistrations: BootstrapRouteRegistration[] = [
  {
    ...BUILD_BOOTSTRAP_ROUTE.JOBS,
    loadPage: () => import('./pages/jobs/index.vue'),
    meta: {
      tabGroup: 'build-jobs',
      pageKind: 'list',
      pageSurface: 'form-detail',
      semanticTitle: jobsTitle,
      breadcrumbTitle: jobsTitle,
      tabTitle: jobsTitle,
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
];
