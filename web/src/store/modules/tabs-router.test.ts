// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it } from 'vitest';

import { LOCALE } from '@/contracts/i18n/locales';

import { useTabsRouterStore } from './tabs-router';

describe('useTabsRouterStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
  });

  it('uses the neutral root entry as the preserved home tab', () => {
    const tabsRouterStore = useTabsRouterStore();

    expect(tabsRouterStore.tabRouters).toHaveLength(1);
    expect(tabsRouterStore.tabRouters[0]?.path).toBe('/');
    expect(tabsRouterStore.tabRouters[0]?.name).toBe('RootEntry');
    expect(tabsRouterStore.tabRouters[0]?.title).toEqual({
      [LOCALE.ZH_CN]: '工作台',
      [LOCALE.EN_US]: 'Workspace',
    });
  });

  it('keeps refresh state ephemeral and restores the tab after refresh completes', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      path: '/users',
      name: 'UserList',
      meta: {
        keepAlive: true,
      },
    });

    tabsRouterStore.startTabRefresh(1);
    tabsRouterStore.setPageSnapshot('/users', { filters: { keyword: 'alice' } });
    expect(tabsRouterStore.refreshing).toBe(true);
    expect(tabsRouterStore.tabRouters[1]?.isAlive).toBe(false);

    tabsRouterStore.finishTabRefresh(1);
    expect(tabsRouterStore.refreshing).toBe(false);
    expect(tabsRouterStore.tabRouters[1]?.isAlive).toBe(true);
  });

  it('clears a tab page snapshot when refreshing a tab', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      tabKey: '/users',
      path: '/users',
      name: 'UserList',
    });
    tabsRouterStore.setPageSnapshot('/users', { filters: { keyword: 'alice' } });

    tabsRouterStore.startTabRefresh(1);

    expect(tabsRouterStore.getPageSnapshot('/users')).toBeUndefined();
  });

  it('heals persisted refresh residue on startup', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      path: '/users',
      name: 'UserList',
      meta: {
        keepAlive: true,
      },
    });

    tabsRouterStore.startTabRefresh(1);
    expect(tabsRouterStore.refreshing).toBe(true);
    expect(tabsRouterStore.tabRouters[1]?.isAlive).toBe(false);

    tabsRouterStore.healPersistedState();
    expect(tabsRouterStore.refreshing).toBe(false);
    expect(tabsRouterStore.tabRouters[0]?.isAlive).toBe(true);
    expect(tabsRouterStore.tabRouters[1]?.isAlive).toBe(true);
  });

  it('heals an empty persisted tab list back to home', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.setActiveTabKey('/missing');
    tabsRouterStore.removeTabRouterList();
    tabsRouterStore.healPersistedState();

    expect(tabsRouterStore.tabRouters.map((route) => route.path)).toEqual(['/']);
    expect(tabsRouterStore.activeTabKey).toBe('/');
  });

  it('resets the active tab after route healing removes stale tabs', () => {
    const tabsRouterStore = useTabsRouterStore();
    const router = {
      getRoutes: () => [{ name: 'RootEntry', path: '/' }],
    };

    tabsRouterStore.appendTabRouterList({
      tabKey: '/removed',
      path: '/removed',
      name: 'RemovedRoute',
    });
    tabsRouterStore.setActiveTabKey('/removed');
    tabsRouterStore.healPersistedRoutes(router as never);

    expect(tabsRouterStore.tabRouters.map((route) => route.path)).toEqual(['/']);
    expect(tabsRouterStore.activeTabKey).toBe('/');
  });

  it('activates the preserved home tab from another active tab', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      tabKey: '/server/system-config',
      path: '/server/system-config',
      name: 'SystemConfigList',
    });
    tabsRouterStore.setActiveTabKey('/server/system-config');

    tabsRouterStore.activateHomeTab();

    expect(tabsRouterStore.tabRouters.map((route) => route.path)).toEqual(['/', '/server/system-config']);
    expect(tabsRouterStore.activeTabKey).toBe('/');
  });

  it('restores the home tab when activating home from a corrupted persisted tab list', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.removeTabRouterList();
    tabsRouterStore.appendTabRouterList({
      tabKey: '/server/system-config',
      path: '/server/system-config',
      name: 'SystemConfigList',
    });
    tabsRouterStore.setActiveTabKey('/server/system-config');

    tabsRouterStore.activateHomeTab();

    expect(tabsRouterStore.tabRouters[0]?.path).toBe('/');
    expect(tabsRouterStore.tabRouters[0]?.isHome).toBe(true);
    expect(tabsRouterStore.tabRouters.map((route) => route.path)).toEqual(['/', '/server/system-config']);
    expect(tabsRouterStore.activeTabKey).toBe('/');
  });

  it('pins tabs, keeps pinned tabs before normal tabs, and persists pinned keys', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      tabKey: '/audit/logs',
      path: '/audit/logs',
      name: 'AuditLogs',
    });
    tabsRouterStore.appendTabRouterList({
      tabKey: '/audit/overview',
      path: '/audit/overview',
      name: 'AuditOverview',
    });

    tabsRouterStore.togglePinnedTab('/audit/overview');

    expect(tabsRouterStore.tabRouters.map((route) => route.path)).toEqual(['/', '/audit/overview', '/audit/logs']);
    expect(tabsRouterStore.tabRouters[1]?.isPinned).toBe(true);
    expect(localStorage.getItem('tabs:pinned')).toBe(JSON.stringify(['/audit/overview']));
  });

  it('closes all closable tabs while preserving home and pinned tabs', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      tabKey: '/audit/overview',
      path: '/audit/overview',
      name: 'AuditOverview',
    });
    tabsRouterStore.appendTabRouterList({
      tabKey: '/audit/logs',
      path: '/audit/logs',
      name: 'AuditLogs',
    });
    tabsRouterStore.appendTabRouterList({
      tabKey: '/access/logs',
      path: '/access/logs',
      name: 'AccessLogs',
    });
    tabsRouterStore.togglePinnedTab('/audit/overview');

    tabsRouterStore.closeAllClosableTabs();

    expect(tabsRouterStore.tabRouters.map((route) => route.path)).toEqual(['/', '/audit/overview']);
    expect(tabsRouterStore.closedTabs.map((route) => route.path)).toEqual(['/audit/logs', '/access/logs']);
  });

  it('resolves the preserved home tab after closing every unpinned business tab', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      tabKey: '/access-control/roles',
      path: '/access-control/roles',
      name: 'RoleListIndex',
    });
    tabsRouterStore.setActiveTabKey('/access-control/roles');

    tabsRouterStore.closeAllClosableTabs();
    const nextRoute =
      tabsRouterStore.tabRouters.find((item) => item.tabKey === tabsRouterStore.activeTabKey) ??
      tabsRouterStore.tabRouters[0];

    expect(tabsRouterStore.tabRouters.map((route) => route.path)).toEqual(['/']);
    expect(tabsRouterStore.resolveNavigationTarget(nextRoute)).toEqual({
      path: '/',
      query: undefined,
    });
  });

  it('keeps pinned tabs when closing other tabs', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      tabKey: '/audit/overview',
      path: '/audit/overview',
      name: 'AuditOverview',
    });
    tabsRouterStore.appendTabRouterList({
      tabKey: '/audit/logs',
      path: '/audit/logs',
      name: 'AuditLogs',
    });
    tabsRouterStore.appendTabRouterList({
      tabKey: '/access/logs',
      path: '/access/logs',
      name: 'AccessLogs',
    });
    tabsRouterStore.togglePinnedTab('/audit/overview');

    tabsRouterStore.subtractTabRouterOther({ path: '/audit/logs', routeIdx: 2 });

    expect(tabsRouterStore.tabRouters.map((route) => route.path)).toEqual(['/', '/audit/overview', '/audit/logs']);
  });

  it('reopens the most recently closed tab with route state', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      tabKey: '/access/logs',
      path: '/access/logs',
      fullPath: '/access/logs?scope=failed-auth',
      name: 'AccessLogs',
      query: {
        scope: 'failed-auth',
      },
      title: {
        [LOCALE.ZH_CN]: '访问日志',
        [LOCALE.EN_US]: 'Access Logs',
      },
    });

    tabsRouterStore.subtractCurrentTabRouter({ tabKey: '/access/logs', path: '/access/logs', routeIdx: 1 });
    const restored = tabsRouterStore.reopenClosedTab();

    expect(restored?.path).toBe('/access/logs');
    expect(restored?.fullPath).toBe('/access/logs?scope=failed-auth');
    expect(restored?.query).toEqual({ scope: 'failed-auth' });
    expect(restored?.title?.[LOCALE.ZH_CN]).toBe('访问日志');
  });

  it('keeps at most twenty closed tabs', () => {
    const tabsRouterStore = useTabsRouterStore();

    Array.from({ length: 22 }).forEach((_, index) => {
      const tabPath = `/audit/logs/${index}`;
      tabsRouterStore.appendTabRouterList({
        tabKey: tabPath,
        path: tabPath,
        name: `AuditLog${index}`,
      });
      tabsRouterStore.subtractCurrentTabRouter({ tabKey: tabPath, path: tabPath, routeIdx: 1 });
    });

    expect(tabsRouterStore.closedTabs).toHaveLength(20);
    expect(tabsRouterStore.closedTabs[0]?.path).toBe('/audit/logs/2');
    expect(tabsRouterStore.closedTabs[19]?.path).toBe('/audit/logs/21');
  });

  it('duplicates a tab with a distinct tab key and copied route state', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      tabKey: '/audit/logs',
      path: '/audit/logs',
      fullPath: '/audit/logs?scope=failed-auth',
      name: 'AuditLogs',
      query: {
        scope: 'failed-auth',
      },
      title: {
        [LOCALE.ZH_CN]: '审计日志',
        [LOCALE.EN_US]: 'Audit Logs',
      },
    });

    const duplicated = tabsRouterStore.duplicateTab('/audit/logs');

    expect(duplicated?.path).toBe('/audit/logs');
    expect(duplicated?.tabKey).not.toBe('/audit/logs');
    expect(duplicated?.query).toEqual({ scope: 'failed-auth' });
    expect(duplicated?.title?.[LOCALE.ZH_CN]).toBe('审计日志(2)');
    expect(tabsRouterStore.tabRouters).toHaveLength(3);
  });

  it('duplicates a tab with a deep-copied page snapshot', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      tabKey: '/audit/logs',
      path: '/audit/logs',
      name: 'AuditLogs',
    });
    tabsRouterStore.setPageSnapshot('/audit/logs', {
      filters: {
        keyword: 'failed-auth',
      },
      pagination: {
        current: 2,
        pageSize: 20,
      },
    });

    const duplicated = tabsRouterStore.duplicateTab('/audit/logs');
    const duplicatedSnapshot = tabsRouterStore.getPageSnapshot<{
      filters: { keyword: string };
      pagination: { current: number; pageSize: number };
    }>(duplicated?.tabKey);

    expect(duplicatedSnapshot).toEqual({
      filters: {
        keyword: 'failed-auth',
      },
      pagination: {
        current: 2,
        pageSize: 20,
      },
    });

    tabsRouterStore.setPageSnapshot('/audit/logs', {
      filters: {
        keyword: 'source-only',
      },
    });

    expect(tabsRouterStore.getPageSnapshot(duplicated?.tabKey)).toEqual(duplicatedSnapshot);
  });

  it('clears page snapshots when closing tabs', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      tabKey: '/audit/logs',
      path: '/audit/logs',
      name: 'AuditLogs',
    });
    tabsRouterStore.appendTabRouterList({
      tabKey: '/access/logs',
      path: '/access/logs',
      name: 'AccessLogs',
    });
    tabsRouterStore.setPageSnapshot('/audit/logs', { filters: { keyword: 'audit' } });
    tabsRouterStore.setPageSnapshot('/access/logs', { filters: { keyword: 'access' } });

    tabsRouterStore.subtractCurrentTabRouter({ tabKey: '/audit/logs', path: '/audit/logs', routeIdx: 1 });

    expect(tabsRouterStore.getPageSnapshot('/audit/logs')).toBeUndefined();
    expect(tabsRouterStore.getPageSnapshot('/access/logs')).toEqual({ filters: { keyword: 'access' } });
  });

  it('keeps duplicate tabs separately addressable even when they share one route path', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      tabKey: '/access-control/permissions',
      path: '/access-control/permissions',
      fullPath: '/access-control/permissions',
      name: 'PermissionListIndex',
    });

    const duplicated = tabsRouterStore.duplicateTab('/access-control/permissions');

    expect(
      tabsRouterStore.tabRouters
        .filter((route) => route.path === '/access-control/permissions')
        .map((route) => route.tabKey),
    ).toEqual(['/access-control/permissions', duplicated?.tabKey]);
    expect(duplicated?.isDuplicate).toBe(true);
    expect(duplicated?.duplicatedFrom).toBe('/access-control/permissions');
  });

  it('keeps a duplicated tab active when the route path matches the source tab', () => {
    const tabsRouterStore = useTabsRouterStore();

    tabsRouterStore.appendTabRouterList({
      tabKey: '/audit/logs',
      path: '/audit/logs',
      fullPath: '/audit/logs?scope=failed-auth',
      name: 'AuditLogs',
    });

    const duplicated = tabsRouterStore.duplicateTab('/audit/logs');
    tabsRouterStore.setActiveTabKey(duplicated?.tabKey ?? '');
    tabsRouterStore.setActiveRoute({
      path: '/audit/logs',
      fullPath: '/audit/logs?scope=failed-auth',
    } as never);

    expect(tabsRouterStore.activeTabKey).toBe(duplicated?.tabKey);
  });
});
