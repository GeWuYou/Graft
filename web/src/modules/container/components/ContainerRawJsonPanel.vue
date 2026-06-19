<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <section class="container-raw-json-panel">
    <header class="container-raw-json-panel__header">
      <div class="container-raw-json-panel__heading">
        <h3>{{ title }}</h3>
      </div>
      <t-alert :theme="policyAlertTheme" :message="policyMessage" />
    </header>

    <div v-if="hasError" class="container-raw-json-panel__empty">
      <t-empty size="small" :description="errorLabel" />
    </div>
    <div v-else-if="isEmpty" class="container-raw-json-panel__empty">
      <t-empty size="small" :description="emptyLabel" />
    </div>
    <template v-else>
      <div class="container-raw-json-panel__meta">
        <t-tag v-for="chip in chips" :key="chip.key" theme="default" variant="light-outline">
          {{ chip.label }}
        </t-tag>
      </div>

      <div class="container-raw-json-panel__viewer">
        <json-viewer-toolbar
          v-model:search-value="searchValue"
          v-model:view-mode="viewMode"
          :collapse-all-label="collapseAllLabel"
          :copy-disabled="copyDisabled"
          :copy-label="copyLabel"
          :copy-tooltip="copyTooltip"
          :expand-all-label="expandAllLabel"
          :expand-disabled="expandActionDisabled"
          :expanded-all="expandedAll"
          :format-disabled="!formattedJson"
          :format-label="formatLabel"
          :search-placeholder="searchPlaceholder"
          :source-label="sourceLabel"
          :tree-label="treeLabel"
          @copy="copyJson"
          @format="handleFormat"
          @toggle-expand-all="toggleExpandAll"
        />

        <div v-if="showSearchEmpty" class="container-raw-json-panel__search-empty">
          <t-alert theme="warning" :message="searchEmptyLabel" />
        </div>

        <div class="container-raw-json-panel__surface">
          <json-tree-viewer
            v-if="viewMode === 'tree'"
            :collapse-label="collapseTreeNodeLabel"
            :empty-label="searchEmptyLabel"
            :expand-label="expandTreeNodeLabel"
            :expanded-all="expandedAll"
            :expand-all-token="expandAllToken"
            :root-label="rootLabel"
            :search-value="searchValue"
            :sensitive-label="sensitiveLabel"
            :value="maskedValue"
          />
          <json-source-viewer
            v-else
            :empty-label="searchEmptyLabel"
            :formatted-json="formattedJson"
            :search-value="searchValue"
          />
        </div>
      </div>
    </template>
  </section>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { copyText, formatLocaleDateTime } from '@/shared/observability';

import JsonSourceViewer from './JsonSourceViewer.vue';
import JsonTreeViewer from './JsonTreeViewer.vue';
import JsonViewerToolbar from './JsonViewerToolbar.vue';

type RawJsonViewMode = 'tree' | 'source';

type ChipItem = {
  key: string;
  label: string;
};

const props = defineProps<{
  collapseAllLabel: string;
  copyValue?: unknown;
  copyDisabledMessage: string;
  copyErrorLabel: string;
  copyLabel: string;
  copyMaskedTooltip: string;
  copySuccessLabel: string;
  description: string;
  emptyLabel: string;
  environmentLabel: string;
  errorLabel: string;
  expandAllLabel: string;
  fieldCountLabel: string;
  formatLabel: string;
  mountedLabel: string;
  networkLabel: string;
  portLabel: string;
  rootLabel: string;
  collapseTreeNodeLabel: string;
  expandTreeNodeLabel: string;
  searchEmptyLabel: string;
  searchPlaceholder: string;
  maskedCountLabel: string;
  sensitiveFieldLabel: string;
  sensitiveLabel: string;
  sourceLabel: string;
  title: string;
  treeLabel: string;
  updatedAtLabel: string;
  policyAlertTheme?: 'info' | 'warning' | 'error' | 'success';
  policyMessage: string;
  rawCopyEnabled?: boolean;
  value: unknown;
}>();

const { locale } = useI18n();

const searchValue = ref('');
const viewMode = ref<RawJsonViewMode>('tree');
const expandedAll = ref(true);
const expandAllToken = ref(0);

const isEmpty = computed(() => {
  const value = props.value;
  if (value === null || value === undefined || value === '') return true;
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === 'object') return Object.keys(value as Record<string, unknown>).length === 0;
  return false;
});

const serializedJson = computed(() => {
  if (isEmpty.value) {
    return { error: false, json: '', value: null as unknown };
  }

  try {
    const value = props.value;
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
      value: null as unknown,
    };
  }
});

const maskedValue = computed(() => serializedJson.value.value);
const formattedJson = computed(() => serializedJson.value.json);
const copyJsonContent = computed(() => {
  try {
    return JSON.stringify(props.copyValue ?? props.value, null, 2);
  } catch {
    return '';
  }
});
const hasError = computed(() => !isEmpty.value && serializedJson.value.error);
const copyDisabled = computed(() => !copyJsonContent.value || props.rawCopyEnabled === false);
const copyTooltip = computed(() => (copyDisabled.value ? props.copyDisabledMessage : props.copyMaskedTooltip));
const visibleText = computed(() => {
  const keyword = searchValue.value.trim().toLowerCase();
  if (!keyword) {
    return formattedJson.value;
  }
  return formattedJson.value
    .split('\n')
    .filter((line) => line.toLowerCase().includes(keyword))
    .join('\n');
});
const showSearchEmpty = computed(
  () => Boolean(searchValue.value.trim()) && !visibleText.value && !hasError.value && !isEmpty.value,
);
const expandActionDisabled = computed(() => viewMode.value !== 'tree');

const chips = computed<ChipItem[]>(() => {
  const record =
    maskedValue.value && typeof maskedValue.value === 'object' && !Array.isArray(maskedValue.value)
      ? (maskedValue.value as Record<string, unknown>)
      : null;
  if (!record) {
    return [];
  }

  const items: ChipItem[] = [];
  items.push({ key: 'fields', label: `${props.fieldCountLabel} ${Object.keys(record).length}` });

  const sensitiveCount = countSensitiveFields(record);
  if (sensitiveCount > 0) {
    items.push({ key: 'sensitive', label: `${props.maskedCountLabel} ${sensitiveCount}` });
  }

  const environmentCount = countArray(record.environment);
  if (environmentCount > 0) {
    items.push({ key: 'environment', label: `${props.environmentLabel} ${environmentCount}` });
  }

  const portCount = countArray(record.ports);
  if (portCount > 0) {
    items.push({ key: 'ports', label: `${props.portLabel} ${portCount}` });
  }

  const mountCount = countArray(record.mounts);
  if (mountCount > 0) {
    items.push({ key: 'mounts', label: `${props.mountedLabel} ${mountCount}` });
  }

  const networkCount = countArray(record.networks);
  if (networkCount > 0) {
    items.push({ key: 'networks', label: `${props.networkLabel} ${networkCount}` });
  }

  const updatedAt = readString(record.inspect_updated_at);
  if (updatedAt) {
    items.push({
      key: 'updatedAt',
      label: `${props.updatedAtLabel} ${formatLocaleDateTime(updatedAt, locale)}`,
    });
  }

  return items;
});

async function copyJson() {
  if (copyDisabled.value) {
    MessagePlugin.error(props.copyDisabledMessage);
    return;
  }
  try {
    const copied = await copyText(copyJsonContent.value);
    if (!copied) {
      MessagePlugin.error(props.copyErrorLabel);
      return;
    }
    MessagePlugin.success(props.copySuccessLabel);
  } catch {
    MessagePlugin.error(props.copyErrorLabel);
  }
}

function handleFormat() {
  if (!formattedJson.value) {
    return;
  }
  viewMode.value = 'source';
}

function toggleExpandAll() {
  expandedAll.value = !expandedAll.value;
  expandAllToken.value += 1;
}

function countArray(value: unknown) {
  return Array.isArray(value) ? value.length : 0;
}

function countSensitiveFields(value: unknown): number {
  if (Array.isArray(value)) {
    return value.reduce((total, item) => total + countSensitiveFields(item), 0);
  }
  if (!value || typeof value !== 'object') {
    return 0;
  }

  const record = value as Record<string, unknown>;
  const current =
    record.masked === true || record.sensitive === true || record.value_masked === true || record.value_hidden === true
      ? 1
      : 0;
  return current + Object.values(record).reduce<number>((total, item) => total + countSensitiveFields(item), 0);
}

function readString(value: unknown) {
  return typeof value === 'string' ? value.trim() : '';
}
</script>
<style scoped lang="less">
.container-raw-json-panel {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-height: 0;
  min-width: 0;
  padding: 0 var(--graft-density-gap-16) var(--graft-density-gap-16);
}

.container-raw-json-panel__header {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
  min-width: 0;
}

.container-raw-json-panel__heading h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
}

.container-raw-json-panel__meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.container-raw-json-panel__viewer {
  background: color-mix(in srgb, var(--td-bg-color-page) 85%, var(--td-bg-color-container));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 78%, transparent);
  border-radius: var(--td-radius-large);
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  height: 100%;
  min-height: 0;
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.container-raw-json-panel__surface {
  background: color-mix(in srgb, var(--td-bg-color-container) 82%, var(--td-bg-color-page));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex: 1;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-10);
}

.container-raw-json-panel__search-empty {
  flex: 0 0 auto;
}

.container-raw-json-panel__empty {
  align-items: center;
  display: flex;
  flex: 1;
  justify-content: center;
  min-height: 240px;
}

@media (width <= 720px) {
  .container-raw-json-panel {
    padding-inline: var(--graft-density-gap-12);
  }

  .container-raw-json-panel__viewer {
    padding: var(--graft-density-gap-10);
  }
}
</style>
