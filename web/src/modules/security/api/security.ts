import { OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import type { SecurityOverviewQuery, SecurityOverviewResponse } from '../types/security';

type SecurityOverviewPath = typeof OPENAPI_RUNTIME_PATH.getSecurityOverview;
type GetSecurityOverviewOperation = paths[SecurityOverviewPath]['get'];
type GetSecurityOverviewResponse = GetSecurityOverviewOperation['responses'][200]['content']['application/json'];
type GetSecurityOverviewResponseData = NonNullable<GetSecurityOverviewResponse['data']>;

/**
 * 获取安全概览数据。
 *
 * @param query - 安全概览查询条件
 * @returns 安全概览响应数据
 */
export function getSecurityOverview(query: SecurityOverviewQuery) {
  return request.get<GetSecurityOverviewResponseData>({
    url: OPENAPI_RUNTIME_PATH.getSecurityOverview,
    params: query,
  }) as Promise<SecurityOverviewResponse>;
}
