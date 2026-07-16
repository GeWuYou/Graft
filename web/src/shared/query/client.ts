import { QueryClient } from '@tanstack/vue-query';

const DEFAULT_STALE_TIME_MS = 30_000;
const DEFAULT_GC_TIME_MS = 5 * 60_000;

/**
 * 共享的服务端数据客户端；查询键和查询函数仍由业务模块拥有，避免壳层形成业务缓存真值。
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

/** 认证会话结束时清理全部服务端数据快照，避免下一个会话读取前一个会话的缓存。 */
export function clearQueryCache() {
  queryClient.clear();
}
