import { useQuery } from '@tanstack/vue-query';
import { computed, type MaybeRef, toValue } from 'vue';

import { queryClient } from '@/shared/query';

import { getDockerNetwork, getDockerNetworks } from '../api/container';
import { containerResourceQueryKeys } from './container-resource-queries';

const dockerNetworkQueryKeys = {
  list: containerResourceQueryKeys.networks,
  detail: (networkId: string) => ['container', 'resources', 'networks', 'detail', networkId] as const,
};

/** 网络详情独立缓存，列表仍复用 Docker 资源页的网络快照。 */
export function useDockerNetworkListQuery() {
  return useQuery({ queryKey: dockerNetworkQueryKeys.list(), queryFn: getDockerNetworks }, queryClient);
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

/** 创建和删除会改变列表及任一已打开的详情快照。 */
export function invalidateDockerNetworkQueries() {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: dockerNetworkQueryKeys.list() }),
    queryClient.invalidateQueries({ queryKey: ['container', 'resources', 'networks', 'detail'] }),
  ]);
}
