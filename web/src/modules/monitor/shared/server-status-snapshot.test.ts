// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

import { flushPromises } from '@vue/test-utils';
import { mount } from '@vue/test-utils';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, nextTick } from 'vue';

import { resetMonitorRefreshPreferencesForTests } from '../composables/use-monitor-refresh-preferences';
import { useServerStatusSnapshot } from './server-status-snapshot';

const apiMocks = vi.hoisted(() => ({
  getServerStatus: vi.fn(),
}));

vi.mock('../api/server-status', () => ({
  getServerStatus: apiMocks.getServerStatus,
}));

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === 'app.refreshControl.status.off') {
        return 'Auto refresh off';
      }
      if (key === 'monitor.serverStatus.nextRefreshPausedByUser') {
        return 'Auto refresh paused';
      }
      if (key === 'monitor.serverStatus.nextRefreshPaused') {
        return 'Next refresh paused while the page is hidden';
      }
      if (key === 'monitor.serverStatus.nextRefreshPending') {
        return 'Preparing the next refresh';
      }
      if (key === 'monitor.serverStatus.nextRefreshIn') {
        return `Next refresh in ${String(params?.seconds ?? '')}s`;
      }
      if (key === 'monitor.serverStatus.nextRefreshRetryIn') {
        return `Retry in ${String(params?.seconds ?? '')}s`;
      }
      if (key === 'monitor.shared.loadFailed') {
        return 'Failed to load server status';
      }
      if (key === 'monitor.serverStatus.refreshInterval5Seconds') {
        return 'Every 5 sec';
      }
      if (key === 'monitor.serverStatus.refreshInterval10Seconds') {
        return 'Every 10 sec';
      }
      if (key === 'monitor.serverStatus.refreshInterval30Seconds') {
        return 'Every 30 sec';
      }
      if (key === 'monitor.serverStatus.refreshInterval1Minute') {
        return 'Every 1 min';
      }
      return key;
    },
  }),
}));

vi.mock('@/shared/localized-api-error', () => ({
  resolveLocalizedErrorMessage: vi.fn(() => 'Failed to load server status'),
}));

const Harness = defineComponent({
  name: 'ServerStatusSnapshotHarness',
  setup() {
    return useServerStatusSnapshot();
  },
  template: '<div />',
});

function setVisibilityState(state: 'visible' | 'hidden') {
  Object.defineProperty(document, 'visibilityState', {
    configurable: true,
    value: state,
  });
}

function createResponse() {
  return {
    observed_at: '2026-05-21T10:30:00Z',
  };
}

describe('useServerStatusSnapshot', () => {
  afterEach(() => {
    resetMonitorRefreshPreferencesForTests();
    vi.useRealTimers();
    vi.clearAllMocks();
    setVisibilityState('visible');
  });

  it('treats non-positive refresh intervals as off and does not schedule follow-up refreshes', async () => {
    vi.useFakeTimers();
    apiMocks.getServerStatus.mockResolvedValue(createResponse());

    const wrapper = mount(Harness);

    await flushPromises();
    await nextTick();

    expect(apiMocks.getServerStatus).toHaveBeenCalledTimes(1);

    (wrapper.vm as { selectedRefreshInterval: number }).selectedRefreshInterval = 0;
    await nextTick();

    expect(wrapper.vm.refreshControlStatus).toBe('off');
    expect(wrapper.vm.refreshCountdownText).toBe('Auto refresh off');
    expect(wrapper.vm.remainingRefreshSeconds).toBeNull();

    await vi.advanceTimersByTimeAsync(7000);
    await flushPromises();

    expect(apiMocks.getServerStatus).toHaveBeenCalledTimes(1);

    document.dispatchEvent(new Event('visibilitychange'));
    await flushPromises();

    expect(apiMocks.getServerStatus).toHaveBeenCalledTimes(1);

    wrapper.unmount();
  });
});
