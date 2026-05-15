import type { AxiosRequestConfig } from 'axios';
import axios from 'axios';

import { localeConfigKey } from '@/locales';
import type { RequestOptions } from '@/types/axios';
import { getAccessToken } from '@/utils/auth-state';

type RequestConfig = AxiosRequestConfig & {
  requestOptions?: RequestOptions;
};

interface RequestInstance {
  get<T>(config: RequestConfig): Promise<T>;
  post<T>(config: RequestConfig): Promise<T>;
  put<T>(config: RequestConfig): Promise<T>;
  delete<T>(config: RequestConfig): Promise<T>;
}

const client = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '',
  withCredentials: true,
});

client.interceptors.request.use((config) => {
  const headers = config.headers ?? {};
  const accessToken = getAccessToken();

  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`;
  }

  try {
    const storedLocale = localStorage.getItem(localeConfigKey);
    if (storedLocale) {
      headers['X-Graft-Locale'] = storedLocale.replace('_', '-');
    }
  } catch {
    // 受限环境下允许 locale 头缺省。
  }

  config.headers = headers;
  return config;
});

async function requestWithMethod<T>(method: 'get' | 'post' | 'put' | 'delete', config: RequestConfig): Promise<T> {
  const response = await client.request<T>({
    method,
    ...config,
  });
  return response.data;
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
