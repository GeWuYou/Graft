import { NETWORK_ROUTE_PATH } from './paths';

export const NETWORK_BOOTSTRAP_ROUTE = {
  OUTBOUND: {
    menuPath: NETWORK_ROUTE_PATH.OUTBOUND,
    routeName: 'PlatformNetworkOutbound',
  },
} as const;
