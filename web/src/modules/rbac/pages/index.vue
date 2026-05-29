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
          <t-button
            v-permission="permissionCodes.ROLE_CREATE"
            theme="primary"
            data-testid="role-create"
            @click="openCreateDrawer"
          >
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
          :table-content-width="tableContentWidth"
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
            <table-action-menu
              :actions="roleRowActions(row)"
              :more-label="t('rbac.roleList.more')"
              more-label-fallback="更多"
              @action="(action) => handleRoleRowAction(action, row)"
            />
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
                      v-permission="permissionCodes.ROLE_CREATE"
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
        <div
          v-if="roleDrawerMode === 'detail' && roleDrawerRole"
          class="inline-warning"
          data-testid="role-lifecycle-summary"
        >
          <span>{{ t('rbac.roleList.lifecycle.statusLabel') }}: {{ roleStatusLabel(roleDrawerRole) }}</span>
          <span>{{ roleDeleteLifecycleHint(roleDrawerRole) }}</span>
        </div>

        <t-form ref="roleFormRef" :data="roleForm" :rules="roleFormRules" label-align="top" @submit="handleRoleSubmit">
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

    <assignment-drawer
      v-model:visible="permissionDrawerVisible"
      :title="t('rbac.roleList.permissionDialog.title')"
      size="860px"
      @close="requestClosePermissionDrawer"
    >
      <template #header>
        <div class="assignment-panel assignment-panel--compact" data-testid="permission-drawer">
          <assignment-header
            :avatar-text="roleAssignmentAvatar"
            :badges="roleAssignmentBadges"
            :description="roleAssignmentDescription"
            :eyebrow="t('rbac.roleList.permissionDialog.headerEyebrow')"
            :stats="roleAssignmentStats"
            :subtitle="roleAssignmentSubtitle"
            :title="roleAssignmentTitle"
          />

          <div class="assignment-toolbar-stack">
            <assignment-toolbar
              v-model:mode-value="permissionMutationMode"
              v-model:search-value="permissionKeyword"
              :disabled="loadingRolePermissions || submittingPermissions || !canAssignPermissions"
              :mode-label="t('rbac.roleList.permissionDialog.saveStrategyLabel')"
              :mode-options="permissionMutationOptions"
              :search-placeholder="t('rbac.roleList.permissionDialog.searchPlaceholder')"
            />
            <label class="assignment-toolbar-toggle">
              <t-checkbox v-model="permissionOnlySelected">
                {{ t('rbac.roleList.permissionDialog.onlySelected') }}
              </t-checkbox>
            </label>
          </div>

          <assignment-summary
            :hint="t('rbac.roleList.permissionDialog.operationHelp')"
            :items="roleAssignmentSummaryItems"
            :warning="permissionDialogStatusMessage"
            :warning-action-label="permissionLoadRetryable ? t('rbac.roleList.permissionDialog.retry') : ''"
            :warning-action-loading="loadingRolePermissions"
            @warning-action="retryPermissionDrawerLoad"
          />
        </div>
      </template>

      <assignment-grid
        :empty="filteredPermissionItems.length === 0"
        :empty-description="t('rbac.roleList.permissionDialog.empty')"
      >
        <t-checkbox-group
          v-model="selectedPermissionIds"
          class="sr-only"
          :disabled="loadingRolePermissions || !permissionSelectionReady || !canAssignPermissions"
          data-testid="permission-checkbox-group"
        />
        <div class="assignment-card-grid permission-card-grid">
          <assignment-card
            v-for="item in filteredPermissionItems"
            :key="item.id"
            :assigned="originalPermissionIds.includes(item.id)"
            :assigned-label="t('rbac.roleList.permissionDialog.assignedBadge')"
            :code="item.code"
            :description="localizedPermissionDescription(item)"
            :disabled="
              loadingRolePermissions ||
              !permissionSelectionReady ||
              !canAssignPermissions ||
              isPermissionCardDisabled(item)
            "
            :selected="selectedPermissionIds.includes(item.id)"
            :tags="[
              { label: t('rbac.roleList.permissionDialog.categoryBadge', { category: item.category || 'general' }) },
            ]"
            :title="localizedPermissionDisplay(item)"
            @toggle="toggleRolePermissionSelection(item.id)"
          />
        </div>
      </assignment-grid>

      <template #footer>
        <assignment-footer
          :cancel-label="t('rbac.roleList.form.cancel')"
          cancel-test-id="permission-drawer-cancel"
          :confirm-disabled="!canSubmitPermissionAssignment"
          :confirm-label="t('rbac.roleList.permissionDialog.confirm')"
          :confirm-loading="submittingPermissions"
          :details="permissionFooterDetails"
          confirm-test-id="permission-drawer-save"
          :summary="permissionFooterSummary"
          @cancel="requestClosePermissionDrawer"
          @confirm="submitPermissionAssignment"
        />
      </template>
    </assignment-drawer>

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
import {
  type FormRule,
  type FormValidateMessage,
  MessagePlugin,
  type SubmitContext,
  type TdBaseTableProps,
} from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { AUDIT_PERMISSION_CODE } from '@/modules/audit/contract/permissions';
import { localizedApiErrorMessage, resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import {
  AssignmentCard,
  AssignmentDrawer,
  AssignmentFooter,
  AssignmentGrid,
  AssignmentHeader,
  AssignmentSummary,
  AssignmentToolbar,
} from '@/shared/components/assignment';
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
import { useAssignmentSelection } from '@/shared/composables';
import { formatHintedMessage, resolveErrorMessageWithCorrelation } from '@/shared/correlation';
import { usePermissionStore } from '@/store';
import { createLogger } from '@/utils/logger';
import { isApiRequestError } from '@/utils/request';

import {
  addRolePermissions,
  createRole,
  deleteRole,
  getPermissions,
  getRoleDetail,
  getRolePermissionBindings,
  getRoles,
  removeRolePermissions,
  replaceRolePermissions,
  updateRole,
  updateRoleStatus,
} from '../api/rbac';
import { RBAC_PERMISSION_CODE } from '../contract/permissions';
import type { RoleListItem } from '../contract/role';
import { resolveRoleFormFieldError, resolveRolePermissionFieldError } from '../error-adapter';
import {
  localizedPermissionDescription as localizePermissionDescription,
  localizedPermissionDisplay as localizePermissionDisplay,
} from '../shared/permission-copy';
import type { PermissionListItem } from '../types/permission';
import type {
  CreateRolePayload,
  ReplaceRolePermissionsPayload,
  RoleDetailResponse,
  RolePermissionBindingResponse,
  RolePermissionMutationPayload,
  UpdateRolePayload,
} from '../types/rbac';

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

type RoleFormInstance = {
  clearValidate: (fields?: Array<keyof RoleFormState>) => void;
  setValidateMessage: (message: FormValidateMessage<RoleFormState>) => void;
};

type RolePermissionMutationMode = 'replace' | 'add' | 'remove';

type RoleRemarkCompat = RoleListItem & {
  remark?: string | null;
};

type RoleStatusCompat = RoleListItem & {
  description?: string | null;
  disabled?: boolean;
  enabled?: boolean;
  status?: string | null;
  deleted_at?: string | null;
};

const DEFAULT_VISIBLE_COLUMNS = ['role', 'builtin', 'permission_count', 'user_count', 'updated_at', 'operation'];

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
const roleFormRef = ref<RoleFormInstance | null>(null);
const roleForm = ref<RoleFormState>({ ...INITIAL_ROLE_FORM });
const submittingRole = ref(false);
const permissionDrawerVisible = ref(false);
const selectedRole = ref<RoleListItem | null>(null);
const originalPermissionIds = ref<number[]>([]);
const permissionDrawerSession = ref(0);
const permissionSelectionReady = ref(false);
const loadingRolePermissions = ref(false);
const submittingPermissions = ref(false);
const permissionMutationMode = ref<RolePermissionMutationMode>('replace');
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
const canDeleteRoles = computed(() => permissionStore.hasPermission(permissionCodes.ROLE_DELETE));
const canToggleRoleStatus = computed(() => permissionStore.hasPermission(permissionCodes.ROLE_STATUS_UPDATE));
const canReadPermissions = computed(() => permissionStore.hasPermission(permissionCodes.PERMISSION_READ));
const canAssignPermissions = computed(
  () => canReadPermissions.value && permissionStore.hasPermission(permissionCodes.ROLE_PERMISSION_ASSIGN),
);
const canOpenPermissionDrawer = computed(() => canReadPermissions.value && permissions.value.length > 0);
const canShowOperationColumn = computed(() =>
  permissionStore.hasAnyPermission([
    permissionCodes.ROLE_UPDATE,
    permissionCodes.ROLE_DELETE,
    permissionCodes.ROLE_STATUS_UPDATE,
    permissionCodes.PERMISSION_READ,
    permissionCodes.ROLE_PERMISSION_ASSIGN,
  ]),
);
const hasPermissionSelectionChanges = computed(() => {
  if (!permissionSelectionReady.value || selectedRole.value === null) {
    return false;
  }

  switch (permissionMutationMode.value) {
    case 'add':
    case 'remove':
      return permissionMutationPayload.value.permission_ids.length > 0;
    default:
      return !arePermissionIDsEqual(originalPermissionIds.value, selectedPermissionIds.value);
  }
});
const canSubmitPermissionAssignment = computed(() => {
  return canAssignPermissions.value && hasPermissionSelectionChanges.value;
});
const hasActiveFilters = computed(() => Boolean(filters.value.keyword.trim()) || Boolean(filters.value.type));
const permissionDialogStatusMessage = computed(() =>
  loadingRolePermissions.value ? t('rbac.roleList.permissionDialog.loadingSelection') : permissionLoadWarning.value,
);
const permissionMutationOptions = computed(() => [
  { label: t('rbac.roleList.permissionActions.replace'), value: 'replace' as const },
  { label: t('rbac.roleList.permissionActions.add'), value: 'add' as const },
  { label: t('rbac.roleList.permissionActions.remove'), value: 'remove' as const },
]);
const permissionMutationPayload = computed<RolePermissionMutationPayload>(() => {
  const original = new Set(originalPermissionIds.value);

  switch (permissionMutationMode.value) {
    case 'add':
      return toRolePermissionMutationPayload(selectedPermissionIds.value.filter((id) => !original.has(id)));
    case 'remove':
      return toRolePermissionMutationPayload(selectedPermissionIds.value.filter((id) => original.has(id)));
    default:
      return toReplaceRolePermissionsPayload(selectedPermissionIds.value);
  }
});
const permissionAddedCount = computed(() => {
  const original = new Set(originalPermissionIds.value);
  return selectedPermissionIds.value.filter((id) => !original.has(id)).length;
});
const permissionRemovedCount = computed(() => {
  const selected = new Set(selectedPermissionIds.value);
  return originalPermissionIds.value.filter((id) => !selected.has(id)).length;
});
const permissionFooterSummary = computed(() =>
  t('rbac.roleList.permissionDialog.selectionCount', {
    selected: selectedPermissionIds.value.length,
    total: permissions.value.length,
  }),
);
const permissionFooterDetails = computed(() => {
  const details = [
    t('rbac.roleList.permissionDialog.modeSummary', {
      mode: t(`rbac.roleList.permissionDialog.modeValue.${permissionMutationMode.value}`),
    }),
  ];

  if (permissionSelectionReady.value && selectedRole.value !== null && permissionMutationMode.value === 'replace') {
    if (permissionAddedCount.value > 0) {
      details.push(
        t('rbac.roleList.permissionDialog.addSelectionCount', {
          count: permissionAddedCount.value,
        }),
      );
    }

    if (permissionRemovedCount.value > 0) {
      details.push(
        t('rbac.roleList.permissionDialog.removeSelectionCount', {
          count: permissionRemovedCount.value,
        }),
      );
    }
  } else if (permissionMutationMode.value === 'add' || permissionMutationMode.value === 'remove') {
    details.push(
      t(
        permissionMutationMode.value === 'add'
          ? 'rbac.roleList.permissionDialog.addSelectionCount'
          : 'rbac.roleList.permissionDialog.removeSelectionCount',
        {
          count: permissionMutationPayload.value.permission_ids.length,
        },
      ),
    );
  }

  return details;
});

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

const roleRowMoreOptions = (role: RoleStatusCompat) => {
  const options: Array<{ content: string; disabled?: boolean; fallbackLabel: string; value: string }> = [];

  options.push({
    content: t('rbac.roleList.detail'),
    fallbackLabel: '详情',
    value: 'detail',
  });

  if (permissionStore.hasPermission(permissionCodes.ROLE_UPDATE)) {
    options.push({
      content: t('rbac.roleList.edit'),
      fallbackLabel: '编辑',
      value: 'edit',
    });
  }

  if (canToggleRoleStatus.value) {
    options.push({
      content: isRoleEnabled(role) ? t('rbac.roleList.moreActions.disable') : t('rbac.roleList.moreActions.enable'),
      disabled: role.builtin,
      fallbackLabel: isRoleEnabled(role) ? '停用角色' : '启用角色',
      value: 'toggle-status',
    });
  }

  if (canDeleteRoles.value) {
    options.push({
      content: t('rbac.roleList.moreActions.delete'),
      disabled: role.builtin,
      fallbackLabel: '删除角色',
      value: 'delete',
    });
  }

  return options;
};

function roleRowActions(role: RoleListItem) {
  const actions: Array<{ disabled?: boolean; label: string; testId?: string; value: string }> = [];

  if (canAssignPermissions.value) {
    actions.push({
      disabled: !canOpenPermissionDrawer.value,
      label: t('rbac.roleList.assignPermissions'),
      testId: 'role-assign-permissions',
      value: 'assign-permissions',
    });
  }

  if (permissionStore.hasPermission(AUDIT_PERMISSION_CODE.READ)) {
    actions.push({
      label: t('rbac.roleList.viewAudit'),
      testId: 'role-view-audit',
      value: 'view-audit',
    });
  }

  return [
    ...actions,
    ...roleRowMoreOptions(role).map((option) => ({
      disabled: option.disabled,
      fallbackLabel: option.fallbackLabel,
      label: option.content,
      testId: option.value === 'detail' ? 'role-detail' : option.value === 'edit' ? 'role-edit' : undefined,
      value: option.value,
    })),
  ];
}

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

const filteredPermissionItems = computed(() => {
  const keyword = permissionKeyword.value.trim().toLowerCase();
  const selected = new Set(selectedPermissionIds.value);

  return permissions.value
    .filter((item) => {
      if (permissionOnlySelected.value && !selected.has(item.id)) {
        return false;
      }

      if (!keyword) {
        return true;
      }

      return `${item.code} ${localizedPermissionDisplay(item)} ${localizedPermissionDescription(item)} ${item.category}`
        .toLowerCase()
        .includes(keyword);
    })
    .slice()
    .sort((left, right) => left.code.localeCompare(right.code));
});
const { selectedIds: selectedPermissionIdsInternal } = useAssignmentSelection({
  active: permissionDrawerVisible,
  mode: permissionMutationMode,
  originalIds: originalPermissionIds,
});
const selectedPermissionIds = selectedPermissionIdsInternal;
const roleAssignmentTitle = computed(() => selectedRole.value?.display || '-');
const roleAssignmentSubtitle = computed(() => selectedRole.value?.name || '-');
const roleAssignmentDescription = computed(
  () =>
    resolveRoleRemark(selectedRole.value ?? ({ remark: '' } as RoleRemarkCompat)) ||
    t('rbac.roleList.permissionDialog.headerDescription'),
);
const roleAssignmentAvatar = computed(() => (selectedRole.value?.display || '?').trim().slice(0, 1).toUpperCase());
const roleAssignmentBadges = computed(() => [
  {
    label: selectedRole.value?.builtin ? t('rbac.roleList.builtinYes') : t('rbac.roleList.builtinNo'),
    theme: selectedRole.value?.builtin ? ('warning' as const) : ('default' as const),
  },
]);
const roleAssignmentStats = computed(() => [
  {
    label: t('rbac.roleList.permissionDialog.stats.permissionCount'),
    value: Number(selectedRole.value?.permission_count ?? 0),
  },
  {
    label: t('rbac.roleList.permissionDialog.stats.userCount'),
    value: Number(selectedRole.value?.user_count ?? 0),
  },
]);
const roleAssignmentSummaryItems = computed(() => [
  {
    label: t('rbac.roleList.columns.updatedAt'),
    value: formatTimestamp(selectedRole.value?.updated_at),
  },
  {
    label: t('rbac.roleList.permissionDialog.summary.assigned'),
    value: currentAssignedPermissionCount.value,
  },
]);
const currentAssignedPermissionCount = computed(() => originalPermissionIds.value.length);

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;

  const allColumns: TdBaseTableProps['columns'] = [
    createTextColumn(t('rbac.roleList.columns.role'), 'role', {
      width: 336,
    }),
    createStatusColumn(t('rbac.roleList.columns.type'), 'builtin', 100),
    createCountColumn(t('rbac.roleList.columns.permissionCount'), 'permission_count', 112),
    createCountColumn(t('rbac.roleList.columns.userCount'), 'user_count', 112),
    createTextColumn(t('rbac.roleList.columns.remark'), 'remark', {
      width: 220,
    }),
    createTimeColumn(t('rbac.roleList.columns.updatedAt'), 'updated_at', 160),
  ];

  if (canShowOperationColumn.value) {
    allColumns.push(createActionColumn(t('components.commonTable.operation'), 160));
  }

  return buildVisibleColumns(allColumns, visibleColumnKeys.value);
});

const visibleColumns = computed(() => {
  if (canShowOperationColumn.value) {
    return columns.value;
  }

  return (columns.value ?? []).filter((column) => column?.colKey !== 'operation');
});

const tableContentWidth = computed(() => calculateTableContentWidth(visibleColumns.value));

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
      permissionCatalogError.value = resolveLocalizedErrorMessage(
        t,
        permissionResult.reason,
        t('rbac.roleList.permissionLoadFailed'),
      );
      MessagePlugin.warning(permissionCatalogError.value);
    }
  } catch (error) {
    roles.value = [];
    logger.error('failed to fetch role page data', error);
    listError.value = resolveLocalizedErrorMessage(t, error, t('rbac.roleList.loadFailed'));
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
  return formatCompactDateTime(value);
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

function isRoleEnabled(role: RoleStatusCompat) {
  if (role.status === 'enabled') {
    return true;
  }

  if (role.status === 'disabled') {
    return false;
  }

  if (typeof role.enabled === 'boolean') {
    return role.enabled;
  }

  if (typeof role.disabled === 'boolean') {
    return !role.disabled;
  }

  return true;
}

function roleStatusLabel(role: RoleStatusCompat) {
  return isRoleEnabled(role) ? t('rbac.roleList.lifecycle.statusEnabled') : t('rbac.roleList.lifecycle.statusDisabled');
}

function roleHasDeleteBlockingBindings(role: RoleStatusCompat) {
  return Number(role.permission_count ?? 0) > 0 || Number(role.user_count ?? 0) > 0;
}

function roleDeleteLifecycleHint(role: RoleStatusCompat) {
  if (role.builtin) {
    return t('rbac.roleList.moreBuiltinHint');
  }
  if (isRoleEnabled(role)) {
    return t('rbac.roleList.lifecycle.deleteNeedsDisable');
  }
  if (roleHasDeleteBlockingBindings(role)) {
    return t('rbac.roleList.lifecycle.deleteNeedsBindingsCleared');
  }
  return t('rbac.roleList.lifecycle.deleteReady');
}

function roleRemark(role: RoleListItem) {
  const remark = resolveRoleRemark(role).trim();
  return remark || '-';
}

function normalizeDescription(description: string) {
  const trimmed = description.trim();
  return trimmed ? trimmed : null;
}

function toCreateRolePayload(form: RoleFormState): CreateRolePayload {
  return {
    name: form.name.trim(),
    display: form.display.trim(),
    description: normalizeDescription(form.description),
  };
}

function toUpdateRolePayload(form: RoleFormState): UpdateRolePayload {
  return {
    name: form.name.trim(),
    display: form.display.trim(),
    description: normalizeDescription(form.description),
  };
}

function sortStableIDs(ids: number[]) {
  return ids.slice().sort((left, right) => left - right);
}

function arePermissionIDsEqual(left: number[], right: number[]) {
  if (left.length !== right.length) {
    return false;
  }

  return left.every((value, index) => value === right[index]);
}

function toReplaceRolePermissionsPayload(permissionIds: number[]): ReplaceRolePermissionsPayload {
  return {
    permission_ids: sortStableIDs(permissionIds),
  };
}

function toRolePermissionMutationPayload(permissionIds: number[]): RolePermissionMutationPayload {
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

function localizedPermissionDisplay(permission: PermissionListItem) {
  return localizePermissionDisplay(t, permission);
}

function localizedPermissionDescription(permission: PermissionListItem) {
  return localizePermissionDescription(t, permission, 'rbac.roleList.permissionDialog.emptyDescription');
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

  if (action === 'toggle-status') {
    void toggleRoleStatus(role);
    return;
  }

  if (action === 'delete') {
    void removeRole(role);
    return;
  }

  if (action === 'detail') {
    void openDetailDrawer(role);
    return;
  }

  void handleMoreAction(role);
}

function handleRoleRowAction(action: string, role: RoleListItem) {
  if (action === 'assign-permissions') {
    void openPermissionDrawer(role);
    return;
  }

  if (action === 'view-audit') {
    void router.push({
      path: '/audit/logs',
      query: {
        resourceType: 'role',
        resourceId: String(role.id),
      },
    });
    return;
  }

  handleRoleMoreAction({ value: action }, role);
}

async function openDetailDrawer(role: RoleListItem) {
  let detail: RoleDetailResponse = {
    ...role,
    created_at: role.updated_at,
  };
  try {
    detail = await getRoleDetail(role.id);
  } catch (error) {
    logger.warn('failed to load role detail, falling back to list item snapshot', error);
  }

  roleDrawerMode.value = 'detail';
  roleDrawerRole.value = detail;
  roleForm.value = {
    name: detail.name,
    display: detail.display,
    description: resolveRoleRemark(detail),
  };
  roleDrawerVisible.value = true;
}

function closeRoleDrawer() {
  roleDrawerVisible.value = false;
  roleDrawerRole.value = null;
  roleForm.value = { ...INITIAL_ROLE_FORM };
  roleFormRef.value?.clearValidate();
  submittingRole.value = false;
}

function setRoleFormFieldError(field: keyof RoleFormState, message: string) {
  roleFormRef.value?.setValidateMessage({
    [field]: [{ type: 'error', message }],
  } as FormValidateMessage<RoleFormState>);
}

async function handleRoleSubmit(ctx: SubmitContext) {
  if (ctx.validateResult !== true || submittingRole.value || roleDrawerMode.value === 'detail') {
    return;
  }

  submittingRole.value = true;
  try {
    if (roleDrawerMode.value === 'create') {
      const created = await createRole(toCreateRolePayload(roleForm.value));
      roles.value = [...roles.value, created].sort((left, right) => left.id - right.id);
      MessagePlugin.success(formatHintedMessage(t, t('rbac.roleList.createSuccess')));
    } else if (roleDrawerRole.value) {
      const updated = await updateRole(roleDrawerRole.value.id, toUpdateRolePayload(roleForm.value));
      roles.value = roles.value.map((item) => (item.id === updated.id ? updated : item));
      roleDrawerRole.value = updated;
      MessagePlugin.success(formatHintedMessage(t, t('rbac.roleList.updateSuccess')));
    }

    closeRoleDrawer();
  } catch (error) {
    logger.error('failed to submit role form', error);
    if (isApiRequestError(error)) {
      const errorMessage =
        localizedApiErrorMessage(t, error.messageKey, error.message) || t('rbac.roleList.submitFailed');
      const field = resolveRoleFormFieldError(error);
      if (field) {
        setRoleFormFieldError(field, errorMessage);
        return;
      }

      MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, errorMessage));
      return;
    }

    MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, t('rbac.roleList.submitFailed')));
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
    originalPermissionIds.value = [];
    selectedPermissionIds.value = [];
    permissionSelectionReady.value = false;
    return false;
  }

  originalPermissionIds.value = normalized;
  selectedPermissionIds.value = normalized;
  permissionSelectionReady.value = true;
  return true;
}

function extractPermissionIDs(response: RolePermissionBindingResponse & { permissionIds?: number[] }) {
  return response.permission_ids ?? response.permissionIds ?? [];
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

    if (!applyRolePermissionSelection(extractPermissionIDs(response))) {
      permissionLoadWarning.value = t('rbac.roleList.permissionDialog.selectionUnavailable');
      permissionLoadRetryable.value = false;
      return false;
    }

    return true;
  } catch (error) {
    if (!isActivePermissionDrawerSession(session)) {
      return false;
    }

    permissionLoadWarning.value = resolveLocalizedErrorMessage(
      t,
      error,
      t('rbac.roleList.permissionDialog.selectionLoadFailed'),
    );
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
  permissionMutationMode.value = 'replace';
  permissionKeyword.value = '';
  permissionOnlySelected.value = false;
  await loadRolePermissionSelection(role.id, session);
}

function closePermissionDrawer() {
  permissionDrawerSession.value += 1;
  permissionDrawerVisible.value = false;
  selectedRole.value = null;
  originalPermissionIds.value = [];
  selectedPermissionIds.value = [];
  permissionSelectionReady.value = false;
  loadingRolePermissions.value = false;
  permissionLoadWarning.value = '';
  permissionLoadRetryable.value = false;
  submittingPermissions.value = false;
  permissionMutationMode.value = 'replace';
  permissionKeyword.value = '';
  permissionOnlySelected.value = false;
}

function requestClosePermissionDrawer() {
  if (submittingPermissions.value) {
    return;
  }

  if (!hasPermissionSelectionChanges.value) {
    closePermissionDrawer();
    return;
  }

  const confirmed = window.confirm(t('rbac.roleList.permissionDialog.unsavedChangesConfirm'));
  if (confirmed) {
    closePermissionDrawer();
  }
}

async function retryPermissionDrawerLoad() {
  if (!selectedRole.value) {
    return;
  }

  await loadRolePermissionSelection(selectedRole.value.id, permissionDrawerSession.value);
}

function isPermissionCardDisabled(item: PermissionListItem) {
  const assigned = originalPermissionIds.value.includes(item.id);

  switch (permissionMutationMode.value) {
    case 'add':
      return assigned;
    case 'remove':
      return !assigned;
    default:
      return false;
  }
}

function toggleRolePermissionSelection(permissionId: number) {
  if (loadingRolePermissions.value || !permissionSelectionReady.value || !canAssignPermissions.value) {
    return;
  }

  if (selectedPermissionIds.value.includes(permissionId)) {
    selectedPermissionIds.value = selectedPermissionIds.value.filter((id) => id !== permissionId);
    return;
  }

  selectedPermissionIds.value = sortStableIDs([...selectedPermissionIds.value, permissionId]);
}

async function mutateRolePermissions(
  roleId: number,
  payload: ReplaceRolePermissionsPayload | RolePermissionMutationPayload,
) {
  switch (permissionMutationMode.value) {
    case 'add':
      return addRolePermissions(roleId, payload);
    case 'remove':
      return removeRolePermissions(roleId, payload);
    default:
      return replaceRolePermissions(roleId, payload);
  }
}

async function submitPermissionAssignment() {
  if (!selectedRole.value || !canSubmitPermissionAssignment.value || loadingRolePermissions.value) {
    return;
  }

  const session = permissionDrawerSession.value;
  const payload = permissionMutationPayload.value;

  permissionLoadWarning.value = '';
  permissionLoadRetryable.value = false;
  submittingPermissions.value = true;
  try {
    await mutateRolePermissions(selectedRole.value.id, payload);

    if (!isActivePermissionDrawerSession(session)) {
      return;
    }

    MessagePlugin.success(formatHintedMessage(t, t('rbac.roleList.assignSuccess')));
    closePermissionDrawer();
    await fetchRolePageData();
  } catch (error) {
    if (isActivePermissionDrawerSession(session)) {
      if (isApiRequestError(error)) {
        const errorMessage =
          localizedApiErrorMessage(t, error.messageKey, error.message) || t('rbac.roleList.assignFailed');
        const field = resolveRolePermissionFieldError(error);

        if (field === 'permission_ids' || error.status === 404) {
          permissionLoadWarning.value = errorMessage;
          permissionLoadRetryable.value = false;
          return;
        }

        MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, errorMessage));
        return;
      }

      MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, t('rbac.roleList.assignFailed')));
    }
  } finally {
    if (permissionDrawerSession.value === session) {
      submittingPermissions.value = false;
    }
  }
}

async function handleMoreAction(role: RoleListItem) {
  if (role.builtin) {
    MessagePlugin.warning(t('rbac.roleList.moreBuiltinHint'));
    return;
  }

  MessagePlugin.warning(t('rbac.roleList.moreCustomHint'));
}

async function toggleRoleStatus(role: RoleStatusCompat) {
  if (!canToggleRoleStatus.value || role.builtin) {
    return;
  }

  try {
    const updated = await updateRoleStatus(role.id, {
      status: isRoleEnabled(role) ? 'disabled' : 'enabled',
    });
    roles.value = roles.value.map((item) => (item.id === updated.id ? updated : item));
    MessagePlugin.success(
      formatHintedMessage(
        t,
        isRoleEnabled(updated) ? t('rbac.roleList.statusEnabledSuccess') : t('rbac.roleList.statusDisabledSuccess'),
      ),
    );
  } catch (error) {
    logger.error('failed to update role status', error);
    if (isApiRequestError(error)) {
      MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, t('rbac.roleList.statusUpdateFailed')));
      return;
    }

    MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, t('rbac.roleList.statusUpdateFailed')));
  }
}

async function removeRole(role: RoleStatusCompat) {
  if (!canDeleteRoles.value || role.builtin) {
    return;
  }
  if (isRoleEnabled(role) || roleHasDeleteBlockingBindings(role)) {
    MessagePlugin.warning(roleDeleteLifecycleHint(role));
    return;
  }

  try {
    await deleteRole(role.id);
    roles.value = roles.value.filter((item) => item.id !== role.id);
    MessagePlugin.success(formatHintedMessage(t, t('rbac.roleList.deleteSuccess')));
  } catch (error) {
    logger.error('failed to delete role', error);
    if (isApiRequestError(error)) {
      MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, t('rbac.roleList.deleteFailed')));
      return;
    }

    MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, t('rbac.roleList.deleteFailed')));
  }
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
