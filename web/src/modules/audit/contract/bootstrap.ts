import { AUDIT_ROUTE_PATH } from './paths';

export const AUDIT_BOOTSTRAP_ROUTE = {
  LOG_LIST: {
    menuPath: AUDIT_ROUTE_PATH.LOGS,
    routeName: 'AuditLogList',
  },
} as const;
