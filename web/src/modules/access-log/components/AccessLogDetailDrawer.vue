<template>
  <t-drawer
    :visible="visible"
    :header="t('accessLog.page.detailTitle')"
    :footer="false"
    destroy-on-close
    placement="right"
    size="720px"
    @update:visible="$emit('update:visible', $event)"
  >
    <div v-if="record" class="access-log-detail">
      <section class="access-log-detail__section">
        <h4>{{ t('accessLog.detail.basic') }}</h4>
        <div class="access-log-detail__grid">
          <div class="access-log-detail__item">
            <span>{{ t('accessLog.columns.occurredAt') }}</span
            ><strong>{{ formatCompactDateTime(record.occurred_at, locale) }}</strong>
          </div>
          <div class="access-log-detail__item">
            <span>{{ t('accessLog.columns.method') }}</span
            ><strong>{{ record.method }}</strong>
          </div>
          <div class="access-log-detail__item">
            <span>{{ t('accessLog.columns.statusCode') }}</span
            ><strong>{{ record.status_code }}</strong>
          </div>
          <div class="access-log-detail__item">
            <span>{{ t('accessLog.columns.durationMs') }}</span
            ><strong>{{ record.duration_ms }} ms</strong>
          </div>
          <div class="access-log-detail__item access-log-detail__item--full">
            <span>{{ t('accessLog.columns.path') }}</span
            ><strong>{{ record.path }}</strong>
          </div>
          <div v-if="accessLogPathSecondary(record)" class="access-log-detail__item access-log-detail__item--full">
            <span>{{ t('accessLog.detail.route') }}</span
            ><strong>{{ accessLogPathSecondary(record) }}</strong>
          </div>
          <div class="access-log-detail__item access-log-detail__item--full">
            <span>{{ t('accessLog.detail.requestId') }}</span>
            <div class="access-log-detail__copy-line">
              <strong class="access-log-detail__mono">{{ record.request_id }}</strong>
              <t-button size="small" theme="default" variant="text" @click="copyValue(record.request_id)">
                {{ t('accessLog.actions.copy') }}
              </t-button>
            </div>
          </div>
          <div class="access-log-detail__item access-log-detail__item--full">
            <span>{{ t('accessLog.detail.traceId') }}</span>
            <div class="access-log-detail__copy-line">
              <strong class="access-log-detail__mono">{{ record.trace_id || '-' }}</strong>
              <t-button
                v-if="record.trace_id"
                size="small"
                theme="default"
                variant="text"
                @click="copyValue(record.trace_id)"
              >
                {{ t('accessLog.actions.copy') }}
              </t-button>
            </div>
          </div>
          <div class="access-log-detail__item">
            <span>{{ t('accessLog.detail.userId') }}</span
            ><strong>{{ accessLogUserSecondary(record, t) }}</strong>
          </div>
          <div class="access-log-detail__item">
            <span>{{ t('accessLog.detail.user') }}</span
            ><strong>{{ accessLogUserPrimary(record, t) }}</strong>
          </div>
        </div>
      </section>

      <section class="access-log-detail__section">
        <h4>{{ t('accessLog.detail.network') }}</h4>
        <div class="access-log-detail__grid">
          <div class="access-log-detail__item">
            <span>{{ t('accessLog.detail.clientIp') }}</span
            ><strong>{{ record.client_ip || '-' }}</strong>
          </div>
          <div class="access-log-detail__item access-log-detail__item--full">
            <span>{{ t('accessLog.detail.userAgent') }}</span
            ><strong>{{ record.user_agent || '-' }}</strong>
          </div>
        </div>
      </section>

      <section class="access-log-detail__section">
        <h4>{{ t('accessLog.detail.size') }}</h4>
        <div class="access-log-detail__grid">
          <div class="access-log-detail__item">
            <span>{{ t('accessLog.detail.requestSize') }}</span
            ><strong>{{ record.request_size ?? '-' }}</strong>
          </div>
          <div class="access-log-detail__item">
            <span>{{ t('accessLog.detail.responseSize') }}</span
            ><strong>{{ record.response_size ?? '-' }}</strong>
          </div>
        </div>
      </section>

      <section class="access-log-detail__section">
        <h4>{{ t('accessLog.detail.context') }}</h4>
        <details class="access-log-detail__context">
          <summary>{{ t('accessLog.detail.toggleContext') }}</summary>
          <pre>{{ JSON.stringify(record, null, 2) }}</pre>
        </details>
      </section>

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
import { MessagePlugin } from 'tdesign-vue-next';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { buildAuditRequestLocation, buildAuditTraceLocation } from '@/modules/audit/contract/deep-link';
import { formatCompactDateTime } from '@/shared/components/management';
import { copyText } from '@/shared/observability';

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
  if (record.trace_id) {
    actions.push({
      key: 'audit-trace',
      label: t('accessLog.actions.openAuditByTraceId'),
      onClick: () => void router.push(buildAuditTraceLocation(record.trace_id)),
    });
  }
  return actions;
});

async function copyValue(value: string) {
  try {
    const copied = await copyText(value);
    if (!copied) {
      MessagePlugin.error(t('accessLog.actions.copyFail'));
      return;
    }
    MessagePlugin.success(t('accessLog.actions.copySuccess'));
  } catch {
    MessagePlugin.error(t('accessLog.actions.copyFail'));
  }
}
</script>
<style scoped lang="less">
.access-log-detail {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.access-log-detail__section h4 {
  margin: 0 0 12px;
}

.access-log-detail__grid {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.access-log-detail__item {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px;
}

.access-log-detail__item--full {
  grid-column: 1 / -1;
}

.access-log-detail__copy-line {
  align-items: center;
  display: flex;
  gap: 8px;
  justify-content: space-between;
}

.access-log-detail__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.access-log-detail__context {
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-border);
  border-radius: 12px;
  padding: 12px;
}

.access-log-detail__context summary {
  cursor: pointer;
  margin-bottom: 8px;
}

.access-log-detail__context pre {
  margin: 0;
  overflow: auto;
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.access-log-detail__mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}

@media (width <= 768px) {
  .access-log-detail__grid {
    grid-template-columns: 1fr;
  }
}
</style>
