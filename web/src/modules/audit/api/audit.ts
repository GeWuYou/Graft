import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { AUDIT_API_PATH } from '../contract/paths';
import type { AuditLogListResponse, AuditLogQuery } from '../types/audit';

type AuditLogsPath = (typeof AUDIT_API_PATH)['LOGS'];
type GetAuditLogsOperation = paths[AuditLogsPath]['get'];
type GetAuditLogsResponse = GetAuditLogsOperation['responses'][200]['content']['application/json'];
type GetAuditLogsResponseData = NonNullable<GetAuditLogsResponse['data']>;

export function getAuditLogs(query: AuditLogQuery) {
  return request.get<GetAuditLogsResponseData>({
    url: AUDIT_API_PATH.LOGS,
    params: query,
  }) as Promise<AuditLogListResponse>;
}
