<template>
  <div class="template-catalog-detail" data-page-type="catalog-discovery">
    <management-page-content>
      <management-page-header
        title-key="project.route.createTemplateDetail.title"
        :description="templateRecord?.description || t('project.templateCatalog.detailDescription')"
        :source="{ labelKey: 'project.creation.eyebrow', fallback: t('project.creation.eyebrow') }"
      >
        <template #actions>
          <t-space size="small">
            <t-button variant="outline" @click="backToCatalog">{{
              t('project.templateCatalog.backToCatalog')
            }}</t-button>
            <t-button v-if="templateRecord" theme="primary" :loading="selecting" @click="useTemplate">{{
              t('project.sourceCreate.useTemplate')
            }}</t-button>
          </t-space>
        </template>
        <template #meta>
          <t-space v-if="templateRecord" size="small">
            <t-tag variant="light-outline">{{ categoryLabel(templateRecord.category) }}</t-tag>
            <t-tag variant="light-outline">{{ templateRecord.deployment_adapter_kind }}</t-tag>
            <t-tag variant="light-outline">{{
              t('project.templates.versionValue', { version: templateRecord.version.version_number })
            }}</t-tag>
          </t-space>
        </template>
      </management-page-header>

      <t-loading :loading="loading" class="template-catalog-detail__loading">
        <t-alert v-if="errorMessage" theme="error" :message="errorMessage"
          ><template #operation
            ><t-button theme="danger" variant="text" @click="loadTemplate">{{
              t('project.templates.retry')
            }}</t-button></template
          ></t-alert
        >
        <template v-else-if="templateRecord">
          <div class="template-catalog-detail__grid">
            <t-card bordered :title="t('project.templateCatalog.readmeTitle')">
              <markdown-viewer v-if="documentation.readme_markdown" :source="documentation.readme_markdown" />
              <t-empty v-else :description="t('project.templateCatalog.noReadme')" />
            </t-card>
            <t-card bordered :title="t('project.templateCatalog.variablesTitle')">
              <t-table
                v-if="documentation.variables?.length"
                :data="documentation.variables"
                :columns="variableColumns"
                size="small"
                row-key="name"
                :pagination="undefined"
              >
                <template #required="{ row }"
                  ><t-tag :theme="row.required ? 'warning' : 'default'" variant="light-outline">{{
                    row.required ? t('project.templateCatalog.required') : t('project.templateCatalog.optional')
                  }}</t-tag></template
                >
              </t-table>
              <t-empty v-else :description="t('project.templateCatalog.noVariables')" />
            </t-card>
          </div>
          <t-card
            bordered
            class="template-catalog-detail__workspace"
            :title="t('project.templateCatalog.workspaceTitle')"
          >
            <t-table
              :data="workspaceEntries"
              :columns="workspaceColumns"
              size="small"
              row-key="path"
              :pagination="undefined"
            >
              <template #kind="{ row }"
                ><t-tag variant="light-outline">{{
                  row.node_type === 'directory'
                    ? t('project.templateCatalog.directory')
                    : t('project.templateCatalog.file')
                }}</t-tag></template
              >
              <template #preview="{ row }"
                ><code v-if="row.node_type === 'file'">{{ preview(row.content) }}</code
                ><span v-else>-</span></template
              >
              <template #empty><t-empty :description="t('project.templateCatalog.noWorkspace')" /></template>
            </t-table>
          </t-card>
        </template>
      </t-loading>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { MarkdownViewer } from '@/shared/components/markdown';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { getPublishedApplicationTemplate } from '../../api/project';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import type { ApplicationTemplate, ApplicationTemplateCategory } from '../../types/project';

// 目录详情只能读取当前发布快照，避免创建者页面通过模板管理接口接触草稿或归档版本。
defineOptions({ name: 'ApplicationTemplateCatalogDetailIndex' });

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const templateRecord = ref<ApplicationTemplate | null>(null);
const loading = ref(false);
const selecting = ref(false);
const errorMessage = ref('');
const loadRequestId = ref(0);
const templateId = computed(() => String(route.params.templateId || ''));
const definition = computed(() => (templateRecord.value?.version.definition ?? {}) as Record<string, unknown>);
const documentation = computed(
  () =>
    (definition.value.catalog_documentation ?? {}) as {
      readme_markdown?: string;
      variables?: { name: string; required: boolean; description: string }[];
    },
);
const workspaceEntries = computed(() =>
  Array.isArray(definition.value.workspace_entries)
    ? definition.value.workspace_entries.map((entry) => ({
        path: String((entry as Record<string, unknown>).path || ''),
        node_type: (entry as Record<string, unknown>).node_type === 'directory' ? 'directory' : 'file',
        content: String((entry as Record<string, unknown>).content || ''),
      }))
    : [],
);
const variableColumns = computed<TableProps['columns']>(() => [
  { colKey: 'name', title: t('project.templateCatalog.variableName'), width: 180 },
  { colKey: 'required', title: t('project.templateCatalog.variableRequired'), width: 110 },
  { colKey: 'description', title: t('project.templateCatalog.variableDescription') },
]);
const workspaceColumns = computed<TableProps['columns']>(() => [
  { colKey: 'path', title: t('project.templateCatalog.path'), width: 300 },
  { colKey: 'kind', title: t('project.templateCatalog.kind'), width: 120 },
  { colKey: 'preview', title: t('project.templateCatalog.preview') },
]);

watch(templateId, () => void loadTemplate(), { immediate: true });
async function loadTemplate() {
  if (!templateId.value) return;
  const requestId = ++loadRequestId.value;
  const requestedTemplateId = templateId.value;
  loading.value = true;
  errorMessage.value = '';
  try {
    const template = await getPublishedApplicationTemplate(requestedTemplateId);
    if (requestId !== loadRequestId.value || requestedTemplateId !== templateId.value) return;
    templateRecord.value = template;
  } catch (error) {
    if (requestId !== loadRequestId.value || requestedTemplateId !== templateId.value) return;
    templateRecord.value = null;
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('project.templateCatalog.detailLoadFailed'));
  } finally {
    if (requestId === loadRequestId.value && requestedTemplateId === templateId.value) {
      loading.value = false;
    }
  }
}
async function useTemplate() {
  if (!templateRecord.value) return;
  if (route.query.deployment !== 'compose' || !/^\d+$/.test(String(route.query.runtime_target_id || ''))) {
    MessagePlugin.warning(t('project.runtimeTarget.unavailableTooltip'));
    backToCatalog();
    return;
  }
  selecting.value = true;
  await router.push({
    name: PROJECT_BOOTSTRAP_ROUTE.CREATE_BLANK.pageRouteName,
    query: {
      ...route.query,
      template_id: templateRecord.value.template_id,
      template_version_id: templateRecord.value.version.template_version_id,
    },
  });
  selecting.value = false;
}
function backToCatalog() {
  void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.CREATE_TEMPLATE.pageRouteName, query: route.query });
}
function categoryLabel(value: ApplicationTemplateCategory) {
  return t(`project.templateCatalog.categories.${value}`);
}
function preview(value: string) {
  return value.replace(/\s+/gu, ' ').trim().slice(0, 160) || '-';
}
</script>
<style scoped>
.template-catalog-detail__loading,
.template-catalog-detail__workspace {
  margin-top: var(--graft-density-gap-16);
}

.template-catalog-detail__grid {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: minmax(0, 1.5fr) minmax(320px, 1fr);
  margin-top: var(--graft-density-gap-16);
}

@media (width <= 960px) {
  .template-catalog-detail__grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
