<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-drawer
    :visible="visible"
    :header="t('audit.logList.detailTitle')"
    :footer="false"
    destroy-on-close
    placement="right"
    size="820px"
    @update:visible="$emit('update:visible', $event)"
  >
    <div v-if="record" class="audit-detail">
      <section class="audit-detail__hero">
        <div>
          <strong>{{ actionTitle(record, t) }}</strong>
          <p>{{ heroDescription }}</p>
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
            <strong>{{ formatAuditTimestamp(record.created_at, locale) }}</strong>
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
        </div>
      </section>

      <section v-if="isSecurityEvent" class="audit-detail__section">
        <h4>{{ t('audit.logList.drawer.sections.security') }}</h4>
        <div class="audit-detail__security-panel">
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.eventType') }}</span>
            <strong class="audit-detail__mono">{{ eventTypeForRecord(record) }}</strong>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.permission') }}</span>
            <strong class="audit-detail__mono">{{ permissionForRecord(record) }}</strong>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.securityTarget') }}</span>
            <strong>{{ securityTargetForRecord(record, t) }}</strong>
          </div>
          <div class="audit-detail__item">
            <span>{{ t('audit.logList.drawer.fields.traceId') }}</span>
            <strong class="audit-detail__mono">{{ traceIdForRecord(record) }}</strong>
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
          <t-button
            v-if="requestIdForRecord(record) !== '-'"
            size="small"
            theme="default"
            variant="outline"
            @click="openRelatedRequest"
          >
            {{ relatedRequestActionLabel }}
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
                  {{ item.action }} · {{ formatAuditTimestamp(item.created_at, locale) }}
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
          <t-tag v-if="isSecurityEvent" theme="danger" variant="light-outline">
            {{ t('audit.logList.drawer.risk.securityEvent') }}
          </t-tag>
        </div>
      </section>

      <t-tabs v-model="activeTab">
        <t-tab-panel value="context" :label="t('audit.logList.drawer.sections.context')">
          <log-json-panel
            v-bind="jsonPanelBindings"
            :title="t('audit.logList.drawer.sections.context')"
            :value="structuredAuditContext"
          />
        </t-tab-panel>
        <t-tab-panel value="metadata" :label="t('audit.logList.drawer.sections.metadata')">
          <log-json-panel
            v-bind="jsonPanelBindings"
            :title="t('audit.logList.drawer.sections.metadata')"
            :empty-text="t('audit.logList.drawer.metadataEmpty')"
            :value="record.metadata"
          />
        </t-tab-panel>
        <t-tab-panel value="raw" :label="t('audit.logList.drawer.sections.rawJson')">
          <log-json-panel
            v-bind="jsonPanelBindings"
            :title="t('audit.logList.drawer.sections.rawJson')"
            :empty-text="t('audit.logList.drawer.rawJsonEmpty')"
            :value="record"
          />
        </t-tab-panel>
      </t-tabs>
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import type { MonitorOriginContext } from '@/modules/monitor/contract/navigation';
import { buildMonitorLocationFromOrigin } from '@/modules/monitor/contract/navigation';
import { LogJsonPanel } from '@/shared/observability';

import {
  buildAccessLogRequestLocationWithOrigin,
  buildAuditRelatedActorLocation,
  buildAuditRelatedRecordLocation,
  buildAuditRelatedResourceLocation,
} from '../contract/navigation';
import {
  actionTitle,
  actorLabel,
  eventTypeForRecord,
  formatAuditTimestamp,
  isSensitiveAction,
  metadataLookup,
  permissionForRecord,
  reasonForRecord,
  requestIdForRecord,
  resourceDetailLabel,
  resourceLabel,
  resultLabel,
  riskLabel,
  riskTone,
  securityTargetForRecord,
  sessionIdForRecord,
  sourceLabel,
  traceIdForRecord,
} from '../shared/presentation';
import { copyAuditRequestId } from '../shared/request-id-copy';
import type { AuditLogListItem } from '../types/audit';

const props = defineProps<{
  initialTab?: 'context' | 'metadata' | 'raw';
  record: AuditLogListItem | null;
  rows: AuditLogListItem[];
  visible: boolean;
  monitorOrigin?: MonitorOriginContext | null;
}>();

defineEmits<{
  (e: 'update:visible', value: boolean): void;
}>();

const { t, locale } = useI18n();
const router = useRouter();
const activeTab = ref<'context' | 'metadata' | 'raw'>('context');

const jsonPanelBindings = computed(() => ({
  expandLabel: t('audit.logList.drawer.actions.expandJson'),
  collapseLabel: t('audit.logList.drawer.actions.collapseJson'),
  copyLabel: t('audit.logList.drawer.actions.copyJson'),
  copySuccessLabel: t('audit.logList.drawer.actions.copyJsonSuccess'),
  copyFailLabel: t('audit.logList.drawer.actions.copyJsonFail'),
  emptyText: t('audit.logList.drawer.contextEmpty'),
}));

const heroDescription = computed(() => {
  const record = props.record;
  if (!record) {
    return t('audit.logList.drawer.messageFallback');
  }

  const summary = reasonForRecord(record, t).trim();
  if (summary && summary !== actionTitle(record, t)) {
    return summary;
  }

  return sourceLabel(record, t);
});

async function copyRequestId(record: AuditLogListItem) {
  await copyAuditRequestId(requestIdForRecord(record), t);
}

const monitorReturnLocation = computed(() =>
  props.monitorOrigin ? buildMonitorLocationFromOrigin(props.monitorOrigin) : null,
);
const isSecurityEvent = computed(() => props.record?.source === 'SECURITY_EVENT');
const relatedRequestActionLabel = computed(() =>
  isSecurityEvent.value
    ? t('audit.logList.drawer.actions.viewAccessLogRequest')
    : t('audit.logList.drawer.actions.viewRelatedRequest'),
);
const structuredAuditContext = computed(() => {
  const record = props.record;
  if (!record) {
    return {};
  }
  const requestId = requestIdForRecord(record);
  const traceId = traceIdForRecord(record);
  const target = record.target ?? null;

  return {
    eventOverview: {
      name: actionTitle(record, t),
      key: record.action,
      category: sourceLabel(record, t),
      type: eventTypeForRecord(record),
      result: resultLabel(record, t),
      riskLevel: record.risk_level ?? null,
      occurredAt: record.created_at,
    },
    operator: {
      userId: record.actor_user_id ?? null,
      username: record.actor_username || null,
      anonymous: !record.actor_user_id && !record.actor_username,
      ip: record.ip || null,
      userAgent: record.user_agent || null,
    },
    auditTarget: {
      targetType: record.target_type || target?.type || metadataValue(record, 'targetType', 'target_type') || null,
      targetId: target?.id || metadataValue(record, 'targetId', 'target_id') || null,
      targetName: record.target_label || target?.label || metadataValue(record, 'targetName', 'target_name') || null,
      resourceType: record.resource_type || null,
      resourceId: record.resource_id || null,
      resourceName: record.resource_name || null,
    },
    requestContext: {
      requestId: requestId === '-' ? null : requestId,
      traceId: traceId === '-' ? null : traceId,
      method: record.request_method || metadataLookup(record, 'request_method') || null,
      path: record.request_path || metadataLookup(record, 'request_path') || metadataLookup(record, 'path') || null,
      route: metadataValue(record, 'route', 'route_ref') || target?.route_ref || null,
    },
    evidenceChain: {
      accessLog: requestId === '-' ? null : { requestId },
      appLog:
        requestId === '-' && traceId === '-' ? null : { requestId: requestId === '-' ? null : requestId, traceId },
      securityEvent: isSecurityEvent.value ? { eventId: record.id, type: eventTypeForRecord(record) } : null,
      incident: metadataValue(record, 'incident', 'incident_id', 'incidentId') || null,
      evidenceLinks: metadataValue(record, 'evidence_links', 'evidenceLinks') || [],
    },
    changes: {
      before: metadataValue(record, 'before', 'old_value', 'oldValue') || null,
      after: metadataValue(record, 'after', 'new_value', 'newValue') || null,
      diff: metadataValue(record, 'diff', 'changes') || null,
      metadata: record.metadata ?? {},
    },
  };
});

function metadataValue(record: AuditLogListItem, ...keys: string[]) {
  const metadata = record.metadata;
  if (!metadata || typeof metadata !== 'object') {
    return undefined;
  }

  for (const key of keys) {
    if (Object.prototype.hasOwnProperty.call(metadata, key)) {
      return metadata[key];
    }
  }
  return undefined;
}

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      activeTab.value = props.initialTab ?? 'context';
    }
  },
);

function openMonitorContext() {
  if (!monitorReturnLocation.value) {
    return;
  }

  void router.push(monitorReturnLocation.value);
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

  void router.push(buildAccessLogRequestLocationWithOrigin(requestId, props.monitorOrigin));
}

function openRelatedRequest() {
  const requestId = props.record ? requestIdForRecord(props.record) : '-';
  if (!requestId || requestId === '-') {
    return;
  }

  void router.push(buildAccessLogRequestLocationWithOrigin(requestId, props.monitorOrigin));
}

function openRelatedActor(row: AuditLogListItem) {
  const actor = row.actor_user_id?.toString() || row.actor_username || row.actor_display_name;
  if (!actor) {
    return;
  }

  void router.push(buildAuditRelatedActorLocation(actor, row.actor_user_id, props.monitorOrigin));
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
  gap: var(--graft-density-gap-8);
  margin-bottom: var(--graft-density-gap-12);
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
  gap: var(--graft-density-gap-20);
}

.audit-detail__hero,
.audit-detail__related-grid,
.audit-detail__grid,
.audit-detail__security-panel,
.audit-detail__tags {
  display: grid;
  gap: var(--graft-density-gap-12);
}

.audit-detail__hero {
  align-items: flex-start;
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  grid-template-columns: minmax(0, 1fr) auto;
  padding: var(--graft-density-gap-16);
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

.audit-detail__security-panel {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-default);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  padding: var(--graft-density-gap-12);
}

.audit-detail__item {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
}

.audit-detail__item--full {
  grid-column: 1 / -1;
}

.audit-detail__mono {
  font-family: var(--td-font-family-mono, ui-monospace, SFMono-Regular, Consolas, 'Liberation Mono', Menlo, monospace);
  overflow-wrap: anywhere;
}

.audit-detail__copy-line {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
}

.audit-detail__related-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.audit-detail__related-card {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  gap: var(--graft-density-gap-10);
  padding: var(--graft-density-gap-14);
}

.audit-detail__related-card ul {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  margin: 0;
  padding-left: var(--graft-density-gap-16);
}

.audit-detail__tags {
  grid-template-columns: repeat(auto-fit, minmax(0, max-content));
}

@media (width <= 768px) {
  .audit-detail__hero,
  .audit-detail__grid,
  .audit-detail__security-panel,
  .audit-detail__related-grid {
    grid-template-columns: 1fr;
  }
}
</style>
