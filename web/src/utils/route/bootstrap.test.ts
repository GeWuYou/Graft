import { describe, expect, it } from 'vitest';

import { transformBootstrapMenusToRoutes } from './bootstrap';

describe('transformBootstrapMenusToRoutes', () => {
  it('只为当前 web 已接入的 bootstrap 菜单生成动态路由', () => {
    const routes = transformBootstrapMenusToRoutes([
      {
        code: 'user.list',
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

    expect(routes).toHaveLength(1);
    expect(routes[0]?.path).toBe('/users');
    expect(routes[0]?.redirect).toBe('/users/index');
    expect(routes[0]?.children?.[0]?.name).toBe('UserListIndex');
  });
});
