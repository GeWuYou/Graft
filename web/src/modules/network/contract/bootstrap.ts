import { NETWORK_ROUTE_PATH } from './paths';

export const NETWORK_BOOTSTRAP_ROUTE = {
  CONNECTIVITY: {
    menuPath: NETWORK_ROUTE_PATH.CONNECTIVITY,
    routeName: 'PlatformNetworkConnectivity',
  },
} as const;

export const NETWORK_GLOBAL_ROUTE = {
  OUTBOUND: {
    path: NETWORK_ROUTE_PATH.OUTBOUND,
    routeName: 'PlatformNetworkOutbound',
  },
  CONNECTIVITY_DIAGNOSTICS: {
    path: NETWORK_ROUTE_PATH.CONNECTIVITY_DIAGNOSTICS,
    routeName: 'PlatformNetworkConnectivityDiagnostics',
  },
} as const;
