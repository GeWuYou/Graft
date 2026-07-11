import { PROJECT_ROUTE_PATH } from './paths';

export const PROJECT_BOOTSTRAP_ROUTE = {
  LIST: {
    menuPath: PROJECT_ROUTE_PATH.LIST,
    routeName: 'ProjectList',
  },
  IMPORT: {
    path: PROJECT_ROUTE_PATH.IMPORT,
    pageRouteName: 'ProjectImportIndex',
    routeName: 'ProjectImport',
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
  CONFIGURATION_WORKSPACE: {
    path: PROJECT_ROUTE_PATH.CONFIGURATION_WORKSPACE,
    pageRouteName: 'ProjectConfigurationWorkspaceIndex',
    routeName: 'ProjectConfigurationWorkspace',
  },
} as const;
