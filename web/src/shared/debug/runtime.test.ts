import { beforeEach, describe, expect, it, vi } from 'vitest';

const { loggerMocks } = vi.hoisted(() => ({
  loggerMocks: {
    debug: vi.fn(),
  },
}));

vi.mock('@/utils/logger', () => ({
  createLogger: () => loggerMocks,
}));

import { useDebugStore } from '@/store/modules/debug';
import { store } from '@/store/pinia';

import { emitDebugLog, formatDebugLine, initDebugRuntime, isDebugFlagEnabled } from './runtime';

describe('debug runtime', () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
    localStorage.clear();
    loggerMocks.debug.mockReset();
    const debugStore = useDebugStore(store);
    debugStore.clearRuntimeFlag();
  });

  it('resolves namespaced env flags through the debug store', async () => {
    vi.stubEnv('VITE_DEBUG_PROJECT_MONACO', 'true');

    const debugStore = useDebugStore(store);
    debugStore.recompute();

    expect(isDebugFlagEnabled('project.monaco')).toBe(true);
  });

  it('resolves the navigation diagnostic flag through the debug store', () => {
    vi.stubEnv('VITE_DEBUG_NAVIGATION', 'true');

    const debugStore = useDebugStore(store);
    debugStore.recompute();

    expect(isDebugFlagEnabled('navigation')).toBe(true);
  });

  it('resolves the management table layout diagnostic flag through the debug store', () => {
    vi.stubEnv('VITE_DEBUG_MANAGEMENT_TABLE_LAYOUT', 'true');

    const debugStore = useDebugStore(store);
    debugStore.recompute();

    expect(isDebugFlagEnabled('management.table-layout')).toBe(true);
  });

  it('resolves the project log diagnostic flag through the debug store', () => {
    vi.stubEnv('VITE_DEBUG_PROJECT_LOGS', 'true');

    const debugStore = useDebugStore(store);
    debugStore.recompute();

    expect(isDebugFlagEnabled('project.logs')).toBe(true);
  });

  it('resolves the project workspace diagnostic flag through the debug store', () => {
    vi.stubEnv('VITE_DEBUG_PROJECT_WORKSPACE', 'true');

    const debugStore = useDebugStore(store);
    debugStore.recompute();

    expect(isDebugFlagEnabled('project.workspace')).toBe(true);
  });

  it('resolves the project template diagnostic flag through the debug store', () => {
    vi.stubEnv('VITE_DEBUG_PROJECT_TEMPLATES', 'true');

    const debugStore = useDebugStore(store);
    debugStore.recompute();

    expect(isDebugFlagEnabled('project.templates')).toBe(true);
  });

  it('resolves the container raw JSON diagnostic flag through the debug store', () => {
    vi.stubEnv('VITE_DEBUG_CONTAINER_RAW_JSON', 'true');

    const debugStore = useDebugStore(store);
    debugStore.recompute();

    expect(isDebugFlagEnabled('container.raw-json')).toBe(true);
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

  it('emits a workspace startup line through the shared logger when the workspace diagnostic flag is enabled', () => {
    vi.stubEnv('VITE_DEBUG_PROJECT_WORKSPACE', 'true');
    const debugStore = useDebugStore(store);
    debugStore.recompute();

    initDebugRuntime();

    expect(loggerMocks.debug).toHaveBeenCalledWith(expect.stringContaining('[debug:project.workspace] runtime-ready'));
  });

  it('does not emit through the shared logger when the diagnostic flag is disabled', () => {
    emitDebugLog('project.monaco', 'worker-created', {
      kind: 'yaml',
    });

    expect(loggerMocks.debug).not.toHaveBeenCalled();
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
