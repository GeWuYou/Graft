import { describe, expect, it } from 'vitest';

import type { MenuRoute } from '@/utils/types';

import {
  findExpandedMenuPaths,
  flattenMixHeaderMenus,
  resolveMenuNavigationPath,
  resolveSidebarMotionMode,
  selectMixSidebarMenu,
} from './layout-navigation';

describe('layout navigation helpers', () => {
  it('uses the wide-table sidebar motion for container and log list routes', () => {
    expect(resolveSidebarMotionMode('/infrastructure/docker/containers')).toBe('wide-table');
    expect(resolveSidebarMotionMode('/observability/application-logs')).toBe('wide-table');
    expect(resolveSidebarMotionMode('/observability/access-logs')).toBe('wide-table');
    expect(resolveSidebarMotionMode('/security/audit')).toBe('wide-table');
  });

  it('uses the default sidebar motion outside wide-table list routes', () => {
    expect(resolveSidebarMotionMode('/infrastructure/docker/containers/container-1')).toBe('default');
    expect(resolveSidebarMotionMode('/observability/service-status')).toBe('default');
  });

  it('resolves a grouped monitor menu to the first visible leaf page', () => {
    const monitorMenu: MenuRoute = {
      path: '/server',
      children: [
        {
          path: 'overview',
          meta: { titleKey: 'menu.monitor.overview.title' },
        },
      ],
    };

    expect(resolveMenuNavigationPath(monitorMenu)).toBe('/server/overview');
  });

  it('prefers the route redirect when a menu entry already defines one', () => {
    const userMenu: MenuRoute = {
      path: '/security/users',
      redirect: '/security/users/index',
    };

    expect(resolveMenuNavigationPath(userMenu)).toBe('/security/users/index');
  });

  it('follows redirected child groups until the first visible leaf page', () => {
    const monitorMenu: MenuRoute = {
      path: '/security',
      redirect: '/security/audit',
      children: [
        {
          path: 'logs',
          meta: { titleKey: 'menu.audit.logs.title' },
        },
      ],
    };

    expect(resolveMenuNavigationPath(monitorMenu)).toBe('/security/audit');
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
          titleKey: 'menu.domain.observability.title',
        },
      },
    ]);

    expect(menus).toHaveLength(1);
    expect(menus[0]?.path).toBe('/server/overview');
    expect(menus[0]?.children).toEqual([]);
    expect(menus[0]?.redirect).toBeUndefined();
    expect(menus[0]?.meta?.single).toBe(true);
  });

  it('selects the owning mixed-layout sidebar branch for nested menu routes', () => {
    const menus: MenuRoute[] = [
      {
        path: 'domain.application',
        meta: { navigationTargetPath: '/applications/projects' },
        children: [
          {
            path: 'application.list',
            meta: { navigationTargetPath: '/applications/projects' },
          },
        ],
      },
      {
        path: 'domain.infrastructure',
        meta: { navigationTargetPath: '/infrastructure/runtime-targets' },
        children: [
          {
            path: 'runtime-target.list',
            meta: { navigationTargetPath: '/infrastructure/runtime-targets' },
          },
          {
            path: 'docker',
            children: [
              {
                path: 'container.list',
                meta: { navigationTargetPath: '/infrastructure/docker/containers' },
              },
            ],
          },
        ],
      },
    ];

    const sidebarMenu = selectMixSidebarMenu(menus, '/infrastructure/docker/containers/container-1');

    expect(sidebarMenu).toHaveLength(1);
    expect(sidebarMenu[0]?.path).toBe('domain.infrastructure');
    expect(sidebarMenu[0]?.meta?.expanded).toBe(true);
    expect(sidebarMenu[0]?.children).toEqual(menus[1]?.children);
  });

  it('keeps the full mixed-layout sidebar menu when no branch owns the route', () => {
    const menus: MenuRoute[] = [
      {
        path: 'domain.platform',
        meta: { navigationTargetPath: '/platform/scheduled-tasks' },
        children: [
          {
            path: 'scheduled-task.list',
            meta: { navigationTargetPath: '/platform/scheduled-tasks' },
          },
        ],
      },
    ];

    expect(selectMixSidebarMenu(menus, '/platforms/scheduled-tasks')).toBe(menus);
  });

  it('derives expanded submenu ancestors from the menu tree', () => {
    const expanded = findExpandedMenuPaths(
      [
        {
          path: '/security',
          children: [
            {
              path: 'audit',
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
      '/security/audit/access',
    );

    expect(expanded).toEqual(['/security', 'audit']);
  });

  it('keeps grouped parent menus expanded for descendant detail routes', () => {
    const expanded = findExpandedMenuPaths(
      [
        {
          path: '/ops',
          children: [
            {
              path: 'containers',
              meta: { titleKey: 'menu.docker.title' },
            },
          ],
        },
      ],
      '/ops/containers/container-1',
    );

    expect(expanded).toEqual(['/ops']);
  });

  it('uses bootstrap menu codes as the submenu expanded values', () => {
    const expanded = findExpandedMenuPaths(
      [
        {
          path: 'domain.platform',
          meta: { navigationTargetPath: '/platform/scheduled-tasks' },
          children: [
            {
              path: 'scheduled-task.list',
              meta: { navigationTargetPath: '/platform/scheduled-tasks' },
            },
          ],
        },
      ],
      '/platform/scheduled-tasks',
    );

    expect(expanded).toEqual(['domain.platform']);
  });
});
