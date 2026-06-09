<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-drawer
    :visible="visible"
    :header="t('appLog.page.detailTitle')"
    :footer="false"
    destroy-on-close
    placement="right"
    size="820px"
    @update:visible="$emit('update:visible', $event)"
  >
    <div v-if="record" class="app-log-detail">
      <section class="app-log-detail__section">
        <h4>{{ t('appLog.detail.basic') }}</h4>
        <div class="app-log-detail__grid">
          <div class="app-log-detail__item">
            <span>{{ t('appLog.columns.occurredAt') }}</span>
            <strong>{{ formatCompactDateTime(record.occurred_at, locale) }}</strong>
          </div>
          <div class="app-log-detail__item">
            <span>{{ t('appLog.columns.severity') }}</span>
            <t-tag :theme="appLogSeverityTheme(record.severity)" variant="light-outline" size="small">
              {{ record.severity.toUpperCase() }}
            </t-tag>
          </div>
          <div class="app-log-detail__item">
            <span>{{ t('appLog.columns.component') }}</span>
            <strong>{{ record.component }}</strong>
          </div>
          <div class="app-log-detail__item">
            <span>{{ t('appLog.columns.operation') }}</span>
            <strong>{{ appLogOperationText(record, t) }}</strong>
          </div>
          <div class="app-log-detail__item app-log-detail__item--full">
            <span>{{ t('appLog.detail.message') }}</span>
            <strong>{{ record.message }}</strong>
          </div>
          <div class="app-log-detail__item app-log-detail__item--full">
            <span>{{ t('appLog.detail.error') }}</span>
            <strong>{{ record.error || t('appLog.values.noError') }}</strong>
          </div>
        </div>
      </section>

      <section class="app-log-detail__section">
        <h4>{{ t('appLog.detail.correlation') }}</h4>
        <div class="app-log-detail__grid">
          <div class="app-log-detail__item app-log-detail__item--full">
            <span>{{ t('appLog.filters.requestId') }}</span>
            <div class="app-log-detail__copy-line">
              <strong class="app-log-detail__mono">{{ record.request_id || t('appLog.values.emptyField') }}</strong>
              <t-button
                v-if="record.request_id"
                size="small"
                theme="default"
                variant="text"
                @click="copyValue(record.request_id)"
              >
                {{ t('appLog.actions.copy') }}
              </t-button>
            </div>
          </div>
          <div class="app-log-detail__item app-log-detail__item--full">
            <span>{{ t('appLog.filters.traceId') }}</span>
            <div class="app-log-detail__copy-line">
              <strong class="app-log-detail__mono">{{ record.trace_id || t('appLog.values.emptyField') }}</strong>
              <t-button
                v-if="record.trace_id"
                size="small"
                theme="default"
                variant="text"
                @click="copyValue(record.trace_id)"
              >
                {{ t('appLog.actions.copy') }}
              </t-button>
            </div>
          </div>
          <div class="app-log-detail__item">
            <span>{{ t('appLog.filters.route') }}</span>
            <strong>{{ record.route || t('appLog.values.emptyField') }}</strong>
          </div>
          <div class="app-log-detail__item">
            <span>{{ t('appLog.filters.method') }}</span>
            <strong>{{ record.method || t('appLog.values.emptyField') }}</strong>
          </div>
        </div>
      </section>

      <log-json-panel
        v-for="panel in jsonPanels"
        :key="panel.key"
        :title="panel.title"
        :expand-label="jsonPanelLabels.expand"
        :collapse-label="jsonPanelLabels.collapse"
        :copy-label="jsonPanelLabels.copy"
        :copy-success-label="jsonPanelLabels.copySuccess"
        :copy-fail-label="jsonPanelLabels.copyFail"
        :empty-text="jsonPanelLabels.empty"
        :value="panel.value"
      />
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { formatCompactDateTime } from '@/shared/components/management';
import { copyText, LogJsonPanel } from '@/shared/observability';

import { appLogOperationText, appLogSeverityTheme } from '../shared/presentation';
import type { AppLogItem } from '../types/app-log';

const props = defineProps<{
  record: AppLogItem | null;
  visible: boolean;
}>();

defineEmits<{
  (e: 'update:visible', value: boolean): void;
}>();

const { t, locale } = useI18n();

const jsonPanelLabels = computed(() => ({
  expand: t('appLog.detail.expandContext'),
  collapse: t('appLog.detail.collapseContext'),
  copy: t('appLog.detail.copyContext'),
  copySuccess: t('appLog.detail.copyContextSuccess'),
  copyFail: t('appLog.detail.copyContextFail'),
  empty: t('appLog.detail.contextEmpty'),
}));

const jsonPanels = computed(() => {
  if (!props.record) {
    return [];
  }

  return [
    { key: 'fields', title: t('appLog.detail.fields'), value: props.record.fields },
    { key: 'full', title: t('appLog.detail.fullContext'), value: props.record },
  ];
});

async function copyValue(value: string) {
  try {
    const copied = await copyText(value);
    if (!copied) {
      MessagePlugin.error(t('appLog.actions.copyFail'));
      return;
    }
    MessagePlugin.success(t('appLog.actions.copySuccess'));
  } catch {
    MessagePlugin.error(t('appLog.actions.copyFail'));
  }
}
</script>
<style scoped lang="less">
.app-log-detail {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-24);
}

.app-log-detail__section h4 {
  margin: 0 0 var(--graft-density-gap-12);
}

.app-log-detail__grid {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.app-log-detail__item {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.app-log-detail__item span {
  color: var(--td-text-color-secondary);
}

.app-log-detail__item strong {
  overflow-wrap: anywhere;
}

.app-log-detail__item--full {
  grid-column: 1 / -1;
}

.app-log-detail__copy-line {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.app-log-detail__mono {
  font-family: var(--td-font-family-mono, monospace);
}

@media (width <= 768px) {
  .app-log-detail__grid {
    grid-template-columns: 1fr;
  }
}
</style>
