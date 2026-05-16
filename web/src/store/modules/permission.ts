import cloneDeep from 'lodash/cloneDeep';
import { defineStore } from 'pinia';
import type { RouteRecordRaw } from 'vue-router';

import type { BootstrapResponse } from '@/api/model/authModel';
import { store } from '@/store';
import { transformBootstrapMenusToRoutes } from '@/utils/route/bootstrap';

export const usePermissionStore = defineStore('permission', {
  state: () => ({
    whiteListRouters: ['/login'] as string[],
    bootstrapSnapshot: null as BootstrapResponse | null,
    routers: [] as RouteRecordRaw[],
    removeRoutes: [] as RouteRecordRaw[],
    asyncRoutes: [] as RouteRecordRaw[],
    routesInitialized: false,
  }),
  actions: {
    setBootstrapSnapshot(snapshot: BootstrapResponse | null) {
      this.bootstrapSnapshot = snapshot;
    },
    async initRoutes() {
      const accessedRouters = this.asyncRoutes;

      // 菜单展示只保留 bootstrap 动态菜单，避免继续暴露 starter 静态演示菜单。
      this.routers = cloneDeep(accessedRouters);
    },
    async buildAsyncRoutes() {
      this.asyncRoutes = transformBootstrapMenusToRoutes(this.bootstrapSnapshot?.menus ?? []);
      await this.initRoutes();
      this.routesInitialized = true;
      return this.asyncRoutes;
    },
    async restoreRoutes() {
      this.bootstrapSnapshot = null;
      this.asyncRoutes = [];
      this.removeRoutes = [];
      this.routesInitialized = false;
      await this.initRoutes();
    },
  },
});

export function getPermissionStore() {
  return usePermissionStore(store);
}
