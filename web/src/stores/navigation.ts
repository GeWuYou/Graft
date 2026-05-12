import { defineStore } from 'pinia';

export interface NavigationItem {
  title: string;
  path: string;
  icon?: string;
  plugin: string;
  permissionCode?: string;
}

const staticItems: NavigationItem[] = [
  {
    title: '仪表盘',
    path: '/dashboard',
    icon: 'dashboard',
    plugin: 'core',
    permissionCode: 'dashboard.view',
  },
];

/**
 * Mirrors the backend menu contract in a frontend-safe shape.
 * Keeping `plugin` and `permissionCode` explicit here makes the later switch to
 * server-driven menus a data-source replacement instead of a layout rewrite.
 */
export const useNavigationStore = defineStore('navigation', {
  state: () => ({
    items: staticItems,
    activePath: '/dashboard',
  }),
  actions: {
    setActivePath(path: string) {
      this.activePath = path;
    },
    firstAccessiblePath(permissions: string[]) {
      const item = this.items.find(
        (candidate) =>
          !candidate.permissionCode || permissions.includes(candidate.permissionCode),
      );

      return item?.path ?? '';
    },
  },
});
