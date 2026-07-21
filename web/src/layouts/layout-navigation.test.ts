import { describe, expect, it } from 'vitest';

import type { MenuRoute } from '@/utils/types';

import {
  findAllExpandedMenuPaths,
  findExpandedMenuPaths,
  flattenMixHeaderMenus,
  resolveMenuNavigationPath,
  resolveSidebarMotionMode,
  resolveSidebarPresentation,
  selectMixSidebarMenu,
} from './layout-navigation';

describe('layout navigation helpers', () => {
  it('maps shell density to desktop, compact rail, and drawer presentations', () => {
    expect(resolveSidebarPresentation('spacious')).toBe('desktop');
    expect(resolveSidebarPresentation('comfortable')).toBe('compact');
    expect(resolveSidebarPresentation('compact')).toBe('drawer');
  });

  it('uses wide-table motion for paged list routes', () => {
    expect(resolveSidebarMotionMode({ pageKind: 'list' })).toBe('wide-table');
    expect(resolveSidebarMotionMode({ pageKind: 'list', pageSurface: 'paged-table' })).toBe('wide-table');
  });

  it('uses explicit motion overrides and preserves default motion for non-paged surfaces', () => {
    expect(resolveSidebarMotionMode({ pageKind: 'overview', pageSurface: 'paged-table' })).toBe('default');
    expect(resolveSidebarMotionMode({ pageKind: 'overview', sidebarMotion: 'wide-table' })).toBe('wide-table');
    expect(resolveSidebarMotionMode({ pageKind: 'list', pageSurface: 'form-detail' })).toBe('default');
    expect(resolveSidebarMotionMode({ pageKind: 'list', sidebarMotion: 'default' })).toBe('default');
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

  it('collects every visible submenu and ignores hidden or single menu branches', () => {
    const menus: MenuRoute[] = [
      {
        path: 'domain.infrastructure',
        children: [
          {
            path: 'docker',
            children: [{ path: 'container.list' }],
          },
          {
            path: 'hidden-group',
            meta: { hidden: true },
            children: [{ path: 'hidden-child' }],
          },
          {
            path: 'direct-route',
            meta: { single: true },
            children: [{ path: 'not-rendered' }],
          },
        ],
      },
    ];

    expect(findAllExpandedMenuPaths(menus)).toEqual(['domain.infrastructure', 'docker']);
  });
});
