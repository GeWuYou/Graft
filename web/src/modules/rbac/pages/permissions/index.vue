<template>
  <div class="permission-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header :title="t('rbac.permissionList.listTitle')" :description="t('rbac.permissionList.hint')">
        <template #eyebrow>{{ t('menu.access_control.title') }}</template>
        <template #meta>
          <t-tag theme="default" variant="light">{{ t('rbac.permissionList.readonlyNotice') }}</t-tag>
        </template>
        <template #actions>
          <t-button theme="default" variant="outline" :loading="loading" @click="fetchPermissions">
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
            <t-button theme="primary" variant="outline" @click="fetchPermissions">
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
  </div>
</template>
<script setup lang="ts">
import { MessagePlugin, type TdBaseTableProps } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  ManagementEmptyState,
  ManagementPageContent,
  ManagementPageHeader,
  ManagementTableCard,
  ManagementTablePagination,
  ManagementToolbar,
} from '@/shared/components/management';
import { createLogger } from '@/utils/logger';

import { getPermissions } from '../../api/rbac';
import { PERMISSION_COPY_BY_CODE } from '../../contract/permission-copy';
import type { PermissionFilters, PermissionListItem } from '../../types/permission';

defineOptions({
  name: 'PermissionIndex',
});

const logger = createLogger('rbac.permissionList');

type PermissionFilterState = {
  keyword: string;
  category: string;
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
const visibleColumnKeys = ref(['permission', 'category', 'code', 'description', 'role_count', 'updated_at']);
const pagination = ref({
  current: 1,
  pageSize: 10,
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
]);

const filteredPermissions = computed(() => permissions.value);

const pagedPermissions = computed(() => {
  const start = (pagination.value.current - 1) * pagination.value.pageSize;
  return filteredPermissions.value.slice(start, start + pagination.value.pageSize);
});

const visibleColumns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;

  const allColumns: TdBaseTableProps['columns'] = [
    { title: t('rbac.permissionList.columns.permission'), colKey: 'permission', minWidth: 320, fixed: 'left' },
    { title: t('rbac.permissionList.columns.module'), colKey: 'category', width: 160 },
    { title: t('rbac.permissionList.columns.code'), colKey: 'code', minWidth: 240 },
    { title: t('rbac.permissionList.columns.description'), colKey: 'description', minWidth: 260 },
    { title: t('rbac.permissionList.columns.roleCount'), colKey: 'role_count', width: 120 },
    { title: t('rbac.permissionList.columns.createdAt'), colKey: 'created_at', width: 196 },
    { title: t('rbac.permissionList.columns.updatedAt'), colKey: 'updated_at', width: 196 },
  ];

  const visibleKeys = new Set(visibleColumnKeys.value);
  return allColumns.filter((column) => visibleKeys.has(String(column.colKey)));
});

async function fetchPermissions() {
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
    pagination.value.current = 1;
  } catch (error) {
    permissions.value = [];
    logger.error('failed to fetch permissions', error);
    listError.value = t('rbac.permissionList.loadFailed');
    MessagePlugin.error(listError.value);
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

function localizedMessage(messageKey: string, fallback?: string | null) {
  const translated = t(messageKey);
  if (translated !== messageKey) {
    return translated;
  }

  return fallback?.trim() || '';
}

function localizedPermissionDisplay(permission: PermissionListItem) {
  const copyEntry = PERMISSION_COPY_BY_CODE[permission.code];
  if (!copyEntry) {
    return permission.display;
  }

  return localizedMessage(copyEntry.displayKey, permission.display) || permission.display;
}

function localizedPermissionDescription(permission: PermissionListItem) {
  const copyEntry = PERMISSION_COPY_BY_CODE[permission.code];
  if (copyEntry) {
    const localized = localizedMessage(copyEntry.descriptionKey, permission.description);
    if (localized) {
      return localized;
    }
  }

  return permission.description?.trim() || t('rbac.permissionList.emptyDescription');
}

function formatTimestamp(value?: string | null) {
  if (!value) {
    return '-';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(locale.value === 'zh-CN' ? 'zh-CN' : 'en-US', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date);
}

onMounted(() => {
  fetchPermissions();
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
  gap: 16px;
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
  gap: 4px;
  padding: 12px 14px;
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
  gap: 4px;
}

.permission-cell__name {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.drawer-panel,
.column-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

@media (width <= 768px) {
  .toolbar__search,
  .toolbar__select {
    width: 100%;
  }

  .table-empty-state {
    min-height: 260px;
    padding-inline: 16px;
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
