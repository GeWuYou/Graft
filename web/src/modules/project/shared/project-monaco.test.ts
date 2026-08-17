import { mount } from '@vue/test-utils';
import { afterAll, afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { defineComponent, h, KeepAlive, nextTick, ref } from 'vue';

import { useDebugStore } from '@/store/modules/debug';
import { store } from '@/store/pinia';

import { toMonacoColor } from './project-monaco-color';
import { isProjectMonacoBenignCancellationError, isProjectMonacoDebugEnabled } from './project-monaco-debug';
import { buildProjectMonacoWorker } from './project-monaco-worker';

const originalQueryCommandSupported = document.queryCommandSupported;
document.queryCommandSupported = vi.fn(() => false);
const { createProjectMonacoRelayoutBridge, useProjectMonacoLifecycle } = await import('./project-monaco');

function resetProjectMonacoDebugState() {
  const debugStore = useDebugStore(store);
  debugStore.clearRuntimeFlag();
  localStorage.clear();
}

beforeEach(() => {
  vi.unstubAllEnvs();
  resetProjectMonacoDebugState();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

afterAll(() => {
  document.queryCommandSupported = originalQueryCommandSupported;
});

describe('project-monaco color normalization', () => {
  it('converts srgb browser output into Monaco-safe hex', () => {
    expect(toMonacoColor('color(srgb 0.1379 0.197 0.2236 / 1)', '#1b2230')).toBe('#233239');
  });

  it('preserves alpha as rrggbbaa when the browser returns slash alpha rgb', () => {
    expect(toMonacoColor('rgb(95 155 255 / 0.28)', '#1b2230')).toBe('#5f9bff47');
  });

  it('falls back to the parsed fallback color when the computed value is invalid', () => {
    expect(toMonacoColor('definitely-not-a-color', '#1b2230')).toBe('#1b2230');
  });

  it('never returns raw css color syntax fragments', () => {
    const normalized = toMonacoColor('color(srgb 0.1379 0.197 0.2236 / 1)', '#1b2230');

    expect(normalized.startsWith('#')).toBe(true);
    expect(normalized.includes('color(')).toBe(false);
    expect(normalized.includes('#0.')).toBe(false);
  });
});

describe('project-monaco worker routing', () => {
  it('routes json models to the json worker factory', () => {
    const editorWorker = { kind: 'editor-worker' } as unknown as Worker;
    const jsonWorker = { kind: 'json-worker' } as unknown as Worker;
    const yamlWorker = { kind: 'yaml-worker' } as unknown as Worker;

    const worker = buildProjectMonacoWorker('monaco-editor/esm/vs/language/json/json.worker', 'json', {
      createEditorWorker: () => editorWorker,
      createJsonWorker: () => jsonWorker,
      createYamlWorker: () => yamlWorker,
    });

    expect(worker).toBe(jsonWorker);
  });

  it('routes yaml models to the yaml worker factory', () => {
    const editorWorker = { kind: 'editor-worker' } as unknown as Worker;
    const jsonWorker = { kind: 'json-worker' } as unknown as Worker;
    const yamlWorker = { kind: 'yaml-worker' } as unknown as Worker;

    const worker = buildProjectMonacoWorker('monaco-yaml/yaml.worker', 'yaml', {
      createEditorWorker: () => editorWorker,
      createJsonWorker: () => jsonWorker,
      createYamlWorker: () => yamlWorker,
    });

    expect(worker).toBe(yamlWorker);
  });

  it('routes non-yaml models to the editor worker factory', () => {
    const editorWorker = { kind: 'editor-worker' } as unknown as Worker;
    const jsonWorker = { kind: 'json-worker' } as unknown as Worker;
    const yamlWorker = { kind: 'yaml-worker' } as unknown as Worker;

    const worker = buildProjectMonacoWorker('monaco-editor/esm/vs/editor/editor.worker', 'editorWorkerService', {
      createEditorWorker: () => editorWorker,
      createJsonWorker: () => jsonWorker,
      createYamlWorker: () => yamlWorker,
    });

    expect(worker).toBe(editorWorker);
  });

  it('routes yaml workers by moduleId even when the label is missing', () => {
    const editorWorker = { kind: 'editor-worker' } as unknown as Worker;
    const jsonWorker = { kind: 'json-worker' } as unknown as Worker;
    const yamlWorker = { kind: 'yaml-worker' } as unknown as Worker;

    const worker = buildProjectMonacoWorker('monaco-yaml/yaml.worker', '', {
      createEditorWorker: () => editorWorker,
      createJsonWorker: () => jsonWorker,
      createYamlWorker: () => yamlWorker,
    });

    expect(worker).toBe(yamlWorker);
  });
});

describe('project-monaco debug toggle', () => {
  it('enables debug logging from the explicit namespaced env flag', async () => {
    vi.stubEnv('VITE_DEBUG_PROJECT_MONACO', 'true');

    const { useDebugStore: loadStore } = await import('@/store/modules/debug');
    const debugStore = loadStore(store);
    debugStore.recompute();

    expect(isProjectMonacoDebugEnabled()).toBe(true);
  });

  it('reads the runtime debug store override', () => {
    const debugStore = useDebugStore(store);
    debugStore.setRuntimeFlag('project.monaco', true);

    expect(isProjectMonacoDebugEnabled()).toBe(true);
  });
});

describe('project-monaco cancellation classification', () => {
  it('suppresses Monaco Delayer cancellation stacks from current build chunks', () => {
    const error = Object.assign(new Error('Canceled'), {
      name: 'Canceled',
      stack:
        'Error: Canceled\n at Delayer.cancel (chunk-current.js:1:1)\n at _DisposableStore.clear (chunk-current.js:2:2)',
    });

    expect(isProjectMonacoBenignCancellationError(error)).toBe(true);
  });

  it('keeps unrelated cancellation errors visible', () => {
    const error = Object.assign(new Error('Canceled'), {
      name: 'Canceled',
      stack: 'Error: Canceled\n at unrelatedRequest (app.js:1:1)',
    });

    expect(isProjectMonacoBenignCancellationError(error)).toBe(false);
  });
});

describe('project-monaco relayout bridge', () => {
  it('resolves relayout after the scheduled animation frame runs', async () => {
    const rafCallbacks: FrameRequestCallback[] = [];
    vi.stubGlobal(
      'requestAnimationFrame',
      vi.fn((callback: FrameRequestCallback) => {
        rafCallbacks.push(callback);
        return rafCallbacks.length;
      }),
    );

    const container = document.createElement('div');
    Object.defineProperties(container, {
      clientHeight: { configurable: true, value: 240 },
      clientWidth: { configurable: true, value: 480 },
    });

    const layout = vi.fn();
    const bridge = createProjectMonacoRelayoutBridge({
      getContainer: () => container,
      layout,
      log: vi.fn(),
    });

    let settled = false;
    const relayoutPromise = bridge.relayout('test-frame').then(() => {
      settled = true;
    });

    await Promise.resolve();

    expect(layout).not.toHaveBeenCalled();
    expect(settled).toBe(false);
    expect(rafCallbacks).toHaveLength(1);

    rafCallbacks[0]?.(0);
    await relayoutPromise;

    expect(layout).toHaveBeenCalledTimes(1);
    expect(settled).toBe(true);
  }, 20000);
});

describe('project-monaco lifecycle', () => {
  it('reapplies the active workspace theme when a KeepAlive surface is reactivated', async () => {
    const visible = ref(true);
    const host = document.createElement('div');
    host.style.setProperty('--graft-workspace-editor-surface', 'rgb(255 255 255)');
    document.body.appendChild(host);

    const setTheme = vi.fn();
    const monacoInstance = {
      editor: {
        defineTheme: vi.fn(),
        setTheme,
      },
    } as never;

    const MonacoLifecycleSurface = defineComponent({
      name: 'MonacoLifecycleSurface',
      setup() {
        useProjectMonacoLifecycle({
          createEditor: () => undefined,
          disposeEditor: () => undefined,
          getMonaco: () => monacoInstance,
          getThemeHost: () => host,
        });
        return () => h('div');
      },
    });

    const wrapper = mount(
      defineComponent({
        setup() {
          return () => h(KeepAlive, null, () => (visible.value ? h(MonacoLifecycleSurface) : null));
        },
      }),
    );
    await nextTick();
    const initialThemeApplications = setTheme.mock.calls.length;

    visible.value = false;
    await nextTick();
    visible.value = true;
    await nextTick();

    expect(setTheme.mock.calls.length).toBeGreaterThan(initialThemeApplications);
    wrapper.unmount();
    host.remove();
  }, 20000);
});
