<template>
  <div class="role-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header :title="t('rbac.roleList.listTitle')" :description="t('rbac.roleList.hint')">
        <template #eyebrow>{{ t('menu.access_control.title') }}</template>
        <template #actions>
          <t-button
            theme="default"
            variant="outline"
            :loading="loading"
            data-testid="role-refresh"
            @click="fetchRolePageData"
          >
            {{ t('rbac.roleList.refresh') }}
          </t-button>
          <t-button v-if="canCreateRoles" theme="primary" data-testid="role-create" @click="openCreateDrawer">
            {{ t('rbac.roleList.create') }}
          </t-button>
        </template>
      </management-page-header>

      <management-toolbar>
        <template #filters>
          <t-input
            v-model="filters.keyword"
            clearable
            class="toolbar__search"
            :placeholder="t('rbac.roleList.toolbar.searchPlaceholder')"
          />
          <t-select
            v-model="filters.type"
            clearable
            class="toolbar__select"
            :options="roleTypeOptions"
            :placeholder="t('rbac.roleList.toolbar.typePlaceholder')"
          />
          <t-button theme="default" variant="text" @click="resetFilters">
            {{ t('rbac.roleList.toolbar.clearFilters') }}
          </t-button>
        </template>
        <template #actions>
          <t-button theme="default" variant="outline" @click="columnDrawerVisible = true">
            {{ t('rbac.roleList.columnSettings') }}
          </t-button>
        </template>
      </management-toolbar>

      <management-table-card>
        <template #head>
          <div class="table-head">
            <div>
              <p class="table-head__summary">{{ t('rbac.roleList.summary', { count: filteredRoles.length }) }}</p>
              <p class="table-head__description">{{ t('rbac.roleList.tableHint') }}</p>
            </div>
            <t-button v-if="hasActiveFilters" theme="default" variant="text" @click="resetFilters">
              {{ t('rbac.roleList.toolbar.clearFilters') }}
            </t-button>
          </div>
        </template>

        <div v-if="permissionCatalogError" class="inline-warning">
          <span>{{ permissionCatalogError }}</span>
        </div>

        <management-empty-state
          v-if="listError && !loading"
          tone="error"
          :title="t('rbac.roleList.errorTitle')"
          :description="listError"
        >
          <template #actions>
            <t-button theme="primary" variant="outline" @click="fetchRolePageData">
              {{ t('rbac.roleList.retry') }}
            </t-button>
          </template>
        </management-empty-state>

        <t-table
          v-else
          row-key="id"
          :data="pagedRoles"
          :columns="visibleColumns"
          :loading="loading"
          table-layout="fixed"
          table-content-width="100%"
          cell-empty-content="-"
        >
          <template #role="{ row }">
            <div class="role-identity">
              <span class="role-identity__display">{{ row.display }}</span>
              <span class="role-identity__code">{{ row.name }}</span>
            </div>
          </template>

          <template #builtin="{ row }">
            <t-tag class="role-type-tag" :theme="row.builtin ? 'warning' : 'default'" variant="light">
              {{ row.builtin ? t('rbac.roleList.builtinYes') : t('rbac.roleList.builtinNo') }}
            </t-tag>
          </template>

          <template #permission_count="{ row }">
            <span class="role-count">{{ countLabel(row.permission_count, 'rbac.roleList.permissionCount') }}</span>
          </template>

          <template #user_count="{ row }">
            <span class="role-count">{{ countLabel(row.user_count, 'rbac.roleList.userCount') }}</span>
          </template>

          <template #remark="{ row }">
            <span class="role-remark table-muted">{{ roleRemark(row) }}</span>
          </template>

          <template #updated_at="{ row }">
            <span class="role-date">{{ formatTimestamp(row.updated_at) }}</span>
          </template>

          <template #operation="{ row }">
            <div class="table-actions">
              <t-button
                size="small"
                theme="default"
                variant="outline"
                data-testid="role-detail"
                @click="openDetailDrawer(row)"
              >
                {{ t('rbac.roleList.detail') }}
              </t-button>
              <t-button
                size="small"
                theme="default"
                variant="outline"
                data-testid="role-assign-permissions"
                :disabled="!canOpenPermissionDrawer"
                @click="openPermissionDrawer(row)"
              >
                {{ t('rbac.roleList.assignPermissions') }}
              </t-button>
              <t-button
                v-if="canUpdateRoles"
                size="small"
                theme="default"
                variant="outline"
                data-testid="role-edit"
                @click="openEditDrawer(row)"
              >
                {{ t('rbac.roleList.edit') }}
              </t-button>
              <t-tooltip v-if="row.builtin" :content="t('rbac.roleList.builtinHint')">
                <t-dropdown
                  :options="roleRowMoreOptions(row)"
                  trigger="click"
                  @click="(payload) => handleRoleMoreAction(payload, row)"
                >
                  <t-button size="small" theme="default" variant="outline">
                    {{ t('rbac.roleList.more') }}
                  </t-button>
                </t-dropdown>
              </t-tooltip>
              <t-dropdown
                v-else
                :options="roleRowMoreOptions(row)"
                trigger="click"
                @click="(payload) => handleRoleMoreAction(payload, row)"
              >
                <t-button size="small" theme="default" variant="outline">
                  {{ t('rbac.roleList.more') }}
                </t-button>
              </t-dropdown>
            </div>
          </template>

          <template #empty>
            <div class="table-empty-state">
              <t-empty :title="t('rbac.roleList.emptyTitle')" :description="t('rbac.roleList.emptyDescription')">
                <template #action>
                  <div class="table-empty-state__actions">
                    <t-button
                      v-if="hasActiveFilters"
                      theme="default"
                      variant="outline"
                      data-testid="role-empty-clear-filters"
                      @click="resetFilters"
                    >
                      {{ t('rbac.roleList.toolbar.clearFilters') }}
                    </t-button>
                    <t-button
                      v-if="canCreateRoles"
                      theme="primary"
                      data-testid="role-empty-create"
                      @click="openCreateDrawer"
                    >
                      {{ t('rbac.roleList.emptyCreate') }}
                    </t-button>
                  </div>
                </template>
              </t-empty>
            </div>
          </template>
        </t-table>

        <template #footer>
          <management-table-pagination :summary="t('rbac.roleList.footerTotal', { count: filteredRoles.length })">
            <t-pagination
              v-model:current="pagination.current"
              v-model:page-size="pagination.pageSize"
              :total="filteredRoles.length"
              :page-size-options="[10, 20, 50]"
              :show-page-number="true"
            />
          </management-table-pagination>
        </template>
      </management-table-card>
    </management-page-content>

    <t-drawer
      v-model:visible="roleDrawerVisible"
      :header="roleDrawerTitle"
      size="520px"
      placement="right"
      destroy-on-close
    >
      <div class="drawer-panel">
        <div v-if="roleDrawerRole?.builtin" class="inline-warning">
          <span>{{ t('rbac.roleList.form.builtinNotice') }}</span>
        </div>

        <t-form :data="roleForm" :rules="roleFormRules" label-align="top" @submit="handleRoleSubmit">
          <t-form-item :label="t('rbac.roleList.form.name')" name="name">
            <t-input
              v-model="roleForm.name"
              :disabled="roleDrawerMode === 'detail' || Boolean(roleDrawerRole?.builtin)"
              :placeholder="t('rbac.roleList.form.namePlaceholder')"
            />
          </t-form-item>
          <t-form-item :label="t('rbac.roleList.form.display')" name="display">
            <t-input
              v-model="roleForm.display"
              :disabled="roleDrawerMode === 'detail'"
              :placeholder="t('rbac.roleList.form.displayPlaceholder')"
            />
          </t-form-item>
          <t-form-item :label="t('rbac.roleList.form.description')" name="description">
            <t-textarea
              v-model="roleForm.description"
              :disabled="roleDrawerMode === 'detail'"
              :maxlength="200"
              :placeholder="t('rbac.roleList.form.descriptionPlaceholder')"
            />
          </t-form-item>
          <div class="drawer-actions">
            <t-button variant="outline" data-testid="role-drawer-cancel" @click="closeRoleDrawer">
              {{ t('rbac.roleList.form.cancel') }}
            </t-button>
            <t-button
              v-if="roleDrawerMode !== 'detail'"
              theme="primary"
              type="submit"
              data-testid="role-drawer-save"
              :loading="submittingRole"
            >
              {{ t('rbac.roleList.form.confirm') }}
            </t-button>
          </div>
        </t-form>
      </div>
    </t-drawer>

    <t-drawer
      v-model:visible="permissionDrawerVisible"
      :header="t('rbac.roleList.permissionDialog.title')"
      size="720px"
      placement="right"
      destroy-on-close
    >
      <div class="drawer-panel" data-testid="permission-drawer">
        <div class="drawer-summary">
          <div class="role-identity">
            <div class="role-identity__main">
              <span class="role-identity__display">{{ selectedRole?.display || '-' }}</span>
              <span class="role-identity__code">{{ selectedRole?.name || '-' }}</span>
            </div>
          </div>
          <div class="drawer-summary__grid">
            <div class="drawer-summary__item">
              <span class="drawer-summary__label">{{ t('rbac.roleList.columns.type') }}</span>
              <t-tag :theme="selectedRole?.builtin ? 'warning' : 'default'" variant="light">
                {{ selectedRole?.builtin ? t('rbac.roleList.builtinYes') : t('rbac.roleList.builtinNo') }}
              </t-tag>
            </div>
            <div class="drawer-summary__item">
              <span class="drawer-summary__label">{{ t('rbac.roleList.columns.updatedAt') }}</span>
              <span>{{ formatTimestamp(selectedRole?.updated_at) }}</span>
            </div>
          </div>
        </div>

        <div class="permission-toolbar">
          <t-input
            v-model="permissionKeyword"
            clearable
            class="toolbar__search"
            :placeholder="t('rbac.roleList.permissionDialog.searchPlaceholder')"
          />
          <label class="permission-toolbar__toggle">
            <t-checkbox v-model="permissionOnlySelected">{{
              t('rbac.roleList.permissionDialog.onlySelected')
            }}</t-checkbox>
          </label>
        </div>

        <div v-if="loadingRolePermissions || permissionLoadWarning" class="inline-warning">
          <span>{{ permissionDialogStatusMessage }}</span>
          <t-button
            v-if="permissionLoadRetryable"
            variant="text"
            theme="primary"
            :loading="loadingRolePermissions"
            @click="retryPermissionDrawerLoad"
          >
            {{ t('rbac.roleList.permissionDialog.retry') }}
          </t-button>
        </div>

        <div v-if="filteredPermissionGroups.length > 0" class="permission-groups">
          <section v-for="group in filteredPermissionGroups" :key="group.category" class="permission-group">
            <div class="permission-group__head">
              <div>
                <h3 class="permission-group__title">{{ group.title }}</h3>
                <p class="permission-group__meta">
                  {{ t('rbac.roleList.permissionDialog.groupHint', { count: group.items.length }) }}
                </p>
              </div>
              <div class="table-actions">
                <t-button
                  size="small"
                  theme="default"
                  variant="outline"
                  :disabled="!canAssignPermissions"
                  @click="selectGroupPermissions(group.items)"
                >
                  {{ t('rbac.roleList.permissionDialog.selectGroup') }}
                </t-button>
                <t-button
                  size="small"
                  theme="default"
                  variant="outline"
                  :disabled="!canAssignPermissions"
                  @click="clearGroupPermissions(group.items)"
                >
                  {{ t('rbac.roleList.permissionDialog.clearGroup') }}
                </t-button>
              </div>
            </div>

            <t-checkbox-group
              v-model="selectedPermissionIds"
              :disabled="loadingRolePermissions || !permissionSelectionReady || !canAssignPermissions"
              data-testid="permission-checkbox-group"
            >
              <div class="permission-list">
                <label v-for="item in group.items" :key="item.id" class="permission-card">
                  <t-checkbox :value="item.id">
                    <div class="permission-card__body">
                      <div class="permission-card__head">
                        <span class="permission-card__name">{{ item.display }}</span>
                        <span class="permission-card__code">{{ item.code }}</span>
                      </div>
                      <span class="permission-card__description">
                        {{ item.description || t('rbac.roleList.permissionDialog.emptyDescription') }}
                      </span>
                    </div>
                  </t-checkbox>
                </label>
              </div>
            </t-checkbox-group>
          </section>
        </div>
        <t-empty v-else :description="t('rbac.roleList.permissionDialog.empty')" />

        <div class="drawer-actions drawer-actions--between">
          <span class="table-muted">
            {{
              t('rbac.roleList.permissionDialog.selectionCount', {
                selected: selectedPermissionIds.length,
                total: permissions.length,
              })
            }}
          </span>
          <div class="table-actions">
            <t-button variant="outline" data-testid="permission-drawer-cancel" @click="closePermissionDrawer">
              {{ t('rbac.roleList.form.cancel') }}
            </t-button>
            <t-button
              theme="primary"
              data-testid="permission-drawer-save"
              :disabled="!canSubmitPermissionAssignment"
              :loading="submittingPermissions"
              @click="submitPermissionAssignment"
            >
              {{ t('rbac.roleList.permissionDialog.confirm') }}
            </t-button>
          </div>
        </div>
      </div>
    </t-drawer>

    <t-drawer
      v-model:visible="columnDrawerVisible"
      :header="t('rbac.roleList.columnSettings')"
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
import { type FormRule, MessagePlugin, type SubmitContext, type TdBaseTableProps } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import {
  ManagementEmptyState,
  ManagementPageContent,
  ManagementPageHeader,
  ManagementTableCard,
  ManagementTablePagination,
  ManagementToolbar,
} from '@/shared/components/management';
import { usePermissionStore } from '@/store';
import { createLogger } from '@/utils/logger';

import {
  assignRolePermissions,
  createRole,
  getPermissions,
  getRolePermissionBindings,
  getRoles,
  updateRole,
} from '../api/rbac';
import { RBAC_PERMISSION_CODE } from '../contract/permissions';
import type { RoleListItem } from '../contract/role';
import type { PermissionListItem } from '../types/permission';
import type { CreateRolePayload, ReplaceRolePermissionsPayload } from '../types/rbac';

defineOptions({
  name: 'RolesIndex',
});

const logger = createLogger('rbac.roleList');

type RoleDrawerMode = 'create' | 'detail' | 'update';

type RoleFilters = {
  keyword: string;
  type: '' | 'builtin' | 'custom';
};

type RoleFormState = {
  description: string;
  display: string;
  name: string;
};

type PermissionGroup = {
  category: string;
  items: PermissionListItem[];
  title: string;
};

type RoleRemarkCompat = RoleListItem & {
  remark?: string | null;
};

const DEFAULT_VISIBLE_COLUMNS = [
  'role',
  'builtin',
  'permission_count',
  'user_count',
  'remark',
  'updated_at',
  'operation',
];

const INITIAL_ROLE_FORM: RoleFormState = {
  description: '',
  display: '',
  name: '',
};

const { t, locale } = useI18n();
const route = useRoute();
const router = useRouter();
const permissionStore = usePermissionStore();
const roles = ref<RoleListItem[]>([]);
const permissions = ref<PermissionListItem[]>([]);
const filters = ref<RoleFilters>({
  keyword: '',
  type: '',
});
const visibleColumnKeys = ref<string[]>([...DEFAULT_VISIBLE_COLUMNS]);
const loading = ref(false);
const listError = ref('');
const permissionCatalogError = ref('');
const roleDrawerVisible = ref(false);
const roleDrawerMode = ref<RoleDrawerMode>('create');
const roleDrawerRole = ref<RoleListItem | null>(null);
const roleForm = ref<RoleFormState>({ ...INITIAL_ROLE_FORM });
const submittingRole = ref(false);
const permissionDrawerVisible = ref(false);
const selectedRole = ref<RoleListItem | null>(null);
const selectedPermissionIds = ref<number[]>([]);
const permissionDrawerSession = ref(0);
const permissionSelectionReady = ref(false);
const loadingRolePermissions = ref(false);
const submittingPermissions = ref(false);
const permissionLoadWarning = ref('');
const permissionLoadRetryable = ref(false);
const permissionKeyword = ref('');
const permissionOnlySelected = ref(false);
const columnDrawerVisible = ref(false);
const pagination = ref({
  current: 1,
  pageSize: 10,
});

const permissionCodes = RBAC_PERMISSION_CODE;
const canCreateRoles = computed(() => permissionStore.hasPermission(permissionCodes.ROLE_CREATE));
const canUpdateRoles = computed(() => permissionStore.hasPermission(permissionCodes.ROLE_UPDATE));
const canReadPermissions = computed(() => permissionStore.hasPermission(permissionCodes.PERMISSION_READ));
const canAssignPermissions = computed(
  () => canReadPermissions.value && permissionStore.hasPermission(permissionCodes.ROLE_PERMISSION_ASSIGN),
);
const canOpenPermissionDrawer = computed(() => canReadPermissions.value && permissions.value.length > 0);
const canShowOperationColumn = computed(() =>
  permissionStore.hasAnyPermission([
    permissionCodes.ROLE_UPDATE,
    permissionCodes.PERMISSION_READ,
    permissionCodes.ROLE_PERMISSION_ASSIGN,
  ]),
);
const canSubmitPermissionAssignment = computed(
  () => canAssignPermissions.value && permissionSelectionReady.value && selectedRole.value !== null,
);
const hasActiveFilters = computed(() => Boolean(filters.value.keyword.trim()) || Boolean(filters.value.type));
const permissionDialogStatusMessage = computed(() =>
  loadingRolePermissions.value ? t('rbac.roleList.permissionDialog.loadingSelection') : permissionLoadWarning.value,
);

const roleTypeOptions = computed(() => [
  { label: t('rbac.roleList.toolbar.typeAll'), value: '' },
  { label: t('rbac.roleList.builtinYes'), value: 'builtin' },
  { label: t('rbac.roleList.builtinNo'), value: 'custom' },
]);

const columnSettingOptions = computed(() => [
  { label: t('rbac.roleList.columns.role'), value: 'role' },
  { label: t('rbac.roleList.columns.type'), value: 'builtin' },
  { label: t('rbac.roleList.columns.permissionCount'), value: 'permission_count' },
  { label: t('rbac.roleList.columns.userCount'), value: 'user_count' },
  { label: t('rbac.roleList.columns.remark'), value: 'remark' },
  { label: t('rbac.roleList.columns.updatedAt'), value: 'updated_at' },
  { label: t('components.commonTable.operation'), value: 'operation' },
]);

const filteredRoles = computed(() => {
  const keyword = filters.value.keyword.trim().toLowerCase();

  return roles.value.filter((role) => {
    if (keyword) {
      const haystack = `${role.name} ${role.display} ${resolveRoleRemark(role)}`.toLowerCase();
      if (!haystack.includes(keyword)) {
        return false;
      }
    }

    if (filters.value.type === 'builtin' && !role.builtin) {
      return false;
    }

    if (filters.value.type === 'custom' && role.builtin) {
      return false;
    }

    return true;
  });
});

const pagedRoles = computed(() => {
  const start = (pagination.value.current - 1) * pagination.value.pageSize;
  return filteredRoles.value.slice(start, start + pagination.value.pageSize);
});

const roleRowMoreOptions = (role: RoleListItem) => [
  {
    content: t('rbac.roleList.moreActions.delete'),
    disabled: role.builtin,
    value: 'delete',
  },
];

const roleFormRules = computed<Record<keyof RoleFormState, FormRule[]>>(() => ({
  name: [{ required: true, message: t('rbac.roleList.form.required.name'), type: 'error' }],
  display: [{ required: true, message: t('rbac.roleList.form.required.display'), type: 'error' }],
  description: [],
}));

const roleDrawerTitle = computed(() => {
  switch (roleDrawerMode.value) {
    case 'detail':
      return t('rbac.roleList.form.detailTitle');
    case 'update':
      return t('rbac.roleList.form.editTitle');
    default:
      return t('rbac.roleList.form.createTitle');
  }
});

const filteredPermissionGroups = computed<PermissionGroup[]>(() => {
  const keyword = permissionKeyword.value.trim().toLowerCase();
  const selected = new Set(selectedPermissionIds.value);
  const groups = new Map<string, PermissionListItem[]>();

  permissions.value.forEach((item) => {
    if (permissionOnlySelected.value && !selected.has(item.id)) {
      return;
    }

    if (keyword) {
      const haystack = `${item.code} ${item.display} ${item.description ?? ''} ${item.category}`.toLowerCase();
      if (!haystack.includes(keyword)) {
        return;
      }
    }

    const category = item.category || 'general';
    const collection = groups.get(category) ?? [];
    collection.push(item);
    groups.set(category, collection);
  });

  return Array.from(groups.entries())
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([category, items]) => ({
      category,
      title: t('rbac.roleList.permissionDialog.category', { category }),
      items: items.slice().sort((left, right) => left.code.localeCompare(right.code)),
    }));
});

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;

  const allColumns: TdBaseTableProps['columns'] = [
    {
      title: t('rbac.roleList.columns.role'),
      colKey: 'role',
      minWidth: 220,
      ellipsis: true,
    },
    {
      title: t('rbac.roleList.columns.type'),
      colKey: 'builtin',
      width: 100,
    },
    {
      title: t('rbac.roleList.columns.permissionCount'),
      colKey: 'permission_count',
      width: 100,
    },
    {
      title: t('rbac.roleList.columns.userCount'),
      colKey: 'user_count',
      width: 100,
    },
    {
      title: t('rbac.roleList.columns.remark'),
      colKey: 'remark',
      minWidth: 220,
      ellipsis: true,
    },
    {
      title: t('rbac.roleList.columns.updatedAt'),
      colKey: 'updated_at',
      width: 180,
    },
  ];

  if (canShowOperationColumn.value) {
    allColumns.push({
      title: t('components.commonTable.operation'),
      colKey: 'operation',
      width: 260,
      align: 'right',
      fixed: 'right',
    });
  }

  const visibleKeys = new Set(visibleColumnKeys.value);
  return allColumns.filter((column) => visibleKeys.has(String(column.colKey)));
});

const visibleColumns = computed(() => {
  if (canShowOperationColumn.value) {
    return columns.value;
  }

  return (columns.value ?? []).filter((column) => column?.colKey !== 'operation');
});

async function fetchRolePageData() {
  loading.value = true;
  listError.value = '';

  try {
    const [roleResult, permissionResult] = await Promise.allSettled([
      getRoles(),
      canReadPermissions.value ? getPermissions() : Promise.resolve({ items: [] as PermissionListItem[] }),
    ]);

    if (roleResult.status === 'rejected') {
      throw roleResult.reason;
    }

    roles.value = roleResult.value.items;
    pagination.value.current = 1;

    if (permissionResult.status === 'fulfilled') {
      permissions.value = permissionResult.value.items;
      permissionCatalogError.value = '';
    } else {
      permissions.value = [];
      permissionCatalogError.value =
        permissionResult.reason instanceof Error
          ? permissionResult.reason.message
          : t('rbac.roleList.permissionLoadFailed');
      MessagePlugin.warning(permissionCatalogError.value);
    }
  } catch (error) {
    roles.value = [];
    logger.error('failed to fetch role page data', error);
    listError.value = t('rbac.roleList.loadFailed');
    MessagePlugin.error(listError.value);
  } finally {
    loading.value = false;
  }
}

function resetFilters() {
  filters.value = {
    keyword: '',
    type: '',
  };
  pagination.value.current = 1;
}

function formatTimestamp(value?: string | null) {
  if (!value) {
    return '-';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(locale.value, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date);
}

function countLabel(value: number | undefined, messageKey: string) {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return '-';
  }

  return t(messageKey, { count: value });
}

function resolveRoleRemark(role: RoleRemarkCompat) {
  return role.remark ?? role.description ?? '';
}

function roleRemark(role: RoleListItem) {
  const remark = resolveRoleRemark(role).trim();
  return remark || '-';
}

function normalizeDescription(description: string) {
  const trimmed = description.trim();
  return trimmed ? trimmed : null;
}

function sortStableIDs(ids: number[]) {
  return ids.slice().sort((left, right) => left - right);
}

function toReplaceRolePermissionsPayload(permissionIds: number[]): ReplaceRolePermissionsPayload {
  return {
    permission_ids: sortStableIDs(permissionIds),
  };
}

function normalizeRolePermissionIDs(rawPermissionIDs: number[]) {
  if (!Array.isArray(rawPermissionIDs)) {
    return null;
  }

  const availablePermissionIDs = new Set(permissions.value.map((item) => item.id));
  if (rawPermissionIDs.some((id) => !Number.isInteger(id) || id <= 0 || !availablePermissionIDs.has(id))) {
    return null;
  }

  return Array.from(new Set(rawPermissionIDs)).sort((left, right) => left - right);
}

function openCreateDrawer() {
  roleDrawerMode.value = 'create';
  roleDrawerRole.value = null;
  roleForm.value = { ...INITIAL_ROLE_FORM };
  roleDrawerVisible.value = true;
}

function consumeCreateActionQuery() {
  if (route.query.action !== 'create') {
    return;
  }

  const nextQuery = { ...route.query };
  delete nextQuery.action;
  void router.replace({ query: nextQuery });
}

function openEditDrawer(role: RoleListItem) {
  roleDrawerMode.value = 'update';
  roleDrawerRole.value = role;
  roleForm.value = {
    name: role.name,
    display: role.display,
    description: resolveRoleRemark(role),
  };
  roleDrawerVisible.value = true;
}

function handleRoleMoreAction(
  payload: { value?: string | number | Record<string, unknown> } | string | number,
  role: RoleListItem,
) {
  const action = typeof payload === 'object' && payload ? payload.value : payload;
  if (action === 'edit') {
    openEditDrawer(role);
    return;
  }

  if (action === 'delete') {
    handleMoreAction(role);
    return;
  }

  handleMoreAction(role);
}

function openDetailDrawer(role: RoleListItem) {
  roleDrawerMode.value = 'detail';
  roleDrawerRole.value = role;
  roleForm.value = {
    name: role.name,
    display: role.display,
    description: resolveRoleRemark(role),
  };
  roleDrawerVisible.value = true;
}

function closeRoleDrawer() {
  roleDrawerVisible.value = false;
  roleDrawerRole.value = null;
  roleForm.value = { ...INITIAL_ROLE_FORM };
  submittingRole.value = false;
}

async function handleRoleSubmit(ctx: SubmitContext) {
  if (ctx.validateResult !== true || submittingRole.value || roleDrawerMode.value === 'detail') {
    return;
  }

  submittingRole.value = true;
  try {
    const payload: CreateRolePayload = {
      name: roleForm.value.name.trim(),
      display: roleForm.value.display.trim(),
      description: normalizeDescription(roleForm.value.description),
    };

    if (roleDrawerMode.value === 'create') {
      const created = await createRole(payload);
      roles.value = [...roles.value, created].sort((left, right) => left.id - right.id);
      MessagePlugin.success(t('rbac.roleList.createSuccess'));
    } else if (roleDrawerRole.value) {
      const updated = await updateRole(roleDrawerRole.value.id, payload);
      roles.value = roles.value.map((item) => (item.id === updated.id ? updated : item));
      roleDrawerRole.value = updated;
      MessagePlugin.success(t('rbac.roleList.updateSuccess'));
    }

    closeRoleDrawer();
  } catch (error) {
    MessagePlugin.error(error instanceof Error ? error.message : t('rbac.roleList.submitFailed'));
  } finally {
    submittingRole.value = false;
  }
}

function isActivePermissionDrawerSession(session: number) {
  return permissionDrawerVisible.value && permissionDrawerSession.value === session;
}

function applyRolePermissionSelection(permissionIDs: number[]) {
  const normalized = normalizeRolePermissionIDs(permissionIDs);
  if (normalized === null) {
    selectedPermissionIds.value = [];
    permissionSelectionReady.value = false;
    return false;
  }

  selectedPermissionIds.value = normalized;
  permissionSelectionReady.value = true;
  return true;
}

async function loadRolePermissionSelection(roleId: number, session: number) {
  if (isActivePermissionDrawerSession(session)) {
    loadingRolePermissions.value = true;
    permissionSelectionReady.value = false;
    selectedPermissionIds.value = [];
    permissionLoadWarning.value = '';
    permissionLoadRetryable.value = false;
  }

  try {
    const response = await getRolePermissionBindings(roleId);
    if (!isActivePermissionDrawerSession(session)) {
      return false;
    }

    if (!applyRolePermissionSelection(response.permission_ids)) {
      permissionLoadWarning.value = t('rbac.roleList.permissionDialog.selectionUnavailable');
      permissionLoadRetryable.value = false;
      return false;
    }

    return true;
  } catch (error) {
    if (!isActivePermissionDrawerSession(session)) {
      return false;
    }

    permissionLoadWarning.value =
      error instanceof Error ? error.message : t('rbac.roleList.permissionDialog.selectionLoadFailed');
    permissionLoadRetryable.value = true;
    return false;
  } finally {
    if (isActivePermissionDrawerSession(session)) {
      loadingRolePermissions.value = false;
    }
  }
}

async function openPermissionDrawer(role: RoleListItem) {
  if (!canOpenPermissionDrawer.value) {
    MessagePlugin.warning(permissionCatalogError.value || t('rbac.roleList.permissionUnavailable'));
    return;
  }

  const session = permissionDrawerSession.value + 1;
  permissionDrawerSession.value = session;
  permissionDrawerVisible.value = true;
  selectedRole.value = role;
  permissionKeyword.value = '';
  permissionOnlySelected.value = false;
  await loadRolePermissionSelection(role.id, session);
}

function closePermissionDrawer() {
  permissionDrawerSession.value += 1;
  permissionDrawerVisible.value = false;
  selectedRole.value = null;
  selectedPermissionIds.value = [];
  permissionSelectionReady.value = false;
  loadingRolePermissions.value = false;
  permissionLoadWarning.value = '';
  permissionLoadRetryable.value = false;
  submittingPermissions.value = false;
  permissionKeyword.value = '';
  permissionOnlySelected.value = false;
}

async function retryPermissionDrawerLoad() {
  if (!selectedRole.value) {
    return;
  }

  await loadRolePermissionSelection(selectedRole.value.id, permissionDrawerSession.value);
}

function selectGroupPermissions(items: PermissionListItem[]) {
  if (!canAssignPermissions.value) {
    return;
  }

  const next = new Set(selectedPermissionIds.value);
  items.forEach((item) => next.add(item.id));
  selectedPermissionIds.value = sortStableIDs(Array.from(next));
}

function clearGroupPermissions(items: PermissionListItem[]) {
  if (!canAssignPermissions.value) {
    return;
  }

  const blocked = new Set(items.map((item) => item.id));
  selectedPermissionIds.value = selectedPermissionIds.value.filter((id) => !blocked.has(id));
}

async function submitPermissionAssignment() {
  if (!selectedRole.value || !canSubmitPermissionAssignment.value || loadingRolePermissions.value) {
    return;
  }

  const session = permissionDrawerSession.value;
  const payload = toReplaceRolePermissionsPayload(selectedPermissionIds.value);

  submittingPermissions.value = true;
  try {
    await assignRolePermissions(selectedRole.value.id, payload);

    if (!isActivePermissionDrawerSession(session)) {
      return;
    }

    MessagePlugin.success(t('rbac.roleList.assignSuccess'));
    closePermissionDrawer();
    await fetchRolePageData();
  } catch (error) {
    if (isActivePermissionDrawerSession(session)) {
      MessagePlugin.error(error instanceof Error ? error.message : t('rbac.roleList.assignFailed'));
    }
  } finally {
    if (permissionDrawerSession.value === session) {
      submittingPermissions.value = false;
    }
  }
}

function handleMoreAction(role: RoleListItem) {
  MessagePlugin.warning(role.builtin ? t('rbac.roleList.moreBuiltinHint') : t('rbac.roleList.moreCustomHint'));
}

onMounted(() => {
  fetchRolePageData();
});

watch(
  () => [filters.value.keyword, filters.value.type] as const,
  () => {
    pagination.value.current = 1;
  },
);

watch(
  () => [route.query.action, canCreateRoles.value, roleDrawerVisible.value] as const,
  ([action, allowed, visible]) => {
    if (action !== 'create' || !allowed || visible) {
      return;
    }

    openCreateDrawer();
    consumeCreateActionQuery();
  },
  { immediate: true },
);
</script>
<style lang="less" scoped>
@import './index.less';
</style>
