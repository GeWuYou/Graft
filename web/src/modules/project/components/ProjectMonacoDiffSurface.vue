<template>
  <div ref="containerRef" class="project-monaco-diff-surface" :data-testid="testId" />
</template>
<script setup lang="ts">
import type * as Monaco from 'monaco-editor';
import { ref, watch } from 'vue';

import { createLogger } from '@/utils/logger';

import {
  buildProjectMonacoModelUri,
  createProjectMonacoModelUriSuffix,
  ensureProjectMonacoConfigured,
  observeProjectMonacoResize,
  scheduleProjectMonacoLayout,
  useProjectMonacoLifecycle,
} from '../shared/project-monaco';
import {
  describeProjectMonacoElement,
  disposeProjectMonacoModelDeferred,
  formatProjectMonacoDebugMessage,
  isProjectMonacoDebugEnabled,
} from '../shared/project-monaco-debug';

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
const modelUriSuffix = createProjectMonacoModelUriSuffix();
const logger = createLogger('project.monaco.diffSurface');
let monaco: typeof Monaco | null = null;
let editor: Monaco.editor.IStandaloneDiffEditor | null = null;
let originalModel: Monaco.editor.ITextModel | null = null;
let modifiedModel: Monaco.editor.ITextModel | null = null;
let resizeObserver: ResizeObserver | null = null;

const { applyTheme } = useProjectMonacoLifecycle({
  createEditor,
  disposeEditor() {
    logDiffDebug('dispose', {
      hasEditor: Boolean(editor),
      modelCount: monaco?.editor.getModels().length ?? 0,
    });
    resizeObserver?.disconnect();
    resizeObserver = null;
    const currentEditor = editor;
    const currentOriginalModel = originalModel;
    const currentModifiedModel = modifiedModel;
    editor = null;
    originalModel = null;
    modifiedModel = null;
    currentEditor?.dispose();
    disposeDiffModel(currentOriginalModel, 'dispose-original');
    disposeDiffModel(currentModifiedModel, 'dispose-modified');
  },
  getMonaco: () => monaco,
  getThemeHost: () => containerRef.value,
});

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

    bindModels(monaco, true);
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
  monaco = ensureProjectMonacoConfigured();
  const host = containerRef.value;

  if (!host) {
    return;
  }

  logDiffDebug('create-start', {
    containerHeight: host.clientHeight,
    containerParent: describeProjectMonacoElement(host.parentElement),
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
    automaticLayout: true,
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
    editorContainer: describeProjectMonacoElement(editorContainer),
    modelCount: monaco.editor.getModels().length,
  });

  observeContainerResize();
  await relayout('initial-before-model');
  bindModels(monaco, false);
  await relayout('initial-after-model');
}

function observeContainerResize() {
  if (!editor) {
    return;
  }

  resizeObserver = observeProjectMonacoResize(containerRef.value, resizeObserver, () => {
    const host = containerRef.value;
    logDiffDebug('resize-observer-fired', {
      containerHeight: host?.clientHeight ?? 0,
      containerWidth: host?.clientWidth ?? 0,
    });
    editor?.layout();
  });
}

function bindModels(monacoInstance: typeof Monaco, disposeExisting: boolean) {
  if (!editor) {
    return;
  }

  if (disposeExisting) {
    const previousOriginalModel = originalModel;
    const previousModifiedModel = modifiedModel;
    originalModel = null;
    modifiedModel = null;
    disposeDiffModel(previousOriginalModel, 'replace-original');
    disposeDiffModel(previousModifiedModel, 'replace-modified');
  }

  const nextOriginal = monacoInstance.editor.createModel(
    props.originalValue,
    props.language,
    buildProjectMonacoModelUri(props.originalKey, props.language, `${modelUriSuffix}-original`),
  );
  const nextModified = monacoInstance.editor.createModel(
    props.modifiedValue,
    props.language,
    buildProjectMonacoModelUri(props.modifiedKey, props.language, `${modelUriSuffix}-modified`),
  );

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
  logDiffDebug('set-model-complete', {
    editorHasModel: Boolean(boundModel),
    modelCount: monacoInstance.editor.getModels().length,
    sameModifiedModel: boundModel?.modified === nextModified,
    sameOriginalModel: boundModel?.original === nextOriginal,
  });
}

async function relayout(reason = 'manual') {
  const host = containerRef.value;
  logDiffDebug('layout-scheduled', {
    containerHeight: host?.clientHeight ?? 0,
    containerWidth: host?.clientWidth ?? 0,
    reason,
  });

  await scheduleProjectMonacoLayout(() => {
    const layoutHost = containerRef.value;
    logDiffDebug('layout-run', {
      containerHeight: layoutHost?.clientHeight ?? 0,
      containerWidth: layoutHost?.clientWidth ?? 0,
      reason,
    });
    editor?.layout();
  });
}

function disposeDiffModel(targetModel: Monaco.editor.ITextModel | null, reason: string) {
  disposeProjectMonacoModelDeferred(targetModel, reason, {
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
  if (!isProjectMonacoDebugEnabled()) {
    return;
  }

  logger.debug(`[ProjectMonacoDiffSurface] ${formatProjectMonacoDebugMessage(event, detail)}`, detail);
}

defineExpose({
  relayout,
});
</script>
<style scoped lang="less">
.project-monaco-diff-surface {
  block-size: 100%;
  inline-size: 100%;
  min-block-size: 0;
  min-inline-size: 0;
}
</style>
