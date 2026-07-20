import { useQuery } from '@tanstack/vue-query';
import { computed, type MaybeRef, toValue } from 'vue';

import { queryClient } from '@/shared/query';

import { type DockerNetworkListQuery, getDockerNetwork, getDockerNetworks } from '../api/container';
import { containerResourceQueryKeys } from './container-resource-queries';

const dockerNetworkQueryKeys = {
  list: containerResourceQueryKeys.networks,
  detail: (networkId: string) => ['container', 'resources', 'networks', 'detail', networkId] as const,
};

/** 网络详情独立缓存，列表仍复用 Docker 资源页的网络快照。 */
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
      queryKey: computed(() => dockerNetworkQueryKeys.detail(toValue(networkId))),
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
