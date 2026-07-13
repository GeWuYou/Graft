import { RUNTIME_TARGET_ROUTE_PATH } from './paths';
export const RUNTIME_TARGET_BOOTSTRAP_ROUTE = {
  LIST: { menuPath: RUNTIME_TARGET_ROUTE_PATH.LIST, routeName: 'RuntimeTargetList' },
  DETAIL: { path: RUNTIME_TARGET_ROUTE_PATH.DETAIL, routeName: 'RuntimeTargetDetail' },
} as const;
