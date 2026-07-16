import { QueryClient } from '@tanstack/vue-query';

const DEFAULT_STALE_TIME_MS = 30_000;
const DEFAULT_GC_TIME_MS = 5 * 60_000;

/**
 * Shared client for server-owned data. Module query keys and query functions remain module-owned.
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      gcTime: DEFAULT_GC_TIME_MS,
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: DEFAULT_STALE_TIME_MS,
    },
    mutations: {
      retry: 0,
    },
  },
});

/** Clears every server-data snapshot when the authenticated session ends. */
export function clearQueryCache() {
  queryClient.clear();
}
