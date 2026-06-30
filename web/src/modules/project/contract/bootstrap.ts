import { PROJECT_ROUTE_PATH } from './paths';

export const PROJECT_BOOTSTRAP_ROUTE = {
  LIST: {
    menuPath: PROJECT_ROUTE_PATH.LIST,
    routeName: 'ProjectList',
  },
  CREATE: {
    path: PROJECT_ROUTE_PATH.CREATE,
    pageRouteName: 'ProjectCreateIndex',
    routeName: 'ProjectCreate',
  },
  DETAIL: {
    path: PROJECT_ROUTE_PATH.DETAIL,
    pageRouteName: 'ProjectDetailIndex',
    routeName: 'ProjectDetail',
  },
} as const;
