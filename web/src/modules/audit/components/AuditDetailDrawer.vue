<template>
  <t-drawer
    :visible="visible"
    :header="t('audit.logList.detailTitle')"
    :footer="false"
    destroy-on-close
    placement="right"
    size="640px"
    @update:visible="$emit('update:visible', $event)"
  >
    <div v-if="record" class="audit-detail">
      <section class="audit-detail__hero">
        <div>
          <strong>{{ record.action }}</strong>
          <p>{{ record.message || t('audit.logList.drawer.messageFallback') }}</p>
        </div>
        <t-tag :theme="riskTone(record)" variant="light-outline">{{ riskLabel(record, t) }}</t-tag>
      </section>

      <section class="audit-detail__section">
        <h4>{{ t('audit.logList.drawer.sections.basic') }}</h4>
        <div class="audit-detail__grid">
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.columns.actor') }}</span>
            <strong>{{ actorLabel(record, t) }}</strong>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.target') }}</span>
            <strong>{{ resourceDetailLabel(record, t) }}</strong>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.source') }}</span>
            <strong>{{ sourceLabel(record, t) }}</strong>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.result') }}</span>
            <strong>{{ resultLabel(record, t) }}</strong>
          </div>
          <div class="audit-detail__item audit-detail__item--full">
            <span>{{ t('audit.logList.drawer.fields.reason') }}</span>
            <strong>{{ reasonForRecord(record, t) }}</strong>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.columns.createdAt') }}</span>
            <strong>{{ formatAuditTimestamp(record.created_at) }}</strong>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.requestId') }}</span>
            <div class="audit-detail__copy-line">
              <strong class="audit-detail__mono">{{ requestIdForRecord(record) }}</strong>
              <t-button size="small" theme="default" variant="text" @click="copyRequestId(record)">
                {{ t('audit.logList.drawer.actions.copyRequestId') }}
              </t-button>
            </div>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.traceId') }}</span>
            <div class="audit-detail__copy-line">
              <strong class="audit-detail__mono">{{ traceIdForRecord(record) }}</strong>
              <t-button size="small" theme="default" variant="text" @click="copyTraceId(record)">
                {{ t('audit.logList.drawer.actions.copyTraceId') }}
              </t-button>
            </div>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.sessionId') }}</span>
            <strong class="audit-detail__mono">{{ sessionIdForRecord(record) }}</strong>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.ip') }}</span>
            <strong>{{ record.ip || '-' }}</strong>
          </div>
          <div class="audit-detail__item audit-detail__item--full">
            <span>{{ t('audit.logList.drawer.fields.userAgent') }}</span>
            <strong>{{ record.user_agent || '-' }}</strong>
          </div>
        </div>
      </section>

      <section class="audit-detail__section">
        <h4>{{ t('audit.logList.drawer.sections.request') }}</h4>
        <div class="audit-detail__grid">
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.method') }}</span>
            <strong>{{ record.request_method || metadataLookup(record, 'request_method') || '-' }}</strong>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.path') }}</span>
            <strong>{{
              record.request_path || metadataLookup(record, 'request_path') || metadataLookup(record, 'path') || '-'
            }}</strong>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.status') }}</span>
            <strong>{{
              record.status_code || metadataLookup(record, 'status_code') || metadataLookup(record, 'status') || '-'
            }}</strong>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.latency') }}</span>
            <strong>{{ metadataLookup(record, 'latency') || '-' }}</strong>
          </div>
        </div>
      </section>

      <section class="audit-detail__section">
        <h4>{{ t('audit.logList.drawer.sections.correlation') }}</h4>
        <div class="audit-detail__actions">
          <t-button
            v-if="monitorReturnLocation"
            size="small"
            theme="primary"
            variant="outline"
            @click="openMonitorContext"
          >
            {{ t('audit.logList.drawer.actions.backToMonitor') }}
          </t-button>
          <t-button v-if="incidentLocation" size="small" theme="default" variant="outline" @click="openIncident">
            {{ t('audit.logList.drawer.actions.openIncident') }}
          </t-button>
          <t-button v-if="record" size="small" theme="default" variant="outline" @click="openRelatedRecord">
            {{ t('audit.logList.drawer.actions.openRelatedEvents') }}
          </t-button>
        </div>
        <div class="audit-detail__related-grid">
          <article class="audit-detail__related-card">
            <strong>{{ t('audit.logList.drawer.related.sameRequest') }}</strong>
            <ul>
              <li v-for="item in sameRequestRows" :key="item.id">
                <button type="button" class="audit-detail__link-button" @click="openRequest(item.request_id)">
                  {{ item.action }} · {{ formatAuditTimestamp(item.created_at) }}
                </button>
              </li>
              <li v-if="sameRequestRows.length === 0">{{ t('audit.logList.drawer.related.empty') }}</li>
            </ul>
          </article>
          <article class="audit-detail__related-card">
            <strong>{{ t('audit.logList.drawer.related.sameActor') }}</strong>
            <ul>
              <li v-for="item in sameActorRows" :key="item.id">
                <button type="button" class="audit-detail__link-button" @click="openRelatedActor(item)">
                  {{ item.action }} · {{ resourceLabel(item, t) }}
                </button>
              </li>
              <li v-if="sameActorRows.length === 0">{{ t('audit.logList.drawer.related.empty') }}</li>
            </ul>
          </article>
          <article class="audit-detail__related-card">
            <strong>{{ t('audit.logList.drawer.related.sameResource') }}</strong>
            <ul>
              <li v-for="item in sameResourceRows" :key="item.id">
                <button type="button" class="audit-detail__link-button" @click="openRelatedResource(item)">
                  {{ actorLabel(item, t) }} · {{ item.action }}
                </button>
              </li>
              <li v-if="sameResourceRows.length === 0">{{ t('audit.logList.drawer.related.empty') }}</li>
            </ul>
          </article>
        </div>
      </section>

      <section class="audit-detail__section">
        <h4>{{ t('audit.logList.drawer.sections.risk') }}</h4>
        <div class="audit-detail__tags">
          <t-tag :theme="riskTone(record)" variant="light-outline">{{ riskLabel(record, t) }}</t-tag>
          <t-tag v-if="record.result === 'FAILED' || record.result === 'ERROR'" theme="danger" variant="light-outline">
            {{ t('audit.logList.drawer.risk.failedOperation') }}
          </t-tag>
          <t-tag v-if="isSensitiveAction(record)" theme="warning" variant="light-outline">
            {{ t('audit.logList.drawer.risk.sensitiveOperation') }}
          </t-tag>
          <t-tag v-if="record.request_id" theme="default" variant="light-outline">
            {{ t('audit.logList.drawer.risk.requestTrace') }}
          </t-tag>
        </div>
      </section>

      <section class="audit-detail__section">
        <h4>{{ t('audit.logList.drawer.sections.metadata') }}</h4>
        <details class="audit-detail__metadata">
          <summary>{{ t('audit.logList.drawer.actions.toggleMetadata') }}</summary>
          <pre class="audit-detail__code">{{ metadataDetail(record.metadata) }}</pre>
        </details>
      </section>
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import type { MonitorOriginContext } from '@/modules/monitor/contract/navigation';
import { buildMonitorLocationFromOrigin } from '@/modules/monitor/contract/navigation';

import {
  buildAuditIncidentLocationWithOrigin,
  buildAuditRelatedActorLocation,
  buildAuditRelatedRecordLocation,
  buildAuditRelatedResourceLocation,
  buildAuditRequestLocationWithOrigin,
} from '../contract/navigation';
import {
  actorLabel,
  formatAuditTimestamp,
  isSensitiveAction,
  metadataDetail,
  metadataLookup,
  reasonForRecord,
  requestIdForRecord,
  resourceDetailLabel,
  resourceLabel,
  resultLabel,
  riskLabel,
  riskTone,
  sessionIdForRecord,
  sourceLabel,
  traceIdForRecord,
} from '../shared/presentation';
import type { AuditLogListItem } from '../types/audit';

const props = defineProps<{
  record: AuditLogListItem | null;
  rows: AuditLogListItem[];
  visible: boolean;
  monitorOrigin?: MonitorOriginContext | null;
}>();

defineEmits<{
  (e: 'update:visible', value: boolean): void;
}>();

const { t } = useI18n();
const router = useRouter();

async function copyTraceId(record: AuditLogListItem) {
  const traceId = traceIdForRecord(record);
  if (!traceId || traceId === '-') {
    return;
  }

  try {
    await navigator.clipboard.writeText(traceId);
    MessagePlugin.success(t('audit.logList.drawer.actions.copySuccess'));
  } catch {
    MessagePlugin.error(t('audit.logList.drawer.actions.copyFail'));
  }
}

async function copyRequestId(record: AuditLogListItem) {
  const requestId = requestIdForRecord(record);
  if (!requestId || requestId === '-') {
    return;
  }

  try {
    await navigator.clipboard.writeText(requestId);
    MessagePlugin.success(t('audit.logList.drawer.actions.copyRequestIdSuccess'));
  } catch {
    MessagePlugin.error(t('audit.logList.drawer.actions.copyRequestIdFail'));
  }
}

const incidentLocation = computed(() => {
  const target = props.record?.target;
  if (target?.kind !== 'incident') {
    return null;
  }

  const eventId = Number(target.id);
  return Number.isFinite(eventId) && eventId > 0
    ? buildAuditIncidentLocationWithOrigin(eventId, props.monitorOrigin)
    : null;
});

const monitorReturnLocation = computed(() =>
  props.monitorOrigin ? buildMonitorLocationFromOrigin(props.monitorOrigin) : null,
);

function openMonitorContext() {
  if (!monitorReturnLocation.value) {
    return;
  }

  void router.push(monitorReturnLocation.value);
}

function openIncident() {
  if (!incidentLocation.value) {
    return;
  }

  void router.push(incidentLocation.value);
}

function openRelatedRecord() {
  if (!props.record) {
    return;
  }

  void router.push(buildAuditRelatedRecordLocation(props.record, props.monitorOrigin));
}

function openRequest(requestId?: string | null) {
  if (!requestId) {
    return;
  }

  void router.push(buildAuditRequestLocationWithOrigin(requestId, props.monitorOrigin));
}

function openRelatedActor(row: AuditLogListItem) {
  const actor = row.actor_display_name || row.actor_username;
  if (!actor) {
    return;
  }

  void router.push(buildAuditRelatedActorLocation(actor, props.monitorOrigin));
}

function openRelatedResource(row: AuditLogListItem) {
  if (!row.resource_type || !row.resource_id) {
    return;
  }

  void router.push(
    buildAuditRelatedResourceLocation(row.resource_type, row.resource_id, row.resource_name, props.monitorOrigin),
  );
}

const sameRequestRows = computed(() => {
  const record = props.record;
  if (!record?.request_id) {
    return [];
  }

  return props.rows.filter((item) => item.id !== record.id && item.request_id === record.request_id).slice(0, 3);
});

const sameActorRows = computed(() => {
  const record = props.record;
  if (!record?.actor_user_id) {
    return [];
  }

  return props.rows.filter((item) => item.id !== record.id && item.actor_user_id === record.actor_user_id).slice(0, 3);
});

const sameResourceRows = computed(() => {
  const record = props.record;
  if (!record?.resource_id) {
    return [];
  }

  return props.rows
    .filter(
      (item) =>
        item.id !== record.id && item.resource_type === record.resource_type && item.resource_id === record.resource_id,
    )
    .slice(0, 3);
});
</script>
<style scoped lang="less">
.audit-detail,
.audit-detail__section,
.audit-detail__related-card {
  display: flex;
  flex-direction: column;
}

.audit-detail__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.audit-detail__link-button {
  background: transparent;
  border: 0;
  color: var(--td-brand-color);
  cursor: pointer;
  padding: 0;
  text-align: left;
}

.audit-detail {
  gap: 20px;
}

.audit-detail__hero,
.audit-detail__related-grid,
.audit-detail__grid,
.audit-detail__tags {
  display: grid;
  gap: 12px;
}

.audit-detail__hero {
  align-items: flex-start;
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  grid-template-columns: minmax(0, 1fr) auto;
  padding: 16px;
}

.audit-detail__hero p,
.audit-detail__item span,
.audit-detail__related-card li {
  color: var(--td-text-color-secondary);
}

.audit-detail__hero p,
.audit-detail__section h4 {
  margin: 0;
}

.audit-detail__grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.audit-detail__item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.audit-detail__item--full {
  grid-column: 1 / -1;
}

.audit-detail__mono {
  font-family: var(--td-font-family-medium);
}

.audit-detail__copy-line {
  align-items: center;
  display: flex;
  gap: 8px;
}

.audit-detail__related-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.audit-detail__related-card {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  gap: 10px;
  padding: 14px;
}

.audit-detail__related-card ul {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding-left: 16px;
}

.audit-detail__tags {
  grid-template-columns: repeat(auto-fit, minmax(120px, max-content));
}

.audit-detail__code {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  margin: 0;
  overflow: auto;
  padding: 12px;
}

.audit-detail__metadata summary {
  cursor: pointer;
  margin-bottom: 8px;
}

@media (width <= 768px) {
  .audit-detail__hero,
  .audit-detail__grid,
  .audit-detail__related-grid {
    grid-template-columns: 1fr;
  }
}
</style>
