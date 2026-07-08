<template>
  <div ref="containerRef" class="project-monaco-diff-surface" :data-testid="testId" />
</template>
<script setup lang="ts">
import type * as Monaco from 'monaco-editor';
import { ref, watch } from 'vue';

import { createLogger } from '@/utils/logger';

import * as projectMonaco from '../shared/project-monaco';
import * as projectMonacoDebug from '../shared/project-monaco-debug';

const props = withDefaults(
  defineProps<{
    editorAriaLabel: string;
    language: string;
    modifiedKey: string;
    modifiedValue: string;
    originalKey: string;
    originalValue: string;
    testId?: string;
  }>(),
  {
    testId: 'project-monaco-diff-surface',
  },
);

const containerRef = ref<HTMLElement | null>(null);
const modelUriSuffix = projectMonaco.createProjectMonacoModelUriSuffix();
const logger = createLogger('project.monaco.diffSurface');
let monaco: typeof Monaco | null = null;
let editor: Monaco.editor.IStandaloneDiffEditor | null = null;
let originalModel: Monaco.editor.ITextModel | null = null;
let modifiedModel: Monaco.editor.ITextModel | null = null;
let relayoutBridge: projectMonaco.ProjectMonacoRelayoutBridge | null = null;
const modelCache = new Map<string, Monaco.editor.ITextModel>();

const { applyTheme } = projectMonaco.useProjectMonacoLifecycle({
  createEditor,
  disposeEditor() {
    logDiffDebug('dispose', {
      hasEditor: Boolean(editor),
      modelCount: monaco?.editor.getModels().length ?? 0,
    });
    relayoutBridge?.disconnect();
    relayoutBridge = null;
    const currentEditor = editor;
    editor = null;
    originalModel = null;
    modifiedModel = null;
    currentEditor?.dispose();
    projectMonaco.disposeProjectMonacoModelCache(modelCache, 'dispose-model', disposeDiffModel);
  },
  getMonaco: () => monaco,
  getThemeHost: () => containerRef.value,
});

function readNestedEditorSize(getEditor: (() => Monaco.editor.IStandaloneCodeEditor) | undefined) {
  try {
    const container = getEditor?.()?.getContainerDomNode();
    return {
      height: container?.clientHeight ?? 0,
      width: container?.clientWidth ?? 0,
    };
  } catch {
    return {
      height: 0,
      width: 0,
    };
  }
}

watch(
  () => [props.originalKey, props.modifiedKey, props.language] as const,
  async () => {
    if (!monaco || !editor) {
      return;
    }

    logDiffDebug('rebind-models-start', {
      language: props.language,
      modifiedKey: props.modifiedKey,
      modifiedTextLength: props.modifiedValue.length,
      originalKey: props.originalKey,
      originalTextLength: props.originalValue.length,
    });

    bindModels(monaco);
    await relayout('rebind-models');
  },
  { flush: 'post' },
);

watch(
  () => props.originalValue,
  (value) => {
    if (!originalModel || originalModel.getValue() === value) {
      return;
    }

    logDiffDebug('original-model-set-value', {
      nextLength: value.length,
      previousLength: originalModel.getValue().length,
    });

    originalModel.setValue(value);
  },
);

watch(
  () => props.modifiedValue,
  (value) => {
    if (!modifiedModel || modifiedModel.getValue() === value) {
      return;
    }

    logDiffDebug('modified-model-set-value', {
      nextLength: value.length,
      previousLength: modifiedModel.getValue().length,
    });

    modifiedModel.setValue(value);
  },
);

async function createEditor() {
  monaco = projectMonaco.ensureProjectMonacoConfigured();
  const host = containerRef.value;

  if (!host) {
    return;
  }

  logDiffDebug('create-start', {
    containerHeight: host.clientHeight,
    containerParent: projectMonacoDebug.describeProjectMonacoElement(host.parentElement),
    containerWidth: host.clientWidth,
    language: props.language,
    modifiedKey: props.modifiedKey,
    modifiedTextLength: props.modifiedValue.length,
    originalKey: props.originalKey,
    originalTextLength: props.originalValue.length,
  });

  applyTheme();
  editor = monaco.editor.createDiffEditor(host, {
    ariaLabel: props.editorAriaLabel,
    automaticLayout: false,
    enableSplitViewResizing: false,
    glyphMargin: false,
    minimap: { enabled: false },
    originalEditable: false,
    readOnly: true,
    renderIndicators: false,
    renderOverviewRuler: false,
    renderSideBySide: true,
    scrollBeyondLastLine: false,
    overviewRulerLanes: 0,
  });

  const editorContainer = editor.getContainerDomNode();
  logDiffDebug('create-complete', {
    containerHeight: host.clientHeight,
    containerMatchesEditor: editorContainer === host,
    containerWidth: host.clientWidth,
    editorContainer: projectMonacoDebug.describeProjectMonacoElement(editorContainer),
    modelCount: monaco.editor.getModels().length,
  });

  observeContainerResize();
  await relayout('initial-before-model');
  bindModels(monaco);
  await relayout('initial-after-model');
}

function observeContainerResize() {
  if (!editor) {
    return;
  }

  relayoutBridge = projectMonaco.createProjectMonacoRelayoutBridge({
    getContainer: () => containerRef.value,
    layout: () => editor?.layout(),
    log: logDiffDebug,
  });
  relayoutBridge.observe();
}

function bindModels(monacoInstance: typeof Monaco) {
  if (!editor) {
    return;
  }

  const nextOriginal = projectMonaco.getOrCreateProjectMonacoModel(monacoInstance, {
    cache: modelCache,
    key: props.originalKey,
    language: props.language,
    suffix: `${modelUriSuffix}-original`,
    value: props.originalValue,
  });
  const nextModified = projectMonaco.getOrCreateProjectMonacoModel(monacoInstance, {
    cache: modelCache,
    key: props.modifiedKey,
    language: props.language,
    suffix: `${modelUriSuffix}-modified`,
    value: props.modifiedValue,
  });

  const nextModel = {
    modified: nextModified,
    original: nextOriginal,
  };

  logDiffDebug('set-model-start', {
    modifiedLanguage: nextModified.getLanguageId(),
    modifiedModelLength: nextModified.getValue().length,
    originalLanguage: nextOriginal.getLanguageId(),
    originalModelLength: nextOriginal.getValue().length,
  });

  editor.setModel(nextModel);
  originalModel = nextOriginal;
  modifiedModel = nextModified;

  const boundModel = editor.getModel();
  const modifiedEditorSize = readNestedEditorSize(editor.getModifiedEditor?.bind(editor));
  const originalEditorSize = readNestedEditorSize(editor.getOriginalEditor?.bind(editor));
  logDiffDebug('set-model-complete', {
    containerHeight: containerRef.value?.clientHeight ?? 0,
    containerWidth: containerRef.value?.clientWidth ?? 0,
    editorHasModel: Boolean(boundModel),
    modifiedEditorHeight: modifiedEditorSize.height,
    modifiedEditorWidth: modifiedEditorSize.width,
    modelCount: monacoInstance.editor.getModels().length,
    originalEditorHeight: originalEditorSize.height,
    originalEditorWidth: originalEditorSize.width,
    sameModifiedModel: boundModel?.modified === nextModified,
    sameOriginalModel: boundModel?.original === nextOriginal,
  });
}

async function relayout(reason = 'manual') {
  await relayoutBridge?.relayout(reason);
}

function disposeDiffModel(targetModel: Monaco.editor.ITextModel | null, reason: string) {
  projectMonacoDebug.disposeProjectMonacoModelDeferred(targetModel, reason, {
    onCancellation: (detail) => {
      logDiffDebug('model-dispose-canceled', detail);
    },
    onDispose: (detail) => {
      logDiffDebug('model-dispose-complete', detail);
    },
    onError: (error, detail) => {
      logger.error(error, detail);
    },
  });
}

function logDiffDebug(event: string, detail: Record<string, unknown>) {
  if (!projectMonacoDebug.isProjectMonacoDebugEnabled()) {
    return;
  }

  projectMonacoDebug.createProjectMonacoDebugLogger('project.monaco.diffSurface')(event, detail);
}

defineExpose({
  relayout,
});
</script>
<style scoped lang="less">
.project-monaco-diff-surface {
  block-size: 100%;
  display: flex;
  flex: 1 1 auto;
  inline-size: 100%;
  min-block-size: 0;
  min-inline-size: 0;
  overflow: hidden;
}
</style>
