import { useQuery } from '@tanstack/vue-query';
import { computed, type MaybeRef, toValue } from 'vue';

import { queryClient } from '@/shared/query';

import { type DockerNetworkListQuery, getDockerNetwork, getDockerNetworks } from '../api/container';

const dockerNetworkQueryScope = ['container', 'resources', 'networks'] as const;
const dockerNetworkQueryKeys = {
  list: () => [...dockerNetworkQueryScope] as const,
  detail: (networkId: string) => [...dockerNetworkQueryScope, 'detail', networkId] as const,
};

/** 网络列表与详情由网络查询边界统一拥有；列表 key 保持既有 tuple 以维持缓存与失效一致性。 */
export function useDockerNetworkListQuery(query: MaybeRef<DockerNetworkListQuery>) {
  return useQuery(
    {
      queryKey: computed(() => [...dockerNetworkQueryKeys.list(), toValue(query)]),
      queryFn: () => getDockerNetworks(toValue(query)),
    },
    queryClient,
  );
}

export function useDockerNetworkDetailQuery(networkId: MaybeRef<string>) {
  return useQuery(
    {
      queryKey: computed(() => dockerNetworkQueryKeys.detail(toValue(networkId) || '')),
      queryFn: ({ queryKey }) => getDockerNetwork(queryKey[4]),
      enabled: computed(() => Boolean(toValue(networkId))),
    },
    queryClient,
  );
}

/** 创建和删除会改变网络列表快照。 */
export function invalidateDockerNetworkQueries() {
  return queryClient.invalidateQueries({ queryKey: dockerNetworkQueryKeys.list() });
}
