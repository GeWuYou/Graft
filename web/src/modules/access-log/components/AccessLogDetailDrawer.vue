<template>
  <t-drawer
    :visible="visible"
    :header="t('accessLog.page.detailTitle')"
    :footer="false"
    destroy-on-close
    placement="right"
    size="820px"
    @update:visible="$emit('update:visible', $event)"
  >
    <div v-if="record" class="access-log-detail">
      <t-descriptions :title="t('accessLog.detail.basic')" bordered :column="2" size="medium">
        <t-descriptions-item :label="t('accessLog.columns.startedAt')">
          {{ formatCompactDateTime(record.started_at, locale) }}
        </t-descriptions-item>
        <t-descriptions-item :label="t('accessLog.columns.occurredAt')">
          {{ formatCompactDateTime(record.occurred_at, locale) }}
        </t-descriptions-item>
        <t-descriptions-item :label="t('accessLog.detail.occurredAtRaw')" :span="2">
          <span class="access-log-detail__mono">{{ record.occurred_at }}</span>
        </t-descriptions-item>
        <t-descriptions-item :label="t('accessLog.columns.method')">{{ record.method }}</t-descriptions-item>
        <t-descriptions-item :label="t('accessLog.columns.statusCode')">
          {{ record.status_code }}
        </t-descriptions-item>
        <t-descriptions-item :label="t('accessLog.columns.durationMs')">
          {{ record.duration_ms }} ms
        </t-descriptions-item>
        <t-descriptions-item :label="t('accessLog.columns.path')" :span="2">{{ record.path }}</t-descriptions-item>
        <t-descriptions-item :label="t('accessLog.detail.route')" :span="2">
          {{ accessLogPathSecondary(record) || '-' }}
        </t-descriptions-item>
      </t-descriptions>

      <t-descriptions :title="t('accessLog.detail.correlation')" bordered :column="2" size="medium">
        <t-descriptions-item :label="t('accessLog.detail.requestId')" :span="2">
          <div class="access-log-detail__copy-line">
            <strong class="access-log-detail__mono">{{ record.request_id }}</strong>
            <t-button size="small" theme="default" variant="text" @click="copyValue(record.request_id)">
              {{ t('accessLog.actions.copy') }}
            </t-button>
          </div>
        </t-descriptions-item>
        <t-descriptions-item :label="t('accessLog.detail.userId')">
          {{ accessLogUserSecondary(record, t) }}
        </t-descriptions-item>
        <t-descriptions-item :label="t('accessLog.detail.user')">
          {{ accessLogUserPrimary(record, t) }}
        </t-descriptions-item>
      </t-descriptions>

      <t-descriptions :title="t('accessLog.detail.network')" bordered :column="2" size="medium">
        <t-descriptions-item :label="t('accessLog.detail.clientIp')">
          {{ record.client_ip || '-' }}
        </t-descriptions-item>
        <t-descriptions-item :label="t('accessLog.detail.userAgent')" :span="2">
          {{ record.user_agent || '-' }}
        </t-descriptions-item>
        <t-descriptions-item :label="t('accessLog.detail.requestSize')">
          {{ record.request_size ?? '-' }}
        </t-descriptions-item>
        <t-descriptions-item :label="t('accessLog.detail.responseSize')">
          {{ record.response_size ?? '-' }}
        </t-descriptions-item>
      </t-descriptions>

      <t-tabs v-model="activeTab">
        <t-tab-panel value="raw" :label="t('accessLog.detail.rawJson')">
          <log-json-panel v-bind="jsonPanelBindings" :title="t('accessLog.detail.rawJson')" :value="sanitizedRecord" />
        </t-tab-panel>
      </t-tabs>

      <section v-if="relatedActions.length" class="access-log-detail__section">
        <h4>{{ t('accessLog.detail.relatedAudit') }}</h4>
        <div class="access-log-detail__actions">
          <t-button
            v-for="action in relatedActions"
            :key="action.key"
            size="small"
            theme="default"
            variant="outline"
            @click="action.onClick"
          >
            {{ action.label }}
          </t-button>
        </div>
      </section>
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { buildAuditRequestLocation } from '@/modules/audit/contract/deep-link';
import { formatCompactDateTime } from '@/shared/components/management';
import { LogJsonPanel, sanitizeTraceFieldsForDisplay } from '@/shared/observability';

import { copyAccessLogValue } from '../shared/clipboard';
import { accessLogPathSecondary, accessLogUserPrimary, accessLogUserSecondary } from '../shared/presentation';
import type { AccessLogItem } from '../types/access-log';

const props = defineProps<{
  record: AccessLogItem | null;
  visible: boolean;
}>();

defineEmits<{
  (e: 'update:visible', value: boolean): void;
}>();

const { t, locale } = useI18n();
const router = useRouter();
const activeTab = ref<'raw'>('raw');
const sanitizedRecord = computed(() => sanitizeTraceFieldsForDisplay(props.record ?? {}));

const jsonPanelBindings = computed(() => ({
  expandLabel: t('accessLog.detail.expandContext'),
  collapseLabel: t('accessLog.detail.collapseContext'),
  copyLabel: t('accessLog.detail.copyContext'),
  copySuccessLabel: t('accessLog.detail.copyContextSuccess'),
  copyFailLabel: t('accessLog.detail.copyContextFail'),
  emptyText: t('accessLog.detail.contextEmpty'),
}));

const relatedActions = computed(() => {
  const record = props.record;
  if (!record) {
    return [];
  }

  const actions = [];
  if (record.request_id) {
    actions.push({
      key: 'audit-request',
      label: t('accessLog.actions.viewRelatedAuditRecords'),
      onClick: () => void router.push(buildAuditRequestLocation(record.request_id)),
    });
  }
  return actions;
});

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      activeTab.value = 'raw';
    }
  },
);

function copyValue(value: string) {
  void copyAccessLogValue(value, t);
}
</script>
<style scoped lang="less">
.access-log-detail {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-24);
}

.access-log-detail__copy-line {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.access-log-detail__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.access-log-detail :deep(.t-descriptions__content) {
  overflow-wrap: anywhere;
}

.access-log-detail__mono {
  font-family: var(--td-font-family-mono, monospace);
}
</style>
