import { defineStore } from 'pinia';

const SESSION_STORAGE_KEY = 'graft:session';

interface SessionPayload {
  token: string;
  userName: string;
  permissions: string[];
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
    return JSON.parse(raw) as SessionPayload;
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
 * Keeps only session data that is shared across pages.
 * The current implementation is intentionally mock-backed so the shell can develop
 * before the server-side login, user, permission, and menu endpoints land.
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
        permissions: ['dashboard.view'],
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
