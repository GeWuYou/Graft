import type { paths } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

import { SECURITY_API_PATH } from '../contract/paths';
import type { SecurityOverviewQuery, SecurityOverviewResponse } from '../types/security';

type SecurityOverviewPath = (typeof SECURITY_API_PATH)['OVERVIEW'];
type GetSecurityOverviewOperation = paths[SecurityOverviewPath]['get'];
type GetSecurityOverviewResponse = GetSecurityOverviewOperation['responses'][200]['content']['application/json'];
type GetSecurityOverviewResponseData = NonNullable<GetSecurityOverviewResponse['data']>;

export function getSecurityOverview(query: SecurityOverviewQuery) {
  return request.get<GetSecurityOverviewResponseData>({
    url: SECURITY_API_PATH.OVERVIEW,
    params: query,
  }) as Promise<SecurityOverviewResponse>;
}
