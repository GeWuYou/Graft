import { STORAGE_KEY } from '@/contracts/storage/keys';

let accessToken = '';
let accessTokenExpiresAt = '';

type PersistedUserSession = {
  token?: string;
  expiresAt?: string;
};

function readPersistedSession(): PersistedUserSession {
  try {
    const raw = localStorage.getItem(STORAGE_KEY.USER_SESSION);
    if (!raw) {
      return {};
    }

    return JSON.parse(raw) as PersistedUserSession;
  } catch {
    return {};
  }
}

export function getAccessToken() {
  if (accessToken) {
    return accessToken;
  }

  // `user` 是 Pinia persist 为 user store 保留的本地快照；当前阶段只依赖其中
  // 的 token / expiresAt 字段做启动期恢复，避免在 store 尚未 hydrate 前丢失会话信息。
  return readPersistedSession().token ?? '';
}

export function getAccessTokenExpiresAt() {
  if (accessTokenExpiresAt) {
    return accessTokenExpiresAt;
  }

  return readPersistedSession().expiresAt ?? '';
}

export function setAccessToken(token: string) {
  accessToken = token;
}

export function setAccessTokenExpiresAt(expiresAt: string) {
  accessTokenExpiresAt = expiresAt;
}

export function clearAccessToken() {
  accessToken = '';
  accessTokenExpiresAt = '';
}
