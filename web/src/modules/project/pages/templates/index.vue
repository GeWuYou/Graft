<template>
  <div ref="pageRef" class="application-template-list" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="project.route.templates.title"
        description-key="project.templates.description"
        action-layout="inline"
        :source="headerSource"
      >
        <template #actions>
          <t-tooltip v-if="isCompact" :content="t('project.templates.create')" placement="bottom">
            <t-button
              :aria-label="t('project.templates.create')"
              class="application-template-list__create-button"
              shape="square"
              theme="primary"
              @click="openCreateDialog"
            >
              <template #icon><add-icon /></template>
            </t-button>
          </t-tooltip>
          <t-button v-else theme="primary" @click="openCreateDialog">
              {{ t('project.templates.create') }}
          </t-button>
        </template>
        <template #meta>
          <management-statistics-bar
            :items="templateSummaryItems"
            :label="t('project.templates.summaryLabel')"
            layout="summary"
          />
        </template>
      </management-page-header>

      <management-toolbar>
        <template #filters>
          <t-input
            v-model="keyword"
            clearable
            class="application-template-list__search"
            :placeholder="t('project.templates.searchPlaceholder')"
          >
            <template #prefix-icon><search-icon /></template>
          </t-input>
          <t-select
            v-model="status"
            clearable
            class="application-template-list__status"
            :options="statusOptions"
            :placeholder="t('project.templates.status')"
          />
        </template>
        <template v-if="!isTablePresentation" #actions>
          <t-tooltip :content="t('project.templates.refresh')" placement="top">
            <t-button
              :aria-label="t('project.templates.refresh')"
              :loading="loading"
              class="application-template-list__refresh-button"
              shape="square"
              theme="default"
              variant="outline"
              @click="loadTemplates"
            >
              <template #icon><refresh-icon /></template>
            </t-button>
          </t-tooltip>
        </template>
      </management-toolbar>

      <t-alert v-if="errorMessage" theme="error" :message="errorMessage" class="application-template-list__feedback">
        <template #operation
          ><t-button theme="danger" variant="text" @click="loadTemplates">{{
            t('project.templates.retry')
          }}</t-button></template
        >
      </t-alert>

      <management-table-card v-else-if="isTablePresentation">
        <template #toolbar>
          <table-view-toolbar :refresh-label="t('project.templates.refresh')" :refresh-loading="loading" @refresh="loadTemplates" />
        </template>
        <t-table
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
          <table-action-menu :actions="templateActions(row)" @action="handleTemplateAction($event, row)" />
        </template>
        <template #empty
          ><t-empty :title="t('project.templates.emptyTitle')" :description="t('project.templates.emptyDescription')"
        /></template>
        </t-table>
      </management-table-card>
      <template-card-list
        v-else
        :actions-for="templateActions"
        :adapter-label="t('project.templates.adapterCompose')"
        :empty-description="t('project.templates.emptyDescription')"
        :empty-title="t('project.templates.emptyTitle')"
        :loading="loading"
        :no-description-label="t('project.templates.noDescription')"
        :status-label="statusLabel"
        :status-theme="statusTheme"
        :templates="filteredTemplates"
        :updated-at-label="updatedAtLabel"
        @action="handleTemplateAction"
        @open="openTemplate"
      />
      <section v-if="showQuickActions" class="application-template-list__quick-actions">
        <t-button theme="primary" @click="openCreateDialog">
          <template #icon><add-icon /></template>
          {{ t('project.templates.create') }}
        </t-button>
        <t-button variant="outline" @click="openComposeImport">
          {{ t('project.templates.importCompose') }}
        </t-button>
        <t-button variant="outline" @click="browseTemplates">
          {{ t('project.templates.browseTemplates') }}
        </t-button>
      </section>
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
import { AddIcon, RefreshIcon, SearchIcon } from 'tdesign-icons-vue-next';
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import {
  ManagementPageContent,
  ManagementPageHeader,
  ManagementStatisticsBar,
  ManagementTableCard,
  ManagementToolbar,
  TableActionMenu,
  TableViewToolbar,
} from '@/shared/components/management';
import { useResponsiveVariant } from '@/shared/composables';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability/time';

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
import TemplateCardList from './TemplateCardList.vue';

defineOptions({ name: 'ApplicationTemplateListIndex' });

// 模板目录只消费管理端完整快照；发布模板选择仍留在 Application 创建向导，避免混入草稿或归档版本。

const { t, locale } = useI18n();
const router = useRouter();
const pageRef = ref<HTMLElement | null>(null);
const responsiveVariant = useResponsiveVariant(pageRef, { layout: 'flow', presentation: 'data' });
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

const isTablePresentation = computed(() => responsiveVariant.value.density === 'spacious');
const isCompact = computed(() => responsiveVariant.value.density === 'compact');
const headerSource = computed(() =>
  isCompact.value ? undefined : { labelKey: 'project.templates.eyebrow', fallback: t('project.templates.eyebrow') },
);
const templateSummaryItems = computed(() => [
  { label: t('project.templates.total'), value: templates.value.length },
  { label: t('project.templates.statusPublished'), value: templates.value.filter((template) => isPublished(template)).length },
  { label: t('project.templates.statusDraft'), value: templates.value.filter((template) => isDraft(template)).length },
]);
const showQuickActions = computed(
  () =>
    !isTablePresentation.value &&
    !loading.value &&
    !errorMessage.value &&
    !keyword.value &&
    !status.value &&
    templates.value.length <= 2,
);

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
  { colKey: 'operation', title: t('project.templates.operation'), width: 152, fixed: 'right' },
]);

function templateActions(template: ApplicationTemplate) {
  return [
    { value: 'detail', label: isDraft(template) ? t('project.templates.edit') : t('project.templates.open') },
    ...(isDraft(template) ? [{ value: 'publish', label: t('project.templates.publish') }] : []),
    { value: 'clone', label: t('project.templates.clone') },
    ...(isPublished(template) ? [{ value: 'withdraw', label: t('project.templates.withdraw') }] : []),
    ...(!isArchived(template) ? [{ value: 'archive', label: t('project.templates.archive') }] : []),
    { value: 'delete', label: t('project.templates.delete') },
  ];
}

function handleTemplateAction(action: string, template: ApplicationTemplate) {
  if (action === 'detail') openTemplate(template);
  else if (action === 'publish') void publishTemplate(template);
  else if (action === 'clone') openCloneDialog(template);
  else if (action === 'withdraw') void withdrawTemplate(template);
  else if (action === 'archive') void archiveTemplate(template);
  else if (action === 'delete') openDeleteDialog(template);
}

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

function openComposeImport() {
  void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.CREATE_IMPORT.pageRouteName });
}

function browseTemplates() {
  void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.CREATE_TEMPLATE.pageRouteName });
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

function updatedAtLabel(template: ApplicationTemplate) {
  const value = formatLocaleDateTime(template.updated_at, locale.value, { dateStyle: 'medium' });
  return t('project.templates.updatedAt', { value });
}
</script>
<style scoped>
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

.application-template-list__quick-actions {
  display: grid;
  gap: var(--graft-density-gap-8);
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.application-template-list__quick-actions :deep(.t-button) {
  min-height: 44px;
  min-width: 0;
}

@media (width <= 991px) {
  .application-template-list :deep(.management-toolbar) {
    min-height: 0;
  }

  .application-template-list :deep(.management-toolbar__filters) {
    width: 100%;
  }

  .application-template-list__search,
  .application-template-list__status {
    flex-basis: 100%;
    min-width: 0;
    width: 100%;
  }

  .application-template-list__search :deep(.t-input),
  .application-template-list__status :deep(.t-input) {
    min-height: 44px;
  }
}

@media (width <= 767px) {
  .application-template-list :deep(.management-page-header) {
    padding: var(--graft-density-gap-16);
  }

  .application-template-list :deep(.page-header__main) {
    align-items: center;
    flex-flow: row wrap;
    gap: var(--graft-density-gap-12);
  }

  .application-template-list :deep(.page-header__side) {
    display: contents;
  }

  .application-template-list :deep(.page-header__actions) {
    align-items: center;
    order: 2;
    width: auto;
  }

  .application-template-list :deep(.page-header__extra) {
    flex-basis: 100%;
    order: 3;
  }

  .application-template-list__create-button {
    min-height: 44px;
    min-width: 44px;
  }

  .application-template-list :deep(.management-toolbar__actions) {
    flex: 0 0 auto;
  }

  .application-template-list__refresh-button {
    min-height: 44px;
    min-width: 44px;
  }

  .application-template-list__quick-actions {
    grid-template-columns: 1fr;
  }

}
</style>
