// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { RouteRecordRaw } from 'vue-router';

const messageError = vi.fn();
const addRoute = vi.fn();
const removeRoute = vi.fn();
const onErrorMock = vi.fn();
const startRouteLoading = vi.fn();
const finishRouteLoadingAfterRender = vi.fn();
const hideRouteLoading = vi.fn();

const guardState = vi.hoisted(() => {
  const beforeEachHandlers: Array<(to: any, from: any, next: (arg?: any) => void) => unknown> = [];
  const afterEachHandlers: Array<(to: any, from?: any) => unknown> = [];

  return {
    beforeEachHandlers,
    afterEachHandlers,
  };
});

const storeState = vi.hoisted(() => ({
  userStore: {
    token: 'restricted-token',
    mustChangePassword: true,
    pendingRestrictedRedirect: '',
    ensureBootstrap: vi.fn(),
    refreshToken: vi.fn(),
    clearSessionState: vi.fn(),
    setPendingRestrictedRedirect: vi.fn(function (this: any, path: string) {
      this.pendingRestrictedRedirect = path;
    }),
  },
  permissionStore: {
    whiteListRouters: ['/login'],
    routesInitialized: true,
    asyncRoutes: [
      {
        path: '/users',
        name: 'UserList',
      },
    ] as RouteRecordRaw[],
    globalRoutes: [] as RouteRecordRaw[],
    setBootstrapSnapshot: vi.fn(),
    buildAsyncRoutes: vi.fn(async function (this: any) {
      return this.asyncRoutes;
    }),
    restoreRoutes: vi.fn(),
  },
}));

vi.mock('nprogress', () => ({
  default: {
    configure: vi.fn(),
    start: vi.fn(),
    done: vi.fn(),
  },
}));

vi.mock('tdesign-vue-next', () => ({
  MessagePlugin: {
    error: messageError,
  },
}));

vi.mock('tdesign-vue-next/es/message', () => ({
  MessagePlugin: {
    error: messageError,
  },
}));

vi.mock('@/router', () => ({
  RESTRICTED_SESSION_PATH: '/auth/restricted-session',
  RESTRICTED_SESSION_ROUTE_NAME: 'RestrictedSession',
  default: {
    addRoute,
    removeRoute,
    hasRoute: vi.fn(() => true),
    beforeEach: (handler: (to: any, from: any, next: (arg?: any) => void) => unknown) => {
      guardState.beforeEachHandlers.push(handler);
    },
    afterEach: (handler: (to: any, from?: any) => unknown) => {
      guardState.afterEachHandlers.push(handler);
    },
    onError: onErrorMock,
  },
}));

vi.mock('@/router/route-loading', () => ({
  finishRouteLoadingAfterRender,
  hideRouteLoading,
  startRouteLoading,
}));

vi.mock('@/store', () => ({
  getPermissionStore: () => storeState.permissionStore,
}));

vi.mock('@/modules/auth/store', () => ({
  useAuthSessionStore: () => storeState.userStore,
}));

async function loadPermissionGuards() {
  vi.resetModules();
  guardState.beforeEachHandlers.length = 0;
  guardState.afterEachHandlers.length = 0;
  const { registerRouteGuards } = await import('@/app/bootstrap/route-guards');
  registerRouteGuards();
  return {
    beforeEach: guardState.beforeEachHandlers[0],
    afterEach: guardState.afterEachHandlers[0],
  };
}

describe('permission restricted session guard', () => {
  beforeEach(() => {
    addRoute.mockReset();
    removeRoute.mockReset();
    onErrorMock.mockReset();
    startRouteLoading.mockReset();
    finishRouteLoadingAfterRender.mockReset();
    hideRouteLoading.mockReset();
    messageError.mockReset();
    storeState.userStore.mustChangePassword = true;
    storeState.userStore.pendingRestrictedRedirect = '';
    storeState.userStore.ensureBootstrap.mockReset();
    storeState.userStore.refreshToken.mockReset();
    storeState.userStore.clearSessionState.mockReset();
    storeState.userStore.setPendingRestrictedRedirect.mockClear();
    storeState.permissionStore.setBootstrapSnapshot.mockReset();
    storeState.permissionStore.buildAsyncRoutes.mockClear();
    storeState.permissionStore.restoreRoutes.mockReset();
    storeState.permissionStore.routesInitialized = true;
    storeState.permissionStore.globalRoutes = [];
    storeState.userStore.ensureBootstrap.mockResolvedValue({
      must_change_password: true,
      roles: ['admin'],
      menus: [],
      permissions: [],
      locale: {
        current_locale: 'zh-CN',
        default_locale: 'zh-CN',
        fallback_locale: 'zh-CN',
        supported_locales: ['zh-CN'],
      },
      user: {
        id: 1,
        username: 'admin',
        display_name: 'Admin',
      },
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('blocks business routes during a restricted session without clearing the token', async () => {
    const { beforeEach } = await loadPermissionGuards();
    const next = vi.fn();

    await beforeEach(
      { path: '/users', fullPath: '/users?tab=active', name: 'UserList', query: { tab: 'active' } },
      { path: '/', fullPath: '/', query: {} },
      next,
    );

    expect(startRouteLoading).toHaveBeenCalledTimes(1);
    expect(storeState.userStore.ensureBootstrap).toHaveBeenCalledTimes(1);
    expect(storeState.userStore.setPendingRestrictedRedirect).toHaveBeenCalledWith('/users?tab=active');
    expect(storeState.userStore.clearSessionState).not.toHaveBeenCalled();
    expect(storeState.permissionStore.restoreRoutes).not.toHaveBeenCalled();
    expect(next).toHaveBeenCalledWith({
      path: '/auth/restricted-session',
      replace: true,
    });
  });

  it('does not show page loading for same-route query state changes', async () => {
    storeState.userStore.mustChangePassword = false;
    const { beforeEach, afterEach } = await loadPermissionGuards();
    const next = vi.fn();
    const from = {
      path: '/ops/containers/container-1',
      fullPath: '/ops/containers/container-1?tab=overview',
      name: 'ContainerDetail',
      query: { tab: 'overview' },
    };
    const to = {
      path: '/ops/containers/container-1',
      fullPath: '/ops/containers/container-1?tab=health',
      name: 'ContainerDetail',
      query: { tab: 'health' },
    };

    await beforeEach(to, from, next);
    await afterEach(to, from);

    expect(startRouteLoading).not.toHaveBeenCalled();
    expect(finishRouteLoadingAfterRender).not.toHaveBeenCalled();
    expect(next).toHaveBeenCalledWith();
  });

  it('replays the original deep link after dynamic bootstrap routes are mounted', async () => {
    storeState.userStore.mustChangePassword = false;
    storeState.permissionStore.routesInitialized = false;
    storeState.permissionStore.asyncRoutes = [
      {
        path: '/access-control',
        name: 'BootstrapGroupAccessControl',
        children: [
          {
            path: 'roles',
            name: 'RoleListIndex',
          },
        ],
      },
    ] as RouteRecordRaw[];
    const { beforeEach } = await loadPermissionGuards();
    const next = vi.fn();

    await beforeEach(
      {
        path: '/access-control/roles',
        fullPath: '/access-control/roles?type=custom',
        name: '404Page',
        query: { type: 'custom' },
        hash: '',
      },
      { path: '/', fullPath: '/', query: {} },
      next,
    );

    expect(storeState.permissionStore.buildAsyncRoutes).toHaveBeenCalledTimes(1);
    expect(addRoute).toHaveBeenCalledWith(storeState.permissionStore.asyncRoutes[0]);
    expect(next).toHaveBeenCalledWith({
      path: '/access-control/roles',
      replace: true,
      query: { type: 'custom' },
      hash: '',
    });
  });

  it('allows the restricted-session route itself without recording a new blocked target', async () => {
    const { beforeEach } = await loadPermissionGuards();
    const next = vi.fn();

    await beforeEach(
      {
        path: '/auth/restricted-session',
        fullPath: '/auth/restricted-session',
        name: 'RestrictedSession',
        query: {},
      },
      { path: '/users', fullPath: '/users', query: {} },
      next,
    );

    expect(storeState.userStore.setPendingRestrictedRedirect).not.toHaveBeenCalled();
    expect(next).toHaveBeenCalledWith();
  });

  it('removes mounted bootstrap routes when the session returns to login', async () => {
    storeState.permissionStore.asyncRoutes = [
      {
        path: '/users',
        name: 'UserList',
        children: [
          {
            path: 'index',
            name: 'UserListIndex',
          },
        ],
      },
    ] as RouteRecordRaw[];
    storeState.permissionStore.globalRoutes = [
      {
        path: '/notifications',
        name: 'NotificationList',
      },
    ] as RouteRecordRaw[];

    const { afterEach } = await loadPermissionGuards();

    await afterEach({ path: '/login' });

    expect(finishRouteLoadingAfterRender).toHaveBeenCalledTimes(1);
    expect(removeRoute).toHaveBeenNthCalledWith(1, 'NotificationList');
    expect(removeRoute).toHaveBeenNthCalledWith(2, 'UserListIndex');
    expect(removeRoute).toHaveBeenNthCalledWith(3, 'UserList');
    expect(storeState.permissionStore.restoreRoutes).toHaveBeenCalledTimes(1);
  });

  it('clears route loading when router navigation errors', async () => {
    await loadPermissionGuards();
    const errorHandler = onErrorMock.mock.calls[0]?.[0] as (() => void) | undefined;

    expect(errorHandler).toBeTypeOf('function');

    errorHandler?.();

    expect(hideRouteLoading).toHaveBeenCalledTimes(1);
  });
});
