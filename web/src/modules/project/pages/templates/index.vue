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
			<t-button theme="primary" @click="openCreateDialog">
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
        <template #operation="{ row }">
          <t-space size="small" class="application-template-list__actions">
            <t-button variant="text" theme="primary" @click="openTemplate(row)">{{
              isDraft(row) ? t('project.templates.edit') : t('project.templates.open')
            }}</t-button>
            <t-button v-if="isDraft(row)" variant="text" theme="success" @click="publishTemplate(row)">{{
              t('project.templates.publish')
            }}</t-button>
            <t-button variant="text" @click="openCloneDialog(row)">{{
              t('project.templates.clone')
            }}</t-button>
            <t-button v-if="isPublished(row)" variant="text" theme="warning" @click="withdrawTemplate(row)">{{
              t('project.templates.withdraw')
            }}</t-button>
            <t-button v-if="!isArchived(row)" variant="text" theme="warning" @click="archiveTemplate(row)">{{
              t('project.templates.archive')
            }}</t-button>
            <t-button variant="text" theme="danger" @click="openDeleteDialog(row)">{{
              t('project.templates.delete')
            }}</t-button>
          </t-space>
        </template>
        <template #empty
          ><t-empty :title="t('project.templates.emptyTitle')" :description="t('project.templates.emptyDescription')"
        /></template>
		</t-table>
	  </management-page-content>

	  <t-dialog
      v-model:visible="cloneVisible"
      :header="t('project.templates.cloneTitle')"
      :confirm-btn="t('project.templates.cloneConfirm')"
      :cancel-btn="t('project.templates.cancel')"
      :confirm-loading="cloning"
      @confirm="cloneTemplate"
    >
      <t-form label-align="top">
        <t-form-item :label="t('project.templates.name')"><t-input v-model="cloneDisplayName" /></t-form-item>
      </t-form>
    </t-dialog>
    <t-dialog
      v-model:visible="deleteVisible"
      theme="danger"
      :header="t('project.templates.deleteTitle')"
      :body="t('project.templates.deleteBody')"
      :confirm-btn="t('project.templates.deleteConfirm')"
      :cancel-btn="t('project.templates.cancel')"
      :confirm-loading="deleting"
      @confirm="deleteTemplate"
    />
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
  deleteApplicationTemplate,
  getApplicationManagedTemplates,
  postApplicationTemplateArchive,
  postApplicationTemplateClone,
  postApplicationTemplatePublish,
  postApplicationTemplateWithdraw,
} from '../../api/project';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { emitApplicationTemplateDebug } from '../../shared/project-template-debug';
import type { ApplicationTemplate } from '../../types/project';

defineOptions({ name: 'ApplicationTemplateListIndex' });

// 模板目录只消费管理端完整快照；发布模板选择仍留在 Application 创建向导，避免混入草稿或归档版本。

const { t } = useI18n();
const router = useRouter();
const templates = ref<ApplicationTemplate[]>([]);
const loading = ref(false);
const cloning = ref(false);
const deleting = ref(false);
const errorMessage = ref('');
const keyword = ref('');
const status = ref('');
const cloneVisible = ref(false);
const deleteVisible = ref(false);
const selectedTemplate = ref<ApplicationTemplate | null>(null);
const cloneDisplayName = ref('');

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
  { colKey: 'operation', title: t('project.templates.operation'), width: 420, fixed: 'right' },
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

function openCreateDialog() {
	emitApplicationTemplateDebug('create-wizard-opened', {
	  routeName: PROJECT_BOOTSTRAP_ROUTE.TEMPLATE_CREATE.pageRouteName,
	});
	void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.TEMPLATE_CREATE.pageRouteName });
}

function openTemplate(template: ApplicationTemplate) {
	emitApplicationTemplateDebug('detail-navigation-requested', {
	  routeName: PROJECT_BOOTSTRAP_ROUTE.TEMPLATE_DETAIL.pageRouteName,
	  templateId: template.template_id,
	});
  void router.push({
    name: PROJECT_BOOTSTRAP_ROUTE.TEMPLATE_DETAIL.pageRouteName,
    params: { templateId: template.template_id },
  });
}

function isArchived(template: ApplicationTemplate) {
  return Boolean(template.archived_at);
}

function templateStatus(template: ApplicationTemplate) {
  return String(template.version.status);
}

function isDraft(template: ApplicationTemplate) {
  return !isArchived(template) && templateStatus(template) === 'draft';
}

function isPublished(template: ApplicationTemplate) {
  return !isArchived(template) && templateStatus(template) === 'published';
}

function openCloneDialog(template: ApplicationTemplate) {
  selectedTemplate.value = template;
  cloneDisplayName.value = template.display_name + ' ' + t('project.templates.cloneSuffix');
  cloneVisible.value = true;
}

function openDeleteDialog(template: ApplicationTemplate) {
  selectedTemplate.value = template;
  deleteVisible.value = true;
}

async function cloneTemplate() {
  if (!selectedTemplate.value || !cloneDisplayName.value.trim()) return;
  cloning.value = true;
  try {
    const template = await postApplicationTemplateClone(selectedTemplate.value.template_id, cloneDisplayName.value.trim());
    cloneVisible.value = false;
    openTemplate(template);
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.cloneFailed')));
  } finally {
    cloning.value = false;
  }
}

async function publishTemplate(template: ApplicationTemplate) {
  try {
    await postApplicationTemplatePublish(template.template_id);
    await loadTemplates();
    MessagePlugin.success(t('project.templates.publishSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.publishFailed')));
  }
}

async function withdrawTemplate(template: ApplicationTemplate) {
  try {
    await postApplicationTemplateWithdraw(template.template_id);
    await loadTemplates();
    MessagePlugin.success(t('project.templates.withdrawSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.withdrawFailed')));
  }
}

async function archiveTemplate(template: ApplicationTemplate) {
  try {
    await postApplicationTemplateArchive(template.template_id);
    await loadTemplates();
    MessagePlugin.success(t('project.templates.archiveSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.archiveFailed')));
  }
}

async function deleteTemplate() {
  if (!selectedTemplate.value) return;
  deleting.value = true;
  try {
    await deleteApplicationTemplate(selectedTemplate.value.template_id);
    deleteVisible.value = false;
    await loadTemplates();
    MessagePlugin.success(t('project.templates.deleteSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.deleteFailed')));
  } finally {
    deleting.value = false;
  }
}

function statusLabel(template: ApplicationTemplate) {
  if (isArchived(template)) return t('project.templates.statusArchived');
  return isDraft(template)
    ? t('project.templates.statusDraft')
    : t('project.templates.statusPublished');
}

function statusTheme(template: ApplicationTemplate) {
  if (isArchived(template)) return 'default';
  return isDraft(template) ? 'warning' : 'success';
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

.application-template-list__actions {
  flex-wrap: wrap;
}
</style>
