import { APPLICATION_ROUTE_PATH } from './paths';

export const PROJECT_BOOTSTRAP_ROUTE = {
  LIST: {
    menuPath: APPLICATION_ROUTE_PATH.LIST,
    pageRouteName: 'ApplicationListIndex',
    routeName: 'ApplicationList',
  },
  CREATE_IMPORT: {
    path: APPLICATION_ROUTE_PATH.CREATE_IMPORT,
    pageRouteName: 'ApplicationImportIndex',
    routeName: 'ApplicationImport',
  },
  CREATE: {
    path: APPLICATION_ROUTE_PATH.CREATE,
    pageRouteName: 'ApplicationCreateMethodIndex',
    routeName: 'ApplicationCreateMethod',
  },
  CREATE_RUNTIME_TARGET: {
    path: APPLICATION_ROUTE_PATH.CREATE_RUNTIME_TARGET,
    pageRouteName: 'ApplicationCreateRuntimeTargetIndex',
    routeName: 'ApplicationCreateRuntimeTarget',
  },
  CREATE_SOURCE: {
    path: APPLICATION_ROUTE_PATH.CREATE_SOURCE,
    pageRouteName: 'ApplicationCreateSourceIndex',
    routeName: 'ApplicationCreateSource',
  },
  CREATE_DISCOVERY: {
    path: APPLICATION_ROUTE_PATH.CREATE_DISCOVERY,
    pageRouteName: 'ApplicationDiscoveryCandidateIndex',
    routeName: 'ApplicationCreateDiscovery',
  },
  CREATE_BLANK: {
    path: APPLICATION_ROUTE_PATH.CREATE_BLANK,
    pageRouteName: 'ApplicationBlankCreateIndex',
    routeName: 'ApplicationBlankCreate',
  },
  CREATE_TEMPLATE: {
    path: APPLICATION_ROUTE_PATH.CREATE_TEMPLATE,
    pageRouteName: 'ApplicationTemplateCreateIndex',
    routeName: 'ApplicationTemplateCreate',
  },
  CREATE_TEMPLATE_DETAIL: {
    path: APPLICATION_ROUTE_PATH.CREATE_TEMPLATE_DETAIL,
    pageRouteName: 'ApplicationTemplateCatalogDetailIndex',
    routeName: 'ApplicationTemplateCatalogDetail',
  },
  TEMPLATES: {
    menuPath: APPLICATION_ROUTE_PATH.TEMPLATES,
    pageRouteName: 'ApplicationTemplatesIndex',
    routeName: 'ApplicationTemplates',
  },
  TEMPLATE_CREATE: {
    path: APPLICATION_ROUTE_PATH.TEMPLATE_CREATE,
    pageRouteName: 'ApplicationTemplateCreateWizardIndex',
    routeName: 'ApplicationTemplateCreateWizard',
  },
  TEMPLATE_DETAIL: {
    path: APPLICATION_ROUTE_PATH.TEMPLATE_DETAIL,
    pageRouteName: 'ApplicationTemplateDetailIndex',
    routeName: 'ApplicationTemplateDetail',
  },
  DETAIL: {
    path: APPLICATION_ROUTE_PATH.DETAIL,
    pageRouteName: 'ApplicationDetailIndex',
    routeName: 'ApplicationDetail',
  },
  CONFIGURATION_WORKSPACE: {
    path: APPLICATION_ROUTE_PATH.CONFIGURATION_WORKSPACE,
    pageRouteName: 'ApplicationConfigurationWorkspaceIndex',
    routeName: 'ApplicationConfigurationWorkspace',
  },
} as const;
