// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';
import type { RouteRecordRaw } from 'vue-router';

const authApiMocks = vi.hoisted(() => ({
  completeRequiredPasswordChange: vi.fn<() => Promise<void>>(),
}));

vi.mock('@/modules/auth/api/auth', () => ({
  completeRequiredPasswordChange: authApiMocks.completeRequiredPasswordChange,
}));

async function loadModule() {
  vi.resetModules();
  return import('./force-password-change');
}

const DummyPage = defineComponent({
  name: 'DummyPage',
  render: () => null,
});

describe('completeRestrictedPasswordChange', () => {
  beforeEach(() => {
    authApiMocks.completeRequiredPasswordChange.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('re-bootstraps after password change and restores the blocked target route', async () => {
    const { completeRestrictedPasswordChange } = await loadModule();
    const callLog: string[] = [];
    const asyncRoutes: RouteRecordRaw[] = [
      {
        path: '/users',
        name: 'UserList',
        component: DummyPage,
      },
    ];

    authApiMocks.completeRequiredPasswordChange.mockImplementation(async () => {
      callLog.push('completeRequiredPasswordChange');
    });

    await completeRestrictedPasswordChange({
      newPassword: 'Password12345',
      bootstrap: async (force) => {
        callLog.push(`bootstrap:${String(force)}`);
      },
      buildAsyncRoutes: async () => {
        callLog.push('buildAsyncRoutes');
        return asyncRoutes;
      },
      consumePendingRestrictedRedirect: (fallbackPath) => {
        callLog.push(`consume:${fallbackPath}`);
        return '/users?tab=active';
      },
      replace: async (path) => {
        callLog.push(`replace:${path}`);
      },
    });

    expect(callLog).toEqual([
      'completeRequiredPasswordChange',
      'bootstrap:true',
      'buildAsyncRoutes',
      'consume:/',
      'replace:/users?tab=active',
    ]);
    expect(authApiMocks.completeRequiredPasswordChange).toHaveBeenCalledWith({
      new_password: 'Password12345',
    });
  });

  it('falls back to the runtime home path when the pending redirect still points at the restricted route', async () => {
    const { completeRestrictedPasswordChange } = await loadModule();
    const replace = vi.fn();

    authApiMocks.completeRequiredPasswordChange.mockResolvedValue();

    await completeRestrictedPasswordChange({
      newPassword: 'Password12345',
      bootstrap: vi.fn(async () => undefined),
      buildAsyncRoutes: vi.fn(async () => [
        {
          path: '/users',
          name: 'UserList',
          component: DummyPage,
        },
      ]),
      consumePendingRestrictedRedirect: () => '/auth/restricted-session',
      replace,
    });

    expect(replace).toHaveBeenCalledWith('/');
  });
});
