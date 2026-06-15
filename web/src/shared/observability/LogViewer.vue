<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <section class="log-viewer">
    <div class="log-viewer__toolbar">
      <div class="log-viewer__toolbar-main">
        <t-button theme="primary" :loading="loading" @click="$emit('refresh')">
          {{ refreshLabel }}
        </t-button>
        <t-button theme="default" variant="outline" :disabled="!displayLines.length" @click="copyContent">
          {{ copyLabel }}
        </t-button>
        <t-select
          v-if="lineLimitOptions.length"
          v-model:value="selectedLineLimit"
          class="log-viewer__limit"
          :options="lineLimitOptions"
          size="small"
          @change="emitLimit"
        />
      </div>
      <div class="log-viewer__toolbar-tools">
        <t-input
          v-model:value="searchKeyword"
          class="log-viewer__search"
          clearable
          type="search"
          :placeholder="searchPlaceholder"
        />
        <label class="log-viewer__switch">
          <span>{{ wrapLabel }}</span>
          <t-switch v-model:value="wrapLines" size="small" />
        </label>
        <label class="log-viewer__switch">
          <span>{{ followTailLabel }}</span>
          <t-switch v-model:value="followTail" size="small" />
        </label>
      </div>
    </div>

    <t-alert v-if="error" theme="error" :title="error" />
    <t-alert v-if="truncated" theme="warning" :title="truncatedLabel" />

    <div ref="viewport" :class="['log-viewer__viewport', { 'log-viewer__viewport--wrap': wrapLines }]">
      <ol v-if="displayLines.length" class="log-viewer__lines">
        <li
          v-for="line in displayLines"
          :key="line.index"
          :class="['log-viewer__line', `log-viewer__line--${line.tone}`]"
        >
          <span class="log-viewer__line-number">{{ line.index }}</span>
          <code class="log-viewer__line-content">
            <span
              v-for="(token, tokenIndex) in line.tokens"
              :key="`${line.index}-${tokenIndex}`"
              :class="tokenClass(token)"
              >{{ token.text }}</span
            >
          </code>
        </li>
      </ol>
      <t-empty v-else size="small" :description="emptyLabel" />
    </div>
  </section>
</template>
<script setup lang="ts">
import type { SelectProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, nextTick, ref, watch } from 'vue';

import { copyText } from './copy';
import { detectLogLevel, getLogLevelTone, type LogToken, tokenizeLogLine } from './log-highlight';

const props = withDefaults(
  defineProps<{
    lines: string[];
    loading?: boolean;
    error?: string;
    truncated?: boolean;
    lineLimit?: number;
    lineLimits?: number[];
    refreshLabel: string;
    copyLabel: string;
    searchPlaceholder: string;
    wrapLabel: string;
    followTailLabel: string;
    emptyLabel: string;
    truncatedLabel: string;
    copySuccessLabel: string;
    copyErrorLabel: string;
  }>(),
  {
    loading: false,
    error: '',
    truncated: false,
    lineLimit: 200,
    lineLimits: () => [100, 200, 500, 1000],
  },
);

const emit = defineEmits<{
  refresh: [];
  'update:lineLimit': [value: number];
}>();

const searchKeyword = ref('');
const wrapLines = ref(false);
const followTail = ref(true);
const selectedLineLimit = ref(props.lineLimit);
const viewport = ref<HTMLElement | null>(null);

type SelectOption = NonNullable<SelectProps['options']>[number];

const lineLimitOptions = computed<SelectOption[]>(() =>
  props.lineLimits.map((value) => ({ label: String(value), value })),
);

const visibleRawLines = computed(() => props.lines.slice(-selectedLineLimit.value));

const displayLines = computed(() =>
  visibleRawLines.value.map((line, index) => {
    const level = detectLogLevel(line);
    return {
      index: props.lines.length - visibleRawLines.value.length + index + 1,
      tone: getLogLevelTone(level),
      tokens: tokenizeLogLine(line, searchKeyword.value),
    };
  }),
);

watch(
  () => props.lineLimit,
  (value) => {
    selectedLineLimit.value = value;
  },
);

watch(
  () => [displayLines.value.length, followTail.value, wrapLines.value],
  () => {
    if (!followTail.value) return;
    void nextTick(() => {
      const node = viewport.value;
      if (node) {
        node.scrollTop = node.scrollHeight;
      }
    });
  },
  { flush: 'post' },
);

function emitLimit(value: SelectProps['value']) {
  if (typeof value === 'number') {
    emit('update:lineLimit', value);
  }
}

async function copyContent() {
  try {
    const copied = await copyText(displayLines.value.map((line) => props.lines[line.index - 1]).join('\n'));
    if (!copied) {
      MessagePlugin.error(props.copyErrorLabel);
      return;
    }
    MessagePlugin.success(props.copySuccessLabel);
  } catch {
    MessagePlugin.error(props.copyErrorLabel);
  }
}

function tokenClass(token: LogToken) {
  return [
    'log-viewer__token',
    `log-viewer__token--${token.type}`,
    token.level ? `log-viewer__token--level-${token.level.toLowerCase()}` : '',
  ];
}
</script>
<style scoped lang="less">
.log-viewer {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
  min-width: 0;
}

.log-viewer__toolbar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-10);
  justify-content: space-between;
  min-width: 0;
}

.log-viewer__toolbar-main,
.log-viewer__toolbar-tools,
.log-viewer__switch {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.log-viewer__limit {
  width: 112px;
}

.log-viewer__search {
  width: min(260px, 100%);
}

.log-viewer__switch {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.log-viewer__viewport {
  background: color-mix(in srgb, var(--td-bg-color-page) 86%, black 14%);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  color: var(--td-text-color-primary);
  max-height: min(62vh, 680px);
  min-height: 360px;
  min-width: 0;
  overflow: auto;
  scrollbar-color: var(--td-scrollbar-color) transparent;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
}

.log-viewer__viewport::-webkit-scrollbar {
  background: transparent;
  height: 8px;
  width: 8px;
}

.log-viewer__viewport::-webkit-scrollbar-track {
  background: transparent;
}

.log-viewer__viewport::-webkit-scrollbar-thumb {
  background-clip: content-box;
  background-color: var(--td-scrollbar-color);
  border: 2px solid transparent;
  border-radius: 6px;
}

.log-viewer__lines {
  counter-reset: none;
  list-style: none;
  margin: 0;
  min-width: max-content;
  padding: var(--graft-density-gap-10) 0;
}

.log-viewer__line {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr);
  min-height: 24px;
}

.log-viewer__line-number {
  color: var(--td-text-color-placeholder);
  font-family: var(--td-font-family-monospace);
  padding: 0 var(--graft-density-gap-10);
  text-align: right;
  user-select: none;
}

.log-viewer__line-content {
  font-family: var(--td-font-family-monospace);
  line-height: var(--td-line-height-body-medium);
  padding-right: var(--graft-density-gap-12);
  white-space: pre;
}

.log-viewer__viewport--wrap .log-viewer__lines {
  min-width: 0;
}

.log-viewer__viewport--wrap .log-viewer__line-content {
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.log-viewer__line--danger {
  background: color-mix(in srgb, var(--td-error-color-5) 10%, transparent);
}

.log-viewer__line--warning {
  background: color-mix(in srgb, var(--td-warning-color-5) 10%, transparent);
}

.log-viewer__line--info {
  background: color-mix(in srgb, var(--td-brand-color) 7%, transparent);
}

.log-viewer__line--muted {
  color: var(--td-text-color-secondary);
}

.log-viewer__token--keyword {
  background: color-mix(in srgb, var(--td-warning-color-5) 28%, transparent);
  border-radius: var(--td-radius-small);
  color: var(--td-text-color-primary);
}

.log-viewer__token--field-key {
  color: var(--td-brand-color);
}

.log-viewer__token--field-value {
  color: var(--td-text-color-secondary);
}

.log-viewer__token--level-error,
.log-viewer__token--level-fatal {
  color: var(--td-error-color);
  font-weight: 600;
}

.log-viewer__token--level-warn {
  color: var(--td-warning-color);
  font-weight: 600;
}

.log-viewer__token--level-info {
  color: var(--td-brand-color);
  font-weight: 600;
}

.log-viewer__token--level-debug,
.log-viewer__token--level-trace {
  color: var(--td-text-color-placeholder);
}
</style>
