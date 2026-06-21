import { STORAGE_KEY } from '@/contracts/storage/keys';

let accessToken = '';

export function getAccessToken() {
  if (accessToken) {
    return accessToken;
  }

  try {
    // `user` 是 Pinia persist 为 user store 保留的本地快照；当前阶段只依赖其中
    // 的 `token` 字段做启动期恢复，避免在 store 尚未 hydrate 前丢失 access token。
    const raw = localStorage.getItem(STORAGE_KEY.USER_SESSION);
    if (!raw) {
      return '';
    }

    const persisted = JSON.parse(raw) as { token?: string };
    return persisted.token ?? '';
  } catch {
    return '';
  }
}

export function setAccessToken(token: string) {
  accessToken = token;
}

export function clearAccessToken() {
  accessToken = '';
}
