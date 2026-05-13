import type { Pinia } from 'pinia';

import axios, { AxiosHeaders } from 'axios';

import { createLocaleRequestHeaders } from '@/app/i18n/request';
import { useLocaleStore } from '@/stores/locale';

export const apiClient = axios.create();

let localeHeaderInterceptorInstalled = false;

/**
 * 统一给共享 axios 实例透传当前 locale。
 * 后续各模块接入真实 API 时，应优先复用 `apiClient`，避免语言请求头分散遗漏。
 */
export function setupApiClient(pinia: Pinia) {
  if (localeHeaderInterceptorInstalled) {
    return;
  }

  apiClient.interceptors.request.use((config) => {
    const localeStore = useLocaleStore(pinia);
    const headers = AxiosHeaders.from(config.headers ?? {});
    const localeHeaders = createLocaleRequestHeaders(localeStore.locale);

    for (const [key, value] of Object.entries(localeHeaders)) {
      headers.set(key, value);
    }

    config.headers = headers;
    return config;
  });

  localeHeaderInterceptorInstalled = true;
}
