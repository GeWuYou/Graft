<template>
  <div class="project-source-page" data-page-type="selector">
    <management-page-content>
      <management-page-header
        title-key="project.create.title"
        description-key="project.create.description"
        :source="{ labelKey: 'project.create.eyebrow', fallback: t('project.create.eyebrow') }"
      />

      <t-alert v-if="loadError" theme="warning" :message="loadError" class="project-source-page__notice" />

      <div class="project-source-page__grid">
        <t-card
          v-for="entry in entries"
          :key="entry.type"
          :bordered="true"
          :title="entry.display_name"
          class="project-source-page__card"
        >
          <template #actions>
            <t-tag :theme="entry.status === 'ready' ? 'success' : 'warning'" variant="light-outline">
              {{ statusLabel(entry.status) }}
            </t-tag>
          </template>

          <t-space direction="vertical" size="small" class="project-source-page__body">
            <p>{{ entry.description }}</p>
            <t-descriptions size="small" :column="1" bordered>
              <t-descriptions-item :label="t('project.createSource.routePath')">
                <code>{{ entry.route_path }}</code>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.createSource.permission')">
                <code>{{ entry.permission }}</code>
              </t-descriptions-item>
              <t-descriptions-item :label="t('project.createSource.metadataFields')">
                {{ entry.metadata_fields.join(', ') || '-' }}
              </t-descriptions-item>
            </t-descriptions>
            <t-alert v-if="entry.status_reason" theme="info" :message="entry.status_reason" />
            <t-button
              theme="primary"
              :disabled="entry.status !== 'ready' && entry.type !== 'git' && entry.type !== 'template'"
              @click="openEntry(entry)"
            >
              {{ actionLabel(entry) }}
            </t-button>
          </t-space>
        </t-card>
      </div>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { LOCALE, type LocalizedTitle } from '@/contracts/i18n/locales';
import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { useTabsRouterStore } from '@/store/modules/tabs-router';

import { getProjectSources } from '../../api/project';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { appendResolvedTab } from '../../shared/navigation';
import type { ProjectSourceEntry } from '../../types/project';

defineOptions({
  name: 'ProjectCreateSourceIndex',
});

const router = useRouter();
const tabsRouterStore = useTabsRouterStore();
const { t } = useI18n();

const entries = ref<ProjectSourceEntry[]>([]);
const loadError = ref('');

onMounted(() => {
  void loadSources();
});

async function loadSources() {
  loadError.value = '';
  try {
    const response = await getProjectSources();
    entries.value = response.items;
  } catch (error) {
    loadError.value = resolveLocalizedErrorMessage(t, error, t('project.createSource.messages.loadFailed'));
  }
}

function statusLabel(status: ProjectSourceEntry['status']) {
  return status === 'ready' ? t('project.createSource.status.ready') : t('project.createSource.status.planned');
}

function actionLabel(entry: ProjectSourceEntry) {
  return entry.status === 'ready'
    ? t('project.createSource.actions.open')
    : t('project.createSource.actions.reviewBoundary');
}

function openEntry(entry: ProjectSourceEntry) {
  const routeName =
    entry.type === 'managed'
      ? PROJECT_BOOTSTRAP_ROUTE.CREATE_MANAGED.pageRouteName
      : entry.type === 'git'
        ? PROJECT_BOOTSTRAP_ROUTE.CREATE_GIT.pageRouteName
        : PROJECT_BOOTSTRAP_ROUTE.CREATE_TEMPLATE.pageRouteName;
  const resolved = router.resolve({ name: routeName });
  const title: LocalizedTitle = {
    [LOCALE.ZH_CN]: entry.display_name,
    [LOCALE.EN_US]: entry.display_name,
  };
  appendResolvedTab(tabsRouterStore, resolved, title);
  void router.push({ name: routeName });
}
</script>
<style scoped>
.project-source-page__notice,
.project-source-page__grid,
.project-source-page__body {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.project-source-page {
  min-height: 100%;
}

.project-source-page__grid {
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  margin-top: var(--graft-density-gap-16);
}

.project-source-page__card :deep(.t-card__body) {
  height: 100%;
}
</style>
