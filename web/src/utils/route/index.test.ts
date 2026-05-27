import { describe, expect, it } from 'vitest';
import type { RouteRecordRaw } from 'vue-router';

import { resolveRuntimeHomePath, RUNTIME_ENTRY_FALLBACK_PATH } from './index';

describe('resolveRuntimeHomePath', () => {
  it('prefers the first visible registered runtime route', () => {
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

    expect(resolveRuntimeHomePath(routes)).toBe('/access-control');
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
