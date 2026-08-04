import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createMemoryHistory, createRouter } from 'vue-router';

import DiagnosticsPage from './diagnostics.vue';

const connectivityStore = vi.hoisted(() => ({
  customTargets: [],
  loadHistory: vi.fn(),
  loadReport: vi.fn(),
  loadTrace: vi.fn(),
  refresh: vi.fn(),
  runTarget: vi.fn(),
  running: false,
  targets: [],
}));

vi.mock('../../store/connectivity', () => ({
  useConnectivityStore: () => connectivityStore,
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    locale: { value: 'zh-CN' },
    t: (key: string) => key,
    te: () => false,
  }),
}));

function createRouterForTest() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/platform/network/:targetId',
        name: 'PlatformNetworkConnectivityDiagnosticsIndex',
        component: DiagnosticsPage,
      },
      {
        path: '/platform/network/outbound',
        name: 'PlatformNetworkOutboundIndex',
        component: { template: '<div />' },
      },
    ],
  });
}

describe('connectivity diagnostics page', () => {
  beforeEach(() => {
    connectivityStore.refresh.mockReset();
    connectivityStore.refresh.mockResolvedValue(undefined);
    connectivityStore.loadHistory.mockReset();
    connectivityStore.loadHistory.mockResolvedValue([]);
    connectivityStore.loadReport.mockReset();
    connectivityStore.loadTrace.mockReset();
    connectivityStore.runTarget.mockReset();
  });

  it('does not load diagnostics for a non-diagnostics route while leaving the page', async () => {
    const router = createRouterForTest();
    await router.push('/platform/network/platform-update');
    await router.isReady();
    mount(DiagnosticsPage, { global: { plugins: [router] } });
    await flushPromises();

    expect(connectivityStore.loadHistory).toHaveBeenCalledWith('platform-update');
    connectivityStore.loadHistory.mockClear();

    await router.push('/platform/network/outbound');
    await flushPromises();

    expect(connectivityStore.loadHistory).not.toHaveBeenCalled();
  });
});
