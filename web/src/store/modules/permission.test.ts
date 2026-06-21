import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/store', () => ({
  store: createPinia(),
}));

import { usePermissionStore } from './permission';

describe('usePermissionStore permission checks', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it('treats an empty all-permissions check as satisfied', () => {
    const store = usePermissionStore();

    expect(store.hasAllPermissions([])).toBe(true);
  });

  it('keeps an empty any-permission check explicit and false', () => {
    const store = usePermissionStore();

    expect(store.hasAnyPermission([])).toBe(false);
  });

  it('checks non-empty permission collections against the bootstrap snapshot', () => {
    const store = usePermissionStore();
    store.setBootstrapSnapshot({
      user: {
        id: 1,
        username: 'admin',
        display_name: 'Admin',
      },
      must_change_password: false,
      roles: ['admin'],
      permissions: ['role.read', 'role.update'],
      menus: [],
      locale: {
        current_locale: 'zh-CN',
        default_locale: 'zh-CN',
        fallback_locale: 'zh-CN',
        supported_locales: ['zh-CN'],
      },
    });

    expect(store.hasAllPermissions(['role.read'])).toBe(true);
    expect(store.hasAllPermissions(['role.read', 'role.create'])).toBe(false);
    expect(store.hasAnyPermission(['role.create', 'role.update'])).toBe(true);
  });

  it('keeps global notification routes out of the sidebar menu routers', async () => {
    const store = usePermissionStore();
    store.setBootstrapSnapshot({
      user: {
        id: 1,
        username: 'admin',
        display_name: 'Admin',
      },
      must_change_password: false,
      roles: ['admin'],
      permissions: ['notification.view'],
      menus: [],
      locale: {
        current_locale: 'zh-CN',
        default_locale: 'zh-CN',
        fallback_locale: 'zh-CN',
        supported_locales: ['zh-CN'],
      },
    });

    const routes = await store.buildAsyncRoutes();

    expect(routes.some((route) => route.path === '/notifications')).toBe(true);
    expect(store.globalRoutes.some((route) => route.path === '/notifications')).toBe(true);
    expect(store.routers.some((route) => route.path === '/notifications')).toBe(false);
  });
});
