import { describe, expect, it } from 'vitest';

import type { MenuRoute } from '@/utils/types';

import { buildGlobalMenuSearchIndex, searchGlobalMenuItems } from './global-menu-search';

function createRoute(overrides: Partial<MenuRoute>): MenuRoute {
  return {
    path: '/placeholder',
    ...overrides,
  } as MenuRoute;
}

describe('useGlobalMenuSearch helpers', () => {
  it('flattens visible menu leaves and excludes hidden routes', () => {
    const routes = [
      createRoute({
        path: '/ops',
        meta: {
          title: { 'zh-CN': '运维管理', 'en-US': 'Operations' },
        },
        children: [
          createRoute({
            name: 'ContainerListIndex',
            path: 'containers',
            meta: {
              title: { 'zh-CN': '容器管理', 'en-US': 'Containers' },
              titleKey: 'container.list.title',
            },
          }),
          createRoute({
            name: 'ContainerDetailIndex',
            path: 'containers/:id',
            meta: {
              hidden: true,
              title: { 'zh-CN': '容器详情', 'en-US': 'Container Detail' },
              titleKey: 'container.detail.title',
            },
          }),
        ],
      }),
    ];

    const index = buildGlobalMenuSearchIndex(routes, { locale: 'zh-CN' });

    expect(index).toEqual([
      expect.objectContaining({
        navigationPath: '/ops/containers',
        parentTitles: ['运维管理'],
        path: '/ops/containers',
        title: '容器管理',
      }),
    ]);
  });

  it('filters empty-path groups and hidden menu routes', () => {
    const routes = [
      createRoute({
        path: '   ',
        meta: {
          title: { 'zh-CN': '无效分组', 'en-US': 'Invalid Group' },
        },
      }),
      createRoute({
        path: '/notifications',
        name: 'NotificationCenterIndex',
        meta: {
          hiddenMenu: true,
          title: { 'zh-CN': '通知中心', 'en-US': 'Notifications' },
          titleKey: 'notification.center.title',
        },
      }),
    ];

    const index = buildGlobalMenuSearchIndex(routes, { locale: 'zh-CN' });

    expect(index).toEqual([]);
  });

  it('deduplicates identical paths and keeps the first visible entry', () => {
    const routes = [
      createRoute({
        path: '/logs',
        meta: {
          title: { 'zh-CN': '日志中心', 'en-US': 'Logs' },
        },
        children: [
          createRoute({
            name: 'AppLogListIndex',
            path: 'application',
            meta: {
              title: { 'zh-CN': '应用日志', 'en-US': 'Application Logs' },
              titleKey: 'appLog.list.title',
            },
          }),
          createRoute({
            name: 'AppLogDuplicateIndex',
            path: 'application',
            meta: {
              title: { 'zh-CN': '应用日志副本', 'en-US': 'Application Logs Duplicate' },
              titleKey: 'appLog.duplicate.title',
            },
          }),
        ],
      }),
    ];

    const index = buildGlobalMenuSearchIndex(routes, { locale: 'zh-CN' });

    expect(index).toHaveLength(1);
    expect(index[0]?.title).toBe('应用日志');
  });

  it('matches title, parent title, and path with stable ranking', () => {
    const routes = [
      createRoute({
        path: '/ops',
        meta: {
          title: { 'zh-CN': '运维管理', 'en-US': 'Operations' },
        },
        children: [
          createRoute({
            name: 'ContainerListIndex',
            path: 'containers',
            meta: {
              title: { 'zh-CN': '容器管理', 'en-US': 'Containers' },
              titleKey: 'container.list.title',
            },
          }),
        ],
      }),
      createRoute({
        path: '/logs',
        meta: {
          title: { 'zh-CN': '日志中心', 'en-US': 'Logs' },
        },
        children: [
          createRoute({
            name: 'AccessLogListIndex',
            path: 'access',
            meta: {
              title: { 'zh-CN': '访问日志', 'en-US': 'Access Logs' },
              titleKey: 'accessLog.list.title',
            },
          }),
          createRoute({
            name: 'ApplicationLogListIndex',
            path: 'application',
            meta: {
              title: { 'zh-CN': '应用日志', 'en-US': 'Application Logs' },
              titleKey: 'appLog.list.title',
            },
          }),
        ],
      }),
    ];

    const index = buildGlobalMenuSearchIndex(routes, { locale: 'zh-CN' });

    expect(searchGlobalMenuItems(index, '容器').map((item) => item.title)).toEqual(['容器管理']);
    expect(searchGlobalMenuItems(index, '运维').map((item) => item.title)).toEqual(['容器管理']);
    expect(searchGlobalMenuItems(index, 'application').map((item) => item.title)).toEqual(['应用日志']);
    expect(searchGlobalMenuItems(index, 'log').map((item) => item.title)).toEqual(['访问日志', '应用日志']);
  });
});
