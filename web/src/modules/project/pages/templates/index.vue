<template>
  <div class="application-template-list" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="project.route.templates.title"
        description-key="project.templates.description"
        :source="{ labelKey: 'project.templates.eyebrow', fallback: t('project.templates.eyebrow') }"
      >
        <template #actions>
          <t-space size="small">
            <t-button variant="outline" @click="legacyImportVisible = true">{{
              t('project.templates.importLegacy')
            }}</t-button>
            <t-button theme="primary" :loading="creating" @click="createBlankDraft">
              {{ t('project.templates.create') }}
            </t-button>
          </t-space>
        </template>
      </management-page-header>

      <t-card bordered class="application-template-list__filters">
        <t-space break-line>
          <t-input v-model="keyword" clearable :placeholder="t('project.templates.searchPlaceholder')" />
          <t-select v-model="status" :options="statusOptions" :placeholder="t('project.templates.status')" clearable />
          <t-button variant="outline" :loading="loading" @click="loadTemplates">{{
            t('project.templates.refresh')
          }}</t-button>
        </t-space>
      </t-card>

      <t-alert v-if="errorMessage" theme="error" :message="errorMessage" class="application-template-list__feedback">
        <template #operation
          ><t-button theme="danger" variant="text" @click="loadTemplates">{{
            t('project.templates.retry')
          }}</t-button></template
        >
      </t-alert>

      <t-table
        v-else
        row-key="template_id"
        :data="filteredTemplates"
        :columns="columns"
        :loading="loading"
        :pagination="undefined"
        table-layout="fixed"
        class="application-template-list__table"
      >
        <template #name="{ row }">
          <button class="application-template-list__name" type="button" @click="openTemplate(row)">
            {{ row.display_name }}
          </button>
          <p>{{ row.description || t('project.templates.noDescription') }}</p>
        </template>
        <template #adapter="{ row }"
          ><t-tag variant="light-outline">{{ row.deployment_adapter_kind }}</t-tag></template
        >
        <template #status="{ row }"
          ><t-tag :theme="statusTheme(row)" variant="light-outline">{{ statusLabel(row) }}</t-tag></template
        >
        <template #version="{ row }">{{
          t('project.templates.versionValue', { version: row.version.version_number })
        }}</template>
        <template #operation="{ row }"
          ><t-button variant="text" theme="primary" @click="openTemplate(row)">{{
            t('project.templates.open')
          }}</t-button></template
        >
        <template #empty
          ><t-empty :title="t('project.templates.emptyTitle')" :description="t('project.templates.emptyDescription')"
        /></template>
      </t-table>
    </management-page-content>

    <t-dialog
      v-model:visible="legacyImportVisible"
      :header="t('project.templates.importLegacyTitle')"
      :confirm-btn="t('project.templates.importLegacyConfirm')"
      :cancel-btn="t('project.templates.cancel')"
      :confirm-loading="importing"
      @confirm="importLegacy"
    >
      <t-form label-align="top">
        <t-form-item :label="t('project.templates.legacyKey')"><t-input v-model="legacyKey" /></t-form-item>
        <t-form-item :label="t('project.templates.name')"><t-input v-model="legacyDisplayName" /></t-form-item>
      </t-form>
    </t-dialog>
  </div>
</template>
<script setup lang="ts">
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import {
  getApplicationManagedTemplates,
  postApplicationTemplate,
  postApplicationTemplateLegacyImport,
} from '../../api/project';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import type { ApplicationTemplate, ApplicationTemplateDraftRequest } from '../../types/project';

defineOptions({ name: 'ApplicationTemplateListIndex' });

// 模板目录只消费管理端完整快照；发布模板选择仍留在 Application 创建向导，避免混入草稿或归档版本。

const { t } = useI18n();
const router = useRouter();
const templates = ref<ApplicationTemplate[]>([]);
const loading = ref(false);
const creating = ref(false);
const importing = ref(false);
const errorMessage = ref('');
const keyword = ref('');
const status = ref('');
const legacyImportVisible = ref(false);
const legacyKey = ref('');
const legacyDisplayName = ref('');

const statusOptions = computed(() => [
  { label: t('project.templates.statusDraft'), value: 'draft' },
  { label: t('project.templates.statusPublished'), value: 'published' },
  { label: t('project.templates.statusArchived'), value: 'archived' },
]);
const filteredTemplates = computed(() => {
  const search = keyword.value.trim().toLocaleLowerCase();
  return templates.value.filter((item) => {
    const itemStatus = item.archived_at ? 'archived' : item.version.status;
    return (
      (!status.value || itemStatus === status.value) &&
      (!search || `${item.display_name} ${item.description}`.toLocaleLowerCase().includes(search))
    );
  });
});
const columns = computed<TableProps['columns']>(() => [
  { colKey: 'name', title: t('project.templates.name'), width: 360 },
  { colKey: 'adapter', title: t('project.templates.adapter'), width: 150 },
  { colKey: 'status', title: t('project.templates.status'), width: 150 },
  { colKey: 'version', title: t('project.templates.version'), width: 110 },
  { colKey: 'operation', title: t('project.templates.operation'), width: 110, fixed: 'right' },
]);

onMounted(() => void loadTemplates());

async function loadTemplates() {
  loading.value = true;
  errorMessage.value = '';
  try {
    templates.value = (await getApplicationManagedTemplates()).items;
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('project.templates.loadFailed'));
  } finally {
    loading.value = false;
  }
}

function blankDraftPayload(): ApplicationTemplateDraftRequest {
  return {
    display_name: t('project.templates.untitled'),
    description: '',
    deployment_adapter_kind: 'compose',
    definition_schema_version: 1,
    definition: {
      compose_file_path: 'compose.yaml',
      workspace_entries: [
        { path: '.env', node_type: 'file', content: '' },
        { path: 'compose.yaml', node_type: 'file', content: t('project.templates.defaultCompose') },
      ],
      lifecycle_configuration: {},
    },
  } as unknown as ApplicationTemplateDraftRequest;
}

async function createBlankDraft() {
  creating.value = true;
  try {
    openTemplate(await postApplicationTemplate(blankDraftPayload()));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.createFailed')));
  } finally {
    creating.value = false;
  }
}

function openTemplate(template: ApplicationTemplate) {
  void router.push({
    name: PROJECT_BOOTSTRAP_ROUTE.TEMPLATE_DETAIL.pageRouteName,
    params: { templateId: template.template_id },
  });
}

async function importLegacy() {
  if (!legacyKey.value.trim()) return;
  importing.value = true;
  try {
    const template = await postApplicationTemplateLegacyImport({
      key: legacyKey.value.trim(),
      display_name: legacyDisplayName.value.trim() || undefined,
    });
    legacyImportVisible.value = false;
    legacyKey.value = '';
    legacyDisplayName.value = '';
    openTemplate(template);
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.importLegacyFailed')));
  } finally {
    importing.value = false;
  }
}

function statusLabel(template: ApplicationTemplate) {
  if (template.archived_at) return t('project.templates.statusArchived');
  return template.version.status === 'draft'
    ? t('project.templates.statusDraft')
    : t('project.templates.statusPublished');
}

function statusTheme(template: ApplicationTemplate) {
  if (template.archived_at) return 'default';
  return template.version.status === 'draft' ? 'warning' : 'success';
}
</script>
<style scoped>
.application-template-list__filters,
.application-template-list__feedback,
.application-template-list__table {
  margin-top: var(--graft-density-gap-16);
}

.application-template-list__name {
  background: none;
  border: 0;
  color: var(--td-brand-color);
  cursor: pointer;
  font: inherit;
  font-weight: 600;
  padding: 0;
  text-align: left;
}

.application-template-list__name + p {
  color: var(--td-text-color-secondary);
  margin: var(--graft-density-gap-4) 0 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
