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

/**
 * 构建指定已保存视图的审计日志 API 路径。
 *
 * @param viewId - 已保存视图的标识符
 * @returns 替换视图标识符后的 API 路径
 */
export function buildAuditSavedViewApiPath(viewId: number) {
  return AUDIT_API_PATH.SAVED_VIEW.replace('{viewId}', String(viewId));
}
