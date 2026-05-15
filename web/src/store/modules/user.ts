import { defineStore } from 'pinia';

import { getBootstrap, login as loginApi, logout as logoutApi, refresh as refreshApi } from '@/api/auth';
import type { BootstrapResponse, LoginResponse } from '@/api/model/authModel';
import { i18n, localeConfigKey, supportedLocales } from '@/locales';
import { usePermissionStore } from '@/store';
import { clearAccessToken, setAccessToken } from '@/utils/auth-state';
import type { UserInfo } from '@/utils/types';

const InitUserInfo: UserInfo = {
  name: '', // 用户名，用于展示在页面右上角头像处
  username: '',
  roles: [],
  permissions: [],
};

export const useUserStore = defineStore('user', {
  state: () => ({
    token: '',
    bootstrapLoaded: false,
    bootstrapSnapshot: null as BootstrapResponse | null,
    userInfo: { ...InitUserInfo },
  }),
  getters: {
    roles: (state) => {
      return state.userInfo?.roles;
    },
    permissions: (state) => {
      return state.userInfo?.permissions ?? [];
    },
  },
  actions: {
    applyLoginResponse(payload: LoginResponse) {
      this.token = payload.access_token;
      setAccessToken(payload.access_token);
      this.userInfo = {
        name: payload.user.display_name || payload.user.username,
        username: payload.user.username,
        roles: [],
        permissions: this.userInfo.permissions,
      };
    },
    applyBootstrap(payload: BootstrapResponse) {
      this.bootstrapSnapshot = payload;
      this.bootstrapLoaded = true;
      syncLocale(payload);
      this.userInfo = {
        name: payload.user.display_name || payload.user.username,
        username: payload.user.username,
        roles: [],
        permissions: payload.permissions,
      };
    },
    async login(userInfo: Record<string, unknown>) {
      const response = await loginApi({
        username: String(userInfo.account ?? ''),
        password: String(userInfo.password ?? ''),
      });
      this.applyLoginResponse(response);
      await this.bootstrap();
    },
    async bootstrap(force = false) {
      if (!this.token) {
        throw new Error('Missing access token');
      }
      if (this.bootstrapLoaded && this.bootstrapSnapshot && !force) {
        return this.bootstrapSnapshot;
      }

      const payload = await getBootstrap();
      this.applyBootstrap(payload);

      const permissionStore = usePermissionStore();
      permissionStore.setBootstrapSnapshot(payload);
      return payload;
    },
    async refreshToken() {
      const response = await refreshApi();
      this.applyLoginResponse(response);
      return response;
    },
    async ensureBootstrap() {
      try {
        return await this.bootstrap();
      } catch {
        await this.refreshToken();
        return this.bootstrap(true);
      }
    },
    clearSessionState() {
      this.token = '';
      clearAccessToken();
      this.bootstrapLoaded = false;
      this.bootstrapSnapshot = null;
      this.userInfo = { ...InitUserInfo };
    },
    async logout() {
      try {
        if (this.token) {
          await logoutApi();
        }
      } finally {
        this.clearSessionState();
      }
    },
  },
  persist: {
    afterHydrate: ({ store }) => {
      setAccessToken(store.token);
      const permissionStore = usePermissionStore();
      permissionStore.initRoutes();
    },
    key: 'user',
    pick: ['token'],
  },
});

function syncLocale(payload: BootstrapResponse) {
  const normalizedLocale = payload.locale.current_locale.replace('-', '_');
  if (!supportedLocales.includes(normalizedLocale as (typeof supportedLocales)[number])) {
    return;
  }

  i18n.global.locale.value = normalizedLocale;

  try {
    localStorage.setItem(localeConfigKey, normalizedLocale);
  } catch {
    // 受限环境下 locale 同步允许降级为内存态。
  }
}
