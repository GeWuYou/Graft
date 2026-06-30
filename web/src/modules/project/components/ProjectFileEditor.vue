<template>
  <content-viewer-frame
    :storage-key="storageKey"
    :fullscreen-label="fullscreenLabel"
    :exit-fullscreen-label="exitFullscreenLabel"
    :resize-handle-label="resizeHandleLabel"
    surface-padding="none"
    fullscreen-surface-padding="none"
  >
    <template #header>
      <div class="project-file-editor__header">
        <strong>{{ title }}</strong>
        <p v-if="description">{{ description }}</p>
      </div>
    </template>

    <template #header-actions>
      <t-space size="small" break-line>
        <t-button size="small" variant="outline" @click="toggleMode">
          {{ mode === 'edit' ? previewLabel : editLabel }}
        </t-button>
        <t-button v-if="mode === 'edit'" size="small" variant="outline" @click="$emit('format')">
          {{ formatLabel }}
        </t-button>
      </t-space>
    </template>

    <div v-if="mode === 'edit'" class="project-file-editor__edit">
      <t-textarea
        v-model="valueProxy"
        class="project-file-editor__textarea"
        :autosize="{ minRows: 16, maxRows: 28 }"
        :placeholder="placeholder"
      />
    </div>

    <pre v-else-if="modelValue.trim()" class="project-file-editor__preview graft-scrollbar">{{ modelValue }}</pre>
    <t-empty v-else class="project-file-editor__empty" :description="emptyLabel" />
  </content-viewer-frame>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import ContentViewerFrame from '@/shared/components/viewer/ContentViewerFrame.vue';

type EditorMode = 'edit' | 'preview';

const props = defineProps<{
  description?: string;
  editLabel: string;
  emptyLabel: string;
  exitFullscreenLabel: string;
  formatLabel: string;
  fullscreenLabel: string;
  mode: EditorMode;
  modelValue: string;
  placeholder: string;
  previewLabel: string;
  resizeHandleLabel: string;
  storageKey: string;
  title: string;
}>();

const emit = defineEmits<{
  format: [];
  'update:mode': [mode: EditorMode];
  'update:modelValue': [value: string];
}>();

const valueProxy = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', String(value ?? '')),
});

function toggleMode() {
  emit('update:mode', props.mode === 'edit' ? 'preview' : 'edit');
}
</script>
<style scoped lang="less">
.project-file-editor__header,
.project-file-editor__edit {
  display: flex;
  flex-direction: column;
}

.project-file-editor__header {
  gap: var(--graft-density-gap-4);
}

.project-file-editor__header strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.project-file-editor__header p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.project-file-editor__edit,
.project-file-editor__preview,
.project-file-editor__empty {
  min-height: 420px;
}

.project-file-editor__textarea {
  width: 100%;
}

:deep(.project-file-editor__textarea.t-textarea),
.project-file-editor__textarea :deep(.t-textarea__inner),
.project-file-editor__preview {
  box-sizing: border-box;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  width: 100%;
}

.project-file-editor__preview {
  background: var(--td-bg-color-page);
  margin: 0;
  min-width: 0;
  overflow: auto;
  padding: var(--graft-density-gap-16);
  white-space: pre-wrap;
}

.project-file-editor__empty {
  align-items: center;
  display: flex;
  justify-content: center;
}
</style>
