// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { defineStore } from 'pinia';
import {
  type RouteLocationNormalizedLoaded,
  type RouteLocationRaw,
  type Router,
  type RouteRecordName,
} from 'vue-router';

import { LOCALE } from '@/contracts/i18n/locales';
import { AUTH_ROUTE_NAME } from '@/modules/auth/contract/routes';
import type { TabPageSnapshot, TRouterInfo, TTabRouterType } from '@/utils/types';

const PINNED_TABS_STORAGE_KEY = 'tabs:pinned';
const MAX_CLOSED_TABS = 20;

const homeRoute: Array<TRouterInfo> = [
  {
    tabKey: '/',
    path: '/',
    fullPath: '/',
    routeIdx: 0,
    title: { [LOCALE.ZH_CN]: '工作台', [LOCALE.EN_US]: 'Workspace' },
    name: 'RootEntry',
    isHome: true,
    isAlive: true,
  },
];

// 不需要做多标签tabs页缓存的列表 值为每个页面对应的name 如 DashboardDetail
// const ignoreCacheRoutes = ['DashboardDetail'];
const ignoreCacheRoutes: string[] = [AUTH_ROUTE_NAME.LOGIN];

function createInitialState(): TTabRouterType {
  return {
    tabRouterList: homeRoute.map((route) => ({ ...route })),
    closedTabStack: [],
    activeTabKey: '/',
    isRefreshing: false,
    pageSnapshots: {},
  };
}

function shouldKeepTabAlive(route: TRouterInfo) {
  return !route.isHome && !ignoreCacheRoutes.includes(route.name as string) && route.meta?.keepAlive !== false;
}

function isBrowserStorageAvailable() {
  return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined';
}

function readPinnedTabKeys() {
  if (!isBrowserStorageAvailable()) {
    return new Set<string>();
  }

  try {
    const parsed = JSON.parse(window.localStorage.getItem(PINNED_TABS_STORAGE_KEY) || '[]') as unknown;
    if (!Array.isArray(parsed)) {
      return new Set<string>();
    }

    return new Set(parsed.filter((item): item is string => typeof item === 'string' && Boolean(item.trim())));
  } catch {
    return new Set<string>();
  }
}

function writePinnedTabKeys(keys: string[]) {
  if (!isBrowserStorageAvailable()) {
    return;
  }

  window.localStorage.setItem(PINNED_TABS_STORAGE_KEY, JSON.stringify([...new Set(keys)]));
}

function normalizeTabKey(value?: string) {
  return typeof value === 'string' ? value.trim() : '';
}

function getTabKey(route: Pick<TRouterInfo, 'path' | 'tabKey'>) {
  return normalizeTabKey(route.tabKey) || normalizeTabKey(route.path) || '/';
}

function cloneTab(route: TRouterInfo): TRouterInfo {
  return {
    ...route,
    query: route.query ? { ...route.query } : undefined,
    params: route.params ? { ...route.params } : undefined,
    meta: route.meta ? { ...route.meta } : undefined,
    title: route.title ? { ...route.title } : undefined,
  };
}

function clonePageSnapshot(snapshot: TabPageSnapshot | undefined): TabPageSnapshot | undefined {
  if (!snapshot) {
    return undefined;
  }

  return JSON.parse(JSON.stringify(snapshot)) as TabPageSnapshot;
}

/**
 * Normalizes route state with computed tab properties.
 *
 * Establishes a unique tab key, ensures a full path, determines pinned status based on
 * persisted preferences, and sets keep-alive behavior.
 *
 * @param route - The router info to normalize
 * @param pinnedKeys - Set of pinned tab keys; defaults to the persisted pinned set
 * @returns The route with computed tab properties
 */
function normalizeRouteState(route: TRouterInfo, pinnedKeys = readPinnedTabKeys()): TRouterInfo {
  const tabKey = getTabKey(route);

  return {
    ...route,
    tabKey,
    fullPath: route.fullPath || route.path,
    isPinned: route.isHome ? false : Boolean(route.isPinned || pinnedKeys.has(tabKey)),
    isAlive: route.isHome ? true : shouldKeepTabAlive(route),
  };
}

/**
 * Resolves which title to preserve when updating a tab route.
 *
 * Keeps the existing title if the next route has no title, or if both routes represent the same page and the existing title exists. Otherwise uses the new route's title.
 *
 * @param current - The existing tab route
 * @param next - The new route information
 * @returns The title to use
 */
function resolveNextTabTitle(current: TRouterInfo, next: TRouterInfo) {
  if (!next.title) {
    return current.title;
  }
  if (
    (current.fullPath === next.fullPath || current.path === next.path || getTabKey(current) === getTabKey(next)) &&
    current.title
  ) {
    return current.title;
  }

  return next.title;
}

/**
 * Orders tabs by home status and pin status.
 *
 * @returns The input tabs ordered with home tabs first, followed by pinned non-home tabs, then unpinned non-home tabs.
 */
function sortTabs(routes: TRouterInfo[]) {
  const homeRoutes = routes.filter((route) => route.isHome);
  const pinnedRoutes = routes.filter((route) => !route.isHome && route.isPinned);
  const normalRoutes = routes.filter((route) => !route.isHome && !route.isPinned);

  return [...homeRoutes, ...pinnedRoutes, ...normalRoutes];
}

function fallbackHomeTabs(pinnedKeys = readPinnedTabKeys()) {
  return homeRoute.map((route) => normalizeRouteState(cloneTab(route), pinnedKeys));
}

function ensureNonEmptyTabs(routes: TRouterInfo[], pinnedKeys = readPinnedTabKeys()) {
  const normalized = routes.map((route) => normalizeRouteState(route, pinnedKeys));
  return sortTabs(normalized.length > 0 ? normalized : fallbackHomeTabs(pinnedKeys));
}

function createRouteRecordMatcher(router: Router) {
  const availableNames = new Set<RouteRecordName>();
  const availablePaths = new Set<string>();

  router.getRoutes().forEach((route) => {
    if (route.name) {
      availableNames.add(route.name);
    }

    availablePaths.add(route.path);
  });

  return (route: TRouterInfo) => {
    if (route.isHome) {
      return true;
    }

    if (route.name && availableNames.has(route.name)) {
      return true;
    }

    return availablePaths.has(route.path);
  };
}

function toRouteLocation(route?: TRouterInfo): RouteLocationRaw | null {
  if (!route) {
    return null;
  }

  if (route.params && route.name) {
    return {
      name: route.name,
      params: route.params,
      query: route.query,
    };
  }

  return {
    path: route.path,
    query: route.query,
  };
}

export const useTabsRouterStore = defineStore('tabsRouter', {
  state: createInitialState,
  getters: {
    tabRouters: (state: TTabRouterType) => state.tabRouterList,
    closedTabs: (state: TTabRouterType) => state.closedTabStack,
    canReopenClosedTab: (state: TTabRouterType) => state.closedTabStack.length > 0,
    refreshing: (state: TTabRouterType) => state.isRefreshing,
  },
  actions: {
    startTabRefresh(routeIdx: number) {
      const route = this.tabRouters[routeIdx];
      if (!route) {
        this.isRefreshing = false;
        return;
      }

      this.isRefreshing = true;
      route.isAlive = false;
      this.clearPageSnapshot(getTabKey(route));
    },
    finishTabRefresh(routeIdx: number) {
      const route = this.tabRouters[routeIdx];
      if (route) {
        route.isAlive = shouldKeepTabAlive(route);
      }

      this.isRefreshing = false;
    },
    healPersistedState() {
      this.isRefreshing = false;
      this.tabRouterList = ensureNonEmptyTabs(this.tabRouters);
      if (!this.tabRouterList.some((route) => getTabKey(route) === this.activeTabKey)) {
        this.activeTabKey = getTabKey(this.tabRouterList[0]);
      }
      this.clearSnapshotsForMissingTabs();
      this.syncPinnedTabsStorage();
    },
    healPersistedRoutes(router: Router) {
      const canKeepRoute = createRouteRecordMatcher(router);
      const pinnedKeys = readPinnedTabKeys();
      const nextTabs = this.tabRouters.filter(canKeepRoute);

      this.tabRouterList = ensureNonEmptyTabs(nextTabs, pinnedKeys);
      if (!this.tabRouterList.some((route) => getTabKey(route) === this.activeTabKey)) {
        this.activeTabKey = getTabKey(this.tabRouterList[0]);
      }
      this.closedTabStack = this.closedTabStack.filter(canKeepRoute).slice(-MAX_CLOSED_TABS).map(cloneTab);
      this.clearSnapshotsForMissingTabs();
      this.syncPinnedTabsStorage();
    },
    // 处理新增
    appendTabRouterList(newRoute: TRouterInfo) {
      // 不要将判断条件newRoute.meta.keepAlive !== false修改为newRoute.meta.keepAlive，starter默认开启保活，所以meta.keepAlive未定义时也需要进行保活，只有显式说明false才禁用保活。
      const normalized = normalizeRouteState(newRoute);
      if (!this.tabRouters.find((route: TRouterInfo) => getTabKey(route) === getTabKey(normalized))) {
        this.tabRouterList = sortTabs(this.tabRouterList.concat(normalized));
      } else {
        this.tabRouterList = sortTabs(
          this.tabRouterList.map((route) =>
            getTabKey(route) === getTabKey(normalized)
              ? {
                  ...route,
                  fullPath: normalized.fullPath,
                  query: normalized.query,
                  params: normalized.params,
                  title: resolveNextTabTitle(route, normalized),
                  name: normalized.name,
                  meta: normalized.meta,
                  isAlive: normalized.isAlive,
                }
              : route,
          ),
        );
      }
    },
    // 处理关闭当前
    subtractCurrentTabRouter(newRoute: TRouterInfo) {
      const { routeIdx, path, tabKey } = newRoute;
      if (routeIdx === undefined) return;
      const routeKey = tabKey || path;
      const target = this.tabRouterList[routeIdx] ?? this.tabRouterList.find((route) => getTabKey(route) === routeKey);
      if (!target?.isHome) {
        this.pushClosedTab(target);
      }
      const targetKey = tabKey || (target ? getTabKey(target) : path);
      this.tabRouterList = this.tabRouterList.filter(
        (route, index) => index !== routeIdx && getTabKey(route) !== targetKey,
      );
      this.clearPageSnapshot(targetKey);
      this.syncPinnedTabsStorage();
    },
    // 处理关闭右侧
    subtractTabRouterBehind(newRoute: TRouterInfo) {
      const { routeIdx } = newRoute;
      if (routeIdx === undefined) return;
      this.closeTabsByPredicate((route, index) => index > routeIdx && !route.isHome && !route.isPinned);
    },
    // 处理关闭左侧
    subtractTabRouterAhead(newRoute: TRouterInfo) {
      const { routeIdx } = newRoute;
      if (routeIdx === undefined) return;
      this.closeTabsByPredicate((route, index) => index < routeIdx && !route.isHome && !route.isPinned);
    },
    // 处理关闭其他
    subtractTabRouterOther(newRoute: TRouterInfo) {
      const { routeIdx } = newRoute;
      if (routeIdx === undefined) return;
      const target =
        this.tabRouterList[routeIdx] ?? this.tabRouterList.find((route) => getTabKey(route) === getTabKey(newRoute));
      const targetKey = target ? getTabKey(target) : getTabKey(newRoute);
      this.closeTabsByPredicate((route) => !route.isHome && !route.isPinned && getTabKey(route) !== targetKey);
    },
    closeAllClosableTabs() {
      this.closeTabsByPredicate((route) => !route.isHome && !route.isPinned);
    },
    togglePinnedTab(routeKey: string) {
      this.tabRouterList = sortTabs(
        this.tabRouterList.map((route) => {
          if (getTabKey(route) !== routeKey || route.isHome) {
            return route;
          }

          return {
            ...route,
            isPinned: !route.isPinned,
          };
        }),
      );
      this.syncPinnedTabsStorage();
    },
    duplicateTab(routeKey: string) {
      const targetIndex = this.tabRouterList.findIndex((route) => getTabKey(route) === routeKey);
      const target = this.tabRouterList[targetIndex];
      if (!target) {
        return null;
      }

      const basePath = target.path;
      const duplicateCount =
        this.tabRouterList.filter((route) => route.path === basePath || route.duplicatedFrom === basePath).length + 1;
      const duplicatedRoute: TRouterInfo = normalizeRouteState({
        ...cloneTab(target),
        tabKey: `${basePath}#copy-${Date.now()}-${duplicateCount}`,
        title: this.createDuplicatedTitle(target.title, duplicateCount),
        isPinned: false,
        isDuplicate: true,
        duplicatedFrom: basePath,
      });
      const nextList = [...this.tabRouterList];
      nextList.splice(targetIndex + 1, 0, duplicatedRoute);
      this.tabRouterList = sortTabs(nextList);
      this.copyPageSnapshot(getTabKey(target), getTabKey(duplicatedRoute));

      return duplicatedRoute;
    },
    reopenClosedTab() {
      const route = this.closedTabStack.pop();
      if (!route) {
        return null;
      }

      const restored = normalizeRouteState({
        ...cloneTab(route),
        isPinned: false,
      });
      this.tabRouterList = sortTabs(this.tabRouterList.concat(restored));
      return restored;
    },
    removeTabRouterList() {
      this.tabRouterList = [];
      this.closedTabStack = [];
      this.pageSnapshots = {};
      this.syncPinnedTabsStorage();
    },
    initTabRouterList(newRoutes: TRouterInfo[]) {
      newRoutes?.forEach((route: TRouterInfo) => this.appendTabRouterList(route));
    },
    getNextRouteAfterClose(routeKey: string) {
      const index = this.tabRouterList.findIndex((route) => getTabKey(route) === routeKey);
      if (index === -1) {
        return this.tabRouterList[0] ?? null;
      }

      return this.tabRouterList[index + 1] || this.tabRouterList[index - 1] || this.tabRouterList[0] || null;
    },
    resolveNavigationTarget(route?: TRouterInfo) {
      return toRouteLocation(route);
    },
    activateHomeTab() {
      const homeTab = fallbackHomeTabs()[0];
      const hasHomeTab = this.tabRouterList.some((route) => route.isHome && getTabKey(route) === getTabKey(homeTab));
      const nextTabs = hasHomeTab
        ? this.tabRouterList
        : [homeTab, ...this.tabRouterList.filter((route) => getTabKey(route) !== getTabKey(homeTab))];

      this.tabRouterList = ensureNonEmptyTabs(nextTabs);
      this.activeTabKey = getTabKey(homeTab);
    },
    setActiveRoute(route: RouteLocationNormalizedLoaded) {
      const currentActiveTab = this.tabRouterList.find((tab) => getTabKey(tab) === this.activeTabKey);
      if (currentActiveTab && currentActiveTab.fullPath === route.fullPath) {
        return;
      }

      const activeTab =
        this.tabRouterList.find((tab) => !tab.isDuplicate && tab.fullPath === route.fullPath) ??
        this.tabRouterList.find((tab) => tab.fullPath === route.fullPath) ??
        this.tabRouterList.find((tab) => !tab.isDuplicate && tab.path === route.path) ??
        this.tabRouterList.find((tab) => tab.path === route.path);
      this.activeTabKey = activeTab ? getTabKey(activeTab) : route.path;
    },
    setActiveTabKey(tabKey: string) {
      this.activeTabKey = tabKey;
    },
    syncPinnedTabsStorage() {
      writePinnedTabKeys(this.tabRouterList.filter((route) => route.isPinned && !route.isHome).map(getTabKey));
    },
    closeTabsByPredicate(predicate: (route: TRouterInfo, index: number) => boolean) {
      const closedRoutes: TRouterInfo[] = [];
      this.tabRouterList = this.tabRouterList.filter((route, index) => {
        const shouldClose = predicate(route, index);
        if (shouldClose) {
          closedRoutes.push(route);
        }

        return !shouldClose;
      });

      closedRoutes.forEach((route) => this.pushClosedTab(route));
      closedRoutes.forEach((route) => this.clearPageSnapshot(getTabKey(route)));
      this.syncPinnedTabsStorage();
    },
    pushClosedTab(route: TRouterInfo) {
      if (route.isHome) {
        return;
      }

      const closedRoute = {
        ...cloneTab(route),
        isPinned: false,
        isAlive: shouldKeepTabAlive(route),
      };
      const dedupedStack = this.closedTabStack.filter((item) => getTabKey(item) !== getTabKey(route));
      this.closedTabStack = dedupedStack.concat(closedRoute).slice(-MAX_CLOSED_TABS);
    },
    createDuplicatedTitle(title: TRouterInfo['title'], count: number) {
      if (!title) {
        return title;
      }

      return {
        ...title,
        [LOCALE.ZH_CN]: `${title[LOCALE.ZH_CN] || ''}(${count})`,
        [LOCALE.EN_US]: `${title[LOCALE.EN_US] || title[LOCALE.ZH_CN] || ''} (${count})`,
      };
    },
    getPageSnapshot<TSnapshot extends TabPageSnapshot>(tabKey?: string) {
      if (!tabKey) {
        return undefined;
      }

      return clonePageSnapshot(this.pageSnapshots[tabKey]) as TSnapshot | undefined;
    },
    setPageSnapshot(tabKey: string | undefined, snapshot: TabPageSnapshot) {
      if (!tabKey) {
        return;
      }

      const clonedSnapshot = clonePageSnapshot(snapshot);
      if (!clonedSnapshot) {
        return;
      }

      this.pageSnapshots = {
        ...this.pageSnapshots,
        [tabKey]: clonedSnapshot,
      };
    },
    clearPageSnapshot(tabKey?: string) {
      if (!tabKey || !this.pageSnapshots[tabKey]) {
        return;
      }

      const nextSnapshots = { ...this.pageSnapshots };
      delete nextSnapshots[tabKey];
      this.pageSnapshots = nextSnapshots;
    },
    copyPageSnapshot(sourceTabKey: string, targetTabKey: string) {
      const clonedSnapshot = clonePageSnapshot(this.pageSnapshots[sourceTabKey]);
      if (!clonedSnapshot) {
        return;
      }

      this.pageSnapshots = {
        ...this.pageSnapshots,
        [targetTabKey]: clonedSnapshot,
      };
    },
    clearSnapshotsForMissingTabs() {
      const aliveKeys = new Set(this.tabRouterList.map(getTabKey));
      const nextSnapshots: Record<string, TabPageSnapshot> = {};

      Object.entries(this.pageSnapshots).forEach(([tabKey, snapshot]) => {
        if (aliveKeys.has(tabKey)) {
          nextSnapshots[tabKey] = snapshot;
        }
      });

      this.pageSnapshots = nextSnapshots;
    },
  },
  persist: {
    pick: ['tabRouterList', 'closedTabStack', 'activeTabKey'],
  },
});
