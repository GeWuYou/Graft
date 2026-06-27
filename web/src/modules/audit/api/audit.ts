import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { AUDIT_API_PATH } from '../contract/paths';
import type {
  AuditIncidentResponse,
  AuditLogDetailResponse,
  AuditLogListResponse,
  AuditLogQuery,
  AuditOverviewQuery,
  AuditOverviewResponse,
  AuditVisibilityDefaultResponse,
  AuditVisibilityDefaultUpdateRequest,
  AuditVisibilityOverrideResponse,
  AuditVisibilityOverrideUpsertRequest,
  AuditVisibilityPolicyResponse,
} from '../types/audit';

type AuditLogsPath = (typeof AUDIT_API_PATH)['LOGS'];
type GetAuditLogsOperation = paths[AuditLogsPath]['get'];
type GetAuditLogsResponse = GetAuditLogsOperation['responses'][200]['content']['application/json'];
type GetAuditLogsResponseData = NonNullable<GetAuditLogsResponse['data']>;

type AuditLogDetailPath = (typeof AUDIT_API_PATH)['DETAIL'];
type GetAuditLogDetailOperation = paths[AuditLogDetailPath]['get'];
type GetAuditLogDetailResponse = GetAuditLogDetailOperation['responses'][200]['content']['application/json'];
type GetAuditLogDetailResponseData = NonNullable<GetAuditLogDetailResponse['data']>;

type AuditOverviewPath = (typeof AUDIT_API_PATH)['OVERVIEW'];
type GetAuditOverviewOperation = paths[AuditOverviewPath]['get'];
type GetAuditOverviewResponse = GetAuditOverviewOperation['responses'][200]['content']['application/json'];
type GetAuditOverviewResponseData = NonNullable<GetAuditOverviewResponse['data']>;

type AuditIncidentPath = (typeof AUDIT_API_PATH)['INCIDENT_DETAIL'];
type GetAuditIncidentOperation = paths[AuditIncidentPath]['get'];
type GetAuditIncidentResponse = GetAuditIncidentOperation['responses'][200]['content']['application/json'];
type GetAuditIncidentResponseData = NonNullable<GetAuditIncidentResponse['data']>;

export function getAuditLogs(query: AuditLogQuery) {
  return request.get<GetAuditLogsResponseData>({
    url: AUDIT_API_PATH.LOGS,
    params: query,
  }) as Promise<AuditLogListResponse>;
}

export function getAuditLogDetail(id: number) {
  return request.get<GetAuditLogDetailResponseData>({
    url: AUDIT_API_PATH.DETAIL.replace('{id}', String(id)),
  }) as Promise<AuditLogDetailResponse>;
}

export function getAuditOverview(query: AuditOverviewQuery) {
  return request.get<GetAuditOverviewResponseData>({
    url: AUDIT_API_PATH.OVERVIEW,
    params: query,
  }) as Promise<AuditOverviewResponse>;
}

export function getAuditIncident(eventId: number) {
  return request.get<GetAuditIncidentResponseData>({
    url: AUDIT_API_PATH.INCIDENT_DETAIL.replace('{event_id}', String(eventId)),
  }) as Promise<AuditIncidentResponse>;
}

export function getAuditVisibilityPolicy() {
  return request.get<AuditVisibilityPolicyResponse>({
    url: AUDIT_API_PATH.VISIBILITY_POLICY,
  });
}

export function updateAuditVisibilityDefault(payload: AuditVisibilityDefaultUpdateRequest) {
  return request.put<AuditVisibilityDefaultResponse>({
    url: AUDIT_API_PATH.VISIBILITY_POLICY,
    data: payload,
  });
}

export function upsertAuditVisibilityOverride(payload: AuditVisibilityOverrideUpsertRequest) {
  return request.put<AuditVisibilityOverrideResponse>({
    url: AUDIT_API_PATH.VISIBILITY_OVERRIDES,
    data: payload,
  });
}

export function deleteAuditVisibilityOverride(source: string, actionKey: string) {
  return request.delete<Record<string, never>>({
    url: AUDIT_API_PATH.VISIBILITY_OVERRIDES,
    params: {
      source,
      action_key: actionKey,
    },
  });
}
