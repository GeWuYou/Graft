import { describe, expect, it } from 'vitest';

import type { MenuRoute } from '@/utils/types';

import { findExpandedMenuPaths, flattenMixHeaderMenus, resolveMenuNavigationPath } from './layout-navigation';

describe('layout navigation helpers', () => {
  it('resolves a grouped monitor menu to the first visible leaf page', () => {
    const monitorMenu: MenuRoute = {
      path: '/server',
      children: [
        {
          path: 'overview',
          meta: { titleKey: 'menu.server.overview.title' },
        },
      ],
    };

    expect(resolveMenuNavigationPath(monitorMenu)).toBe('/server/overview');
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
      path: '/audit',
      redirect: '/audit/overview',
      children: [
        {
          path: 'overview',
          meta: { titleKey: 'menu.audit.overview.title' },
        },
        {
          path: 'logs',
          meta: { titleKey: 'menu.audit.logs.title' },
        },
      ],
    };

    expect(resolveMenuNavigationPath(monitorMenu)).toBe('/audit/overview');
  });

  it('flattens mix header menus into direct leaf navigation targets', () => {
    const menus = flattenMixHeaderMenus([
      {
        path: '/server',
        children: [
          {
            path: 'overview',
          },
        ],
        meta: {
          titleKey: 'menu.server.title',
        },
      },
    ]);

    expect(menus).toHaveLength(1);
    expect(menus[0]?.path).toBe('/server/overview');
    expect(menus[0]?.children).toEqual([]);
    expect(menus[0]?.redirect).toBeUndefined();
    expect(menus[0]?.meta?.single).toBe(true);
  });

  it('derives expanded submenu ancestors from the menu tree', () => {
    const expanded = findExpandedMenuPaths(
      [
        {
          path: '/audit',
          children: [
            {
              path: 'logs',
              children: [
                {
                  path: 'access',
                  meta: { titleKey: 'menu.accessLog.title' },
                },
              ],
            },
          ],
        },
      ],
      '/audit/logs/access',
    );

    expect(expanded).toEqual(['/audit', '/audit/logs']);
  });

  it('keeps grouped parent menus expanded for descendant detail routes', () => {
    const expanded = findExpandedMenuPaths(
      [
        {
          path: '/ops',
          children: [
            {
              path: 'containers',
              meta: { titleKey: 'menu.container.title' },
            },
          ],
        },
      ],
      '/ops/containers/container-1',
    );

    expect(expanded).toEqual(['/ops']);
  });
});
