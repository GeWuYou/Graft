import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { API_CODE, type BootstrapResponse, type LoginResponse } from '@/api/model/authModel';
import { STORAGE_KEY } from '@/contracts/storage/keys';

const authApiMocks = vi.hoisted(() => ({
  getBootstrap: vi.fn<() => Promise<BootstrapResponse>>(),
  login: vi.fn(),
  logout: vi.fn(),
  refresh: vi.fn<() => Promise<LoginResponse>>(),
}));

const { mockPermissionStore } = vi.hoisted(() => ({
  mockPermissionStore: {
    setBootstrapSnapshot: vi.fn(),
    restoreRoutes: vi.fn(),
    initRoutes: vi.fn(),
  },
}));

vi.mock('@/api/auth', () => authApiMocks);
vi.mock('@/store', () => ({
  usePermissionStore: () => mockPermissionStore,
}));

async function loadUserStore() {
  vi.resetModules();
  return import('./user');
}

function createApiRequestError(status: number, code: string, message = code) {
  const error = new Error(message) as Error & {
    status: number;
    code: string;
    traceId: string;
    isApiRequestError: true;
  };
  error.name = 'ApiRequestError';
  error.status = status;
  error.code = code;
  error.traceId = 'trace-auth';
  error.isApiRequestError = true;
  return error;
}

function createBootstrapPayload(): BootstrapResponse {
  return {
    user: {
      id: 7,
      username: 'alice',
      display_name: 'Alice',
    },
    must_change_password: false,
    roles: ['admin'],
    permissions: ['user.read'],
    menus: [],
    locale: {
      current_locale: 'zh-CN',
      default_locale: 'zh-CN',
      fallback_locale: 'zh-CN',
      supported_locales: ['zh-CN', 'en-US'],
    },
  };
}

function createRefreshPayload(token = 'fresh-token'): LoginResponse {
  return {
    access_token: token,
    expires_at: '2026-05-15T00:00:00Z',
    must_change_password: false,
    user: {
      id: 7,
      username: 'alice',
      display_name: 'Alice',
    },
  };
}

describe('useUserStore.ensureBootstrap', () => {
  beforeEach(() => {
    authApiMocks.getBootstrap.mockReset();
    authApiMocks.login.mockReset();
    authApiMocks.logout.mockReset();
    authApiMocks.refresh.mockReset();
    mockPermissionStore.setBootstrapSnapshot.mockReset();
    mockPermissionStore.restoreRoutes.mockReset();
    mockPermissionStore.initRoutes.mockReset();
    localStorage.clear();
    setActivePinia(createPinia());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('refreshes once and retries bootstrap only when the first bootstrap fails with AUTH_TOKEN_EXPIRED', async () => {
    const { useUserStore } = await loadUserStore();
    const store = useUserStore();

    store.token = 'stale-token';
    authApiMocks.getBootstrap
      .mockRejectedValueOnce(createApiRequestError(401, API_CODE.AUTH_TOKEN_EXPIRED))
      .mockResolvedValueOnce(createBootstrapPayload());
    authApiMocks.refresh.mockResolvedValue(createRefreshPayload());

    const payload = await store.ensureBootstrap();

    expect(authApiMocks.getBootstrap).toHaveBeenCalledTimes(2);
    expect(authApiMocks.refresh).toHaveBeenCalledTimes(1);
    expect(store.token).toBe('fresh-token');
    expect(payload).toEqual(createBootstrapPayload());
  });

  it('does not refresh when bootstrap fails with a non-refreshable auth error', async () => {
    const { useUserStore } = await loadUserStore();
    const store = useUserStore();

    store.token = 'stale-token';
    authApiMocks.getBootstrap.mockRejectedValueOnce(createApiRequestError(401, API_CODE.AUTH_TOKEN_INVALID));

    await expect(store.ensureBootstrap()).rejects.toMatchObject({
      code: API_CODE.AUTH_TOKEN_INVALID,
      status: 401,
    });

    expect(authApiMocks.getBootstrap).toHaveBeenCalledTimes(1);
    expect(authApiMocks.refresh).not.toHaveBeenCalled();
  });

  it('does not retry refresh when bootstrap already cleared the session token', async () => {
    const { useUserStore } = await loadUserStore();
    const store = useUserStore();

    store.token = 'stale-token';
    authApiMocks.getBootstrap.mockImplementationOnce(async () => {
      store.clearSessionState();
      throw createApiRequestError(401, API_CODE.AUTH_TOKEN_EXPIRED);
    });

    await expect(store.ensureBootstrap()).rejects.toMatchObject({
      code: API_CODE.AUTH_TOKEN_EXPIRED,
      status: 401,
    });

    expect(authApiMocks.getBootstrap).toHaveBeenCalledTimes(1);
    expect(authApiMocks.refresh).not.toHaveBeenCalled();
  });

  it('preserves the restricted session when bootstrap returns AUTH_FORBIDDEN after login', async () => {
    const { useUserStore } = await loadUserStore();
    const store = useUserStore();

    authApiMocks.login.mockResolvedValue({
      ...createRefreshPayload('restricted-token'),
      must_change_password: true,
    });
    authApiMocks.getBootstrap.mockRejectedValueOnce(createApiRequestError(403, API_CODE.AUTH_FORBIDDEN));

    await expect(store.login({ account: 'graft', password: 'graft-admin' })).resolves.toBeUndefined();

    expect(store.token).toBe('restricted-token');
    expect(store.mustChangePassword).toBe(true);
    expect(authApiMocks.refresh).not.toHaveBeenCalled();
  });

  it('does not refresh or clear state for restricted-session AUTH_FORBIDDEN during ensureBootstrap', async () => {
    const { useUserStore } = await loadUserStore();
    const store = useUserStore();

    store.token = 'restricted-token';
    store.mustChangePassword = true;
    authApiMocks.getBootstrap.mockRejectedValueOnce(createApiRequestError(403, API_CODE.AUTH_FORBIDDEN));

    await expect(store.ensureBootstrap()).rejects.toMatchObject({
      code: API_CODE.AUTH_FORBIDDEN,
      status: 403,
    });

    expect(authApiMocks.refresh).not.toHaveBeenCalled();
    expect(store.token).toBe('restricted-token');
    expect(store.mustChangePassword).toBe(true);
  });
});

describe('useUserStore password change state', () => {
  beforeEach(() => {
    authApiMocks.getBootstrap.mockReset();
    authApiMocks.login.mockReset();
    authApiMocks.logout.mockReset();
    authApiMocks.refresh.mockReset();
    mockPermissionStore.setBootstrapSnapshot.mockReset();
    mockPermissionStore.restoreRoutes.mockReset();
    mockPermissionStore.initRoutes.mockReset();
    localStorage.clear();
    setActivePinia(createPinia());
  });

  it('tracks must_change_password from login and bootstrap responses', async () => {
    const { useUserStore } = await loadUserStore();
    const store = useUserStore();

    store.applyLoginResponse({
      ...createRefreshPayload('login-token'),
      must_change_password: true,
    });

    expect(store.mustChangePassword).toBe(true);

    store.applyBootstrap({
      ...createBootstrapPayload(),
      must_change_password: false,
    });

    expect(store.mustChangePassword).toBe(false);
  });

  it('persists the canonical locale from bootstrap snapshots', async () => {
    const { useUserStore } = await loadUserStore();
    const store = useUserStore();

    store.applyBootstrap({
      ...createBootstrapPayload(),
      locale: {
        current_locale: 'en-US',
        default_locale: 'zh-CN',
        fallback_locale: 'zh-CN',
        supported_locales: ['zh-CN', 'en-US'],
      },
    });

    expect(localStorage.getItem(STORAGE_KEY.LOCALE)).toBe('en-US');
  });

  it('clears must_change_password when the session is cleared', async () => {
    const { useUserStore } = await loadUserStore();
    const store = useUserStore();

    store.applyBootstrap({
      ...createBootstrapPayload(),
      must_change_password: true,
    });

    store.clearSessionState();

    expect(store.mustChangePassword).toBe(false);
    expect(store.bootstrapLoaded).toBe(false);
    expect(store.bootstrapSnapshot).toBeNull();
  });

  it('stores and consumes the pending restricted redirect once', async () => {
    const { useUserStore } = await loadUserStore();
    const store = useUserStore();

    store.setPendingRestrictedRedirect('/users?tab=active');

    expect(store.consumePendingRestrictedRedirect('/dashboard')).toBe('/users?tab=active');
    expect(store.consumePendingRestrictedRedirect('/dashboard')).toBe('/dashboard');
  });
});
