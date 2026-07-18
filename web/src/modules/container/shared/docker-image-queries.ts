import { useQuery } from '@tanstack/vue-query';
import { computed, type MaybeRef, toValue } from 'vue';

import { queryClient } from '@/shared/query';

import { type DockerImageListQuery, getDockerImages } from '../api/container';

export type DockerImageQueryState = {
  keyword: string;
  offset: number;
  pageSize: number;
  unused?: boolean;
};

export const dockerImageQueryKeys = {
  list: (query: DockerImageQueryState) => ['container', 'images', query] as const,
};

export function useDockerImageQuery(query: MaybeRef<DockerImageQueryState>) {
  return useQuery(
    {
      queryKey: computed(() => dockerImageQueryKeys.list(toValue(query))),
      queryFn: ({ queryKey }) => {
        const { keyword, offset, pageSize, unused } = queryKey[2];
        const requestQuery: DockerImageListQuery = {
          limit: pageSize,
          offset,
          ...(keyword ? { keyword } : {}),
          ...(unused ? { unused } : {}),
        };
        return getDockerImages(requestQuery);
      },
      retry: false,
    },
    queryClient,
  );
}
