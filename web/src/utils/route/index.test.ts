// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import type { RouteRecordRaw } from 'vue-router';

import { resolveRuntimeHomePath, RUNTIME_ENTRY_FALLBACK_PATH, RUNTIME_HOME_PATH } from './index';

describe('resolveRuntimeHomePath', () => {
  it('uses the shell workbench as the runtime home when bootstrap routes exist', () => {
    const routes: RouteRecordRaw[] = [
      {
        path: '/access-control',
        redirect: '/access-control/overview',
        children: [
          {
            path: 'overview',
            name: 'AccessControlOverviewIndex',
            component: async () => ({ default: {} }),
          },
        ],
      },
    ];

    expect(resolveRuntimeHomePath(routes)).toBe(RUNTIME_HOME_PATH);
  });

  it('falls back to the runtime exception page when no visible page is registered', () => {
    const routes: RouteRecordRaw[] = [
      {
        path: '',
        children: [
          {
            path: 'index',
            meta: { hidden: true },
            component: async () => ({ default: {} }),
          },
        ],
      },
    ];

    expect(resolveRuntimeHomePath(routes)).toBe(RUNTIME_ENTRY_FALLBACK_PATH);
  });
});
