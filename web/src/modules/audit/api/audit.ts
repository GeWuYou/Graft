import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { AUDIT_API_PATH } from '../contract/paths';
import type { AuditLogListResponse, AuditLogQuery, AuditOverviewQuery, AuditOverviewResponse } from '../types/audit';

type AuditLogsPath = (typeof AUDIT_API_PATH)['LOGS'];
type GetAuditLogsOperation = paths[AuditLogsPath]['get'];
type GetAuditLogsResponse = GetAuditLogsOperation['responses'][200]['content']['application/json'];
type GetAuditLogsResponseData = NonNullable<GetAuditLogsResponse['data']>;

type AuditOverviewPath = (typeof AUDIT_API_PATH)['OVERVIEW'];
type GetAuditOverviewOperation = paths[AuditOverviewPath]['get'];
type GetAuditOverviewResponse = GetAuditOverviewOperation['responses'][200]['content']['application/json'];
type GetAuditOverviewResponseData = NonNullable<GetAuditOverviewResponse['data']>;

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
