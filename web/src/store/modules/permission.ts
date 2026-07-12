import cloneDeep from 'lodash/cloneDeep';
import { defineStore } from 'pinia';
import type { RouteRecordRaw } from 'vue-router';

import { getGlobalRouteRegistrations } from '@/modules';
import { AUTH_ROUTE_PATH } from '@/modules/auth/contract/routes';
import type { BootstrapResponse } from '@/modules/auth/contract/types';
import { store } from '@/store/pinia';
import {
  buildBootstrapNavigationTree,
  transformBootstrapMenusToRoutes,
  transformGlobalRegistrationsToRoutes,
} from '@/utils/route/bootstrap';
import type { MenuRoute } from '@/utils/types';

export const usePermissionStore = defineStore('permission', {
  state: () => ({
    whiteListRouters: [AUTH_ROUTE_PATH.LOGIN] as string[],
    bootstrapSnapshot: null as BootstrapResponse | null,
    routers: [] as Array<RouteRecordRaw | MenuRoute>,
    removeRoutes: [] as RouteRecordRaw[],
    asyncRoutes: [] as RouteRecordRaw[],
    globalRoutes: [] as RouteRecordRaw[],
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
      this.routers = cloneDeep(buildBootstrapNavigationTree(this.bootstrapSnapshot?.menus ?? []));
    },
    async buildAsyncRoutes() {
      this.asyncRoutes = transformBootstrapMenusToRoutes(this.bootstrapSnapshot?.menus ?? []);
      this.globalRoutes = transformGlobalRegistrationsToRoutes(getGlobalRouteRegistrations());
      await this.initRoutes();
      this.routesInitialized = true;
      return [...this.asyncRoutes, ...this.globalRoutes];
    },
    async restoreRoutes() {
      this.bootstrapSnapshot = null;
      this.asyncRoutes = [];
      this.globalRoutes = [];
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
        return true;
      }

      return codes.every((code) => this.permissionCodeSet.has(code));
    },
  },
});

export function getPermissionStore() {
  return usePermissionStore(store);
}
