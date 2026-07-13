import { PROJECT_ROUTE_PATH } from './paths';

export const PROJECT_BOOTSTRAP_ROUTE = {
  LIST: {
    menuPath: PROJECT_ROUTE_PATH.LIST,
    routeName: 'ProjectList',
  },
  CREATE_IMPORT: {
    path: PROJECT_ROUTE_PATH.CREATE_IMPORT,
    pageRouteName: 'ProjectImportIndex',
    routeName: 'ProjectImport',
  },
  CREATE: {
    path: PROJECT_ROUTE_PATH.CREATE,
    pageRouteName: 'ProjectCreateMethodIndex',
    routeName: 'ProjectCreateMethod',
  },
  CREATE_SOURCE: {
    path: PROJECT_ROUTE_PATH.CREATE_SOURCE,
    pageRouteName: 'ProjectCreateSourceIndex',
    routeName: 'ProjectCreateSource',
  },
  CREATE_DISCOVERY: {
    path: PROJECT_ROUTE_PATH.CREATE_DISCOVERY,
    pageRouteName: 'ProjectDiscoveryCandidateIndex',
    routeName: 'ProjectCreateDiscovery',
  },
  CREATE_BLANK: {
    path: PROJECT_ROUTE_PATH.CREATE_BLANK,
    pageRouteName: 'ProjectBlankCreateIndex',
    routeName: 'ProjectBlankCreate',
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
