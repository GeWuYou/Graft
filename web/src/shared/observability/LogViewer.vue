<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <section class="log-viewer">
    <div class="log-viewer__toolbar">
      <div class="log-viewer__toolbar-group log-viewer__toolbar-left">
        <t-button theme="primary" :loading="loading" @click="$emit('refresh')">
          {{ refreshLabel }}
        </t-button>
        <t-button theme="default" variant="outline" :disabled="!displayLines.length" @click="copyContent">
          {{ copyLabel }}
        </t-button>
        <t-button theme="default" variant="outline" :disabled="!displayLines.length" @click="downloadContent">
          {{ downloadLabel }}
        </t-button>
        <span class="log-viewer__select-wrap">
          <t-select
            v-if="lineLimitOptions.length"
            v-model:value="selectedLineLimit"
            class="log-viewer__limit"
            :options="lineLimitOptions"
            size="small"
            @change="emitLimit"
          />
        </span>
        <t-select v-model:value="selectedLevel" class="log-viewer__level-filter" :options="levelOptions" size="small" />
      </div>

      <div class="log-viewer__toolbar-group log-viewer__toolbar-right">
        <t-input
          v-model:value="searchKeyword"
          class="log-viewer__search"
          clearable
          type="search"
          :placeholder="searchPlaceholder"
        />
        <span v-if="normalizedSearchKeyword" class="log-viewer__match-count">
          {{ matchCountLabel.replace('{count}', String(searchMatchCount)) }}
        </span>
        <label class="log-viewer__switch">
          <span>{{ wrapLabel }}</span>
          <t-switch v-model:value="wrapLines" size="small" />
        </label>
        <label class="log-viewer__switch">
          <span>{{ refreshScrollLabel }}</span>
          <t-tooltip :content="refreshScrollTooltipLabel" theme="light">
            <t-switch v-model:value="scrollAfterRefresh" size="small" />
          </t-tooltip>
        </label>
      </div>
    </div>

    <t-alert v-if="error" theme="error" :title="error">
      <template #operation>
        <t-button size="small" theme="danger" variant="text" @click="$emit('refresh')">
          {{ retryLabel }}
        </t-button>
      </template>
    </t-alert>
    <t-alert v-if="truncated" theme="warning" :title="truncatedLabel" />

    <div ref="viewport" :class="['log-viewer__viewport', { 'log-viewer__viewport--wrap': wrapLines }]">
      <t-skeleton v-if="loading && !displayLines.length" animation="gradient" :row-col="skeletonRows" />
      <ol v-else-if="displayLines.length" class="log-viewer__lines">
        <li
          v-for="line in displayLines"
          :key="line.lineNo"
          tabindex="0"
          :class="[
            'log-viewer__line',
            `log-viewer__line--${line.tone}`,
            { 'log-viewer__line--active': isActive(line.lineNo) },
          ]"
          @click="openLineDetail(line)"
          @keydown.enter.prevent="openLineDetail(line)"
          @keydown.space.prevent="openLineDetail(line)"
        >
          <span class="log-viewer__line-number">{{ line.lineNo }}</span>
          <div class="log-viewer__timestamp-cell">
            <t-tooltip v-if="line.timestamp" :content="line.timestamp" placement="top-left" theme="light">
              <time class="log-viewer__timestamp">{{ shortTimestamp(line.timestamp) }}</time>
            </t-tooltip>
            <span v-else class="log-viewer__timestamp log-viewer__timestamp--empty"></span>
          </div>
          <div class="log-viewer__level-cell">
            <t-tag class="log-viewer__level" :theme="levelTheme(line.level)" size="small" variant="light-outline">
              {{ line.level ?? 'LOG' }}
            </t-tag>
          </div>
          <div class="log-viewer__source-cell">
            <t-tooltip v-if="line.source" :content="line.source" placement="top-left" theme="light">
              <span class="log-viewer__source">{{ line.sourceShort || line.source }}</span>
            </t-tooltip>
            <span v-else class="log-viewer__source log-viewer__source--empty"></span>
          </div>
          <div class="log-viewer__content">
            <div class="log-viewer__message-row">
              <code class="log-viewer__message">
                <span
                  v-for="(token, tokenIndex) in line.messageTokens"
                  :key="`${line.lineNo}-message-${tokenIndex}`"
                  :class="tokenClass(token)"
                  >{{ token.text }}</span
                >
              </code>
            </div>
            <div v-if="visibleMetadataTags(line).length" class="log-viewer__metadata-tags" @click.stop>
              <t-tag
                v-for="[key, value] in visibleMetadataTags(line)"
                :key="`${line.lineNo}-${key}`"
                size="small"
                theme="default"
                variant="light"
              >
                {{ key }}={{ formatMetadataValue(value) }}
              </t-tag>
              <t-button
                v-if="hiddenRowFieldCount(line)"
                size="small"
                theme="default"
                variant="text"
                @click="openLineDetail(line)"
              >
                +{{ hiddenRowFieldCount(line) }}
              </t-button>
            </div>
          </div>
          <div class="log-viewer__row-actions" @click.stop>
            <t-tooltip :content="viewDetailLabel" theme="light">
              <t-button
                :aria-label="viewDetailLabel"
                class="log-viewer__icon-action"
                shape="square"
                size="small"
                theme="default"
                variant="text"
                @click="openLineDetail(line)"
              >
                <template #icon>
                  <browse-icon />
                </template>
              </t-button>
            </t-tooltip>
            <t-tooltip :content="copyLineLabel" theme="light">
              <t-button
                :aria-label="copyLineLabel"
                class="log-viewer__icon-action"
                shape="square"
                size="small"
                theme="default"
                variant="text"
                @click="copyLine(line.raw)"
              >
                <template #icon>
                  <copy-icon />
                </template>
              </t-button>
            </t-tooltip>
          </div>
        </li>
      </ol>
      <t-empty v-else size="small" :description="emptyLabel" />
    </div>

    <t-drawer
      v-model:visible="detailDrawerVisible"
      drawer-class-name="log-viewer__drawer"
      :footer="false"
      :header="detailTitleLabel"
      placement="right"
      size="min(600px, 100vw)"
      @close="closeLineDetail"
    >
      <div v-if="selectedLine" class="log-viewer__detail-drawer">
        <section class="log-viewer__summary">
          <div class="log-viewer__summary-main">
            <div class="log-viewer__summary-title">
              <t-tag
                class="log-viewer__summary-level"
                :theme="levelTheme(selectedLine.parsed.display.level)"
                size="small"
                variant="light-outline"
              >
                {{ selectedLine.parsed.display.level ?? 'LOG' }}
              </t-tag>
              <span class="log-viewer__summary-message">{{ selectedLine.parsed.display.title }}</span>
            </div>
            <div v-if="selectedLine.parsed.display.subtitleParts.length" class="log-viewer__summary-meta">
              <template
                v-for="(part, partIndex) in selectedLine.parsed.display.subtitleParts"
                :key="`${selectedLine.lineNo}-summary-${partIndex}`"
              >
                <span v-if="partIndex" aria-hidden="true">·</span>
                <t-tooltip
                  v-if="part === selectedLine.source"
                  :content="selectedLine.source"
                  placement="top-left"
                  theme="light"
                >
                  <span class="log-viewer__summary-source">{{ part }}</span>
                </t-tooltip>
                <span v-else>{{ part }}</span>
              </template>
            </div>
          </div>
        </section>

        <section v-if="selectedLine.parsed.importantFields.length" class="log-viewer__drawer-section">
          <div class="log-viewer__drawer-section-title">{{ importantFieldsLabel }}</div>
          <div class="log-viewer__field-chips">
            <span v-for="field in selectedLine.parsed.importantFields" :key="field.key" class="log-viewer__field-chip">
              <span class="log-viewer__field-key">{{ field.key }}</span>
              <span class="log-viewer__field-equals">=</span>
              <t-tooltip :content="field.value" placement="top-left" theme="light">
                <span class="log-viewer__field-value">{{ field.value }}</span>
              </t-tooltip>
            </span>
          </div>
        </section>

        <section class="log-viewer__drawer-section">
          <div class="log-viewer__drawer-section-title">{{ basicInfoLabel }}</div>
          <div class="log-viewer__basic">
            <div class="log-viewer__descriptions">
              <template v-if="selectedLine.timestamp">
                <div class="log-viewer__description-label">{{ timeLabel }}</div>
                <div class="log-viewer__description-value">{{ selectedLine.timestamp }}</div>
              </template>

              <template v-if="selectedLine.level">
                <div class="log-viewer__description-label">{{ levelLabel }}</div>
                <div class="log-viewer__description-value log-viewer__level-value">
                  <t-tag
                    class="log-viewer__detail-level"
                    :theme="levelTheme(selectedLine.level)"
                    size="small"
                    variant="light-outline"
                  >
                    {{ selectedLine.level }}
                  </t-tag>
                </div>
              </template>

              <template v-if="selectedLine.source">
                <div class="log-viewer__description-label">{{ sourceLabel }}</div>
                <div class="log-viewer__description-value">{{ selectedLine.source }}</div>
              </template>

              <div class="log-viewer__description-label">{{ messageLabel }}</div>
              <div class="log-viewer__description-value">{{ selectedLine.message }}</div>
            </div>
          </div>
        </section>

        <section v-if="selectedLine.metadata" class="log-viewer__drawer-section">
          <div class="log-viewer__drawer-section-header">
            <div class="log-viewer__drawer-section-title">{{ metadataLabel }}</div>
            <t-button size="small" theme="default" variant="text" @click="copySelectedJson">
              {{ copyJsonLabel }}
            </t-button>
          </div>
          <pre class="log-viewer__code-block"><code>{{ formatJson(selectedLine.metadata) }}</code></pre>
        </section>

        <section class="log-viewer__drawer-section">
          <div class="log-viewer__drawer-section-header">
            <div class="log-viewer__drawer-section-title">{{ rawLabel }}</div>
            <t-button size="small" theme="default" variant="text" @click="copySelectedLine">
              {{ copyLineLabel }}
            </t-button>
          </div>
          <pre class="log-viewer__code-block log-viewer__code-block--raw"><code>{{ selectedLine.raw }}</code></pre>
        </section>
      </div>
    </t-drawer>
  </section>
</template>
<script setup lang="ts">
import { BrowseIcon, CopyIcon } from 'tdesign-icons-vue-next';
import type { SelectProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, nextTick, ref, watch } from 'vue';

import { copyText } from './copy';
import type { LogLevel, LogToken } from './log-highlight';
import {
  buildDisplayLogLine,
  type DisplayLogLine,
  formatLogMetadataValue,
  type ParsedLogMetadata,
  parseLogLines,
  summarizeMetadata,
} from './log-parser';

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
    downloadLabel: string;
    retryLabel: string;
    searchPlaceholder: string;
    wrapLabel: string;
    refreshScrollLabel: string;
    refreshScrollTooltipLabel: string;
    levelFilterLabel: string;
    allLevelsLabel: string;
    matchCountLabel: string;
    emptyLabel: string;
    truncatedLabel: string;
    detailTitleLabel: string;
    importantFieldsLabel: string;
    basicInfoLabel: string;
    timeLabel: string;
    levelLabel: string;
    sourceLabel: string;
    viewDetailLabel: string;
    collapseDetailLabel: string;
    metadataLabel: string;
    messageLabel: string;
    rawLabel: string;
    copyMessageLabel: string;
    copyLineLabel: string;
    copyJsonLabel: string;
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

type SelectOption = NonNullable<SelectProps['options']>[number];
type LevelFilter = 'ALL' | LogLevel;

const searchKeyword = ref('');
const wrapLines = ref(true);
const scrollAfterRefresh = ref(true);
const selectedLineLimit = ref(props.lineLimit);
const selectedLevel = ref<LevelFilter>('ALL');
const viewport = ref<HTMLElement | null>(null);
const selectedLineNo = ref<number | null>(null);

const skeletonRows = [
  { height: '22px', width: '96%' },
  { height: '22px', width: '88%' },
  { height: '22px', width: '92%' },
  { height: '22px', width: '76%' },
  { height: '22px', width: '84%' },
];
const levelOptions = computed<SelectOption[]>(() => [
  { label: `${props.levelFilterLabel}: ${props.allLevelsLabel}`, value: 'ALL' },
  { label: 'ERROR', value: 'ERROR' },
  { label: 'WARN', value: 'WARN' },
  { label: 'INFO', value: 'INFO' },
  { label: 'DEBUG', value: 'DEBUG' },
  { label: 'TRACE', value: 'TRACE' },
  { label: 'LOG', value: 'LOG' },
]);
const lineLimitOptions = computed<SelectOption[]>(() =>
  props.lineLimits.map((value) => ({ label: String(value), value })),
);
const normalizedSearchKeyword = computed(() => searchKeyword.value.trim());
const parsedLines = computed(() => parseLogLines(props.lines));
const visibleRawLines = computed(() => parsedLines.value.slice(-selectedLineLimit.value));
const displayLines = computed(() =>
  visibleRawLines.value
    .filter((line) => selectedLevel.value === 'ALL' || line.level === selectedLevel.value)
    .map((line) => buildDisplayLogLine(line, normalizedSearchKeyword.value)),
);
const searchMatchCount = computed(() => displayLines.value.reduce((total, line) => total + line.searchMatchCount, 0));
const selectedLine = computed(() => displayLines.value.find((line) => line.lineNo === selectedLineNo.value) ?? null);
const detailDrawerVisible = computed({
  get: () => selectedLine.value !== null,
  set: (visible: boolean) => {
    if (!visible) {
      selectedLineNo.value = null;
    }
  },
});

watch(
  () => props.lineLimit,
  (value) => {
    selectedLineLimit.value = value;
  },
);

watch(
  () => [props.lines.length, scrollAfterRefresh.value],
  () => {
    if (!scrollAfterRefresh.value) return;
    void nextTick(scrollToBottom);
  },
  { flush: 'post' },
);

function emitLimit(value: SelectProps['value']) {
  if (typeof value === 'number') {
    emit('update:lineLimit', value);
  }
}

async function copyContent() {
  await copyTextWithFeedback(displayLines.value.map((line) => line.raw).join('\n'));
}

async function copyLine(raw: string) {
  await copyTextWithFeedback(raw);
}

async function copyJson(metadata: ParsedLogMetadata) {
  await copyTextWithFeedback(formatJson(metadata));
}

function downloadContent() {
  const blob = new Blob([displayLines.value.map((line) => line.raw).join('\n')], { type: 'text/plain;charset=utf-8' });
  const link = document.createElement('a');
  link.href = URL.createObjectURL(blob);
  link.download = `container-logs-${new Date().toISOString().replace(/[:.]/g, '-')}.log`;
  link.click();
  URL.revokeObjectURL(link.href);
}

function scrollToBottom() {
  const node = viewport.value;
  if (node) {
    node.scrollTop = node.scrollHeight;
  }
}

function openLineDetail(line: DisplayLogLine) {
  selectedLineNo.value = line.lineNo;
}

function closeLineDetail() {
  selectedLineNo.value = null;
}

function isActive(lineNo: number) {
  return selectedLineNo.value === lineNo;
}

async function copySelectedLine() {
  if (selectedLine.value) {
    await copyLine(selectedLine.value.raw);
  }
}

async function copySelectedJson() {
  if (selectedLine.value?.metadata) {
    await copyJson(selectedLine.value.metadata);
  }
}

function visibleMetadataTags(line: DisplayLogLine) {
  if (line.parsed.importantFields.length) {
    return visibleRowImportantFields(line).map((field) => [field.key, field.value] as [string, unknown]);
  }
  return summarizeMetadata(line.metadata).tags;
}

function hiddenMetadataCount(line: DisplayLogLine) {
  return summarizeMetadata(line.metadata).hiddenCount;
}

function hiddenRowFieldCount(line: DisplayLogLine) {
  if (!line.parsed.importantFields.length) {
    return hiddenMetadataCount(line);
  }
  if (line.parsed.format === 'logfmt') {
    return 0;
  }
  return Math.max(0, Object.keys(line.parsed.fields).length - visibleRowImportantFields(line).length);
}

function formatMetadataValue(value: unknown) {
  return formatLogMetadataValue(value);
}

function visibleRowImportantFields(line: DisplayLogLine) {
  const hiddenRowKeys = new Set(['level', 'severity', 'msg', 'message', 'event']);
  return line.parsed.importantFields
    .filter((field) => !hiddenRowKeys.has(field.key))
    .filter((field) => field.value !== line.message)
    .slice(0, 3);
}

function formatJson(value: unknown) {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

function shortTimestamp(timestamp: string) {
  const timeMatch = /(?:T|\s)(\d{2}:\d{2}:\d{2}(?:[.,]\d+)?)/.exec(timestamp);
  return timeMatch?.[1] ?? timestamp;
}

function levelTheme(level: LogLevel | null | undefined) {
  if (level === 'ERROR' || level === 'FATAL') return 'danger';
  if (level === 'WARN') return 'warning';
  if (level === 'INFO') return 'primary';
  return 'default';
}

function tokenClass(token: LogToken) {
  return [
    'log-viewer__token',
    `log-viewer__token--${token.type}`,
    token.level ? `log-viewer__token--level-${token.level.toLowerCase()}` : '',
  ];
}

async function copyTextWithFeedback(value: string) {
  try {
    const copied = await copyText(value);
    if (!copied) {
      MessagePlugin.error(props.copyErrorLabel);
      return;
    }
    MessagePlugin.success(props.copySuccessLabel);
  } catch {
    MessagePlugin.error(props.copyErrorLabel);
  }
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
  background: var(--td-bg-color-container);
  border-bottom: 1px solid var(--td-border-level-1-color);
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-14);
  position: sticky;
  top: 0;
  z-index: 1;
}

.log-viewer__toolbar-group,
.log-viewer__switch {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.log-viewer__toolbar-left {
  flex: 0 0 auto;
}

.log-viewer__toolbar-right {
  flex: 1 1 420px;
  justify-content: flex-end;
}

.log-viewer__limit {
  width: 96px;
}

.log-viewer__level-filter {
  width: 140px;
}

.log-viewer__search {
  flex: 0 0 auto;
  max-width: 36vw;
  min-width: 220px;
  width: 320px;
}

.log-viewer__match-count,
.log-viewer__switch {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.log-viewer__switch {
  white-space: nowrap;
}

.log-viewer__viewport {
  background: color-mix(in srgb, var(--td-bg-color-page) 78%, var(--td-bg-color-container) 22%);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  color: var(--td-text-color-primary);
  max-height: min(62vh, 680px);
  min-height: 360px;
  min-width: 0;
  overflow: auto;
  padding: var(--graft-density-gap-6) var(--graft-density-gap-8);
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
  min-width: max(100%, 760px);
  padding: 0;
}

.log-viewer__line {
  border-left: var(--graft-density-gap-2) solid transparent;
  border-radius: var(--td-radius-small);
  column-gap: var(--graft-density-gap-6);
  display: grid;
  grid-template-columns: 44px 96px 58px minmax(140px, 180px) minmax(0, 1fr) 60px;
  margin-block: var(--graft-density-gap-1);
  min-height: 30px;
  padding: var(--graft-density-gap-4) var(--graft-density-gap-6);
}

.log-viewer__line:hover,
.log-viewer__line:focus-within {
  background: color-mix(in srgb, var(--td-bg-color-container-hover) 54%, transparent);
  outline: none;
}

.log-viewer__line--active {
  background: var(--td-bg-color-container);
  box-shadow: inset 0 0 0 1px var(--td-component-stroke);
}

.log-viewer__line--active.log-viewer__line--default,
.log-viewer__line--active.log-viewer__line--info {
  border-left-color: var(--td-brand-color);
}

.log-viewer__line--active.log-viewer__line--muted {
  border-left-color: var(--td-text-color-placeholder);
}

.log-viewer__line-number {
  align-self: start;
  color: var(--td-text-color-placeholder);
  font-family: var(--td-font-family-monospace);
  font-variant-numeric: tabular-nums;
  line-height: var(--td-line-height-body-medium);
  text-align: right;
  user-select: none;
}

.log-viewer__timestamp-cell,
.log-viewer__level-cell,
.log-viewer__source-cell,
.log-viewer__content,
.log-viewer__row-actions {
  align-self: start;
}

.log-viewer__timestamp-cell,
.log-viewer__level-cell,
.log-viewer__source-cell,
.log-viewer__content {
  min-width: 0;
}

.log-viewer__timestamp-cell,
.log-viewer__source-cell {
  line-height: var(--td-line-height-body-medium);
}

.log-viewer__level-cell {
  display: flex;
  justify-content: flex-start;
  width: 58px;
}

.log-viewer__level {
  flex: 0 0 auto;
  max-width: max-content;
  width: auto;
}

.log-viewer__level-cell :deep(.t-tag) {
  flex: 0 0 auto;
  max-width: max-content;
  padding-inline: var(--graft-density-gap-4);
  width: auto;
}

.log-viewer__message-row {
  align-items: center;
  display: flex;
  min-width: 0;
}

.log-viewer__timestamp,
.log-viewer__source,
.log-viewer__message,
.log-viewer__raw,
.log-viewer__json {
  font-family: var(--td-font-family-monospace);
}

.log-viewer__timestamp,
.log-viewer__source {
  color: var(--td-text-color-placeholder);
  font-variant-numeric: tabular-nums;
}

.log-viewer__source {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-viewer__message {
  color: var(--td-text-color-primary);
  line-height: var(--td-line-height-body-medium);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-viewer__metadata-tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-4);
  margin-top: var(--graft-density-gap-4);
  max-width: 100%;
  min-width: 0;
}

.log-viewer__metadata-tags :deep(.t-tag) {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  line-height: 18px;
  max-width: 240px;
  overflow: hidden;
  padding: 0 var(--graft-density-gap-6);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-viewer__metadata-tags :deep(.t-button) {
  color: var(--td-text-color-secondary);
  min-width: 0;
  padding-inline: var(--graft-density-gap-4);
}

.log-viewer__row-actions {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-2);
  justify-content: flex-end;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.16s ease;
}

.log-viewer__line:hover .log-viewer__row-actions,
.log-viewer__line:focus-within .log-viewer__row-actions,
.log-viewer__line--active .log-viewer__row-actions {
  opacity: 1;
  pointer-events: auto;
}

.log-viewer__icon-action {
  color: var(--td-text-color-secondary);
}

.log-viewer__viewport--wrap .log-viewer__lines {
  min-width: 0;
}

.log-viewer__viewport--wrap .log-viewer__message {
  overflow: visible;
  overflow-wrap: anywhere;
  text-overflow: unset;
  white-space: pre-wrap;
}

.log-viewer__line--danger {
  background: color-mix(in srgb, var(--td-error-color-5) 4%, transparent);
  border-left-color: var(--td-error-color);
}

.log-viewer__line--warning {
  background: color-mix(in srgb, var(--td-warning-color-5) 4%, transparent);
  border-left-color: var(--td-warning-color);
}

.log-viewer__line--muted {
  color: var(--td-text-color-secondary);
}

.log-viewer :deep(.log-viewer__drawer .t-drawer__body) {
  padding: var(--graft-density-gap-24);
}

.log-viewer :deep(.log-viewer__drawer .t-drawer__content-wrapper) {
  max-width: min(720px, 100vw);
}

.log-viewer__detail-drawer {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.log-viewer__summary {
  border-bottom: 1px solid var(--td-component-stroke);
  margin-bottom: var(--graft-density-gap-18);
  min-width: 0;
  padding-bottom: var(--graft-density-gap-16);
}

.log-viewer__summary-level,
.log-viewer__detail-level {
  flex: 0 0 auto;
  max-width: max-content;
  width: fit-content;
}

.log-viewer__summary-title {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.log-viewer__summary-main {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.log-viewer__summary-message {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
  min-width: 0;
  overflow-wrap: anywhere;
}

.log-viewer__summary-meta {
  align-items: center;
  color: var(--td-text-color-secondary);
  column-gap: var(--graft-density-gap-6);
  display: flex;
  flex-wrap: wrap;
  font: var(--td-font-body-small);
  margin-top: var(--graft-density-gap-6);
  min-width: 0;
}

.log-viewer__summary-source {
  display: inline-block;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.log-viewer__drawer-section-header {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  margin-bottom: var(--graft-density-gap-8);
  min-width: 0;
}

.log-viewer__drawer-section {
  display: flex;
  flex-direction: column;
  margin-top: var(--graft-density-gap-18);
  min-width: 0;
}

.log-viewer__drawer-section-title {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  font-weight: 600;
}

.log-viewer__field-chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  margin-top: var(--graft-density-gap-8);
  min-width: 0;
}

.log-viewer__field-chip {
  align-items: center;
  background: var(--td-bg-color-secondarycontainer);
  border-radius: var(--td-radius-small);
  color: var(--td-text-color-secondary);
  display: inline-flex;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-4);
  max-width: 100%;
  min-width: 0;
  padding: var(--graft-density-gap-4) var(--graft-density-gap-8);
}

.log-viewer__field-key {
  color: var(--td-text-color-placeholder);
  flex: 0 0 auto;
}

.log-viewer__field-value {
  color: var(--td-text-color-primary);
  display: inline-block;
  max-width: 260px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.log-viewer__basic {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  margin-top: var(--graft-density-gap-8);
  padding: var(--graft-density-gap-12) var(--graft-density-gap-14);
}

.log-viewer__descriptions {
  display: grid;
  gap: var(--graft-density-gap-8) var(--graft-density-gap-12);
  grid-template-columns: 72px minmax(0, 1fr);
}

.log-viewer__description-label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.log-viewer__description-value {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  min-width: 0;
  overflow-wrap: anywhere;
}

.log-viewer__level-value {
  align-items: center;
  display: inline-flex;
  justify-self: start;
  max-width: max-content;
  width: fit-content;
}

.log-viewer__summary-title :deep(.t-tag),
.log-viewer__level-value :deep(.t-tag) {
  flex: 0 0 auto;
  max-width: max-content;
  width: auto;
}

.log-viewer__code-block {
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  font: var(--td-font-body-small);
  font-family: var(--td-font-family-mono, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace);
  line-height: 1.7;
  margin: 0;
  max-height: 260px;
  overflow: auto;
  padding: var(--graft-density-gap-12) var(--graft-density-gap-14);
  scrollbar-color: var(--td-scrollbar-color) transparent;
  scrollbar-width: thin;
  white-space: pre;
}

.log-viewer__code-block code {
  font-family: inherit;
}

.log-viewer__code-block--raw {
  max-height: 180px;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
  word-break: normal;
}

.log-viewer__code-block::-webkit-scrollbar {
  background: transparent;
  height: 8px;
  width: 8px;
}

.log-viewer__code-block::-webkit-scrollbar-track {
  background: transparent;
}

.log-viewer__code-block::-webkit-scrollbar-thumb {
  background-clip: content-box;
  background-color: var(--td-scrollbar-color);
  border: 2px solid transparent;
  border-radius: 6px;
}

.log-viewer__token--keyword {
  background: color-mix(in srgb, var(--td-warning-color-5) 34%, transparent);
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

@media (width <= 1200px) {
  .log-viewer__toolbar {
    align-items: flex-start;
  }

  .log-viewer__toolbar-left,
  .log-viewer__toolbar-right {
    width: 100%;
  }

  .log-viewer__toolbar-right {
    justify-content: flex-start;
  }

  .log-viewer__search {
    max-width: 100%;
    width: min(360px, 100%);
  }
}

@media (width <= 1024px) {
  .log-viewer__line {
    grid-template-columns: 40px 92px 56px minmax(108px, 132px) minmax(0, 1fr) 56px;
  }

  .log-viewer__level-cell {
    width: 56px;
  }
}

@media (width <= 760px) {
  .log-viewer__lines {
    min-width: 680px;
  }

  .log-viewer__line {
    grid-template-columns: 38px 88px 54px 112px minmax(0, 1fr) 52px;
  }
}
</style>
