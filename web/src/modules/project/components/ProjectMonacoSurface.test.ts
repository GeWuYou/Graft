import { flushPromises, mount } from '@vue/test-utils';
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

const PROJECT_MONACO_SURFACE_TEST_STATE_KEY = '__PROJECT_MONACO_SURFACE_TEST_STATE__';

function createMockState() {
  const callOrder: string[] = [];
  const models: Array<{
    dispose: ReturnType<typeof vi.fn>;
    getFullModelRange: ReturnType<typeof vi.fn>;
    getLanguageId: () => string;
    getValue: () => string;
    pushEditOperations: ReturnType<typeof vi.fn>;
    setValue: ReturnType<typeof vi.fn>;
    uri: string;
  }> = [];

  let currentModel: (typeof models)[number] | null = null;

  const createModel = vi.fn((value: string, language: string, uri: string) => {
    let currentValue = value;
    const model = {
      dispose: vi.fn(),
      getFullModelRange: vi.fn(() => ({ endColumn: 1, endLineNumber: 1, startColumn: 1, startLineNumber: 1 })),
      getLanguageId: () => language,
      getValue: () => currentValue,
      pushEditOperations: vi.fn((_, operations: Array<{ text: string }>) => {
        currentValue = operations[0]?.text ?? currentValue;
        return [];
      }),
      setValue: vi.fn((nextValue: string) => {
        currentValue = nextValue;
      }),
      uri,
    };
    models.push(model);
    return model;
  });

  const editor = {
    dispose: vi.fn(),
    getContainerDomNode: vi.fn(),
    getValue: vi.fn(() => currentModel?.getValue() ?? ''),
    layout: vi.fn(() => {
      callOrder.push('layout');
    }),
    onDidChangeModelContent: vi.fn((handler: () => void) => {
      void handler;
    }),
    setModel: vi.fn((nextModel) => {
      currentModel = nextModel;
      callOrder.push('setModel');
    }),
    updateOptions: vi.fn(),
  };

  const createEditor = vi.fn((container: HTMLElement, options?: { model?: (typeof models)[number] }) => {
    editor.getContainerDomNode.mockReturnValue(container);
    currentModel = options?.model ?? null;
    callOrder.push('create');
    return editor;
  });

  return {
    callOrder,
    createEditor,
    createLayoutController: vi.fn((options: { layout: () => void }) => ({
      disconnect: vi.fn(),
      observe: vi.fn(),
      schedule: vi.fn(() => {
        options.layout();
        return Promise.resolve();
      }),
    })),
    createModel,
    currentModel: () => currentModel,
    editor,
    models,
    reset() {
      callOrder.length = 0;
      models.length = 0;
      currentModel = null;
      createEditor.mockClear();
      this.createLayoutController.mockClear();
      createModel.mockClear();
      editor.dispose.mockClear();
      editor.getContainerDomNode.mockClear();
      editor.getValue.mockClear();
      editor.layout.mockClear();
      editor.onDidChangeModelContent.mockClear();
      editor.setModel.mockClear();
      editor.updateOptions.mockClear();
    },
  };
}

const mockState = createMockState();
(globalThis as typeof globalThis & Record<string, unknown>)[PROJECT_MONACO_SURFACE_TEST_STATE_KEY] = mockState;
const getMockState = () =>
  (globalThis as typeof globalThis & Record<string, unknown>)[PROJECT_MONACO_SURFACE_TEST_STATE_KEY] as ReturnType<
    typeof createMockState
  >;

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    error: vi.fn(),
    warn: vi.fn(),
  }),
}));

vi.mock('../shared/project-monaco-debug', () => ({
  createProjectMonacoDebugLogger: () => vi.fn(),
  describeProjectMonacoElement: () => 'div.test-host',
  disposeProjectMonacoModelDeferred: (
    targetModel: { dispose: () => void } | null,
    reason: string,
    handlers: { onDispose: (detail: Record<string, unknown>) => void },
  ) => {
    if (!targetModel) {
      return;
    }
    targetModel.dispose();
    handlers.onDispose({ reason });
  },
  formatProjectMonacoDebugMessage: (event: string) => event,
  isProjectMonacoDebugEnabled: () => false,
}));

vi.mock('../shared/project-monaco', async () => {
  const { onBeforeUnmount, onMounted } = await import('vue');

  return {
    buildProjectMonacoModelUri: (key: string, language: string, suffix?: string) =>
      `inmemory://${language}/${key}/${suffix ?? 'default'}`,
    createProjectMonacoModelUriSuffix: () => 'test-suffix',
    createProjectMonacoRelayoutBridge: (options: {
      layout: () => void;
      log: (event: string, detail: Record<string, unknown>) => void;
    }) => ({
      disconnect: vi.fn(),
      observe: vi.fn(),
      relayout: vi.fn(async (reason = 'manual') => {
        options.log('layout-scheduled', { reason });
        options.layout();
      }),
    }),
    disposeProjectMonacoModelCache: (
      cache: Map<string, { dispose: () => void }>,
      reason: string,
      disposeModel: (targetModel: { dispose: () => void }, reason: string) => void,
    ) => {
      for (const model of cache.values()) {
        disposeModel(model, reason);
      }
      cache.clear();
    },
    ensureProjectMonacoConfigured: () => ({
      editor: {
        create: getMockState().createEditor,
        createModel: getMockState().createModel,
        getModels: () => getMockState().models,
      },
    }),
    getOrCreateProjectMonacoModel: (
      monacoInstance: { editor: { createModel: (...args: any[]) => any } },
      options: { cache: Map<string, any>; key: string; language: string; suffix: string; value: string },
    ) => {
      const uri = `inmemory://${options.language}/${options.key}/${options.suffix}`;
      const existingModel = options.cache.get(uri);
      if (existingModel) {
        if (existingModel.getValue() !== options.value) {
          existingModel.setValue(options.value);
        }
        return existingModel;
      }

      const nextModel = monacoInstance.editor.createModel(options.value, options.language, uri);
      options.cache.set(uri, nextModel);
      return nextModel;
    },
    useProjectMonacoLifecycle: (options: { createEditor: () => void | Promise<void>; disposeEditor: () => void }) => {
      onMounted(() => {
        void options.createEditor();
      });
      onBeforeUnmount(() => {
        options.disposeEditor();
      });
      return {
        applyTheme: () => undefined,
      };
    },
  };
});

describe('ProjectMonacoSurface', () => {
  let ProjectMonacoSurface: any;

  beforeAll(async () => {
    ProjectMonacoSurface = (await import('./ProjectMonacoSurface.vue')).default;
  });

  beforeEach(() => {
    mockState.reset();
  });

  it('creates the editor with one model and relayouts once on mount', async () => {
    mount(ProjectMonacoSurface, {
      attachTo: document.body,
      props: {
        editorAriaLabel: 'Editor',
        language: 'yaml',
        modelKey: 'docker-compose.yml',
        modelValue: 'services:\n  api:\n    image: demo\n',
      },
    });

    await flushPromises();

    expect(mockState.callOrder).toEqual(['create', 'layout']);
    expect(mockState.createModel).toHaveBeenCalledTimes(1);
  });

  it('updates the current model value without recreating the model', async () => {
    const wrapper = mount(ProjectMonacoSurface, {
      props: {
        editorAriaLabel: 'Editor',
        language: 'yaml',
        modelKey: 'docker-compose.yml',
        modelValue: 'version: 1\n',
      },
    });

    await flushPromises();
    const initialModel = mockState.currentModel();
    expect(initialModel).not.toBeNull();

    await wrapper.setProps({
      modelValue: 'version: 2\n',
    });
    await flushPromises();

    expect(mockState.createModel).toHaveBeenCalledTimes(1);
    expect(initialModel?.pushEditOperations).toHaveBeenCalledTimes(1);
    expect(initialModel?.getValue()).toBe('version: 2\n');
  });

  it('reuses cached models when switching back to a previously opened file', async () => {
    const wrapper = mount(ProjectMonacoSurface, {
      props: {
        editorAriaLabel: 'Editor',
        language: 'yaml',
        modelKey: 'docker-compose.yml',
        modelValue: 'services:\n',
      },
    });

    await flushPromises();
    expect(mockState.createModel).toHaveBeenCalledTimes(1);

    await wrapper.setProps({
      modelKey: '.env',
      modelValue: 'A=1\n',
    });
    await flushPromises();
    expect(mockState.createModel).toHaveBeenCalledTimes(2);

    await wrapper.setProps({
      modelKey: 'docker-compose.yml',
      modelValue: 'services:\n',
    });
    await flushPromises();

    expect(mockState.createModel).toHaveBeenCalledTimes(2);
    expect(mockState.editor.setModel).toHaveBeenCalledTimes(2);

    await wrapper.unmount();
    expect(mockState.models[0]?.dispose).toHaveBeenCalledTimes(1);
    expect(mockState.models[1]?.dispose).toHaveBeenCalledTimes(1);
  });
});
