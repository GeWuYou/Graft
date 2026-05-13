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
 * 这里先用前端安全的数据结构镜像后端菜单契约。
 * 保留 `plugin` 与 `permissionCode` 字段，后续切到服务端驱动菜单时，
 * 只需要替换数据来源，不必重写布局与权限映射。
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
