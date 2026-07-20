import { useQuery } from '@tanstack/vue-query';
import { computed, type Ref } from 'vue';

import { queryClient } from '@/shared/query';

import { getDockerNetworks, getDockerSystem, getDockerVolumes } from '../api/container';

const CONTAINER_RESOURCE_QUERY_SCOPE = ['container', 'resources'] as const;

export type DockerResourceTab = 'containers' | 'networks' | 'volumes' | 'system';

export const containerResourceQueryKeys = {
  networks: () => [...CONTAINER_RESOURCE_QUERY_SCOPE, 'networks'] as const,
  volumes: () => [...CONTAINER_RESOURCE_QUERY_SCOPE, 'volumes'] as const,
  system: () => [...CONTAINER_RESOURCE_QUERY_SCOPE, 'system'] as const,
};

/**
 * useDockerResourceQueries 仅缓存 Docker 静态资源快照；当前 tab 仍是页面本地交互状态。
 *
 * 列表和详情页的实时订阅、轮询、日志及编辑器状态不属于这里的 Query 缓存边界。
 */
export function useDockerResourceQueries(activeTab: Ref<DockerResourceTab>) {
  const networks = useQuery(
    {
      queryKey: containerResourceQueryKeys.networks(),
      queryFn: () => getDockerNetworks(),
      enabled: computed(() => activeTab.value === 'networks'),
    },
    queryClient,
  );
  const volumes = useQuery(
    {
      queryKey: containerResourceQueryKeys.volumes(),
      queryFn: getDockerVolumes,
      enabled: computed(() => activeTab.value === 'volumes'),
    },
    queryClient,
  );
  const system = useQuery(
    {
      queryKey: containerResourceQueryKeys.system(),
      queryFn: getDockerSystem,
      enabled: computed(() => activeTab.value === 'system'),
    },
    queryClient,
  );

  return { networks, system, volumes };
}
