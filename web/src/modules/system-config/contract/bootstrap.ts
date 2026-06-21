import { SYSTEM_CONFIG_ROUTE_PATH } from './paths';

export const SYSTEM_CONFIG_BOOTSTRAP_ROUTE = {
  LIST: {
    menuPath: SYSTEM_CONFIG_ROUTE_PATH.LIST,
    routeName: 'SystemConfigList',
  },
} as const;

export type SystemConfigBootstrapRouteName =
  (typeof SYSTEM_CONFIG_BOOTSTRAP_ROUTE)[keyof typeof SYSTEM_CONFIG_BOOTSTRAP_ROUTE]['routeName'];
