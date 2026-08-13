import { REGISTRY_ROUTE_PATH } from './paths';

export const REGISTRY_BOOTSTRAP_ROUTE = {
  LIST: {
    menuPath: REGISTRY_ROUTE_PATH.LIST,
    routeName: 'RegistryConnectionList',
  },
  DETAIL: {
    path: REGISTRY_ROUTE_PATH.DETAIL,
    pageRouteName: 'RegistryConnectionDetailIndex',
    routeName: 'RegistryConnectionDetail',
  },
} as const;
