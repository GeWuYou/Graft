<template>
  <section class="scheduled-task-config-json-editor">
    <div class="scheduled-task-config-json-editor__head">
      <strong>{{ title }}</strong>
      <t-space size="small" break-line>
        <t-button v-if="mode === 'preview'" size="small" variant="outline" @click="switchMode('edit')">
          {{ editLabel }}
        </t-button>
        <t-button v-else size="small" variant="outline" @click="switchMode('preview')">
          {{ doneLabel }}
        </t-button>
        <t-button v-if="mode === 'edit'" size="small" variant="outline" @click="$emit('format')">
          {{ formatLabel }}
        </t-button>
      </t-space>
    </div>
    <pre v-if="mode === 'preview'" class="scheduled-task-config-json-editor__preview graft-scrollbar">{{
      previewText
    }}</pre>
    <t-form-item
      v-else
      class="scheduled-task-config-json-editor__form-item"
      :label="editorLabel"
      :name="fieldName"
      :status="error ? 'error' : undefined"
      :tips="error"
    >
      <t-textarea
        v-model="jsonValue"
        class="scheduled-task-config-json-editor__textarea"
        :autosize="{ minRows: 4, maxRows: 8 }"
        :placeholder="placeholder"
        @change="$emit('change')"
      />
    </t-form-item>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';

type JsonEditorMode = 'preview' | 'edit';

const props = withDefaults(
  defineProps<{
    doneLabel: string;
    editLabel: string;
    editorLabel: string;
    error?: string;
    fieldName?: string;
    formatLabel: string;
    mode: JsonEditorMode;
    modelValue: string;
    placeholder: string;
    previewText: string;
    title: string;
  }>(),
  {
    error: '',
    fieldName: 'configJson',
  },
);

const emit = defineEmits<{
  change: [];
  format: [];
  'update:mode': [mode: JsonEditorMode];
  'update:modelValue': [value: string];
}>();

const jsonValue = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', String(value ?? '')),
});

function switchMode(mode: JsonEditorMode) {
  emit('update:mode', mode);
}
</script>
<style scoped>
.scheduled-task-config-json-editor {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.scheduled-task-config-json-editor__head {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
}

.scheduled-task-config-json-editor__head strong {
  color: var(--td-text-color-primary);
}

.scheduled-task-config-json-editor__preview {
  background: var(--td-bg-color-page);
  border-radius: var(--td-radius-small);
  box-sizing: border-box;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  margin: 0;
  max-width: 100%;
  min-height: 96px;
  overflow: auto;
  overflow-wrap: anywhere;
  padding: var(--graft-density-gap-8);
  white-space: pre-wrap;
  width: 100%;
}

.scheduled-task-config-json-editor__form-item {
  margin-bottom: 0;
}

.scheduled-task-config-json-editor__textarea {
  width: 100%;
}

:deep(.scheduled-task-config-json-editor__textarea.t-textarea),
.scheduled-task-config-json-editor__textarea :deep(.t-textarea__inner) {
  box-sizing: border-box;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  width: 100%;
}

@media (width <= 640px) {
  .scheduled-task-config-json-editor__head {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
