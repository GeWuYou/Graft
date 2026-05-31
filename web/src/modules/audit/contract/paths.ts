export const AUDIT_ROUTE_PATH = {
  OVERVIEW: '/audit/overview',
  LOGS: '/audit/logs',
  INCIDENT_DETAIL: '/audit/incidents/:event_id',
} as const;

export const AUDIT_API_PATH = {
  LOGS: '/api/audit/logs',
  OVERVIEW: '/api/audit/overview',
  INCIDENT_DETAIL: '/api/audit/incidents/{event_id}',
} as const;
