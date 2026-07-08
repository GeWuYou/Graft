<template>
  <div ref="containerRef" class="project-monaco-diff-surface" :data-testid="testId" />
</template>
<script setup lang="ts">
import type * as Monaco from 'monaco-editor';
import { ref, watch } from 'vue';

import {
  buildProjectMonacoModelUri,
  createProjectMonacoModelUriSuffix,
  ensureProjectMonacoConfigured,
  scheduleProjectMonacoLayout,
  useProjectMonacoLifecycle,
} from '../shared/project-monaco';

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
let monaco: typeof Monaco | null = null;
let editor: Monaco.editor.IStandaloneDiffEditor | null = null;
let originalModel: Monaco.editor.ITextModel | null = null;
let modifiedModel: Monaco.editor.ITextModel | null = null;

const { applyTheme } = useProjectMonacoLifecycle({
  createEditor,
  disposeEditor() {
    editor?.dispose();
    editor = null;
    originalModel?.dispose();
    modifiedModel?.dispose();
    originalModel = null;
    modifiedModel = null;
  },
  getMonaco: () => monaco,
  getThemeHost: () => containerRef.value,
});

watch(
  () => [props.originalKey, props.modifiedKey, props.language] as const,
  () => {
    if (!monaco || !editor) {
      return;
    }
    bindModels(monaco, true);
  },
);

watch(
  () => props.originalValue,
  (value) => {
    if (!originalModel || originalModel.getValue() === value) {
      return;
    }
    originalModel.setValue(value);
  },
);

watch(
  () => props.modifiedValue,
  (value) => {
    if (!modifiedModel || modifiedModel.getValue() === value) {
      return;
    }
    modifiedModel.setValue(value);
  },
);

async function createEditor() {
  monaco = ensureProjectMonacoConfigured();
  if (!containerRef.value) {
    return;
  }

  applyTheme();
  editor = monaco.editor.createDiffEditor(containerRef.value, {
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

  bindModels(monaco, false);
  await scheduleProjectMonacoLayout(() => {
    editor?.layout();
  });
}

function bindModels(monacoInstance: typeof Monaco, disposeExisting: boolean) {
  if (!editor) {
    return;
  }

  if (disposeExisting) {
    editor.setModel(null);
    originalModel?.dispose();
    modifiedModel?.dispose();
    originalModel = null;
    modifiedModel = null;
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

  editor.setModel({
    modified: nextModified,
    original: nextOriginal,
  });
  originalModel = nextOriginal;
  modifiedModel = nextModified;
}
</script>
<style scoped lang="less">
.project-monaco-diff-surface {
  block-size: 100%;
  inline-size: 100%;
  min-block-size: 0;
  min-inline-size: 0;
}
</style>
