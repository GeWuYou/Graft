<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="permission-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="rbac.permissionList.listTitle"
        :title="t('rbac.permissionList.listTitle')"
        description-key="rbac.permissionList.hint"
        :description="t('rbac.permissionList.hint')"
        :source="{ labelKey: 'menu.access_control.title', fallback: t('menu.access_control.title') }"
      >
        <template #meta>
          <t-tag theme="default" variant="light">{{ t('rbac.permissionList.readonlyNotice') }}</t-tag>
        </template>
        <template #actions>
          <t-button theme="default" variant="outline" :loading="loading" @click="() => fetchPermissions()">
            {{ t('rbac.permissionList.refresh') }}
          </t-button>
        </template>
      </management-page-header>

      <management-toolbar>
        <template #filters>
          <t-input
            v-model="filters.keyword"
            clearable
            class="toolbar__search"
            :placeholder="t('rbac.permissionList.toolbar.searchPlaceholder')"
          />
          <t-select
            v-model="filters.category"
            clearable
            class="toolbar__select"
            :options="categoryOptions"
            :placeholder="t('rbac.permissionList.toolbar.modulePlaceholder')"
          />
          <t-button theme="default" variant="text" @click="resetFilters">
            {{ t('rbac.permissionList.toolbar.clearFilters') }}
          </t-button>
        </template>
        <template #actions>
          <t-button theme="default" variant="outline" @click="columnDrawerVisible = true">
            {{ t('rbac.permissionList.columnSettings') }}
          </t-button>
        </template>
      </management-toolbar>

      <management-table-card>
        <template #head>
          <div class="table-head">
            <div>
              <p class="table-head__summary">
                {{ t('rbac.permissionList.summary', { count: filteredPermissions.length }) }}
              </p>
              <p class="table-head__description">{{ t('rbac.permissionList.tableHint') }}</p>
            </div>
          </div>
        </template>

        <div class="inline-note">
          <p>{{ t('rbac.permissionList.readonlyDescription') }}</p>
          <p>{{ t('rbac.permissionList.factSourceHint') }}</p>
        </div>

        <management-empty-state
          v-if="listError && !loading"
          tone="error"
          :title="t('rbac.permissionList.errorTitle')"
          :description="listError"
        >
          <template #actions>
            <t-button theme="primary" variant="outline" @click="() => fetchPermissions()">
              {{ t('rbac.permissionList.retry') }}
            </t-button>
          </template>
        </management-empty-state>

        <t-table
          v-else
          row-key="id"
          :data="pagedPermissions"
          :columns="visibleColumns"
          :loading="loading"
          table-layout="fixed"
          :table-content-width="tableContentWidth"
          cell-empty-content="-"
        >
          <template #permission="{ row }">
            <div class="permission-cell">
              <span class="permission-cell__name">{{ localizedPermissionDisplay(row) }}</span>
              <span class="permission-cell__code">{{ row.code }}</span>
            </div>
          </template>

          <template #category="{ row }">
            <t-tag theme="default" variant="light">{{ row.category || '-' }}</t-tag>
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
        </t-table>

        <template #footer>
          <management-table-pagination
            :summary="t('rbac.permissionList.footerTotal', { count: filteredPermissions.length })"
          >
            <t-pagination
              v-model:current="pagination.current"
              v-model:page-size="pagination.pageSize"
              :total="filteredPermissions.length"
              :page-size-options="[10, 20, 50]"
              :show-page-number="true"
            />
          </management-table-pagination>
        </template>
      </management-table-card>
    </management-page-content>

    <t-drawer
      v-model:visible="columnDrawerVisible"
      :header="t('rbac.permissionList.columnSettings')"
      size="360px"
      placement="right"
      destroy-on-close
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
    </t-drawer>

    <t-drawer
      v-model:visible="detailDrawerVisible"
      :header="t('rbac.permissionList.detailTitle')"
      size="480px"
      placement="right"
      destroy-on-close
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
            <t-tag theme="default" variant="light">{{ detailRecord.category || '-' }}</t-tag>
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
    </t-drawer>
  </div>
</template>
<script setup lang="ts">
import { MessagePlugin, type TdBaseTableProps } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { buildAuditResourceLocation } from '@/modules/audit/contract/deep-link';
import { openCorrelationErrorNotification, requestIdFromError } from '@/modules/audit/shared/correlation-actions';
import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import {
  buildVisibleColumns,
  calculateTableContentWidth,
  createActionColumn,
  createCountColumn,
  createStatusColumn,
  createTextColumn,
  createTimeColumn,
  formatCompactDateTime,
  ManagementEmptyState,
  ManagementPageContent,
  ManagementPageHeader,
  ManagementTableCard,
  ManagementTablePagination,
  ManagementToolbar,
  TableActionMenu,
} from '@/shared/components/management';
import { useTabPageSnapshot } from '@/shared/composables';
import { resolveErrorMessageWithCorrelation } from '@/shared/correlation';
import { createLogger } from '@/utils/logger';

import { getPermissionDetail, getPermissions } from '../../api/rbac';
import {
  localizedPermissionDescription as localizePermissionDescription,
  localizedPermissionDisplay as localizePermissionDisplay,
} from '../../shared/permission-copy';
import type { PermissionDetailResponse, PermissionFilters, PermissionListItem } from '../../types/permission';

defineOptions({
  name: 'PermissionIndex',
});

const logger = createLogger('rbac.permissionList');
const router = useRouter();

type PermissionFilterState = {
  keyword: string;
  category: string;
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
const loading = ref(false);
const listError = ref('');
const permissions = ref<PermissionListItem[]>([]);
const filters = ref<PermissionFilterState>({
  keyword: '',
  category: '',
});
const columnDrawerVisible = ref(false);
const visibleColumnKeys = ref(['permission', 'category', 'code', 'role_count', 'updated_at', 'operation']);
const detailDrawerVisible = ref(false);
const detailDrawerPermission = ref<PermissionListItem | null>(null);
const detailRecord = ref<PermissionDetailResponse | null>(null);
const detailLoading = ref(false);
const detailError = ref('');
const pagination = ref({
  current: 1,
  pageSize: 10,
});

useTabPageSnapshot<PermissionPageSnapshot>({
  apply(snapshot) {
    filters.value = { ...snapshot.filters };
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

const categoryOptions = computed(() => {
  const categories = Array.from(new Set(permissions.value.map((item) => item.category).filter(Boolean))).sort();
  return categories.map((category) => ({ label: category, value: category }));
});

const hasActiveFilters = computed(() => Boolean(filters.value.keyword.trim() || filters.value.category));

const columnSettingOptions = computed(() => [
  { label: t('rbac.permissionList.columns.permission'), value: 'permission' },
  { label: t('rbac.permissionList.columns.module'), value: 'category' },
  { label: t('rbac.permissionList.columns.code'), value: 'code' },
  { label: t('rbac.permissionList.columns.description'), value: 'description' },
  { label: t('rbac.permissionList.columns.roleCount'), value: 'role_count' },
  { label: t('rbac.permissionList.columns.createdAt'), value: 'created_at' },
  { label: t('rbac.permissionList.columns.updatedAt'), value: 'updated_at' },
  { label: t('rbac.permissionList.columns.operation'), value: 'operation' },
]);

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
    createStatusColumn(t('rbac.permissionList.columns.module'), 'category', 100),
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

const tableContentWidth = computed(() => calculateTableContentWidth(visibleColumns.value));

async function fetchPermissions(preservePagination = false) {
  loading.value = true;
  listError.value = '';

  try {
    const requestFilters: PermissionFilters = {};
    const keyword = filters.value.keyword.trim();
    if (keyword) {
      requestFilters.keyword = keyword;
    }
    if (filters.value.category) {
      requestFilters.category = filters.value.category;
    }

    const permissionResult = await getPermissions(requestFilters);
    permissions.value = permissionResult.items;
    if (!preservePagination) {
      pagination.value.current = 1;
    }
  } catch (error) {
    permissions.value = [];
    logger.error('failed to fetch permissions', error);
    listError.value = resolveLocalizedErrorMessage(t, error, t('rbac.permissionList.loadFailed'));
    MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, listError.value));
  } finally {
    loading.value = false;
  }
}

function resetFilters() {
  filters.value = {
    keyword: '',
    category: '',
  };
  pagination.value.current = 1;
}

function localizedPermissionDisplay(permission: PermissionListItem) {
  return localizePermissionDisplay(t, permission);
}

function localizedPermissionDescription(permission: PermissionListItem) {
  return localizePermissionDescription(t, permission, 'rbac.permissionList.emptyDescription');
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

onMounted(() => {
  fetchPermissions(true);
});

watch(
  () => [filters.value.keyword, filters.value.category] as const,
  () => {
    pagination.value.current = 1;
    fetchPermissions();
  },
);
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
  .toolbar__search,
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
