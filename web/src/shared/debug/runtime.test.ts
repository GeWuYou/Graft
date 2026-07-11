import { beforeEach, describe, expect, it, vi } from 'vitest';

import { useDebugStore } from '@/store/modules/debug';
import { store } from '@/store/pinia';

import { formatDebugLine, initDebugRuntime, isDebugFlagEnabled } from './runtime';

describe('debug runtime', () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
    localStorage.clear();
    const debugStore = useDebugStore(store);
    debugStore.clearRuntimeFlag();
  });

  it('resolves namespaced env flags through the debug store', async () => {
    vi.stubEnv('VITE_DEBUG_PROJECT_MONACO', 'true');

    const debugStore = useDebugStore(store);
    debugStore.recompute();

    expect(isDebugFlagEnabled('project.monaco')).toBe(true);
  });

  it('resolves the project log diagnostic flag through the debug store', () => {
    vi.stubEnv('VITE_DEBUG_PROJECT_LOGS', 'true');

    const debugStore = useDebugStore(store);
    debugStore.recompute();

    expect(isDebugFlagEnabled('project.logs')).toBe(true);
  });

  it('exposes a window developer API backed by the debug store', () => {
    initDebugRuntime();

    expect(window.__GRAFT_DEBUG__?.isEnabled('tabs')).toBe(false);
    expect(window.__GRAFT_DEBUG__?.enable('tabs.store')).toBe(true);
    expect(window.__GRAFT_DEBUG__?.isEnabled('tabs.store')).toBe(true);
    expect(window.__GRAFT_DEBUG__?.state().runtimeOverrides).toMatchObject({
      'tabs.store': true,
    });
  });

  it('formats debug lines as flat key-value output', () => {
    expect(
      formatDebugLine('project.monaco', 'worker-created', {
        kind: 'yaml',
        label: 'yaml',
        error: new Error('boom'),
      }),
    ).toContain('[debug:project.monaco] worker-created kind=yaml label=yaml error=Error:boom');
  });
});
