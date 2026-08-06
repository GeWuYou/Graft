import { onlineManager, QueryClient } from '@tanstack/vue-query';

const DEFAULT_STALE_TIME_MS = 30_000;
const DEFAULT_GC_TIME_MS = 5 * 60_000;
let platformOnline = true;

/**
 * 共享的服务端数据客户端；查询键和查询函数仍由业务模块拥有，避免壳层形成业务缓存真值。
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      gcTime: DEFAULT_GC_TIME_MS,
      refetchOnWindowFocus: false,
      retry: (failureCount) => platformOnline && failureCount < 1,
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

/** 平台不可达时暂停 Query 的 retry 与网络恢复行为，恢复后仅重新开放统一网络闸门。 */
export function setPlatformQueryOnline(online: boolean) {
  platformOnline = online;
  onlineManager.setOnline(online);
}
