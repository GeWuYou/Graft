import { computed, onBeforeUnmount, watch } from 'vue';
import { useRoute } from 'vue-router';

import { useTabsRouterStore } from '@/store';

type TabRefreshHandler = () => void | Promise<void>;

const tabRefreshHandlers = new Map<string, TabRefreshHandler>();

function normalizeTabKey(value?: string | null) {
  return typeof value === 'string' ? value.trim() : '';
}

function registerTabRefreshHandler(tabKey: string, handler: TabRefreshHandler) {
  const normalizedTabKey = normalizeTabKey(tabKey);
  if (!normalizedTabKey) {
    return () => undefined;
  }

  tabRefreshHandlers.set(normalizedTabKey, handler);

  return () => {
    const currentHandler = tabRefreshHandlers.get(normalizedTabKey);
    if (currentHandler === handler) {
      tabRefreshHandlers.delete(normalizedTabKey);
    }
  };
}

export function resolveTabRefreshHandler(tabKey?: string | null) {
  const normalizedTabKey = normalizeTabKey(tabKey);
  if (!normalizedTabKey) {
    return undefined;
  }

  return tabRefreshHandlers.get(normalizedTabKey);
}

export function useCurrentTabRefresh(handler: TabRefreshHandler) {
  const route = useRoute();
  const tabsRouterStore = useTabsRouterStore();

  const activeTabKey = computed(() => {
    const activeTabRoute = tabsRouterStore.tabRouters.find(
      (tabRoute) => tabRoute.tabKey === tabsRouterStore.activeTabKey,
    );
    if (activeTabRoute?.path === route.path || activeTabRoute?.fullPath === route.fullPath) {
      return activeTabRoute.tabKey ?? activeTabRoute.path;
    }

    const matchedRoute =
      tabsRouterStore.tabRouters.find((tabRoute) => tabRoute.fullPath === route.fullPath) ??
      tabsRouterStore.tabRouters.find((tabRoute) => tabRoute.path === route.path);
    return matchedRoute?.tabKey ?? matchedRoute?.path ?? route.path;
  });

  let unregister = registerTabRefreshHandler(activeTabKey.value, handler);

  watch(activeTabKey, (nextTabKey) => {
    unregister();
    unregister = registerTabRefreshHandler(nextTabKey, handler);
  });

  onBeforeUnmount(() => {
    unregister();
  });
}
