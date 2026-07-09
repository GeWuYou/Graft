<template>
  <div ref="containerRef" class="project-monaco-surface" :data-testid="testId" />
</template>
<script setup lang="ts">
import type * as Monaco from 'monaco-editor';
import { ref, watch } from 'vue';

import { createLogger } from '@/utils/logger';

import * as projectMonaco from '../shared/project-monaco';
import * as projectMonacoDebug from '../shared/project-monaco-debug';

type MonacoEditorSurfaceOptions = Monaco.editor.IStandaloneEditorConstructionOptions;

const props = withDefaults(
  defineProps<{
    editorAriaLabel: string;
    language: string;
    markers?: Monaco.editor.IMarkerData[];
    modelKey: string;
    modelValue: string;
    options?: MonacoEditorSurfaceOptions;
    readOnly?: boolean;
    testId?: string;
  }>(),
  {
    markers: () => [],
    options: () => ({}),
    readOnly: false,
    testId: 'project-monaco-surface',
  },
);

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

const containerRef = ref<HTMLElement | null>(null);
const modelUriSuffix = projectMonaco.createProjectMonacoModelUriSuffix();
const logger = createLogger('project.monaco.surface');
const logProjectMonacoSurfaceDebug = projectMonacoDebug.createProjectMonacoDebugLogger('project.monaco.surface');
let monaco: typeof Monaco | null = null;
let editor: Monaco.editor.IStandaloneCodeEditor | null = null;
let model: Monaco.editor.ITextModel | null = null;
let syncingFromEditor = false;
let syncingFromProps = false;
let relayoutBridge: projectMonaco.ProjectMonacoRelayoutBridge | null = null;
const modelCache = new Map<string, Monaco.editor.ITextModel>();
const markerOwner = 'project-monaco-surface';

type ProjectMonacoMarker = Monaco.editor.IMarker;

const { applyTheme } = projectMonaco.useProjectMonacoLifecycle({
  createEditor,
  disposeEditor() {
    logSurfaceDebug('dispose-start', {
      hasEditor: Boolean(editor),
      hasModel: Boolean(model),
      modelCount: monaco?.editor.getModels().length ?? 0,
    });
    relayoutBridge?.disconnect();
    relayoutBridge = null;
    const currentEditor = editor;
    editor = null;
    model = null;
    currentEditor?.dispose();
    projectMonaco.disposeProjectMonacoModelCache(modelCache, 'dispose-editor-model', disposeSurfaceModel);
    logSurfaceDebug('dispose-complete', {
      hasEditor: Boolean(currentEditor),
      hasModel: modelCache.size > 0,
    });
  },
  getMonaco: () => monaco,
  getThemeHost: () => containerRef.value,
});

watch(
  () => props.modelValue,
  (value) => {
    if (!model || syncingFromEditor) {
      return;
    }

    const normalizedValue = String(value ?? '');
    if (model.getValue() === normalizedValue) {
      return;
    }

    logSurfaceDebug('model-value-update', {
      modelKey: props.modelKey,
      nextLength: normalizedValue.length,
      previousLength: model.getValue().length,
    });

    syncingFromProps = true;
    model.pushEditOperations([], [{ range: model.getFullModelRange(), text: normalizedValue }], () => null);
    syncingFromProps = false;
  },
);

watch(
  () => [props.modelKey, props.language] as const,
  async () => {
    if (!monaco || !editor) {
      return;
    }

    const previousModel = model;
    const nextModel = projectMonaco.getOrCreateProjectMonacoModel(monaco, {
      cache: modelCache,
      key: props.modelKey,
      language: props.language,
      suffix: modelUriSuffix,
      value: String(props.modelValue ?? ''),
    });
    logSurfaceDebug('model-rebind-start', {
      containerHeight: containerRef.value?.clientHeight ?? 0,
      containerWidth: containerRef.value?.clientWidth ?? 0,
      language: props.language,
      modelKey: props.modelKey,
      nextModelLength: nextModel.getValue().length,
      previousModelLength: previousModel?.getValue().length ?? 0,
    });
    if (previousModel === nextModel) {
      return;
    }
    editor.setModel(nextModel);
    model = nextModel;
    syncModelMarkers();
    await relayout('rebind-model');
    logSurfaceDebug('model-rebind-complete', {
      currentModelLength: model.getValue().length,
      modelCacheSize: modelCache.size,
      modelCount: monaco.editor.getModels().length,
    });
  },
  { flush: 'post' },
);

watch(
  () => props.readOnly,
  (readOnly) => {
    editor?.updateOptions({ readOnly });
  },
);

watch(
  () => props.options,
  (options) => {
    editor?.updateOptions(options);
  },
  { deep: true },
);

watch(
  () => props.markers,
  () => {
    syncModelMarkers();
  },
  { deep: true },
);

async function createEditor() {
  monaco = projectMonaco.ensureProjectMonacoConfigured();
  const host = containerRef.value;

  if (!host) {
    return;
  }

  logSurfaceDebug('create-start', {
    containerHeight: host.clientHeight,
    containerParent: projectMonacoDebug.describeProjectMonacoElement(host.parentElement),
    containerWidth: host.clientWidth,
    language: props.language,
    modelKey: props.modelKey,
    textLength: String(props.modelValue ?? '').length,
  });

  applyTheme();
  model = projectMonaco.getOrCreateProjectMonacoModel(monaco, {
    cache: modelCache,
    key: props.modelKey,
    language: props.language,
    suffix: modelUriSuffix,
    value: String(props.modelValue ?? ''),
  });
  editor = monaco.editor.create(host, {
    ariaLabel: props.editorAriaLabel,
    automaticLayout: false,
    glyphMargin: false,
    insertSpaces: true,
    language: props.language,
    lineNumbers: 'on',
    lineNumbersMinChars: 3,
    minimap: { enabled: false },
    padding: { top: 14, bottom: 14 },
    readOnly: props.readOnly,
    renderLineHighlightOnlyWhenFocus: true,
    roundedSelection: false,
    scrollBeyondLastLine: false,
    tabSize: 2,
    wordWrap: 'off',
    ...props.options,
    model,
  });
  syncModelMarkers();

  logSurfaceDebug('create-complete', {
    containerHeight: host.clientHeight,
    containerMatchesEditor: editor.getContainerDomNode() === host,
    containerWidth: host.clientWidth,
    modelCount: monaco.editor.getModels().length,
    textLength: model.getValue().length,
  });

  editor.onDidChangeModelContent(() => {
    if (!editor || syncingFromProps) {
      return;
    }
    const value = editor.getValue();
    if (value === props.modelValue) {
      return;
    }
    logSurfaceDebug('editor-content-change', {
      modelKey: props.modelKey,
      textLength: value.length,
    });
    syncingFromEditor = true;
    emit('update:modelValue', value);
    syncingFromEditor = false;
  });

  observeContainerResize();
  await relayout('initial-create');
}

function observeContainerResize() {
  if (!editor) {
    return;
  }

  relayoutBridge = projectMonaco.createProjectMonacoRelayoutBridge({
    getContainer: () => containerRef.value,
    layout: () => editor?.layout(),
    log: logSurfaceDebug,
  });
  relayoutBridge.observe();
}

async function relayout(reason = 'manual') {
  await relayoutBridge?.relayout(reason);
}

function syncModelMarkers() {
  if (!monaco || !model) {
    return;
  }

  monaco.editor.setModelMarkers(model, markerOwner, props.markers ?? []);
}

function getSortedMarkers() {
  if (!monaco || !model) {
    return [] as ProjectMonacoMarker[];
  }

  return monaco.editor
    .getModelMarkers({
      resource: model.uri,
    })
    .slice()
    .sort((left, right) => {
      if (left.startLineNumber !== right.startLineNumber) {
        return left.startLineNumber - right.startLineNumber;
      }
      if (left.startColumn !== right.startColumn) {
        return left.startColumn - right.startColumn;
      }
      return right.severity - left.severity;
    });
}

async function waitForDiagnostics(options?: { quietMs?: number; timeoutMs?: number }) {
  if (!monaco || !model) {
    return [] as ProjectMonacoMarker[];
  }

  const monacoInstance = monaco;
  const quietMs = Math.max(0, options?.quietMs ?? 120);
  const timeoutMs = Math.max(quietMs, options?.timeoutMs ?? 900);

  return await new Promise<ProjectMonacoMarker[]>((resolve) => {
    let settled = false;
    let idleTimer: ReturnType<typeof setTimeout> | null = null;

    const finalize = () => {
      if (settled) {
        return;
      }
      settled = true;
      if (idleTimer) {
        clearTimeout(idleTimer);
      }
      markerListener.dispose();
      clearTimeout(timeoutTimer);
      resolve(getSortedMarkers());
    };

    const scheduleIdleFinalize = () => {
      if (idleTimer) {
        clearTimeout(idleTimer);
      }
      idleTimer = setTimeout(finalize, quietMs);
    };

    const markerListener = monacoInstance.editor.onDidChangeMarkers((resources) => {
      const currentModel = model;
      if (!currentModel) {
        finalize();
        return;
      }
      if (resources.some((resource) => resource.toString() === currentModel.uri.toString())) {
        scheduleIdleFinalize();
      }
    });

    const timeoutTimer = setTimeout(finalize, timeoutMs);
    scheduleIdleFinalize();
  });
}

function revealMarker(marker: ProjectMonacoMarker | null | undefined) {
  if (!editor || !marker) {
    return false;
  }

  const lineNumber = Math.max(1, marker.startLineNumber || marker.endLineNumber || 1);
  const column = Math.max(1, marker.startColumn || marker.endColumn || 1);
  editor.setPosition({ column, lineNumber });
  editor.revealLineInCenter(lineNumber);
  editor.focus();
  return true;
}

function disposeSurfaceModel(targetModel: Monaco.editor.ITextModel | null, reason: string) {
  projectMonacoDebug.disposeProjectMonacoModelDeferred(targetModel, reason, {
    onCancellation: (detail) => {
      logSurfaceDebug('model-dispose-canceled', detail);
    },
    onDispose: (detail) => {
      logSurfaceDebug('model-dispose-complete', detail);
    },
    onError: (error, detail) => {
      logger.error(error, detail);
    },
  });
}

function logSurfaceDebug(event: string, detail: Record<string, unknown>) {
  if (!projectMonacoDebug.isProjectMonacoDebugEnabled()) {
    return;
  }

  logProjectMonacoSurfaceDebug(event, detail);
}

defineExpose({
  getModelKey: () => props.modelKey,
  getMarkers: getSortedMarkers,
  relayout,
  revealMarker,
  waitForDiagnostics,
});
</script>
<style scoped lang="less">
.project-monaco-surface {
  block-size: 100%;
  inline-size: 100%;
  min-block-size: 0;
  min-inline-size: 0;
}
</style>
