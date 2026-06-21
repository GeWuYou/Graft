import type { AxiosError, AxiosResponse } from 'axios';
import axios from 'axios';

import { API_CODE, type ApiResponseCode } from '@/contracts/api/codes';
import type { ApiEnvelope, ApiErrorEnvelope } from '@/contracts/api/envelope';
import { AUTH_SCHEME, HTTP_HEADER } from '@/contracts/api/headers';
import { MESSAGE_KEY } from '@/contracts/api/messages';
import { getDefaultLocale, normalizeLocale } from '@/contracts/i18n/locales';
import { STORAGE_KEY } from '@/contracts/storage/keys';
import { i18n } from '@/locales';
import { AUTH_API_PATH } from '@/modules/auth/contract/paths';
import { AUTH_ROUTE_PATH } from '@/modules/auth/contract/routes';
import type { LoginResponse } from '@/modules/auth/contract/types';
import type { ApiRequestError, AxiosRequestConfigRetry, RequestOptions } from '@/types/axios';
import { clearAccessToken, getAccessToken, setAccessToken } from '@/utils/auth-state';
import { patchGlobalLoggerContext } from '@/utils/logger';

type RequestConfig = AxiosRequestConfigRetry & {
  requestOptions?: RequestOptions;
};

interface RequestInstance {
  get<T>(config: RequestConfig): Promise<T>;
  post<T>(config: RequestConfig): Promise<T>;
  put<T>(config: RequestConfig): Promise<T>;
  delete<T>(config: RequestConfig): Promise<T>;
}

type AuthSessionBridge = {
  applyLoginResponse(payload: LoginResponse): void | Promise<void>;
  handleAuthFailure(): void | Promise<void>;
};

const AUTH_REFRESH_URL = AUTH_API_PATH.REFRESH;
let authSessionBridge: AuthSessionBridge | null = null;

export function serializeRequestParams(params: Record<string, unknown>) {
  const searchParams = new URLSearchParams();

  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null) {
      continue;
    }

    if (Array.isArray(value)) {
      for (const item of value) {
        if (item === undefined || item === null || item === '') {
          continue;
        }
        searchParams.append(key, String(item));
      }
      continue;
    }

    if (value === '') {
      continue;
    }

    searchParams.append(key, String(value));
  }

  return searchParams.toString();
}

function resolveBaseURL() {
  if (import.meta.env.VITE_IS_REQUEST_PROXY === 'true') {
    return '';
  }

  const apiTarget = import.meta.env.VITE_API_TARGET || '';
  return apiTarget.replace(/\/+$/, '');
}

const client = axios.create({
  baseURL: resolveBaseURL(),
  withCredentials: true,
  paramsSerializer: {
    serialize: serializeRequestParams,
  },
});

client.interceptors.request.use((config) => {
  const headers = config.headers ?? {};
  const accessToken = getAccessToken();

  if (accessToken) {
    headers[HTTP_HEADER.AUTHORIZATION] = `${AUTH_SCHEME.BEARER} ${accessToken}`;
  }

  const runtimeLocale = normalizeLocale(i18n.global.locale.value);

  try {
    const storedLocale = normalizeLocale(localStorage.getItem(STORAGE_KEY.LOCALE));
    headers[HTTP_HEADER.LOCALE] = runtimeLocale ?? storedLocale ?? getDefaultLocale();
  } catch {
    headers[HTTP_HEADER.LOCALE] = runtimeLocale ?? getDefaultLocale();
  }

  config.headers = headers;
  return config;
});

client.interceptors.response.use(
  async (response) => unwrapResponse(response),
  async (error: AxiosError<ApiErrorEnvelope>) => {
    const requestError = normalizeAxiosError(error);
    const config = error.config as AxiosRequestConfigRetry | undefined;

    if (shouldRefresh(requestError, config)) {
      return tryRefreshAndReplay(config!);
    }

    if (shouldExitToLogin(requestError)) {
      await clearClientSession();
    }

    throw requestError;
  },
);

async function requestWithMethod<T>(method: 'get' | 'post' | 'put' | 'delete', config: RequestConfig): Promise<T> {
  const response = await client.request<T>({
    method,
    ...config,
  });
  return response as T;
}

function unwrapResponse<T>(response: AxiosResponse<T | ApiEnvelope<T>>): T {
  const payload = response.data;

  if (!isApiEnvelope(payload)) {
    return payload as T;
  }

  if (!payload.success) {
    syncLoggerCorrelation(payload.traceId);
    throw buildApiRequestError(response.status, payload);
  }

  syncLoggerCorrelation(payload.traceId);
  return payload.data;
}

function isApiEnvelope<T>(payload: unknown): payload is ApiEnvelope<T> {
  if (!payload || typeof payload !== 'object') {
    return false;
  }

  const candidate = payload as Partial<ApiEnvelope<T>>;
  return (
    typeof candidate.success === 'boolean' &&
    typeof candidate.code === 'string' &&
    typeof candidate.message === 'string'
  );
}

function normalizeAxiosError(error: AxiosError<ApiErrorEnvelope>): ApiRequestError {
  const status = error.response?.status ?? 0;
  const payload = error.response?.data;

  if (payload && isApiEnvelope(payload) && !payload.success) {
    syncLoggerCorrelation(payload.traceId);
    return buildApiRequestError(status, payload);
  }

  const fallbackMessage = error.message || i18n.global.t('app.request.failed');
  return buildApiRequestError(status, {
    success: false,
    code: API_CODE.COMMON_INTERNAL_ERROR,
    message: fallbackMessage,
    traceId: '',
  });
}

function buildApiRequestError(status: number, payload: ApiErrorEnvelope): ApiRequestError {
  const error = new Error(payload.message) as ApiRequestError;
  error.name = 'ApiRequestError';
  error.status = status;
  error.code = payload.code;
  error.traceId = payload.traceId;
  error.messageKey = payload.messageKey;
  error.locale = payload.locale;
  error.responseData = payload;
  error.isApiRequestError = true;
  return error;
}

function syncLoggerCorrelation(traceId: string | undefined) {
  const normalized = (traceId || '').trim();
  if (!normalized) {
    return;
  }

  // syncLoggerCorrelation 按后端 request/audit 契约把同一个 normalized 同时写入
  // patchGlobalLoggerContext 的 requestId 与 traceId；当前运行时中 traceId 就是 request id，
  // 因此前端不能在这里再发明第二个相关性值。
  patchGlobalLoggerContext({
    requestId: normalized,
    traceId: normalized,
  });
}

function shouldRefresh(error: ApiRequestError, config?: AxiosRequestConfigRetry) {
  if (!config) {
    return false;
  }

  if (config._skipAuthRefresh || config._authRefreshAttempted) {
    return false;
  }

  if (config.url === AUTH_REFRESH_URL) {
    return false;
  }

  return error.status === 401 && error.code === API_CODE.AUTH_TOKEN_EXPIRED;
}

function shouldExitToLogin(error: ApiRequestError) {
  return (
    error.status === 401 && (error.code === API_CODE.AUTH_TOKEN_INVALID || error.code === API_CODE.AUTH_TOKEN_MISSING)
  );
}

async function tryRefreshAndReplay<T>(config: AxiosRequestConfigRetry) {
  try {
    const payload = await requestWithMethod<LoginResponse>('post', {
      url: AUTH_REFRESH_URL,
      _skipAuthRefresh: true,
    } as RequestConfig);
    await syncAuthStateAfterRefresh(payload);
  } catch (refreshError) {
    // 受限首次改密态会显式拒绝 refresh；此时保留当前受限会话，交给页面继续完成改密。
    if (isRestrictedPasswordChangeRefreshError(refreshError)) {
      throw refreshError;
    }

    // refresh 已经进入会话清理路径时，不再重复执行 store 侧副作用。
    if (!isApiRequestError(refreshError) || !shouldExitToLogin(refreshError)) {
      await clearClientSession();
    }
    throw refreshError;
  }

  return client.request<T>({
    ...config,
    _authRefreshAttempted: true,
  } as AxiosRequestConfigRetry);
}

async function syncAuthStateAfterRefresh(payload: LoginResponse) {
  if (authSessionBridge) {
    await authSessionBridge.applyLoginResponse(payload);
    return;
  }

  setAccessToken(payload.access_token);

  try {
    const raw = localStorage.getItem(STORAGE_KEY.USER_SESSION);
    if (raw) {
      const persisted = JSON.parse(raw) as Record<string, unknown>;
      localStorage.setItem(STORAGE_KEY.USER_SESSION, JSON.stringify({ ...persisted, token: payload.access_token }));
    }
  } catch {
    // 受限环境下允许只更新内存 token。
  }
}

async function clearClientSession() {
  let clearedByStore = false;

  if (authSessionBridge) {
    await authSessionBridge.handleAuthFailure();
    clearedByStore = true;
  }

  if (!clearedByStore) {
    clearAccessToken();

    try {
      localStorage.removeItem(STORAGE_KEY.USER_SESSION);
    } catch {
      // 受限环境下允许只清空内存 token。
    }
  }

  if (typeof window !== 'undefined' && window.location.pathname !== AUTH_ROUTE_PATH.LOGIN) {
    const redirect = encodeURIComponent(`${window.location.pathname}${window.location.search}${window.location.hash}`);
    window.location.replace(`${AUTH_ROUTE_PATH.LOGIN}?redirect=${redirect}`);
  }
}

// 仅提供 starter 页面所需的最小请求适配，避免引入完整请求基础设施。
export const request: RequestInstance = {
  get<T>(config: RequestConfig) {
    return requestWithMethod<T>('get', config);
  },
  post<T>(config: RequestConfig) {
    return requestWithMethod<T>('post', config);
  },
  put<T>(config: RequestConfig) {
    return requestWithMethod<T>('put', config);
  },
  delete<T>(config: RequestConfig) {
    return requestWithMethod<T>('delete', config);
  },
};

// registerAuthSessionBridge 让请求层显式复用 user store 的会话同步与清理入口，
// 避免动态 import store 带来的构建告警与双源登录态漂移。
export function registerAuthSessionBridge(bridge: AuthSessionBridge | null) {
  authSessionBridge = bridge;
}

export function isApiRequestError(error: unknown): error is ApiRequestError {
  return Boolean(error && typeof error === 'object' && (error as Partial<ApiRequestError>).isApiRequestError);
}

export function shouldAttemptRefreshByError(status: number, code: ApiResponseCode) {
  return status === 401 && code === API_CODE.AUTH_TOKEN_EXPIRED;
}

function isRestrictedPasswordChangeRefreshError(error: unknown) {
  return (
    isApiRequestError(error) &&
    error.status === 403 &&
    error.code === API_CODE.AUTH_FORBIDDEN &&
    error.messageKey === MESSAGE_KEY.AUTH_FORBIDDEN
  );
}
