import { OPENAPI_RUNTIME_PATH } from '@/contracts/generated/openapi-runtime-paths';
import type { components } from '@/contracts/openapi/generated/schema';
import { request } from '@/utils/request';

export type DashboardSystemConfigItem = components['schemas']['system-config-item'];
type DashboardSystemConfigListResponse = components['schemas']['system-config-list-response'];

/** 快捷操作配置沿用系统配置权限边界，缺失或无效配置由调用方回退默认策略。 */
export function getDashboardSystemConfigs() {
  return request.get<DashboardSystemConfigListResponse>({
    url: OPENAPI_RUNTIME_PATH.getSystemConfigs,
  }) as Promise<DashboardSystemConfigListResponse>;
}
