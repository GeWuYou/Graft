<template>
  <div ref="containerRef" class="project-monaco-diff-surface" :data-testid="testId" />
</template>
<script setup lang="ts">
import type * as Monaco from 'monaco-editor';
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';

import { buildProjectMonacoModelUri, ensureProjectMonacoConfigured } from '../shared/project-monaco';

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
let monaco: typeof Monaco | null = null;
let editor: Monaco.editor.IStandaloneDiffEditor | null = null;
let originalModel: Monaco.editor.ITextModel | null = null;
let modifiedModel: Monaco.editor.ITextModel | null = null;

onMounted(() => {
  monaco = ensureProjectMonacoConfigured();
  void createEditor();
});

onBeforeUnmount(() => {
  editor?.dispose();
  editor = null;
  originalModel?.dispose();
  modifiedModel?.dispose();
  originalModel = null;
  modifiedModel = null;
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
  if (!monaco || !containerRef.value) {
    return;
  }

  editor = monaco.editor.createDiffEditor(containerRef.value, {
    ariaLabel: props.editorAriaLabel,
    automaticLayout: true,
    glyphMargin: false,
    minimap: { enabled: false },
    originalEditable: false,
    readOnly: true,
    renderSideBySide: true,
    scrollBeyondLastLine: false,
  });

  bindModels(monaco, false);
  await nextTick();
  requestAnimationFrame(() => {
    editor?.layout();
  });
}

function bindModels(monacoInstance: typeof Monaco, disposeExisting: boolean) {
  if (!editor) {
    return;
  }

  const nextOriginal = monacoInstance.editor.createModel(
    props.originalValue,
    props.language,
    buildProjectMonacoModelUri(props.originalKey, props.language),
  );
  const nextModified = monacoInstance.editor.createModel(
    props.modifiedValue,
    props.language,
    buildProjectMonacoModelUri(props.modifiedKey, props.language),
  );

  editor.setModel({
    modified: nextModified,
    original: nextOriginal,
  });

  if (disposeExisting) {
    originalModel?.dispose();
    modifiedModel?.dispose();
  }

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
