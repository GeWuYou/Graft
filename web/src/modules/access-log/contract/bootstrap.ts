import { ACCESS_LOG_ROUTE_PATH } from './paths';

export const ACCESS_LOG_BOOTSTRAP_ROUTE = {
  LIST: {
    menuPath: ACCESS_LOG_ROUTE_PATH.LIST,
    routeName: 'AccessLogList',
  },
} as const;
