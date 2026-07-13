export const AUDIT_ROUTE_PATH = {
  LOGS: '/security/audit',
  INCIDENT_DETAIL: '/security/audit/incidents/:event_id',
} as const;

export const AUDIT_API_PATH = {
  DETAIL: '/api/audit/logs/{id}',
  LOGS: '/api/audit/logs',
  SAVED_VIEWS: '/api/audit/logs/saved-views',
  SAVED_VIEW: '/api/audit/logs/saved-views/{viewId}',
  INCIDENT_DETAIL: '/api/audit/incidents/{event_id}',
  VISIBILITY_POLICY: '/api/audit/policies/visibility',
  VISIBILITY_OVERRIDES: '/api/audit/policies/visibility/overrides',
} as const;

export function buildAuditSavedViewApiPath(viewId: number) {
  return AUDIT_API_PATH.SAVED_VIEW.replace('{viewId}', String(viewId));
}
