import { PROJECT_ROUTE_PATH } from './paths';

export const PROJECT_BOOTSTRAP_ROUTE = {
  LIST: {
    menuPath: PROJECT_ROUTE_PATH.LIST,
    routeName: 'ProjectList',
  },
  CREATE: {
    path: PROJECT_ROUTE_PATH.CREATE,
    pageRouteName: 'ProjectCreateSourceIndex',
    routeName: 'ProjectCreateSource',
  },
  CREATE_DISCOVERY: {
    path: PROJECT_ROUTE_PATH.CREATE_DISCOVERY,
    pageRouteName: 'ProjectDiscoveryCandidateIndex',
    routeName: 'ProjectCreateDiscovery',
  },
  CREATE_MANAGED: {
    path: PROJECT_ROUTE_PATH.CREATE_MANAGED,
    pageRouteName: 'ProjectManagedCreateIndex',
    routeName: 'ProjectManagedCreate',
  },
  CREATE_GIT: {
    path: PROJECT_ROUTE_PATH.CREATE_GIT,
    pageRouteName: 'ProjectGitCreateIndex',
    routeName: 'ProjectGitCreate',
  },
  CREATE_TEMPLATE: {
    path: PROJECT_ROUTE_PATH.CREATE_TEMPLATE,
    pageRouteName: 'ProjectTemplateCreateIndex',
    routeName: 'ProjectTemplateCreate',
  },
  DETAIL: {
    path: PROJECT_ROUTE_PATH.DETAIL,
    pageRouteName: 'ProjectDetailIndex',
    routeName: 'ProjectDetail',
  },
} as const;
