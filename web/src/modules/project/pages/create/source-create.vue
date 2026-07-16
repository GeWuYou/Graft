<template>
  <management-page-content>
    <management-page-header
      title-key="project.route.createTemplate.title"
      :description="t('project.sourceCreate.templateDescription')"
      :source="{ labelKey: 'project.creation.eyebrow', fallback: t('project.creation.eyebrow') }"
    >
      <template #actions>
        <t-space size="small">
          <t-button variant="outline" @click="goToSource">{{ t('project.create.actions.backToSource') }}</t-button>
          <t-button @click="loadTemplates">{{ t('project.create.actions.refresh') }}</t-button>
        </t-space>
      </template>
    </management-page-header>

    <t-alert v-if="loadError" theme="warning" :message="loadError" />
    <div v-else class="source-create__grid" :aria-busy="loading">
      <t-card v-for="item in templates" :key="item.template_id" bordered class="source-create__card">
        <template #header>
          <div class="source-create__header">
            <div>
              <h2>{{ item.display_name }}</h2>
              <p>{{ item.description || t('project.sourceCreate.noDescription') }}</p>
            </div>
            <t-tag theme="success" variant="light-outline">{{
              t('project.sourceCreate.versionLabel', { version: item.version.version_number })
            }}</t-tag>
          </div>
        </template>
        <t-descriptions size="small" :column="1">
          <t-descriptions-item :label="t('project.sourceCreate.adapter')">{{
            item.deployment_adapter_kind
          }}</t-descriptions-item>
          <t-descriptions-item :label="t('project.sourceCreate.version')">{{
            item.version.template_version_id
          }}</t-descriptions-item>
        </t-descriptions>
        <template #footer>
          <t-button theme="primary" :loading="selecting === item.template_id" @click="selectTemplate(item)">
            {{ t('project.sourceCreate.useTemplate') }}
          </t-button>
        </template>
      </t-card>
      <t-empty v-if="!loading && templates.length === 0" :description="t('project.sourceCreate.emptyTemplates')" />
    </div>
  </management-page-content>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { getApplicationTemplates } from '../../api/project';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { navigateToApplicationCreateSource } from '../../shared/navigation';
import type { ApplicationTemplate } from '../../types/project';

// 模板目录页只读取创建者可见的发布快照；选择后交由统一受管创建编辑器持有可修改的应用草稿。
defineOptions({ name: 'ApplicationTemplateCatalog' });

const { t } = useI18n();
const router = useRouter();
const route = useRoute();
const templates = ref<ApplicationTemplate[]>([]);
const loading = ref(false);
const selecting = ref('');
const loadError = ref('');

onMounted(() => void loadTemplates());

async function loadTemplates() {
  loading.value = true;
  loadError.value = '';
  try {
    templates.value = (await getApplicationTemplates()).items.filter(
      (item) => item.deployment_adapter_kind === 'compose' && item.version.status === 'published',
    );
  } catch (error) {
    loadError.value = resolveLocalizedErrorMessage(t, error, t('project.sourceCreate.templatesLoadFailed'));
  } finally {
    loading.value = false;
  }
}

async function selectTemplate(item: ApplicationTemplate) {
  if (route.query.deployment !== 'compose' || !/^\d+$/.test(String(route.query.runtime_target_id || ''))) {
    MessagePlugin.warning(t('project.runtimeTarget.unavailableTooltip'));
    goToSource();
    return;
  }
  selecting.value = item.template_id;
  await router.push({
    name: PROJECT_BOOTSTRAP_ROUTE.CREATE_BLANK.pageRouteName,
    query: { ...route.query, template_id: item.template_id, template_version_id: item.version.template_version_id },
  });
  selecting.value = '';
}

function goToSource() {
  navigateToApplicationCreateSource(router, route.query);
}
</script>
<style scoped>
.source-create__grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  margin-top: var(--graft-density-gap-16);
}

.source-create__card {
  display: flex;
  flex-direction: column;
}

.source-create__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.source-create__header h2,
.source-create__header p {
  margin: 0;
}

.source-create__header p {
  color: var(--td-text-color-secondary);
  margin-top: var(--graft-density-gap-4);
}
</style>
