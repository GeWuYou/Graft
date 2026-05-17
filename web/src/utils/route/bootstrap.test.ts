import { describe, expect, it } from 'vitest';

import { transformBootstrapMenusToRoutes } from './bootstrap';

describe('transformBootstrapMenusToRoutes', () => {
  it('只为当前 web 已接入的 bootstrap 菜单生成动态路由', () => {
    const routes = transformBootstrapMenusToRoutes([
      {
        code: 'role.list',
        title_key: 'menu.role_list.title',
        title: '角色管理',
        path: '/roles',
        icon: 'secured',
        permission: 'role.read',
      },
      {
        code: 'user.list',
        title_key: 'menu.user_list.title',
        title: '用户管理',
        path: '/users',
        icon: 'usergroup',
        permission: 'user.read',
      },
      {
        code: 'unknown.feature',
        title: '未知功能',
        path: '/unknown',
        icon: 'app',
        permission: '',
      },
    ]);

    expect(routes).toHaveLength(2);
    expect(routes[0]?.path).toBe('/roles');
    expect(routes[0]?.redirect).toBe('/roles/index');
    expect(routes[0]?.children?.[0]?.name).toBe('RoleListIndex');
    expect(routes[0]?.meta?.titleKey).toBe('menu.role_list.title');
    expect(routes[0]?.meta?.title).toEqual({ 'zh-CN': '角色管理', 'en-US': 'Role Management' });
    expect(routes[1]?.path).toBe('/users');
    expect(routes[1]?.redirect).toBe('/users/index');
    expect(routes[1]?.children?.[0]?.name).toBe('UserListIndex');
    expect(routes[1]?.meta?.titleKey).toBe('menu.user_list.title');
    expect(routes[1]?.meta?.title).toEqual({ 'zh-CN': '用户管理', 'en-US': 'User Management' });
  });
});
