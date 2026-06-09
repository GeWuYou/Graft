// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { AUDIT_API_PATH } from '../contract/paths';
import type {
  AuditIncidentResponse,
  AuditLogListResponse,
  AuditLogQuery,
  AuditOverviewQuery,
  AuditOverviewResponse,
} from '../types/audit';

type AuditLogsPath = (typeof AUDIT_API_PATH)['LOGS'];
type GetAuditLogsOperation = paths[AuditLogsPath]['get'];
type GetAuditLogsResponse = GetAuditLogsOperation['responses'][200]['content']['application/json'];
type GetAuditLogsResponseData = NonNullable<GetAuditLogsResponse['data']>;

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
