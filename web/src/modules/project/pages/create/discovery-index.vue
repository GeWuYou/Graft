<template>
  <div class="project-discovery-page" data-page-type="detail">
    <management-page-content>
      <management-page-header
        title-key="project.route.createDiscovery.title"
        description-key="project.discovery.description"
        :source="{ labelKey: 'project.create.eyebrow', fallback: t('project.create.eyebrow') }"
      >
        <template #actions>
          <t-space size="small" break-line>
            <t-button theme="primary" variant="outline" :loading="loading" @click="loadCandidates">
              {{ t('project.discovery.refresh') }}
            </t-button>
          </t-space>
        </template>
      </management-page-header>

      <t-alert v-if="loadError" theme="warning" :message="loadError" class="project-discovery-page__notice" />
      <t-alert
        v-else-if="response?.status_reason"
        theme="info"
        :message="response.status_reason"
        class="project-discovery-page__notice"
      />

      <t-card :bordered="true" class="project-discovery-page__summary">
        <t-descriptions size="small" :column="1" bordered>
          <t-descriptions-item :label="t('project.discovery.authorityRoot')">
            <code>{{ response?.authority_root || '-' }}</code>
          </t-descriptions-item>
          <t-descriptions-item :label="t('project.discovery.capabilities')">
            {{
              response?.supports_scan || response?.supports_auto_discovery
                ? t('project.discovery.capabilitiesReady')
                : t('project.discovery.capabilitiesBlocked')
            }}
          </t-descriptions-item>
        </t-descriptions>
      </t-card>

      <div v-if="items.length" class="project-discovery-page__grid">
        <t-card v-for="item in items" :key="item.candidate_key" :title="item.display_name" :bordered="true">
          <template #actions>
            <t-tag :theme="statusTheme(item.status)" variant="light-outline">
              {{ statusLabel(item.status) }}
            </t-tag>
          </template>
          <t-space direction="vertical" size="small" class="project-discovery-page__body">
            <t-descriptions size="small" :column="1" bordered>
              <t-descriptions-item :label="t('project.discovery.workingDirectory')">
                <code>{{ item.working_directory }}</code>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.discovery.candidateKind')">
                {{ candidateKindLabel(item.candidate_kind) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.discovery.recommendedAction')">
                {{ recommendedActionLabel(item.recommended_action) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.discovery.serviceCount')">
                {{ item.service_count }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.discovery.composeFiles')">
                {{
                  item.compose_files
                    .map((file: ProjectDiscoveryCandidate['compose_files'][number]) => file.display_path)
                    .join(', ') || '-'
                }}
              </t-descriptions-item>
            </t-descriptions>
            <t-alert v-if="item.status_reason" theme="info" :message="item.status_reason" />
            <t-alert
              v-if="item.conflicts.length"
              theme="warning"
              :message="t('project.discovery.conflictsValue', { value: item.conflicts.join(', ') })"
            />
          </t-space>
        </t-card>
      </div>

      <management-empty-state
        v-else
        tone="default"
        :title="t('project.discovery.emptyTitle')"
        :description="t('project.discovery.emptyDescription')"
      />
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { ManagementEmptyState, ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { getProjectDiscoveryCandidates } from '../../api/project';
import type {
  ProjectDiscoveryCandidate,
  ProjectDiscoveryCandidatesResponse,
  ProjectDiscoveryCandidateStatus,
} from '../../types/project';

defineOptions({
  name: 'ProjectDiscoveryCandidateIndex',
});

const { t } = useI18n();
const loading = ref(false);
const loadError = ref('');
const response = ref<ProjectDiscoveryCandidatesResponse | null>(null);
const items = ref<ProjectDiscoveryCandidate[]>([]);

onMounted(() => {
  void loadCandidates();
});

async function loadCandidates() {
  loading.value = true;
  loadError.value = '';
  try {
    const result = await getProjectDiscoveryCandidates();
    response.value = result;
    items.value = result.items;
  } catch (error) {
    loadError.value = resolveLocalizedErrorMessage(t, error, t('project.discovery.loadFailed'));
    response.value = null;
    items.value = [];
  } finally {
    loading.value = false;
  }
}

function statusTheme(status: ProjectDiscoveryCandidateStatus) {
  if (status === 'ready') return 'success';
  if (status === 'conflict') return 'warning';
  return 'default';
}

function statusLabel(status: ProjectDiscoveryCandidateStatus) {
  return t(`project.discovery.status.${status}`);
}

function candidateKindLabel(kind: ProjectDiscoveryCandidate['candidate_kind']) {
  return t(`project.discovery.kinds.${kind}`);
}

function recommendedActionLabel(action: ProjectDiscoveryCandidate['recommended_action']) {
  return t(`project.discovery.actions.${action}`);
}
</script>
<style scoped>
.project-discovery-page__notice,
.project-discovery-page__grid,
.project-discovery-page__body {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.project-discovery-page__summary {
  margin-top: var(--graft-density-gap-16);
}

.project-discovery-page__grid {
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  margin-top: var(--graft-density-gap-16);
}
</style>
