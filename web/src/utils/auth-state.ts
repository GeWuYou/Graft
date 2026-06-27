import { STORAGE_KEY } from '@/contracts/storage/keys';

let accessToken = '';
let accessTokenExpiresAt = '';

type PersistedUserSession = {
  token?: string;
  expiresAt?: string;
};

/**
 * 读取已持久化的用户会话信息。
 *
 * @returns 本地存储中的会话数据；读取失败或不存在时返回空对象。
 */
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

/**
 * 获取当前访问令牌。
 *
 * @returns 当前可用的访问令牌字符串；如果内存和持久化会话中都不存在，则返回空字符串。
 */
export function getAccessToken() {
  if (accessToken) {
    return accessToken;
  }

  // `user` 是 Pinia persist 为 user store 保留的本地快照；当前阶段只依赖其中
  // 的 token / expiresAt 字段做启动期恢复，避免在 store 尚未 hydrate 前丢失会话信息。
  return readPersistedSession().token ?? '';
}

/**
 * 获取访问令牌的过期时间。
 *
 * @returns 过期时间字符串；如果未设置则返回空字符串。
 */
export function getAccessTokenExpiresAt() {
  if (accessTokenExpiresAt) {
    return accessTokenExpiresAt;
  }

  return readPersistedSession().expiresAt ?? '';
}

/**
 * 设置当前访问令牌。
 *
 * @param token - 要缓存的访问令牌
 */
export function setAccessToken(token: string) {
  accessToken = token;
}

/**
 * 设置内存中的访问令牌过期时间。
 *
 * @param expiresAt - 访问令牌的过期时间字符串
 */
export function setAccessTokenExpiresAt(expiresAt: string) {
  accessTokenExpiresAt = expiresAt;
}

/**
 * 清除当前访问令牌及其过期时间。
 */
export function clearAccessToken() {
  accessToken = '';
  accessTokenExpiresAt = '';
}
