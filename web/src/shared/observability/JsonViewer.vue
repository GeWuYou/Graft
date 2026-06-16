<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <section class="json-viewer">
    <div class="json-viewer__toolbar">
      <div class="json-viewer__title">
        <strong>{{ title }}</strong>
        <span v-if="description">{{ description }}</span>
      </div>
      <div class="json-viewer__actions">
        <t-button theme="default" variant="outline" :disabled="isEmpty || hasError" @click="copyJson">
          {{ copyLabel }}
        </t-button>
        <t-button theme="default" variant="outline" :disabled="isEmpty || hasError" @click="toggleSource">
          {{ sourceMode ? treeLabel : sourceLabel }}
        </t-button>
      </div>
    </div>

    <t-alert v-if="hasError" theme="error" :title="errorLabel" />

    <div class="json-viewer__viewport">
      <t-empty v-if="isEmpty" size="small" :description="emptyLabel" />
      <t-empty v-else-if="hasError" size="small" :description="errorLabel" />
      <pre v-else-if="sourceMode" class="json-viewer__source" v-html="highlightedSource"></pre>
      <json-node v-else :node-key="rootLabel" :value="maskedValue" root />
    </div>
  </section>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { type Component, computed, defineComponent, h, ref, type VNode } from 'vue';

import { copyText } from './copy';
import { maskSensitiveJson } from './json-sanitize';

const props = defineProps<{
  value: unknown;
  title: string;
  description?: string;
  rootLabel: string;
  sourceLabel: string;
  treeLabel: string;
  copyLabel: string;
  copySuccessLabel: string;
  copyErrorLabel: string;
  emptyLabel: string;
  errorLabel: string;
}>();

const sourceMode = ref(false);

const isEmpty = computed(() => {
  const value = props.value;
  if (value === null || value === undefined || value === '') return true;
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === 'object') return Object.keys(value as Record<string, unknown>).length === 0;
  return false;
});

const serializedJson = computed(() => {
  if (isEmpty.value) {
    return {
      error: false,
      json: '',
      value: null,
    };
  }

  try {
    const value = maskSensitiveJson(props.value);
    const json = JSON.stringify(value, null, 2);
    return {
      error: !json,
      json: json ?? '',
      value,
    };
  } catch {
    return {
      error: true,
      json: '',
      value: null,
    };
  }
});

const maskedValue = computed(() => serializedJson.value.value);
const formattedJson = computed(() => serializedJson.value.json);
const hasError = computed(() => !isEmpty.value && serializedJson.value.error);
const highlightedSource = computed(() => highlightJson(formattedJson.value));

function toggleSource() {
  sourceMode.value = !sourceMode.value;
}

async function copyJson() {
  try {
    const copied = await copyText(formattedJson.value);
    if (!copied) {
      MessagePlugin.error(props.copyErrorLabel);
      return;
    }
    MessagePlugin.success(props.copySuccessLabel);
  } catch {
    MessagePlugin.error(props.copyErrorLabel);
  }
}

function highlightJson(value: string) {
  return escapeHtml(value).replace(
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
      return `<span class="json-viewer__token json-viewer__token--${tokenType}">${match}</span>`;
    },
  );
}

function escapeHtml(value: string) {
  return value.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

function valuePreview(value: unknown) {
  if (typeof value === 'string') return `"${value}"`;
  if (typeof value === 'number' || typeof value === 'boolean' || value === null) return String(value);
  if (Array.isArray(value)) return `Array(${value.length})`;
  return `Object(${Object.keys((value as Record<string, unknown>) ?? {}).length})`;
}

function valueType(value: unknown) {
  if (value === null) return 'null';
  if (Array.isArray(value)) return 'array';
  return typeof value;
}

const JsonNode: Component = defineComponent({
  name: 'JsonNode',
  props: {
    nodeKey: {
      type: String,
      required: true,
    },
    value: {
      type: null,
      required: false,
      default: null,
    },
    root: {
      type: Boolean,
      default: false,
    },
  },
  setup(nodeProps): () => VNode {
    const expanded = ref(Boolean(nodeProps.root));

    return (): VNode => {
      const value = nodeProps.value;
      const isObjectLike = value !== null && typeof value === 'object';
      const entries = Array.isArray(value)
        ? value.map((item, index) => [String(index), item] as const)
        : Object.entries((value as Record<string, unknown>) ?? {});

      if (!isObjectLike) {
        return h('div', { class: 'json-viewer__node json-viewer__node--leaf' }, [
          h('span', { class: 'json-viewer__key' }, nodeProps.nodeKey),
          h('span', { class: 'json-viewer__punctuation' }, ': '),
          h(
            'span',
            { class: `json-viewer__primitive json-viewer__primitive--${valueType(value)}` },
            valuePreview(value),
          ),
        ]);
      }

      return h('div', { class: 'json-viewer__node' }, [
        h(
          'button',
          {
            class: 'json-viewer__node-toggle',
            type: 'button',
            onClick: () => {
              expanded.value = !expanded.value;
            },
          },
          [
            h('span', { class: 'json-viewer__caret' }, expanded.value ? 'v' : '>'),
            h('span', { class: 'json-viewer__key' }, nodeProps.nodeKey),
            h('span', { class: 'json-viewer__preview' }, valuePreview(value)),
          ],
        ),
        expanded.value
          ? h(
              'div',
              { class: 'json-viewer__children' },
              entries.map(([key, childValue]) =>
                h(JsonNode, {
                  key,
                  nodeKey: key,
                  value: childValue,
                }),
              ),
            )
          : null,
      ]);
    };
  },
});
</script>
<style scoped lang="less">
.json-viewer {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
  min-width: 0;
}

.json-viewer__toolbar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-10);
  justify-content: space-between;
}

.json-viewer__title {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.json-viewer__title strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.json-viewer__title span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.json-viewer__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.json-viewer__viewport {
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  max-height: min(62vh, 680px);
  min-height: 360px;
  min-width: 0;
  overflow: auto;
  padding: var(--graft-density-gap-12);
  scrollbar-color: var(--td-scrollbar-color) transparent;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
}

.json-viewer__viewport::-webkit-scrollbar {
  background: transparent;
  height: 8px;
  width: 8px;
}

.json-viewer__viewport::-webkit-scrollbar-track {
  background: transparent;
}

.json-viewer__viewport::-webkit-scrollbar-thumb {
  background-clip: content-box;
  background-color: var(--td-scrollbar-color);
  border: 2px solid transparent;
  border-radius: 6px;
}

.json-viewer__source {
  color: var(--td-text-color-primary);
  font-family: var(--td-font-family-monospace);
  line-height: var(--td-line-height-body-medium);
  margin: 0;
  min-width: max-content;
  white-space: pre;
}

.json-viewer__node {
  color: var(--td-text-color-primary);
  font-family: var(--td-font-family-monospace);
  line-height: var(--td-line-height-body-medium);
}

.json-viewer__node-toggle {
  align-items: center;
  background: transparent;
  border: 0;
  color: inherit;
  cursor: pointer;
  display: inline-flex;
  gap: var(--graft-density-gap-6);
  padding: 0;
}

.json-viewer__caret {
  color: var(--td-text-color-placeholder);
  width: 14px;
}

.json-viewer__children {
  border-left: 1px solid var(--td-component-stroke);
  margin-left: var(--graft-density-gap-6);
  padding-left: var(--graft-density-gap-14);
}

.json-viewer__key,
.json-viewer__source :deep(.json-viewer__token--key) {
  color: var(--td-brand-color);
}

.json-viewer__preview {
  color: var(--td-text-color-secondary);
}

.json-viewer__primitive--string,
.json-viewer__source :deep(.json-viewer__token--string) {
  color: var(--td-success-color);
}

.json-viewer__primitive--number,
.json-viewer__source :deep(.json-viewer__token--number) {
  color: var(--td-warning-color);
}

.json-viewer__primitive--boolean,
.json-viewer__source :deep(.json-viewer__token--boolean) {
  color: var(--td-brand-color);
}

.json-viewer__primitive--null,
.json-viewer__source :deep(.json-viewer__token--null) {
  color: var(--td-text-color-placeholder);
}
</style>
