import { describe, expect, it } from 'vitest';

import type { MenuRoute } from '@/utils/types';

import { flattenMixHeaderMenus, resolveMenuNavigationPath } from './layout-navigation';

describe('layout navigation helpers', () => {
  it('resolves a grouped monitor menu to the first visible leaf page', () => {
    const monitorMenu: MenuRoute = {
      path: '/monitor',
      children: [
        {
          path: 'server-status',
          redirect: 'overview',
          children: [
            { path: 'overview', meta: { titleKey: 'menu.monitor.server_status.overview.title' } },
            { path: 'runtime', meta: { titleKey: 'menu.monitor.server_status.runtime.title' } },
          ],
        },
      ],
    };

    expect(resolveMenuNavigationPath(monitorMenu)).toBe('/monitor/server-status/overview');
  });

  it('prefers the route redirect when a menu entry already defines one', () => {
    const userMenu: MenuRoute = {
      path: '/users',
      redirect: '/users/index',
    };

    expect(resolveMenuNavigationPath(userMenu)).toBe('/users/index');
  });

  it('follows redirected child groups until the first visible leaf page', () => {
    const monitorMenu: MenuRoute = {
      path: '/monitor',
      redirect: '/monitor/server-status',
      children: [
        {
          path: 'server-status',
          redirect: 'overview',
          children: [
            { path: 'overview', meta: { titleKey: 'menu.monitor.server_status.overview.title' } },
            { path: 'runtime', meta: { titleKey: 'menu.monitor.server_status.runtime.title' } },
          ],
        },
      ],
    };

    expect(resolveMenuNavigationPath(monitorMenu)).toBe('/monitor/server-status/overview');
  });

  it('flattens mix header menus into direct leaf navigation targets', () => {
    const menus = flattenMixHeaderMenus([
      {
        path: '/monitor',
        redirect: '/monitor/server-status',
        children: [
          {
            path: 'server-status',
            redirect: 'overview',
            children: [{ path: 'overview' }],
          },
        ],
        meta: {
          titleKey: 'monitor.sectionTitle',
        },
      },
    ]);

    expect(menus).toHaveLength(1);
    expect(menus[0]?.path).toBe('/monitor/server-status/overview');
    expect(menus[0]?.children).toEqual([]);
    expect(menus[0]?.redirect).toBeUndefined();
    expect(menus[0]?.meta?.single).toBe(true);
  });
});
