import { flushPromises, mount } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import ProjectMonacoDiffSurface from './ProjectMonacoDiffSurface.vue';

const mockState = vi.hoisted(() => {
  const callOrder: string[] = [];
  const layoutReasons: string[] = [];
  const models: Array<{
    dispose: ReturnType<typeof vi.fn>;
    getLanguageId: () => string;
    getValue: () => string;
    setValue: ReturnType<typeof vi.fn>;
    uri: string;
  }> = [];

  let currentModel: {
    modified: (typeof models)[number];
    original: (typeof models)[number];
  } | null = null;

  const createModel = vi.fn((value: string, language: string, uri: string) => {
    let currentValue = value;
    const model = {
      dispose: vi.fn(),
      getLanguageId: () => language,
      getValue: () => currentValue,
      setValue: vi.fn((nextValue: string) => {
        currentValue = nextValue;
      }),
      uri,
    };
    models.push(model);
    return model;
  });

  const diffEditor = {
    dispose: vi.fn(),
    getContainerDomNode: vi.fn(),
    getModel: vi.fn(() => currentModel),
    layout: vi.fn(() => {
      callOrder.push('layout');
    }),
    setModel: vi.fn((model) => {
      callOrder.push(model ? 'setModel' : 'setModel:null');
      currentModel = model;
    }),
  };

  const createDiffEditor = vi.fn((container: HTMLElement) => {
    callOrder.push('create');
    diffEditor.getContainerDomNode.mockReturnValue(container);
    return diffEditor;
  });

  const disconnect = vi.fn();

  return {
    callOrder,
    createDiffEditor,
    createModel,
    currentModel: () => currentModel,
    diffEditor,
    disconnect,
    layoutReasons,
    models,
    reset() {
      callOrder.length = 0;
      layoutReasons.length = 0;
      models.length = 0;
      currentModel = null;
      createDiffEditor.mockClear();
      createModel.mockClear();
      diffEditor.dispose.mockClear();
      diffEditor.getContainerDomNode.mockClear();
      diffEditor.getModel.mockClear();
      diffEditor.layout.mockClear();
      diffEditor.setModel.mockClear();
      disconnect.mockClear();
    },
  };
});

vi.mock('@/utils/logger', () => ({
  createLogger: () => ({
    debug: vi.fn(),
    error: vi.fn(),
  }),
}));

vi.mock('../shared/project-monaco-debug', () => ({
  describeProjectMonacoElement: (element: HTMLElement | null) => element?.tagName.toLowerCase() ?? 'null',
  disposeProjectMonacoModelDeferred: (
    targetModel: { dispose: () => void } | null,
    _reason: string,
    handlers: {
      onCancellation: (detail: Record<string, unknown>) => void;
      onDispose: (detail: Record<string, unknown>) => void;
      onError: (error: Error, detail: Record<string, unknown>) => void;
    },
  ) => {
    if (!targetModel) {
      return;
    }

    try {
      targetModel.dispose();
      handlers.onDispose({});
    } catch (error) {
      handlers.onError(error instanceof Error ? error : new Error(String(error)), {});
    }
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
    ensureProjectMonacoConfigured: () => ({
      editor: {
        createDiffEditor: mockState.createDiffEditor,
        createModel: mockState.createModel,
        getModels: () => mockState.models,
      },
    }),
    observeProjectMonacoResize: () => ({
      disconnect: mockState.disconnect,
    }),
    scheduleProjectMonacoLayout: (layout: () => void) => {
      mockState.layoutReasons.push('scheduled');
      layout();
      return Promise.resolve();
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

describe('ProjectMonacoDiffSurface', () => {
  beforeEach(() => {
    mockState.reset();
  });

  it('creates the diff editor before binding models and relayouts again after binding', async () => {
    mount(ProjectMonacoDiffSurface, {
      attachTo: document.body,
      props: {
        editorAriaLabel: 'Diff Viewer',
        language: 'yaml',
        modifiedKey: 'modified.yml',
        modifiedValue: 'services:\n  api:\n    image: newer\n',
        originalKey: 'original.yml',
        originalValue: 'services:\n  api:\n    image: older\n',
      },
    });

    await flushPromises();

    expect(mockState.callOrder).toEqual(['create', 'layout', 'setModel', 'layout']);
    expect(mockState.createDiffEditor).toHaveBeenCalledTimes(1);
    expect(mockState.createModel).toHaveBeenCalledTimes(2);

    const boundModel = mockState.currentModel();
    expect(boundModel).not.toBeNull();
    expect(boundModel?.original.getValue()).toContain('older');
    expect(boundModel?.modified.getValue()).toContain('newer');
    expect(mockState.diffEditor.getModel()).toBe(boundModel);
  });

  it('rebinds both models when the diff identity changes', async () => {
    const wrapper = mount(ProjectMonacoDiffSurface, {
      props: {
        editorAriaLabel: 'Diff Viewer',
        language: 'yaml',
        modifiedKey: 'modified.yml',
        modifiedValue: 'version: 2\n',
        originalKey: 'original.yml',
        originalValue: 'version: 1\n',
      },
    });

    await flushPromises();
    mockState.callOrder.length = 0;

    await wrapper.setProps({
      language: 'json',
      modifiedKey: 'modified.json',
      modifiedValue: '{"version":2}\n',
      originalKey: 'original.json',
      originalValue: '{"version":1}\n',
    });
    await flushPromises();

    expect(mockState.callOrder).toEqual(['setModel', 'layout']);
    const reboundModel = mockState.currentModel();
    expect(reboundModel?.original.getLanguageId()).toBe('json');
    expect(reboundModel?.modified.getLanguageId()).toBe('json');
  });
});
