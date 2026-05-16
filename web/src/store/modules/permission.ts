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
  getters: {
    permissionCodes: (state) => {
      return state.bootstrapSnapshot?.permissions ?? [];
    },
    permissionCodeSet(): Set<string> {
      return new Set(this.permissionCodes);
    },
  },
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
    hasPermission(code: string) {
      if (!code) {
        return false;
      }

      return this.permissionCodeSet.has(code);
    },
    hasAnyPermission(codes: string[]) {
      if (codes.length === 0) {
        return false;
      }

      return codes.some((code) => this.permissionCodeSet.has(code));
    },
    hasAllPermissions(codes: string[]) {
      if (codes.length === 0) {
        return false;
      }

      return codes.every((code) => this.permissionCodeSet.has(code));
    },
  },
});

export function getPermissionStore() {
  return usePermissionStore(store);
}
