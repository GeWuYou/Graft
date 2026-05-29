<template>
  <div class="audit-incident-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header :title="incidentTitle" :description="incidentDescription">
        <template #eyebrow>{{ t('menu.audit.title') }}</template>
        <template #actions>
          <t-space size="small" wrap>
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
                  {{ formatAuditTimestamp(incident.incident.started_at) }} -
                  {{ formatAuditTimestamp(incident.incident.ended_at) }}
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
                <p class="audit-incident-page__text">{{ incident.monitor_context.reason }}</p>
              </t-space>
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
                    <span>{{ formatAuditTimestamp(item.created_at) }}</span>
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
                      >{{ formatAuditTimestamp(request.started_at) }} -
                      {{ formatAuditTimestamp(request.ended_at) }}</span
                    >
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

import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import { ManagementEmptyState, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { createLogger } from '@/utils/logger';

import { getAuditIncident } from '../../api/audit';
import { buildAuditRequestLocation } from '../../contract/deep-link';
import { formatAuditTimestamp, resourceLabel, riskTone } from '../../shared/presentation';
import type { AuditIncidentResponse } from '../../types/audit';

defineOptions({
  name: 'AuditIncidentIndex',
});

const route = useRoute();
const router = useRouter();
const { t } = useI18n();
const logger = createLogger('audit.incident');
const loading = ref(false);
const errorMessage = ref('');
const incident = ref<AuditIncidentResponse | null>(null);

const eventId = computed(() => Number(route.params.eventId));
const incidentTitle = computed(() => incident.value?.incident.title ?? t('audit.incident.title'));
const incidentDescription = computed(() => incident.value?.incident.summary ?? t('audit.incident.description'));

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

function openSeedRequest() {
  const requestId = incident.value?.seed_event.request_id;
  if (!requestId) {
    return;
  }
  void router.push(buildAuditRequestLocation(requestId));
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
</style>
