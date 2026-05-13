import { beforeEach, describe, expect, it, vi } from 'vitest';

describe('setupApiClient', () => {
  beforeEach(() => {
    vi.resetModules();
    window.localStorage.clear();
  });

  it('injects the current locale into request headers', async () => {
    const { createTestingPinia } = await import('@/test/helpers');
    const { useLocaleStore } = await import('@/stores/locale');
    const { apiClient, setupApiClient } = await import('./client');

    const pinia = createTestingPinia();
    const localeStore = useLocaleStore(pinia);

    localeStore.setLocale('zh-CN');
    setupApiClient(pinia);

    const response = await apiClient.get('/locale-check', {
      adapter: async (config) => ({
        config,
        data: null,
        headers: {},
        status: 200,
        statusText: 'OK',
      }),
    });

    expect(response.config.headers?.get('Accept-Language')).toBe('zh-CN');
  });
});
