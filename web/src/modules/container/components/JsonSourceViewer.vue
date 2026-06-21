<template>
  <div class="json-source-viewer" data-testid="container-raw-source-viewer">
    <div v-if="!lines.length" class="json-source-viewer__empty">
      {{ emptyLabel }}
    </div>
    <div v-else class="json-source-viewer__viewport graft-scrollbar">
      <div
        v-for="line in lines"
        :key="line.lineNumber"
        class="json-source-viewer__line"
        :class="{ 'json-source-viewer__line--match': line.matched }"
      >
        <span class="json-source-viewer__line-number">{{ line.lineNumber }}</span>
        <code class="json-source-viewer__line-content" v-html="line.html"></code>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  emptyLabel: string;
  formattedJson: string;
  searchValue: string;
}>();

const lines = computed(() => {
  if (!props.formattedJson) {
    return [];
  }

  const normalizedSearch = props.searchValue.trim().toLowerCase();
  return props.formattedJson.split('\n').map((line, index) => ({
    html: highlightJsonLine(line),
    lineNumber: index + 1,
    matched: normalizedSearch ? line.toLowerCase().includes(normalizedSearch) : false,
  }));
});

function highlightJsonLine(line: string) {
  const escapedLine = escapeHtml(line);
  return escapedLine.replace(
    /("(?:\\u[\da-fA-F]{4}|\\[^u]|[^\\"])*"(\s*:)?|\btrue\b|\bfalse\b|\bnull\b|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)/g,
    (match) => {
      let tokenType = 'number';
      if (match.startsWith('"')) {
        tokenType = match.endsWith(':') ? 'key' : 'string';
      } else if (match === 'true' || match === 'false') {
        tokenType = 'boolean';
      } else if (match === 'null') {
        tokenType = 'null';
      }
      return `<span class="json-source-viewer__token json-source-viewer__token--${tokenType}">${match}</span>`;
    },
  );
}

function escapeHtml(value: string) {
  return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
</script>
<style scoped lang="less">
.json-source-viewer {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
}

.json-source-viewer__empty {
  align-items: center;
  color: var(--td-text-color-placeholder);
  display: flex;
  flex: 1;
  justify-content: center;
  min-height: 0;
}

.json-source-viewer__viewport {
  min-height: 0;
  min-width: 0;
  overflow: auto;
}

.json-source-viewer__line {
  align-items: stretch;
  display: grid;
  grid-template-columns: 52px minmax(0, 1fr);
  min-width: max-content;
}

.json-source-viewer__line--match {
  background: color-mix(in srgb, var(--td-warning-color-1) 72%, transparent);
}

.json-source-viewer__line-number {
  background: inherit;
  border-right: 1px solid color-mix(in srgb, var(--td-component-stroke) 70%, transparent);
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  left: 0;
  line-height: 22px;
  padding: 0 var(--graft-density-gap-12) 0 0;
  position: sticky;
  text-align: right;
  user-select: none;
}

.json-source-viewer__line-content {
  color: var(--td-text-color-primary);
  display: block;
  font-family: var(
    --td-font-family-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    monospace
  );
  font-size: var(--td-font-size-body-small);
  font-weight: var(--td-font-weight-regular);
  line-height: 22px;
  margin: 0;
  padding: 0 0 0 var(--graft-density-gap-12);
  white-space: pre;
}

.json-source-viewer__line-content :deep(.json-source-viewer__token--key) {
  color: var(--td-brand-color);
}

.json-source-viewer__line-content :deep(.json-source-viewer__token--string) {
  color: var(--td-success-color-6);
}

.json-source-viewer__line-content :deep(.json-source-viewer__token--number) {
  color: var(--td-warning-color-7);
}

.json-source-viewer__line-content :deep(.json-source-viewer__token--boolean) {
  color: var(--td-error-color-6);
}

.json-source-viewer__line-content :deep(.json-source-viewer__token--null) {
  color: var(--td-text-color-placeholder);
}
</style>
