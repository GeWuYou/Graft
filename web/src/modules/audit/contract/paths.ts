export const AUDIT_ROUTE_PATH = {
  LOGS: '/security/audit',
  INCIDENT_DETAIL: '/security/audit/incidents/:event_id',
} as const;

export const AUDIT_API_PATH = {
  DETAIL: '/api/audit/logs/{id}',
  LOGS: '/api/audit/logs',
  INCIDENT_DETAIL: '/api/audit/incidents/{event_id}',
  VISIBILITY_POLICY: '/api/audit/policies/visibility',
  VISIBILITY_OVERRIDES: '/api/audit/policies/visibility/overrides',
} as const;
