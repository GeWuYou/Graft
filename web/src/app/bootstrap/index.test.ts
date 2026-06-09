// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest';

const healPersistedState = vi.fn();
const afterEachMock = vi.fn();
const registerRouteGuards = vi.fn();
const registerPermissionDirective = vi.fn();
const loggerError = vi.fn();
const patchGlobalLoggerContext = vi.fn();
const useMock = vi.fn();
const mountMock = vi.fn();

vi.mock('vue', () => ({
  createApp: () => ({
    config: {},
    use: useMock,
    mount: mountMock,
  }),
}));

vi.mock('tdesign-vue-next', () => ({
  default: {},
}));

vi.mock('@/App.vue', () => ({
  default: {},
}));

vi.mock('@/router', () => ({
  default: {
    currentRoute: {
      value: {
        path: '/login',
      },
    },
    afterEach: afterEachMock,
  },
}));

vi.mock('@/locales', () => ({
  i18n: {},
}));

vi.mock('@/store', () => ({
  store: {},
  useTabsRouterStore: () => ({
    healPersistedState,
  }),
}));

vi.mock('./route-guards', () => ({
  registerRouteGuards,
}));

vi.mock('./permission-directive', () => ({
  registerPermissionDirective,
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    withContext: () => ({
      error: loggerError,
    }),
  }),
  patchGlobalLoggerContext,
}));

describe('bootstrapApp', () => {
  beforeEach(() => {
    vi.resetModules();
    afterEachMock.mockReset();
    healPersistedState.mockReset();
    registerRouteGuards.mockReset();
    registerPermissionDirective.mockReset();
    loggerError.mockReset();
    patchGlobalLoggerContext.mockReset();
    useMock.mockReset();
    mountMock.mockReset();
  });

  it('heals persisted tab refresh residue before mounting the app', async () => {
    const { bootstrapApp } = await import('./index');

    bootstrapApp();

    expect(registerRouteGuards).toHaveBeenCalledTimes(1);
    expect(afterEachMock).toHaveBeenCalledTimes(1);
    expect(patchGlobalLoggerContext).toHaveBeenCalledWith({
      route: '/login',
    });
    expect(healPersistedState).toHaveBeenCalledTimes(1);
    expect(useMock).toHaveBeenCalledTimes(4);
    expect(registerPermissionDirective).toHaveBeenCalledTimes(1);
    expect(mountMock).toHaveBeenCalledWith('#app');
    expect(healPersistedState.mock.invocationCallOrder[0]).toBeLessThan(mountMock.mock.invocationCallOrder[0]);
  });
});
