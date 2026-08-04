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

  it('discards a stale target history response after navigating to another target', async () => {
    let resolveFirstHistory: (value: Array<{ check_id: number }>) => void;
    const firstHistory = new Promise<Array<{ check_id: number }>>((resolve) => {
      resolveFirstHistory = resolve;
    });
    connectivityStore.loadHistory.mockImplementationOnce(() => firstHistory).mockResolvedValueOnce([]);
    const router = createRouterForTest();
    await router.push('/platform/network/platform-update');
    await router.isReady();
    mount(DiagnosticsPage, { global: { plugins: [router] } });
    await flushPromises();

    await router.push('/platform/network/marketplace');
    await flushPromises();
    resolveFirstHistory!([{ check_id: 42 }]);
    await flushPromises();

    expect(connectivityStore.loadHistory).toHaveBeenNthCalledWith(1, 'platform-update');
    expect(connectivityStore.loadHistory).toHaveBeenNthCalledWith(2, 'marketplace');
    expect(connectivityStore.loadReport).not.toHaveBeenCalledWith('platform-update', 42);
  });
});
