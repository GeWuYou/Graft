import type { AxiosRequestConfig } from 'axios';
import axios from 'axios';

import type { RequestOptions } from '@/types/axios';

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
