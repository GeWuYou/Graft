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
      <t-descriptions :title="t('appLog.detail.basic')" bordered :column="2" size="medium">
        <t-descriptions-item :label="t('appLog.columns.occurredAt')">
          {{ formatCompactDateTime(record.occurred_at, locale) }}
        </t-descriptions-item>
        <t-descriptions-item :label="t('appLog.columns.severity')">
          <t-tag :theme="appLogSeverityTheme(record.severity)" variant="light-outline" size="small">
            {{ record.severity.toUpperCase() }}
          </t-tag>
        </t-descriptions-item>
        <t-descriptions-item :label="t('appLog.columns.component')">{{ record.component }}</t-descriptions-item>
        <t-descriptions-item :label="t('appLog.columns.operation')">
          {{ appLogOperationText(record, t) }}
        </t-descriptions-item>
        <t-descriptions-item :label="t('appLog.detail.message')" :span="2">{{ record.message }}</t-descriptions-item>
        <t-descriptions-item :label="t('appLog.detail.error')" :span="2">
          {{ record.error || t('appLog.values.noError') }}
        </t-descriptions-item>
      </t-descriptions>

      <t-descriptions :title="t('appLog.detail.correlation')" bordered :column="2" size="medium">
        <t-descriptions-item :label="t('appLog.filters.requestId')" :span="2">
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
        </t-descriptions-item>
        <t-descriptions-item :label="t('appLog.filters.route')">
          {{ record.route || t('appLog.values.emptyField') }}
        </t-descriptions-item>
        <t-descriptions-item :label="t('appLog.filters.method')">
          {{ record.method || t('appLog.values.emptyField') }}
        </t-descriptions-item>
      </t-descriptions>

      <t-tabs v-model="activeTab">
        <t-tab-panel value="fields" :label="t('appLog.detail.fields')">
          <log-json-panel v-bind="jsonPanelBindings" :title="t('appLog.detail.fields')" :value="sanitizedFields" />
        </t-tab-panel>
        <t-tab-panel value="raw" :label="t('appLog.detail.rawJson')">
          <log-json-panel v-bind="jsonPanelBindings" :title="t('appLog.detail.rawJson')" :value="sanitizedRecord" />
        </t-tab-panel>
      </t-tabs>
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { formatCompactDateTime } from '@/shared/components/management';
import { copyText, LogJsonPanel, sanitizeTraceFieldsForDisplay } from '@/shared/observability';

import { appLogOperationText, appLogSeverityTheme } from '../shared/presentation';
import type { AppLogItem } from '../types/app-log';

const props = defineProps<{
  initialTab?: 'fields' | 'raw';
  record: AppLogItem | null;
  visible: boolean;
}>();

defineEmits<{
  (e: 'update:visible', value: boolean): void;
}>();

const { t, locale } = useI18n();
const activeTab = ref<'fields' | 'raw'>(props.initialTab ?? 'fields');
const sanitizedFields = computed(() => sanitizeTraceFieldsForDisplay(props.record?.fields ?? {}));
const sanitizedRecord = computed(() => sanitizeTraceFieldsForDisplay(props.record ?? {}));

const jsonPanelBindings = computed(() => ({
  expandLabel: t('appLog.detail.expandContext'),
  collapseLabel: t('appLog.detail.collapseContext'),
  copyLabel: t('appLog.detail.copyContext'),
  copySuccessLabel: t('appLog.detail.copyContextSuccess'),
  copyFailLabel: t('appLog.detail.copyContextFail'),
  emptyText: t('appLog.detail.contextEmpty'),
}));

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      activeTab.value = props.initialTab ?? 'fields';
    }
  },
  { immediate: true },
);

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

.app-log-detail :deep(.t-descriptions__content) {
  overflow-wrap: anywhere;
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
</style>
