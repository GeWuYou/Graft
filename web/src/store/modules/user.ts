import { defineStore } from 'pinia';

import { getBootstrap, login as loginApi, logout as logoutApi, refresh as refreshApi } from '@/api/auth';
import { API_CODE, type ApiResponseCode, type BootstrapResponse, type LoginResponse } from '@/api/model/authModel';
import { STORAGE_KEY } from '@/contracts/storage/keys';
import { i18n, supportedLocales } from '@/locales';
import { usePermissionStore } from '@/store';
import type { ApiRequestError } from '@/types/axios';
import { clearAccessToken, setAccessToken } from '@/utils/auth-state';
import { isApiRequestError, registerAuthSessionBridge } from '@/utils/request';
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
    mustChangePassword: false,
    pendingRestrictedRedirect: '',
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
      this.mustChangePassword = payload.must_change_password;
      setAccessToken(payload.access_token);
      this.userInfo = {
        name: payload.user.display_name || payload.user.username,
        username: payload.user.username,
        roles: [],
        // login 响应只负责建立会话；权限仍以后续 bootstrap 快照为准，因此这里
        // 保留现有 permissions，避免在 refresh/login 后短暂清空权限状态。
        permissions: this.userInfo.permissions,
      };
    },
    applyBootstrap(payload: BootstrapResponse) {
      this.bootstrapSnapshot = payload;
      this.bootstrapLoaded = true;
      this.mustChangePassword = payload.must_change_password;
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
        throw createAuthStateError(401, API_CODE.AUTH_TOKEN_MISSING, 'Missing access token');
      }
      // bootstrap 是前端恢复真实用户、权限、菜单和 locale 快照的唯一入口；
      // 非 force 模式下优先复用已加载快照，避免每次导航都重复请求。
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
      } catch (error) {
        // 如果会话已在请求层失败路径中被清空，这里不要再发第二次 refresh。
        if (!isRefreshableAuthError(error) || !this.token) {
          throw error;
        }

        // 当 access token 过期且 refresh cookie 仍有效时，先刷新 token 再强制
        // 重新拉取 bootstrap，保持路由守卫只消费最新后端契约快照。
        await this.refreshToken();
        return this.bootstrap(true);
      }
    },
    clearSessionState() {
      this.token = '';
      clearAccessToken();
      this.bootstrapLoaded = false;
      this.bootstrapSnapshot = null;
      this.mustChangePassword = false;
      this.pendingRestrictedRedirect = '';
      this.userInfo = { ...InitUserInfo };
    },
    setPendingRestrictedRedirect(path: string) {
      this.pendingRestrictedRedirect = path;
    },
    consumePendingRestrictedRedirect(fallbackPath: string) {
      const path = this.pendingRestrictedRedirect || fallbackPath;
      this.pendingRestrictedRedirect = '';
      return path;
    },
    handleAuthFailure() {
      this.clearSessionState();
      const permissionStore = usePermissionStore();
      void permissionStore.restoreRoutes();
    },
    async logout() {
      try {
        if (this.token) {
          await logoutApi();
        }
      } finally {
        this.handleAuthFailure();
      }
    },
  },
  persist: {
    afterHydrate: ({ store }) => {
      setAccessToken(store.token);
      const permissionStore = usePermissionStore();
      permissionStore.initRoutes();
    },
    key: STORAGE_KEY.USER_SESSION,
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
    localStorage.setItem(STORAGE_KEY.LOCALE, normalizedLocale);
  } catch {
    // 受限环境下 locale 同步允许降级为内存态。
  }
}

function isRefreshableAuthError(error: unknown) {
  return isApiRequestError(error) && error.status === 401 && error.code === API_CODE.AUTH_TOKEN_EXPIRED;
}

function createAuthStateError(status: number, code: ApiResponseCode, message: string): ApiRequestError {
  const error = new Error(message) as ApiRequestError;
  error.name = 'ApiRequestError';
  error.status = status;
  error.code = code;
  error.traceId = '';
  error.isApiRequestError = true;
  return error;
}

registerAuthSessionBridge({
  applyLoginResponse(payload) {
    useUserStore().applyLoginResponse(payload);
  },
  handleAuthFailure() {
    useUserStore().handleAuthFailure();
  },
});
