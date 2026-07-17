import { defineStore } from 'pinia';
import {
  type RouteLocationNormalizedLoaded,
  type RouteLocationRaw,
  type RouteLocationResolved,
  type Router,
  type RouteRecordName,
} from 'vue-router';

import { LOCALE, type LocalizedTitle } from '@/contracts/i18n/locales';
import { AUTH_ROUTE_NAME } from '@/modules/auth/contract/routes';
import { createLogger } from '@/utils/logger';
import { PAGE_NOT_FOUND_ROUTE } from '@/utils/route/constant';
import { localizeRouteTitleKey } from '@/utils/route/title';
import { formatTabDebugTitle, formatTabsDebugSummary, logTabsDebug } from '@/utils/tabs-debug';
import type { TabPageSnapshot, TRouterInfo, TTabRouterType } from '@/utils/types';

const PINNED_TABS_STORAGE_KEY = 'tabs:pinned';
const REMOVED_LEGACY_TAB_KEYS = new Set([
  '/access-control/overview',
  '/access-control/users',
  '/access-control/roles',
  '/access-control/permissions',
  '/audit/overview',
]);
const LEGACY_ACCESS_CONTROL_TITLE_MARKERS = ['访问控制', 'Access Control'];
const MAX_CLOSED_TABS = 20;
const ROOT_ENTRY_TITLE_KEY = 'app.home.title';
const logger = createLogger('store.tabsRouter');

const homeRoute: Array<TRouterInfo> = [
  {
    tabKey: '/',
    path: '/',
    fullPath: '/',
    routeIdx: 0,
    title: localizeRouteTitleKey(ROOT_ENTRY_TITLE_KEY),
    name: 'RootEntry',
    isHome: true,
    isAlive: true,
  },
];

const ignoreCacheRoutes: string[] = [AUTH_ROUTE_NAME.LOGIN];

/**
 * 创建标签路由存储的初始状态。
 *
 * @returns 包含首页标签、关闭栈、当前激活标签、刷新标记、刷新 nonce 映射和页面快照的初始状态对象。
 */
function createInitialState(): TTabRouterType {
  return {
    tabRouterList: homeRoute.map((route) => ({ ...route })),
    closedTabStack: [],
    activeTabKey: '/',
    refreshingTabKey: undefined,
    refreshNonceByTabKey: {},
    pageSnapshots: {},
  };
}

function shouldKeepTabAlive(route: TRouterInfo) {
  return !route.isHome && !ignoreCacheRoutes.includes(route.name as string) && route.meta?.keepAlive !== false;
}

function isBrowserStorageAvailable() {
  return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined';
}

/**
 * 读取并清理持久化的置顶标签键。
 *
 * @returns 有效置顶标签键的集合；存储不可用、数据格式无效或读取失败时返回空集合
 */
function readPinnedTabKeys() {
  if (!isBrowserStorageAvailable()) {
    return new Set<string>();
  }

  try {
    const parsed = JSON.parse(window.localStorage.getItem(PINNED_TABS_STORAGE_KEY) || '[]') as unknown;
    if (!Array.isArray(parsed)) {
      return new Set<string>();
    }

    const keys = parsed.filter(
      (item): item is string => typeof item === 'string' && Boolean(item.trim()) && !REMOVED_LEGACY_TAB_KEYS.has(item),
    );
    if (keys.length !== parsed.length) {
      writePinnedTabKeys(keys);
    }
    return new Set(keys);
  } catch {
    return new Set<string>();
  }
}

function writePinnedTabKeys(keys: string[]) {
  if (!isBrowserStorageAvailable()) {
    return;
  }

  try {
    window.localStorage.setItem(PINNED_TABS_STORAGE_KEY, JSON.stringify([...new Set(keys)]));
  } catch (error) {
    logger.warn('pinned tabs storage write failed', {
      storageKey: PINNED_TABS_STORAGE_KEY,
      error,
    });
  }
}

function normalizeTabKey(value?: string) {
  return typeof value === 'string' ? value.trim() : '';
}

/**
 * 获取标签页的唯一标识。
 *
 * @param route - 包含标签标识或路径的路由信息
 * @returns 规范化后的标签标识；两者均为空时返回根路径 `/`
 */
function getTabKey(route: Pick<TRouterInfo, 'path' | 'tabKey'>) {
  return normalizeTabKey(route.tabKey) || normalizeTabKey(route.path) || '/';
}

/**
 * 判断标签是否属于需要移除的旧版遗留标签。
 *
 * @param route - 要检查的标签路由
 * @returns 如果标签标识符或标题包含旧版遗留标记则为 `true`，否则为 `false`
 */
function isRemovedLegacyTab(route: TRouterInfo) {
  const identifiers = [route.tabKey, route.path, route.fullPath, route.duplicatedFrom]
    .map(normalizeTabKey)
    .filter(Boolean);
  if (identifiers.some((identifier) => REMOVED_LEGACY_TAB_KEYS.has(identifier))) {
    return true;
  }

  return Object.values(route.title ?? {}).some((title) =>
    LEGACY_ACCESS_CONTROL_TITLE_MARKERS.some((marker) => title.includes(marker)),
  );
}

/**
 * 移除已废弃的旧版标签页。
 *
 * @param routes - 待清理的标签页路由列表
 * @returns 不包含已废弃旧版标签页的路由列表
 */
function removeLegacyTabs(routes: TRouterInfo[]) {
  return routes.filter((route) => !isRemovedLegacyTab(route));
}

/**
 * 创建标签路由的浅克隆副本，并复制其查询参数、路径参数、元数据和标题。
 *
 * @param route - 要克隆的标签路由
 * @returns 克隆后的标签路由
 */
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

function formatTabsSummary(routes: TRouterInfo[]) {
  return formatTabsDebugSummary(
    routes.map((route) => ({
      fullPath: route.fullPath,
      name: route.name,
      path: route.path,
      tabKey: getTabKey(route),
      title: route.title,
    })),
  );
}

/**
 * 规范化路由状态并补充计算得到的标签属性。
 *
 * 该过程会建立唯一标签键、补齐完整路径，根据持久化偏好确定置顶状态，
 * 并设置页面保活行为。
 *
 * @param route - 待规范化的路由信息
 * @param pinnedKeys - 置顶标签键集合；默认读取持久化的置顶集合
 * @returns 补充计算标签属性后的路由
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
 * 确定更新标签路由时应保留的标题。
 *
 * 当新路由没有标题，或两个路由表示同一页面且旧标题存在时保留旧标题；否则使用新路由标题。
 *
 * @param current - 当前标签路由
 * @param next - 新的路由信息
 * @returns 应使用的标题
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
 * 按首页状态和置顶状态排列标签。
 *
 * @returns 按首页、非首页置顶、非首页未置顶顺序排列的标签
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

  router.getRoutes().forEach((route) => {
    if (route.name) {
      availableNames.add(route.name);
    }
  });

  return (route: TRouterInfo) => {
    if (route.isHome) {
      return true;
    }

    if (route.name && availableNames.has(route.name)) {
      return true;
    }

    const resolved = resolvePersistedRoute(router, route);
    return isResolvedTabRouteValid(resolved, route);
  };
}

function resolvePersistedRoute(router: Router, route: TRouterInfo) {
  const target = toRouteLocation(route);
  if (!target) {
    return null;
  }

  try {
    return router.resolve(target);
  } catch {
    return null;
  }
}

function isResolvedTabRouteValid(resolved: RouteLocationResolved | null, route: TRouterInfo) {
  if (!resolved || resolved.name === PAGE_NOT_FOUND_ROUTE.name) {
    return false;
  }

  const matchesCurrentPath = resolved.path === route.path;
  if (!matchesCurrentPath) {
    return false;
  }

  if (!route.name) {
    return resolved.matched.length > 0;
  }

  return resolved.matched.some((record) => record.name === route.name);
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
    refreshing: (state: TTabRouterType) => Boolean(state.refreshingTabKey),
  },
  actions: {
    startTabRefresh(tabKey: string) {
      const route = this.tabRouters.find((item) => getTabKey(item) === tabKey);
      if (!route) {
        this.refreshingTabKey = undefined;
        return;
      }

      this.refreshingTabKey = tabKey;
      route.isAlive = false;
      this.refreshNonceByTabKey = {
        ...this.refreshNonceByTabKey,
        [tabKey]: (this.refreshNonceByTabKey[tabKey] ?? 0) + 1,
      };
      this.clearPageSnapshot(tabKey);
    },
    finishTabRefresh(tabKey: string) {
      const route = this.tabRouters.find((item) => getTabKey(item) === tabKey);
      if (route) {
        route.isAlive = shouldKeepTabAlive(route);
      }

      if (this.refreshingTabKey === tabKey) {
        this.refreshingTabKey = undefined;
      }
    },
    healPersistedState() {
      logTabsDebug(
        'tabs.store',
        () => `tabs debug: healPersistedState before active=${this.activeTabKey} ${formatTabsSummary(this.tabRouters)}`,
      );
      this.refreshingTabKey = undefined;
      this.tabRouterList = ensureNonEmptyTabs(removeLegacyTabs(this.tabRouters));
      if (!this.tabRouterList.some((route) => getTabKey(route) === this.activeTabKey)) {
        this.activeTabKey = getTabKey(this.tabRouterList[0]);
      }
      this.clearSnapshotsForMissingTabs();
      this.clearRefreshNonceForMissingTabs();
      this.syncPinnedTabsStorage();
      logTabsDebug(
        'tabs.store',
        () => `tabs debug: healPersistedState after active=${this.activeTabKey} ${formatTabsSummary(this.tabRouters)}`,
      );
    },
    healPersistedRoutes(router: Router) {
      const canKeepRoute = createRouteRecordMatcher(router);
      const pinnedKeys = readPinnedTabKeys();
      logTabsDebug(
        'tabs.store',
        () =>
          `tabs debug: healPersistedRoutes before active=${this.activeTabKey} ${formatTabsSummary(this.tabRouters)}`,
      );
      const nextTabs = this.tabRouters.filter(canKeepRoute);

      this.tabRouterList = ensureNonEmptyTabs(nextTabs, pinnedKeys);
      if (!this.tabRouterList.some((route) => getTabKey(route) === this.activeTabKey)) {
        this.activeTabKey = getTabKey(this.tabRouterList[0]);
      }
      this.closedTabStack = removeLegacyTabs(this.closedTabStack)
        .filter(canKeepRoute)
        .slice(-MAX_CLOSED_TABS)
        .map(cloneTab);
      this.clearSnapshotsForMissingTabs();
      this.clearRefreshNonceForMissingTabs();
      this.syncPinnedTabsStorage();
      logTabsDebug(
        'tabs.store',
        () => `tabs debug: healPersistedRoutes after active=${this.activeTabKey} ${formatTabsSummary(this.tabRouters)}`,
      );
    },
    appendTabRouterList(newRoute: TRouterInfo) {
      logTabsDebug(
        'tabs.store',
        () =>
          `tabs debug: appendTabRouterList input active=${this.activeTabKey} incoming=[key=${getTabKey(
            newRoute,
          )} path=${newRoute.path} fullPath=${newRoute.fullPath || ''} name=${String(newRoute.name || '')} title=${formatTabDebugTitle(
            newRoute.title,
          )}] ${formatTabsSummary(this.tabRouters)}`,
      );
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
      logTabsDebug(
        'tabs.store',
        () => `tabs debug: appendTabRouterList after active=${this.activeTabKey} ${formatTabsSummary(this.tabRouters)}`,
      );
    },
    updateActiveTabTitle(
      expectedRouteName: RouteRecordName,
      route: Pick<RouteLocationNormalizedLoaded, 'name' | 'path'>,
      title: LocalizedTitle,
    ) {
      if (route.name !== expectedRouteName) {
        return;
      }

      const activeTab = this.tabRouterList.find((tab) => getTabKey(tab) === this.activeTabKey);
      if (!activeTab || activeTab.path !== route.path || activeTab.name !== route.name) {
        return;
      }

      this.tabRouterList = this.tabRouterList.map((tab) =>
        getTabKey(tab) === this.activeTabKey ? { ...tab, title } : tab,
      );
    },
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
    discardTabRouter(newRoute: TRouterInfo) {
      const { routeIdx, path, tabKey } = newRoute;
      const routeKey = tabKey || path;
      const target =
        routeIdx === undefined
          ? this.tabRouterList.find((route) => getTabKey(route) === routeKey)
          : (this.tabRouterList[routeIdx] ?? this.tabRouterList.find((route) => getTabKey(route) === routeKey));
      const targetKey = tabKey || (target ? getTabKey(target) : path);
      if (!targetKey) return;

      const wasActive = this.activeTabKey === targetKey;
      const nextActiveTab = wasActive ? this.getNextRouteAfterClose(targetKey) : null;

      this.tabRouterList = this.tabRouterList.filter((route) => getTabKey(route) !== targetKey);
      this.closedTabStack = this.closedTabStack.filter((route) => getTabKey(route) !== targetKey);
      this.clearPageSnapshot(targetKey);
      const remainingNonces = { ...this.refreshNonceByTabKey };
      delete remainingNonces[targetKey];
      this.refreshNonceByTabKey = remainingNonces;
      if (this.refreshingTabKey === targetKey) {
        this.refreshingTabKey = undefined;
      }
      if (wasActive) {
        this.activeTabKey = getTabKey(nextActiveTab ?? this.tabRouterList[0] ?? homeRoute[0]);
      }
      this.syncPinnedTabsStorage();
    },
    subtractTabRouterBehind(newRoute: TRouterInfo) {
      const { routeIdx } = newRoute;
      if (routeIdx === undefined) return;
      this.closeTabsByPredicate((route, index) => index > routeIdx && !route.isHome && !route.isPinned);
    },
    subtractTabRouterAhead(newRoute: TRouterInfo) {
      const { routeIdx } = newRoute;
      if (routeIdx === undefined) return;
      this.closeTabsByPredicate((route, index) => index < routeIdx && !route.isHome && !route.isPinned);
    },
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
      this.refreshingTabKey = undefined;
      this.refreshNonceByTabKey = {};
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
      logTabsDebug(
        'tabs.store',
        () =>
          `tabs debug: setActiveRoute input active=${this.activeTabKey} route=[path=${route.path} fullPath=${
            route.fullPath
          } name=${String(route.name || '')}] ${formatTabsSummary(this.tabRouters)}`,
      );
      const currentActiveTab = this.tabRouterList.find((tab) => getTabKey(tab) === this.activeTabKey);
      if (currentActiveTab && currentActiveTab.fullPath === route.fullPath) {
        logTabsDebug(
          'tabs.store',
          () =>
            `tabs debug: setActiveRoute skipped same fullPath active=${this.activeTabKey} ${formatTabsSummary(
              this.tabRouters,
            )}`,
        );
        return;
      }

      const activeTab =
        this.tabRouterList.find((tab) => !tab.isDuplicate && tab.fullPath === route.fullPath) ??
        this.tabRouterList.find((tab) => tab.fullPath === route.fullPath) ??
        this.tabRouterList.find((tab) => !tab.isDuplicate && tab.path === route.path) ??
        this.tabRouterList.find((tab) => tab.path === route.path);
      this.activeTabKey = activeTab ? getTabKey(activeTab) : route.path;
      logTabsDebug(
        'tabs.store',
        () =>
          `tabs debug: setActiveRoute after active=${this.activeTabKey} resolved=${
            activeTab
              ? `[key=${getTabKey(activeTab)} path=${activeTab.path} fullPath=${activeTab.fullPath || ''} name=${String(
                  activeTab.name || '',
                )} title=${formatTabDebugTitle(activeTab.title)}]`
              : 'null'
          } ${formatTabsSummary(this.tabRouters)}`,
      );
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
    clearRefreshNonceForMissingTabs() {
      const aliveKeys = new Set(this.tabRouterList.map(getTabKey));
      const nextNonceByTabKey: Record<string, number> = {};

      Object.entries(this.refreshNonceByTabKey).forEach(([tabKey, nonce]) => {
        if (aliveKeys.has(tabKey)) {
          nextNonceByTabKey[tabKey] = nonce;
        }
      });

      this.refreshNonceByTabKey = nextNonceByTabKey;
      if (this.refreshingTabKey && !aliveKeys.has(this.refreshingTabKey)) {
        this.refreshingTabKey = undefined;
      }
    },
  },
  persist: {
    afterHydrate: ({ store }) => {
      logTabsDebug(
        'tabs.store',
        () =>
          `tabs debug: persist afterHydrate active=${store.activeTabKey} tabs=${formatTabsSummary(
            store.tabRouterList,
          )} closed=${formatTabsSummary(store.closedTabStack)}`,
      );
    },
    pick: ['tabRouterList', 'closedTabStack', 'activeTabKey'],
  },
});
