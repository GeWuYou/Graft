import { defineStore } from 'pinia';

const SESSION_STORAGE_KEY = 'graft:session';

interface SessionPayload {
  token: string;
  userName: string;
  permissions: string[];
}

const MOCK_ADMIN_PERMISSIONS = ['dashboard.view'] as const;

function isSessionPayload(value: unknown): value is SessionPayload {
  if (!value || typeof value !== 'object') {
    return false;
  }

  const session = value as Record<string, unknown>;

  return (
    typeof session.token === 'string' &&
    typeof session.userName === 'string' &&
    Array.isArray(session.permissions) &&
    session.permissions.every((permission) => typeof permission === 'string')
  );
}

function loadSession(): SessionPayload | null {
  if (typeof window === 'undefined') {
    return null;
  }

  const raw = window.localStorage.getItem(SESSION_STORAGE_KEY);

  if (!raw) {
    return null;
  }

  try {
    const parsed = JSON.parse(raw) as unknown;

    if (!isSessionPayload(parsed)) {
      window.localStorage.removeItem(SESSION_STORAGE_KEY);
      return null;
    }

    return parsed;
  } catch {
    window.localStorage.removeItem(SESSION_STORAGE_KEY);
    return null;
  }
}

function persistSession(payload: SessionPayload | null) {
  if (typeof window === 'undefined') {
    return;
  }

  if (!payload) {
    window.localStorage.removeItem(SESSION_STORAGE_KEY);
    return;
  }

  window.localStorage.setItem(SESSION_STORAGE_KEY, JSON.stringify(payload));
}

/**
 * 这里只保存跨页面共享的会话数据。
 * 当前实现故意保留 mock 支撑，让壳层在后端登录、用户、权限、菜单接口落地前
 * 仍能先稳定联通路由与权限流程。
 */
export const useAuthStore = defineStore('auth', {
  state: () => {
    const session = loadSession();

    return {
      token: session?.token ?? '',
      userName: session?.userName ?? '',
      permissions: session?.permissions ?? [],
    };
  },
  getters: {
    isAuthenticated: (state) => Boolean(state.token),
  },
  actions: {
    login(userName: string) {
      const session: SessionPayload = {
        token: 'mock-session-token',
        userName,
        permissions: [...MOCK_ADMIN_PERMISSIONS],
      };

      this.token = session.token;
      this.userName = session.userName;
      this.permissions = session.permissions;
      persistSession(session);
    },
    logout() {
      this.token = '';
      this.userName = '';
      this.permissions = [];
      persistSession(null);
    },
    hasPermission(permission?: string) {
      if (!permission) {
        return true;
      }

      return this.permissions.includes(permission);
    },
  },
});
