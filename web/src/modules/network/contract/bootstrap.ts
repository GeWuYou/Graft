import { NETWORK_ROUTE_PATH } from './paths';

export const NETWORK_BOOTSTRAP_ROUTE = {
  OUTBOUND: {
    menuPath: NETWORK_ROUTE_PATH.OUTBOUND,
    routeName: 'PlatformNetworkOutbound',
  },
  CONNECTIVITY: {
    menuPath: NETWORK_ROUTE_PATH.CONNECTIVITY,
    routeName: 'PlatformNetworkConnectivity',
  },
  CONNECTIVITY_DIAGNOSTICS: {
    menuPath: NETWORK_ROUTE_PATH.CONNECTIVITY_DIAGNOSTICS,
    routeName: 'PlatformNetworkConnectivityDiagnostics',
  },
} as const;
