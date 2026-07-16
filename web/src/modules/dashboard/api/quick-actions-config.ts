import type { components } from '@/contracts/openapi/generated/schema';
import { SYSTEM_CONFIG_API_PATH } from '@/modules/system-config/contract/paths';
import { request } from '@/utils/request';

export type DashboardSystemConfigItem = components['schemas']['system-config-item'];
type DashboardSystemConfigListResponse = components['schemas']['system-config-list-response'];

/** 快捷操作配置沿用系统配置权限边界，缺失或无效配置由调用方回退默认策略。 */
export function getDashboardSystemConfigs() {
  return request.get<DashboardSystemConfigListResponse>({
    url: SYSTEM_CONFIG_API_PATH.LIST,
  }) as Promise<DashboardSystemConfigListResponse>;
}
