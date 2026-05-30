<template>
  <div class="audit-incident-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header :title="incidentTitle" :description="incidentDescription">
        <template #eyebrow>{{ t('menu.audit.title') }}</template>
        <template #actions>
          <t-space size="small" wrap>
            <t-button v-if="monitorReturnLocation" theme="primary" variant="outline" @click="returnToMonitor">
              {{ t('audit.incident.actions.backToMonitor') }}
            </t-button>
            <t-button theme="default" variant="outline" @click="openSeedRequest">
              {{ t('audit.incident.actions.openRequest') }}
            </t-button>
            <t-button theme="default" variant="outline" :loading="loading" @click="fetchIncident">
              {{ t('audit.incident.actions.refresh') }}
            </t-button>
          </t-space>
        </template>
      </management-page-header>

      <management-empty-state
        v-if="errorMessage && !loading"
        tone="error"
        :title="t('audit.incident.errorTitle')"
        :description="errorMessage"
      >
        <template #actions>
          <t-button theme="primary" variant="outline" @click="fetchIncident">
            {{ t('audit.incident.actions.retry') }}
          </t-button>
        </template>
      </management-empty-state>

      <template v-else-if="incident">
        <t-row :gutter="[16, 16]">
          <t-col :xs="12" :xl="8">
            <t-card :title="t('audit.incident.sections.summary')">
              <t-descriptions bordered :column="1">
                <t-descriptions-item :label="t('audit.incident.fields.riskLevel')">
                  <t-tag :theme="riskTone(incident.seed_event)" variant="light-outline">
                    {{ t(`audit.common.risk.${incident.incident.risk_level}`) }}
                  </t-tag>
                </t-descriptions-item>
                <t-descriptions-item :label="t('audit.incident.fields.window')">
                  {{ formatAuditTimestamp(incident.incident.started_at, locale) }} -
                  {{ formatAuditTimestamp(incident.incident.ended_at, locale) }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('audit.incident.fields.reason')">
                  {{ incident.incident.correlation_reason }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('audit.incident.fields.seedAction')">
                  {{ incident.seed_event.action }}
                </t-descriptions-item>
                <t-descriptions-item :label="t('audit.incident.fields.seedResource')">
                  {{ resourceLabel(incident.seed_event, t) }}
                </t-descriptions-item>
              </t-descriptions>
            </t-card>
          </t-col>

          <t-col :xs="12" :xl="4">
            <t-card :title="t('audit.incident.sections.monitorContext')">
              <t-space direction="vertical" size="small" style="width: 100%">
                <t-tag :theme="monitorStateTheme(incident.monitor_context.state)" variant="light-outline">
                  {{ t(`audit.incident.monitorState.${incident.monitor_context.state}`) }}
                </t-tag>
                <p class="audit-incident-page__text">{{ incident.monitor_context.summary }}</p>
                <p
                  v-if="incident.monitor_context.reason"
                  class="audit-incident-page__text audit-incident-page__text--subtle"
                >
                  {{ incident.monitor_context.reason }}
                </p>
                <t-space v-if="monitorContextMeta.length" size="8px" wrap>
                  <t-tag
                    v-for="item in monitorContextMeta"
                    :key="item.label"
                    theme="default"
                    variant="light-outline"
                    size="small"
                  >
                    {{ item.label }}
                  </t-tag>
                </t-space>
                <t-button
                  v-if="monitorReturnLocation"
                  theme="primary"
                  variant="text"
                  size="small"
                  @click="returnToMonitor"
                >
                  {{ t('audit.incident.actions.openMonitorContext') }}
                </t-button>
              </t-space>
            </t-card>
          </t-col>
        </t-row>

        <t-row
          v-if="incident.monitor_context.evidence_links.length"
          :gutter="[16, 16]"
          class="audit-incident-page__panels"
        >
          <t-col :xs="12">
            <t-card :title="t('audit.incident.sections.evidenceLinks')">
              <t-list split>
                <t-list-item
                  v-for="(link, index) in incident.monitor_context.evidence_links"
                  :key="`${link.target_kind}-${index}`"
                >
                  <t-space direction="vertical" size="4">
                    <strong>{{ link.title }}</strong>
                    <span class="audit-incident-page__text">{{ evidenceStateLabel(link.link_state) }}</span>
                    <span v-if="link.reason" class="audit-incident-page__text">{{ link.reason }}</span>
                    <span v-if="link.time_window" class="audit-incident-page__text audit-incident-page__text--subtle">
                      {{
                        t('audit.incident.evidenceWindow', {
                          from: formatAuditTimestamp(link.time_window.created_from, locale),
                          to: formatAuditTimestamp(link.time_window.created_to, locale),
                        })
                      }}
                    </span>
                    <t-button
                      v-if="evidenceTargetLocation(link)"
                      size="small"
                      theme="primary"
                      variant="text"
                      @click="openEvidenceLink(link)"
                    >
                      {{ t('audit.incident.actions.openEvidenceLink') }}
                    </t-button>
                  </t-space>
                </t-list-item>
              </t-list>
            </t-card>
          </t-col>
        </t-row>

        <t-row :gutter="[16, 16]" class="audit-incident-page__panels">
          <t-col :xs="12" :xl="6">
            <t-card :title="t('audit.incident.sections.relatedEvents')">
              <t-list split>
                <t-list-item v-for="item in incident.related_events" :key="item.id">
                  <t-space direction="vertical" size="2">
                    <strong>{{ item.action }}</strong>
                    <span>{{ resourceLabel(item, t) }}</span>
                    <span>{{ formatAuditTimestamp(item.created_at, locale) }}</span>
                    <t-button
                      v-if="item.request_id"
                      size="small"
                      theme="primary"
                      variant="text"
                      @click="openRequest(item.request_id)"
                    >
                      {{ t('audit.incident.actions.openRelatedRequest') }}
                    </t-button>
                  </t-space>
                </t-list-item>
              </t-list>
            </t-card>
          </t-col>

          <t-col :xs="12" :xl="6">
            <t-card :title="t('audit.incident.sections.relatedActors')">
              <t-list split>
                <t-list-item
                  v-for="actor in incident.related_actors"
                  :key="`${actor.actor_user_id ?? 'guest'}-${actor.actor_username ?? actor.actor_display_name ?? 'unknown'}`"
                >
                  <t-space direction="vertical" size="2">
                    <strong>{{
                      actor.actor_display_name || actor.actor_username || t('audit.common.unknownActor')
                    }}</strong>
                    <span>{{ t('audit.incident.eventCount', { count: actor.event_count }) }}</span>
                    <t-button
                      v-if="actor.actor_user_id || actor.actor_username || actor.actor_display_name"
                      size="small"
                      theme="primary"
                      variant="text"
                      @click="openActor(actor.actor_username || actor.actor_display_name || '', actor.actor_user_id)"
                    >
                      {{ t('audit.incident.actions.openActorEvents') }}
                    </t-button>
                  </t-space>
                </t-list-item>
              </t-list>
            </t-card>
          </t-col>

          <t-col :xs="12" :xl="6">
            <t-card :title="t('audit.incident.sections.relatedResources')">
              <t-list split>
                <t-list-item
                  v-for="resource in incident.related_resources"
                  :key="`${resource.resource_type}:${resource.resource_id}`"
                >
                  <t-space direction="vertical" size="2">
                    <strong>{{ resource.resource_name || resource.resource_type }}</strong>
                    <span>{{ resource.resource_type }} / {{ resource.resource_id }}</span>
                    <span>{{ t('audit.incident.eventCount', { count: resource.event_count }) }}</span>
                    <t-button
                      size="small"
                      theme="primary"
                      variant="text"
                      @click="
                        openResource(resource.resource_type, resource.resource_id, resource.resource_name ?? undefined)
                      "
                    >
                      {{ t('audit.incident.actions.openResourceEvents') }}
                    </t-button>
                  </t-space>
                </t-list-item>
              </t-list>
            </t-card>
          </t-col>

          <t-col :xs="12" :xl="6">
            <t-card :title="t('audit.incident.sections.relatedRequests')">
              <t-list split>
                <t-list-item v-for="request in incident.related_requests" :key="request.request_id">
                  <t-space direction="vertical" size="2">
                    <strong>{{ request.request_id }}</strong>
                    <span>{{ t('audit.incident.eventCount', { count: request.event_count }) }}</span>
                    <span
                      >{{ formatAuditTimestamp(request.started_at, locale) }} -
                      {{ formatAuditTimestamp(request.ended_at, locale) }}</span
                    >
                    <t-button size="small" theme="primary" variant="text" @click="openRequest(request.request_id)">
                      {{ t('audit.incident.actions.openRelatedRequest') }}
                    </t-button>
                  </t-space>
                </t-list-item>
              </t-list>
            </t-card>
          </t-col>
        </t-row>
      </template>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { buildAccessLogRequestLocation } from '@/modules/access-log/contract/deep-link';
import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import { ManagementEmptyState, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { createLogger } from '@/utils/logger';

import { getAuditIncident } from '../../api/audit';
import { buildAuditEvidenceTargetLocation } from '../../contract/deep-link';
import {
  buildAuditRelatedActorLocation,
  buildAuditRelatedResourceLocation,
  buildMonitorReturnLocation,
  resolveAuditNavigationContext,
  withMonitorOrigin,
} from '../../contract/navigation';
import { formatAuditTimestamp, resourceLabel, riskTone } from '../../shared/presentation';
import type { AuditIncidentMonitorContext, AuditIncidentResponse, EvidenceLink } from '../../types/audit';

defineOptions({
  name: 'AuditIncidentIndex',
});

const route = useRoute();
const router = useRouter();
const { t, locale } = useI18n();
const logger = createLogger('audit.incident');
const loading = ref(false);
const errorMessage = ref('');
const incident = ref<AuditIncidentResponse | null>(null);

const eventId = computed(() => Number(route.params.eventId));
const incidentTitle = computed(() => incident.value?.incident.title ?? t('audit.incident.title'));
const incidentDescription = computed(() => incident.value?.incident.summary ?? t('audit.incident.description'));
const navigationContext = computed(() => resolveAuditNavigationContext(route.query));
const monitorReturnLocation = computed(() => buildMonitorReturnLocation(route.query));
const monitorContextMeta = computed(() => {
  const context = incident.value?.monitor_context;
  if (!context) {
    return [];
  }

  return [
    context.anomaly_key ? { label: t(`audit.incident.anomalyKey.${context.anomaly_key}`) } : null,
    context.scope_kind && context.scope_ref
      ? {
          label: t('audit.incident.scopeLabel', {
            kind: t(`audit.incident.scopeKind.${context.scope_kind}`),
            ref: context.scope_ref,
          }),
        }
      : null,
    context.observed_at
      ? { label: t('audit.incident.observedAt', { value: formatAuditTimestamp(context.observed_at, locale.value) }) }
      : null,
  ].filter((item): item is { label: string } => Boolean(item));
});

function monitorStateTheme(state: AuditIncidentResponse['monitor_context']['state']) {
  switch (state) {
    case 'available':
      return 'success';
    case 'partial':
      return 'warning';
    default:
      return 'default';
  }
}

function evidenceStateLabel(state: AuditIncidentMonitorContext['evidence_links'][number]['link_state']) {
  return t(`audit.incident.evidenceState.${state}`);
}

function evidenceTargetLocation(link: EvidenceLink) {
  return buildAuditEvidenceTargetLocation(link, navigationContext.value.monitorOrigin);
}

function openEvidenceLink(link: EvidenceLink) {
  const target = evidenceTargetLocation(link);
  if (!target) {
    return;
  }

  void router.push(target);
}

function openSeedRequest() {
  const requestId = incident.value?.seed_event.request_id;
  if (!requestId) {
    return;
  }
  void router.push(withMonitorOrigin(buildAccessLogRequestLocation(requestId), navigationContext.value.monitorOrigin));
}

function openRequest(requestId: string) {
  void router.push(withMonitorOrigin(buildAccessLogRequestLocation(requestId), navigationContext.value.monitorOrigin));
}

function openActor(actor: string, actorUserId?: number | null) {
  void router.push(
    buildAuditRelatedActorLocation(
      actor || String(actorUserId ?? ''),
      actorUserId,
      navigationContext.value.monitorOrigin,
    ),
  );
}

function openResource(resourceType: string, resourceId: string, resourceName?: string) {
  void router.push(
    buildAuditRelatedResourceLocation(resourceType, resourceId, resourceName, navigationContext.value.monitorOrigin),
  );
}

function returnToMonitor() {
  if (!monitorReturnLocation.value) {
    return;
  }
  void router.push(monitorReturnLocation.value);
}

async function fetchIncident() {
  if (!Number.isFinite(eventId.value) || eventId.value <= 0) {
    incident.value = null;
    errorMessage.value = t('audit.incident.invalidEventId');
    return;
  }

  loading.value = true;
  errorMessage.value = '';

  try {
    incident.value = await getAuditIncident(eventId.value);
  } catch (error) {
    incident.value = null;
    logger.error('failed to fetch audit incident', error);
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('audit.incident.loadFailed'));
    MessagePlugin.error(errorMessage.value);
  } finally {
    loading.value = false;
  }
}
watch(() => route.params.eventId, fetchIncident);
onMounted(fetchIncident);
</script>
<style scoped>
.audit-incident-page__panels {
  margin-top: 16px;
}

.audit-incident-page__text {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.audit-incident-page__text--subtle {
  color: var(--td-text-color-placeholder);
}
</style>
