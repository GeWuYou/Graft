<template>
  <div class="permission-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="rbac.permissionList.listTitle"
        :title="t('rbac.permissionList.listTitle')"
        description-key="rbac.permissionList.hint"
        :description="t('rbac.permissionList.hint')"
        :source="{ labelKey: 'menu.domain.security.title', fallback: t('menu.domain.security.title') }"
      >
        <template #meta>
          <t-tag theme="default" variant="light">{{ t('rbac.permissionList.readonlyNotice') }}</t-tag>
        </template>
      </management-page-header>

      <advanced-query-filter-builder
        active-preset="all"
        :add-filter-label="`+ ${t('rbac.permissionList.toolbar.addFilter')}`"
        add-sorter-label=""
        :builder-hint="t('rbac.permissionList.hint')"
        :builder-title="t('rbac.permissionList.toolbar.filterPanelTitle')"
        :field-values="permissionFilterFieldValues"
        :fields="permissionFilterDefinitions"
        :filters-group-label="t('rbac.permissionList.toolbar.filterPanelTitle')"
        :keyword="filters.keyword"
        :keyword-placeholder="t('rbac.permissionList.toolbar.searchPlaceholder')"
        :loading="loading"
        move-down-label=""
        move-up-label=""
        preset-label=""
        :presets="[]"
        remove-sorter-label=""
        :reset-label="t('rbac.permissionList.toolbar.clearFilters')"
        :search-label="t('rbac.permissionList.toolbar.query')"
        selected-field-key="module"
        :sort-direction-options="[]"
        sort-direction-placeholder=""
        sort-field-key="sorter"
        :sort-field-options-by-index="[]"
        sort-field-placeholder=""
        :sorters="[]"
        :show-sorter-builder="false"
        :tags="permissionFilterTags"
        time-field-key="timeRange"
        :time-fields="[]"
        @close-tag="clearPermissionFilterTag"
        @reset="resetFilters"
        @search="applyPermissionFilters"
        @update:field="updatePermissionFilterField"
        @update:keyword="filters.keyword = $event"
      >
        <template #saved-query-views><saved-query-view-control :controller="savedViews" /></template>
      </advanced-query-filter-builder>

      <management-empty-state
        v-if="listError && !loading"
        tone="error"
        :title="t('rbac.permissionList.errorTitle')"
        :description="listError"
      >
        <template #actions>
          <t-button theme="primary" variant="outline" @click="refreshPermissions">
            {{ t('rbac.permissionList.retry') }}
          </t-button>
        </template>
      </management-empty-state>

      <management-paged-table
        v-else
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :column-sets="permissionColumnSets"
        :columns="visibleColumns"
        :empty-description="
          hasActiveFilters ? t('rbac.permissionList.emptyFilteredDescription') : t('rbac.permissionList.empty')
        "
        :empty-title="t('rbac.permissionList.emptyTitle')"
        :footer-summary="t('rbac.permissionList.footerTotal', { count: filteredPermissions.length })"
        :loading="loading"
        :rows="pagedPermissions"
        :total="filteredPermissions.length"
      >
        <template #head>
          <div class="table-head">
            <div>
              <p class="table-head__summary">
                {{ t('rbac.permissionList.summary', { count: filteredPermissions.length }) }}
              </p>
              <p class="table-head__description">{{ t('rbac.permissionList.tableHint') }}</p>
              <div class="inline-note">
                <p>{{ t('rbac.permissionList.readonlyDescription') }}</p>
                <p>{{ t('rbac.permissionList.factSourceHint') }}</p>
              </div>
            </div>
          </div>
        </template>
        <template #toolbar>
          <table-view-toolbar
            :column-settings-label="t('rbac.permissionList.columnSettings')"
            :refresh-label="t('rbac.permissionList.refresh')"
            :refresh-loading="loading"
            @column-settings="columnDrawerVisible = true"
            @refresh="refreshPermissions"
          />
        </template>
        <template #permission="{ row }">
          <div class="permission-cell">
            <span class="permission-cell__name">{{ localizedPermissionDisplay(row) }}</span>
            <span class="permission-cell__code">{{ row.code }}</span>
          </div>
        </template>

        <template #module="{ row }">
          <t-tag theme="default" variant="light">{{ row.module || '-' }}</t-tag>
        </template>

        <template #description="{ row }">
          <span class="permission-description">{{ localizedPermissionDescription(row) }}</span>
        </template>

        <template #created_at="{ row }">
          <span>{{ formatTimestamp(row.created_at) }}</span>
        </template>

        <template #updated_at="{ row }">
          <span>{{ formatTimestamp(row.updated_at) }}</span>
        </template>

        <template #role_count="{ row }">
          <span>{{ row.role_binding_count ?? '-' }}</span>
        </template>

        <template #operation="{ row }">
          <table-action-menu
            :actions="[
              {
                label: t('rbac.permissionList.detail'),
                testId: 'permission-detail',
                value: 'detail',
              },
              {
                label: t('rbac.permissionList.viewAudit'),
                testId: 'permission-view-audit',
                value: 'view-audit',
              },
            ]"
            :more-label="t('rbac.permissionList.more')"
            :more-label-fallback="t('rbac.permissionList.more')"
            @action="(action) => handlePermissionAction(action, row)"
          />
        </template>

        <template #empty>
          <div class="table-empty-state">
            <t-empty
              :title="t('rbac.permissionList.emptyTitle')"
              :description="
                hasActiveFilters ? t('rbac.permissionList.emptyFilteredDescription') : t('rbac.permissionList.empty')
              "
            >
              <template #action>
                <div v-if="hasActiveFilters" class="table-empty-state__actions">
                  <t-button
                    theme="default"
                    variant="outline"
                    data-testid="permission-empty-clear-filters"
                    @click="resetFilters"
                  >
                    {{ t('rbac.permissionList.toolbar.clearFilters') }}
                  </t-button>
                </div>
              </template>
            </t-empty>
          </div>
        </template>
      </management-paged-table>
    </management-page-content>

    <responsive-dialog
      v-model:visible="columnDrawerVisible"
      :close-label="t('components.common.close')"
      :title="t('rbac.permissionList.columnSettings')"
      purpose="form"
      size="compact"
    >
      <div class="drawer-panel">
        <t-checkbox-group v-model="visibleColumnKeys">
          <div class="column-grid">
            <t-checkbox v-for="column in columnSettingOptions" :key="column.value" :value="column.value">
              {{ column.label }}
            </t-checkbox>
          </div>
        </t-checkbox-group>
      </div>
    </responsive-dialog>

    <responsive-dialog
      v-model:visible="detailDrawerVisible"
      :close-label="t('components.common.close')"
      :title="t('rbac.permissionList.detailTitle')"
      purpose="detail"
      size="medium"
    >
      <div class="drawer-panel permission-detail-panel">
        <div v-if="detailError" class="inline-warning detail-warning">
          <span>{{ t('rbac.permissionList.detailLoadFailedTitle') }}</span>
          <span>{{ detailError }}</span>
          <t-button v-if="detailDrawerPermission" theme="default" variant="text" @click="retryDetail">
            {{ t('rbac.permissionList.retry') }}
          </t-button>
        </div>

        <template v-if="detailRecord">
          <div class="detail-header">
            <div>
              <p class="detail-header__title">{{ localizedPermissionDisplay(detailRecord) }}</p>
              <p class="detail-header__code">{{ detailRecord.code }}</p>
            </div>
            <t-tag theme="default" variant="light">{{ detailRecord.module || '-' }}</t-tag>
          </div>

          <div class="detail-grid">
            <div class="detail-item">
              <span class="detail-item__label">{{ t('rbac.permissionList.detailFields.description') }}</span>
              <span class="detail-item__value">{{ localizedPermissionDescription(detailRecord) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">{{ t('rbac.permissionList.columns.roleCount') }}</span>
              <span class="detail-item__value">{{ detailRecord.role_binding_count ?? '-' }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">{{ t('rbac.permissionList.columns.createdAt') }}</span>
              <span class="detail-item__value">{{ formatTimestamp(detailRecord.created_at) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">{{ t('rbac.permissionList.columns.updatedAt') }}</span>
              <span class="detail-item__value">{{ formatTimestamp(detailRecord.updated_at) }}</span>
            </div>
          </div>
        </template>
      </div>
    </responsive-dialog>
  </div>
</template>
<script setup lang="ts">
import type { TdBaseTableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { buildAuditResourceLocation } from '@/modules/audit/contract/deep-link';
import { openCorrelationErrorNotification, requestIdFromError } from '@/modules/audit/shared/correlation-actions';
import {
  buildVisibleColumns,
  createActionColumn,
  createCountColumn,
  createStatusColumn,
  createTextColumn,
  createTimeColumn,
  formatCompactDateTime,
  ManagementEmptyState,
  ManagementPageContent,
  ManagementPageHeader,
  TableActionMenu,
  TableViewToolbar,
} from '@/shared/components/management';
import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';
import {
  AdvancedQueryFilterBuilder,
  type AdvancedQueryFilterFieldDefinition,
  type AdvancedQueryFilterTag,
  applySavedQueryViewPresentation,
  normalizeSavedQueryView,
  SavedQueryViewControl,
  serializeSavedQueryViewRequest,
  useSavedQueryViews,
} from '@/shared/components/query-list';
import ResponsiveDialog from '@/shared/components/responsive/ResponsiveDialog.vue';
import { useTabPageSnapshot } from '@/shared/composables/useTabPageSnapshot';
import { resolveErrorMessageWithCorrelation } from '@/shared/correlation';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { createLogger } from '@/utils/logger';

import {
  deletePermissionSavedView,
  getPermissionDetail,
  getPermissionSavedViews,
  postPermissionSavedView,
  putPermissionSavedView,
} from '../../api/rbac';
import {
  localizedPermissionDescription as localizePermissionDescription,
  localizedPermissionDisplay as localizePermissionDisplay,
} from '../../shared/permission-copy';
import { usePermissionListQuery } from '../../shared/rbac-queries';
import type { PermissionDetailResponse, PermissionListItem } from '../../types/permission';

defineOptions({
  name: 'PermissionIndex',
});

/** 权限目录页按筛选键消费 Query 快照；详情抽屉保留独立加载和失败状态以隔离会话。 */
const logger = createLogger('rbac.permissionList');
const router = useRouter();

type PermissionFilterState = {
  keyword: string;
  module: string;
};

type PermissionSavedViewState = {
  pageSize: number;
  queryState: PermissionFilterState;
  visibleColumns: string[];
};

type PermissionPageSnapshot = {
  columnDrawerVisible: boolean;
  filters: PermissionFilterState;
  pagination: {
    current: number;
    pageSize: number;
  };
  visibleColumnKeys: string[];
};

const { t, locale } = useI18n();
const filters = ref<PermissionFilterState>({
  keyword: '',
  module: '',
});
const appliedFilters = ref<PermissionFilterState>({ ...filters.value });
const columnDrawerVisible = ref(false);
const visibleColumnKeys = ref(['permission', 'module', 'code', 'role_count', 'updated_at', 'operation']);
const detailDrawerVisible = ref(false);
const detailDrawerPermission = ref<PermissionListItem | null>(null);
const detailRecord = ref<PermissionDetailResponse | null>(null);
const detailLoading = ref(false);
const detailError = ref('');
const pagination = ref({
  current: 1,
  pageSize: 10,
});
const permissionListQuery = usePermissionListQuery(
  computed(() => ({ keyword: appliedFilters.value.keyword, module: appliedFilters.value.module })),
);
const permissions = computed(() => permissionListQuery.data.value?.items ?? []);
const loading = computed(() => permissionListQuery.isFetching.value);
const listError = computed(() =>
  permissionListQuery.isError.value
    ? resolveLocalizedErrorMessage(t, permissionListQuery.error.value, t('rbac.permissionList.loadFailed'))
    : '',
);

useTabPageSnapshot<PermissionPageSnapshot>({
  apply(snapshot) {
    filters.value = { ...snapshot.filters };
    appliedFilters.value = { ...snapshot.filters };
    visibleColumnKeys.value = [...snapshot.visibleColumnKeys];
    pagination.value = { ...snapshot.pagination };
    columnDrawerVisible.value = snapshot.columnDrawerVisible;
  },
  read() {
    return {
      columnDrawerVisible: columnDrawerVisible.value,
      filters: { ...filters.value },
      pagination: { ...pagination.value },
      visibleColumnKeys: [...visibleColumnKeys.value],
    };
  },
});

const moduleOptions = computed(() => {
  const modules = Array.from(
    new Set(permissions.value.map((item) => item.module).filter((module): module is string => Boolean(module))),
  ).sort();
  return modules.map((module) => ({ label: module, value: module }));
});

const hasActiveFilters = computed(() => Boolean(appliedFilters.value.keyword.trim() || appliedFilters.value.module));
const permissionFilterDefinitions = computed<AdvancedQueryFilterFieldDefinition[]>(() => [
  {
    key: 'module',
    kind: 'select',
    label: t('rbac.permissionList.toolbar.modulePlaceholder'),
    options: moduleOptions.value,
  },
]);
const permissionFilterFieldValues = computed(() => ({ module: filters.value.module }));
const permissionFilterTags = computed<AdvancedQueryFilterTag[]>(() => {
  const tags: AdvancedQueryFilterTag[] = [];
  if (appliedFilters.value.keyword.trim()) tags.push({ key: 'keyword', label: appliedFilters.value.keyword.trim() });
  if (appliedFilters.value.module)
    tags.push({
      key: 'module',
      label: `${t('rbac.permissionList.toolbar.modulePlaceholder')}: ${appliedFilters.value.module}`,
    });
  return tags;
});

const columnSettingOptions = computed(() => [
  { label: t('rbac.permissionList.columns.permission'), value: 'permission' },
  { label: t('rbac.permissionList.columns.module'), value: 'module' },
  { label: t('rbac.permissionList.columns.code'), value: 'code' },
  { label: t('rbac.permissionList.columns.description'), value: 'description' },
  { label: t('rbac.permissionList.columns.roleCount'), value: 'role_count' },
  { label: t('rbac.permissionList.columns.createdAt'), value: 'created_at' },
  { label: t('rbac.permissionList.columns.updatedAt'), value: 'updated_at' },
  { label: t('rbac.permissionList.columns.operation'), value: 'operation' },
]);

const savedViews = useSavedQueryViews<PermissionSavedViewState, number>({
  adapter: {
    list: async () =>
      (await getPermissionSavedViews()).map((view) =>
        normalizeSavedQueryView<PermissionSavedViewState['queryState'], number>(view),
      ),
    create: async (input) =>
      normalizeSavedQueryView<PermissionSavedViewState['queryState'], number>(
        await postPermissionSavedView(serializeSavedQueryViewRequest(input)),
      ),
    update: async (id, input) =>
      normalizeSavedQueryView<PermissionSavedViewState['queryState'], number>(
        await putPermissionSavedView(id, serializeSavedQueryViewRequest(input)),
      ),
    remove: deletePermissionSavedView,
  },
  applyView: (view) => {
    const savedFilters = view.state.queryState;
    filters.value = {
      keyword: savedFilters.keyword ?? '',
      module: savedFilters.module ?? '',
    };
    appliedFilters.value = { ...filters.value };
    applySavedQueryViewPresentation(view.state, {
      pagination: pagination.value,
      supportedColumns: columnSettingOptions.value.map((option) => option.value),
      visibleColumnKeys,
    });
  },
  onError: (error) => MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('rbac.permissionList.loadFailed'))),
  serializeCurrentState: () => ({
    pageSize: pagination.value.pageSize,
    queryState: { ...filters.value },
    visibleColumns: [...visibleColumnKeys.value],
  }),
});

const filteredPermissions = computed(() => permissions.value);

const pagedPermissions = computed(() => {
  const start = (pagination.value.current - 1) * pagination.value.pageSize;
  return filteredPermissions.value.slice(start, start + pagination.value.pageSize);
});

const visibleColumns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;

  const allColumns: TdBaseTableProps['columns'] = [
    createTextColumn(t('rbac.permissionList.columns.permission'), 'permission', {
      width: 340,
      fixed: 'left',
    }),
    createStatusColumn(t('rbac.permissionList.columns.module'), 'module', 140),
    createTextColumn(t('rbac.permissionList.columns.code'), 'code', {
      width: 240,
    }),
    createTextColumn(t('rbac.permissionList.columns.description'), 'description', {
      width: 220,
    }),
    createCountColumn(t('rbac.permissionList.columns.roleCount'), 'role_count', 120),
    createTimeColumn(t('rbac.permissionList.columns.createdAt'), 'created_at', 160),
    createTimeColumn(t('rbac.permissionList.columns.updatedAt'), 'updated_at', 160),
    createActionColumn(t('rbac.permissionList.columns.operation'), 112),
  ];

  return buildVisibleColumns(allColumns, visibleColumnKeys.value);
});

const permissionColumnSets = {
  compact: ['permission', 'module', 'operation'],
};

async function refreshPermissions() {
  await permissionListQuery.refetch();
}

function resetFilters() {
  filters.value = {
    keyword: '',
    module: '',
  };
  appliedFilters.value = { ...filters.value };
  pagination.value.current = 1;
}

function applyPermissionFilters() {
  appliedFilters.value = { ...filters.value };
  pagination.value.current = 1;
}

function updatePermissionFilterField(payload: { key: string; value: string | string[] }) {
  if (payload.key !== 'module') return;
  filters.value.module = Array.isArray(payload.value) ? (payload.value[0] ?? '') : payload.value;
}

function clearPermissionFilterTag(key: string) {
  if (key === 'keyword') filters.value.keyword = '';
  if (key === 'module') filters.value.module = '';
  applyPermissionFilters();
}

function localizedPermissionDisplay(permission: PermissionListItem) {
  return localizePermissionDisplay(t, permission, locale.value);
}

function localizedPermissionDescription(permission: PermissionListItem) {
  return localizePermissionDescription(t, permission, 'rbac.permissionList.emptyDescription', locale.value);
}

async function loadPermissionDetail(permissionId: number) {
  detailLoading.value = true;
  detailError.value = '';

  try {
    detailRecord.value = await getPermissionDetail(permissionId);
  } catch (error) {
    detailRecord.value = null;
    logger.warn('failed to fetch permission detail', error);
    detailError.value = resolveLocalizedErrorMessage(t, error, t('rbac.permissionList.detailLoadFailed'));
    const message = resolveErrorMessageWithCorrelation(t, error, detailError.value);
    MessagePlugin.error(message);
    openCorrelationErrorNotification({
      router,
      title: t('audit.correlation.errorTitle'),
      message,
      requestId: requestIdFromError(error),
      translate: t,
    });
  } finally {
    detailLoading.value = false;
  }
}

function handlePermissionAction(action: string, permission: PermissionListItem) {
  if (action === 'view-audit') {
    void router.push(
      buildAuditResourceLocation('permission', String(permission.id), permission.code || permission.display),
    );
    return;
  }

  void openDetailDrawer(permission);
}

async function openDetailDrawer(permission: PermissionListItem) {
  detailDrawerPermission.value = permission;
  detailRecord.value = permission;
  detailDrawerVisible.value = true;
  await loadPermissionDetail(permission.id);
}

async function retryDetail() {
  if (!detailDrawerPermission.value || detailLoading.value) {
    return;
  }

  await loadPermissionDetail(detailDrawerPermission.value.id);
}

function formatTimestamp(value?: string | null) {
  return formatCompactDateTime(value);
}

watch(
  () => [appliedFilters.value.keyword, appliedFilters.value.module] as const,
  () => {
    pagination.value.current = 1;
  },
);

onMounted(() => void savedViews.load());
</script>
<style scoped lang="less">
@import '../../shared/list-page.less';

.permission-page {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  .management-list-header();
  .management-list-table-empty();
  .management-list-table-shell();
}

.inline-note {
  background: color-mix(in srgb, var(--td-brand-color) 4%, var(--td-bg-color-container));
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  color: var(--td-text-color-secondary);
  display: grid;
  gap: var(--graft-density-gap-4);
  padding: var(--graft-density-gap-12) var(--graft-density-gap-14);
}

.inline-note p {
  margin: 0;
}

.table-head__summary,
.table-head__description,
.permission-cell__code,
.permission-description {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.permission-cell {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
}

.permission-cell__name {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.drawer-panel,
.column-grid {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.table-actions {
  display: flex;
  justify-content: flex-start;
}

.permission-detail-panel {
  gap: var(--graft-density-gap-16);
}

.detail-warning {
  align-items: flex-start;
}

.detail-header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.detail-header__title,
.detail-header__code,
.detail-item__label,
.detail-item__value {
  margin: 0;
}

.detail-header__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.detail-header__code,
.detail-item__label {
  color: var(--td-text-color-secondary);
}

.detail-grid {
  display: grid;
  gap: var(--graft-density-gap-12);
}

.detail-item {
  background: var(--td-bg-color-container-hover);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  padding: var(--graft-density-gap-12) var(--graft-density-gap-14);
}

.detail-item__value {
  color: var(--td-text-color-primary);
}

@media (width <= 768px) {
  .toolbar__select {
    width: 100%;
  }

  .table-empty-state {
    min-height: 260px;
    padding-inline: var(--graft-density-gap-16);
  }

  .permission-page :deep(.management-toolbar__filters) {
    flex-wrap: wrap;
  }

  .table-head {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
