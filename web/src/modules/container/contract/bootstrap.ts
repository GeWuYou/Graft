import { CONTAINER_ROUTE_PATH } from './paths';

export const CONTAINER_BOOTSTRAP_ROUTE = {
  LIST: {
    menuPath: CONTAINER_ROUTE_PATH.LIST,
    routeName: 'ContainerList',
  },
  DETAIL: {
    path: CONTAINER_ROUTE_PATH.DETAIL,
    pageRouteName: 'ContainerDetailIndex',
    routeName: 'ContainerDetail',
  },
  RESOURCES: {
    path: CONTAINER_ROUTE_PATH.RESOURCES,
    pageRouteName: 'DockerResourcesIndex',
    routeName: 'DockerResources',
  },
} as const;

export type ContainerBootstrapRouteName =
  (typeof CONTAINER_BOOTSTRAP_ROUTE)[keyof typeof CONTAINER_BOOTSTRAP_ROUTE]['routeName'];
