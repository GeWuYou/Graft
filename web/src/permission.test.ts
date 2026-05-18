import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const messageError = vi.fn();
const addRoute = vi.fn();

const guardState = vi.hoisted(() => {
  const beforeEachHandlers: Array<(to: any, from: any, next: (arg?: any) => void) => unknown> = [];
  const afterEachHandlers: Array<(to: any) => unknown> = [];

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
    ],
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

vi.mock('@/router', () => ({
  RESTRICTED_SESSION_PATH: '/auth/restricted-session',
  RESTRICTED_SESSION_ROUTE_NAME: 'RestrictedSession',
  default: {
    addRoute,
    hasRoute: vi.fn(() => true),
    beforeEach: (handler: (to: any, from: any, next: (arg?: any) => void) => unknown) => {
      guardState.beforeEachHandlers.push(handler);
    },
    afterEach: (handler: (to: any) => unknown) => {
      guardState.afterEachHandlers.push(handler);
    },
  },
}));

vi.mock('@/store', () => ({
  getPermissionStore: () => storeState.permissionStore,
  useUserStore: () => storeState.userStore,
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

    expect(storeState.userStore.ensureBootstrap).toHaveBeenCalledTimes(1);
    expect(storeState.userStore.setPendingRestrictedRedirect).toHaveBeenCalledWith('/users?tab=active');
    expect(storeState.userStore.clearSessionState).not.toHaveBeenCalled();
    expect(storeState.permissionStore.restoreRoutes).not.toHaveBeenCalled();
    expect(next).toHaveBeenCalledWith({
      path: '/auth/restricted-session',
      replace: true,
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
});
