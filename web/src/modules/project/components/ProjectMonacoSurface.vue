<template>
  <div ref="containerRef" class="project-monaco-surface" :data-testid="testId" />
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

type MonacoEditorSurfaceOptions = Monaco.editor.IStandaloneEditorConstructionOptions;

const props = withDefaults(
  defineProps<{
    editorAriaLabel: string;
    language: string;
    modelKey: string;
    modelValue: string;
    options?: MonacoEditorSurfaceOptions;
    readOnly?: boolean;
    testId?: string;
  }>(),
  {
    options: () => ({}),
    readOnly: false,
    testId: 'project-monaco-surface',
  },
);

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

const containerRef = ref<HTMLElement | null>(null);
const modelUriSuffix = createProjectMonacoModelUriSuffix();
let monaco: typeof Monaco | null = null;
let editor: Monaco.editor.IStandaloneCodeEditor | null = null;
let model: Monaco.editor.ITextModel | null = null;
let syncingFromEditor = false;
let syncingFromProps = false;

const { applyTheme } = useProjectMonacoLifecycle({
  createEditor,
  disposeEditor() {
    editor?.dispose();
    editor = null;
    model?.dispose();
    model = null;
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

    syncingFromProps = true;
    model.pushEditOperations([], [{ range: model.getFullModelRange(), text: normalizedValue }], () => null);
    syncingFromProps = false;
  },
);

watch(
  () => [props.modelKey, props.language] as const,
  () => {
    if (!monaco || !editor) {
      return;
    }
    editor.setModel(null);
    model?.dispose();
    const nextModel = createModel(monaco);
    editor.setModel(nextModel);
    model = nextModel;
  },
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

async function createEditor() {
  monaco = ensureProjectMonacoConfigured();
  const host = containerRef.value;

  if (!host) {
    return;
  }

  applyTheme();
  model = createModel(monaco);
  editor = monaco.editor.create(host, {
    ariaLabel: props.editorAriaLabel,
    automaticLayout: true,
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

  editor.onDidChangeModelContent(() => {
    if (!editor || syncingFromProps) {
      return;
    }
    const value = editor.getValue();
    if (value === props.modelValue) {
      return;
    }
    syncingFromEditor = true;
    emit('update:modelValue', value);
    syncingFromEditor = false;
  });

  await scheduleProjectMonacoLayout(() => {
    editor?.layout();
  });
}

function createModel(monacoInstance: typeof Monaco) {
  return monacoInstance.editor.createModel(
    String(props.modelValue ?? ''),
    props.language,
    buildProjectMonacoModelUri(props.modelKey, props.language, modelUriSuffix),
  );
}
</script>
<style scoped lang="less">
.project-monaco-surface {
  block-size: 100%;
  inline-size: 100%;
  min-block-size: 0;
  min-inline-size: 0;
}
</style>
