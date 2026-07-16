import { useQuery } from '@tanstack/vue-query';

import { queryClient } from '@/shared/query';

import { getSystemConfigs } from '../api/system-config';
import type { SystemConfigItem, SystemConfigListResponse } from '../types/system-config';

const SYSTEM_CONFIG_QUERY_SCOPE = ['system-config'] as const;

export const systemConfigQueryKeys = {
  list: () => [...SYSTEM_CONFIG_QUERY_SCOPE, 'list'] as const,
};

/** 系统配置集合快照只由 Query cache 持有，树选择和编辑器草稿仍由页面管理。 */
export function useSystemConfigsQuery() {
  return useQuery({ queryKey: systemConfigQueryKeys.list(), queryFn: getSystemConfigs }, queryClient);
}

/** 将更新或重置接口返回的权威配置项精确写回已缓存的集合。 */
export function upsertSystemConfigCache(updated: SystemConfigItem) {
  queryClient.setQueryData<SystemConfigListResponse>(systemConfigQueryKeys.list(), (current) => {
    if (!current) {
      return current;
    }

    const index = current.items.findIndex((item) => item.key === updated.key);
    const items = [...current.items];
    if (index >= 0) {
      items[index] = updated;
    } else {
      items.push(updated);
    }

    return { ...current, items };
  });
}
