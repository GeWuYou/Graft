import { useQuery } from '@tanstack/vue-query';

import { queryClient } from '@/shared/query';

import { getDockerImages } from '../api/container';

const dockerImageQueryKey = ['container', 'images'] as const;

export function useDockerImageQuery() {
  return useQuery({ queryKey: dockerImageQueryKey, queryFn: getDockerImages }, queryClient);
}
