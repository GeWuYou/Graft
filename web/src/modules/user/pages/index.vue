<template>
  <div class="user-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header :title="t('user.userList.listTitle')" :description="t('user.userList.hint')">
        <template #eyebrow>{{ t('menu.access_control.title') }}</template>
        <template #actions>
          <t-button theme="default" variant="outline" :loading="loading" data-testid="user-refresh" @click="fetchUsers">
            {{ t('user.userList.refresh') }}
          </t-button>
          <t-button
            v-permission="userPermissionCodes.CREATE"
            theme="primary"
            data-testid="user-create"
            @click="openUserDrawer('create')"
          >
            {{ t('user.userList.create') }}
          </t-button>
        </template>
      </management-page-header>

      <management-toolbar>
        <template #filters>
          <t-input
            v-model="filters.keyword"
            clearable
            class="toolbar__search"
            :placeholder="t('user.userList.toolbar.searchPlaceholder')"
          />
          <t-select
            v-model="filters.status"
            clearable
            class="toolbar__select"
            :options="statusOptions"
            :placeholder="t('user.userList.toolbar.statusPlaceholder')"
          />
          <t-select
            v-model="filters.roleId"
            clearable
            class="toolbar__select"
            :options="roleOptions"
            :loading="roleCatalogLoading"
            :placeholder="t('user.userList.toolbar.rolePlaceholder')"
          />
          <t-button theme="default" variant="text" @click="resetFilters">
            {{ t('user.userList.toolbar.clearFilters') }}
          </t-button>
        </template>
        <template #actions>
          <t-button theme="default" variant="outline" @click="columnDrawerVisible = true">
            {{ t('user.userList.columnSettings') }}
          </t-button>
        </template>
      </management-toolbar>

      <management-table-card>
        <template #head>
          <div class="table-head">
            <div>
              <p class="table-head__summary">{{ t('user.userList.summary', { count: filteredUsers.length }) }}</p>
              <p class="table-head__description">{{ t('user.userList.tableHint') }}</p>
            </div>
            <t-button v-if="hasActiveFilters" theme="default" variant="text" @click="resetFilters">
              {{ t('user.userList.toolbar.clearFilters') }}
            </t-button>
          </div>
        </template>

        <template #batch>
          <div v-if="selectedRowKeys.length > 0" class="batch-bar">
            <span>{{ t('user.userList.batch.selected', { count: selectedRowKeys.length }) }}</span>
            <div class="batch-bar__actions">
              <t-button size="small" variant="outline" disabled>{{ t('user.userList.batch.enable') }}</t-button>
              <t-button size="small" variant="outline" disabled>{{ t('user.userList.batch.disable') }}</t-button>
              <t-button
                v-permission="{ allOf: userRoleManagePermissionCodes }"
                size="small"
                theme="primary"
                variant="outline"
                data-testid="user-batch-manage-roles"
                @click="openBatchUserRoleDrawer"
              >
                {{ t('user.userList.batch.assignRoles') }}
              </t-button>
              <t-button size="small" theme="default" variant="text" @click="selectedRowKeys = []">
                {{ t('user.userList.batch.cancelSelection') }}
              </t-button>
            </div>
          </div>
        </template>

        <management-empty-state
          v-if="listError && !loading"
          tone="error"
          :title="t('user.userList.errorTitle')"
          :description="listError"
        >
          <template #actions>
            <t-button theme="primary" variant="outline" @click="fetchUsers">
              {{ t('user.userList.retry') }}
            </t-button>
          </template>
        </management-empty-state>

        <t-table
          v-else
          row-key="id"
          :data="pagedUsers"
          :columns="visibleColumns"
          :loading="loading"
          table-layout="fixed"
          :table-content-width="tableContentWidth"
          :selected-row-keys="selectedRowKeys"
          cell-empty-content="-"
          @select-change="handleSelectChange"
        >
          <template #user="{ row }">
            <div class="user-cell">
              <div class="user-cell__avatar">{{ userInitial(row.display || row.username) }}</div>
              <div class="user-cell__meta">
                <span class="user-cell__display">{{ row.display || row.username }}</span>
                <span class="user-cell__username">{{ row.email || `@${row.username}` }}</span>
              </div>
            </div>
          </template>

          <template #status="{ row }">
            <t-tag :theme="statusTheme(row.status)" variant="light">
              {{ statusLabel(row.status) }}
            </t-tag>
          </template>

          <template #roles="{ row }">
            <div class="role-tag-list">
              <template v-if="(row.roles ?? []).length > 0">
                <t-tag
                  v-for="role in (row.roles ?? []).slice(0, 2)"
                  :key="role.id"
                  theme="default"
                  variant="light-outline"
                  size="small"
                >
                  {{ role.display }}
                </t-tag>
                <t-tag v-if="(row.roles ?? []).length > 2" theme="default" variant="light-outline" size="small">
                  +{{ (row.roles ?? []).length - 2 }}
                </t-tag>
              </template>
              <span v-else class="table-muted">{{ t('user.userList.roleSummary.empty') }}</span>
            </div>
          </template>

          <template #last_login_at="{ row }">
            <span>{{ formatTimestamp(row.last_login_at) }}</span>
          </template>

          <template #created_at="{ row }">
            <span>{{ formatTimestamp(row.created_at) }}</span>
          </template>

          <template #updated_at="{ row }">
            <span>{{ formatTimestamp(row.updated_at) }}</span>
          </template>

          <template #operation="{ row }">
            <table-action-menu
              :actions="userRowActions(row)"
              :more-label="t('user.userList.more')"
              more-label-fallback="更多"
              @action="(action) => handleUserRowAction(action, row)"
            />
          </template>

          <template #empty>
            <div class="table-empty-state">
              <t-empty :title="t('user.userList.emptyTitle')" :description="t('user.userList.emptyDescription')">
                <template #action>
                  <div class="table-empty-state__actions">
                    <t-button
                      v-if="hasActiveFilters"
                      theme="default"
                      variant="outline"
                      data-testid="user-empty-clear-filters"
                      @click="resetFilters"
                    >
                      {{ t('user.userList.toolbar.clearFilters') }}
                    </t-button>
                    <t-button
                      v-permission="userPermissionCodes.CREATE"
                      theme="primary"
                      data-testid="user-empty-create"
                      @click="openUserDrawer('create')"
                    >
                      {{ t('user.userList.create') }}
                    </t-button>
                  </div>
                </template>
              </t-empty>
            </div>
          </template>
        </t-table>

        <template #footer>
          <management-table-pagination :summary="t('user.userList.footerTotal', { count: filteredUsers.length })">
            <t-pagination
              v-model:current="pagination.current"
              v-model:page-size="pagination.pageSize"
              :total="filteredUsers.length"
              :page-size-options="[10, 20, 50]"
              :show-page-number="true"
            />
          </management-table-pagination>
        </template>
      </management-table-card>
    </management-page-content>

    <t-drawer
      v-model:visible="userDrawerVisible"
      :header="userDrawerMode === 'create' ? t('user.userList.form.createTitle') : t('user.userList.form.editTitle')"
      size="520px"
      placement="right"
      destroy-on-close
    >
      <div class="drawer-panel">
        <t-form ref="userFormRef" :data="userForm" :rules="userFormRules" label-align="top" @submit="handleUserSubmit">
          <t-form-item :label="t('user.userList.form.username')" name="username">
            <t-input v-model="userForm.username" :placeholder="t('user.userList.form.usernamePlaceholder')" />
          </t-form-item>
          <t-form-item :label="t('user.userList.form.display')" name="display">
            <t-input v-model="userForm.display" :placeholder="t('user.userList.form.displayPlaceholder')" />
          </t-form-item>
          <t-form-item
            v-if="userDrawerMode === 'create'"
            :label="t('user.userList.form.password')"
            :tips="passwordFieldError ? '' : t('user.userList.form.passwordPolicy.hint')"
            name="password"
          >
            <t-input
              v-model="userForm.password"
              type="password"
              :placeholder="t('user.userList.form.passwordPlaceholder')"
            />
          </t-form-item>
          <div class="drawer-actions">
            <t-button variant="outline" @click="closeUserDrawer">
              {{ t('user.userList.form.cancel') }}
            </t-button>
            <t-button theme="primary" type="submit" :loading="submittingUser">
              {{ t('user.userList.form.confirm') }}
            </t-button>
          </div>
        </t-form>
      </div>
    </t-drawer>

    <t-drawer
      v-model:visible="detailDrawerVisible"
      :header="t('user.userList.detailTitle')"
      size="520px"
      placement="right"
      destroy-on-close
    >
      <div v-if="detailUser" class="drawer-panel user-detail-panel">
        <div class="detail-header">
          <div class="user-cell">
            <div class="user-cell__avatar">{{ userInitial(detailUser.display || detailUser.username) }}</div>
            <div class="user-cell__meta">
              <span class="user-cell__display">{{ detailUser.display || detailUser.username }}</span>
              <span class="user-cell__username">@{{ detailUser.username }}</span>
            </div>
          </div>
          <t-tag :theme="statusTheme(detailUser.status)" variant="light">
            {{ statusLabel(detailUser.status) }}
          </t-tag>
        </div>

        <div class="detail-grid">
          <div class="detail-item">
            <span class="detail-item__label">{{ t('user.userList.detailFields.email') }}</span>
            <span class="detail-item__value">{{ detailUser.email || '-' }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-item__label">{{ t('user.userList.columns.roles') }}</span>
            <span class="detail-item__value">{{ roleSummaryText(detailUser) }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-item__label">{{ t('user.userList.columns.lastLoginAt') }}</span>
            <span class="detail-item__value">{{ formatTimestamp(detailUser.last_login_at) }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-item__label">{{ t('user.userList.columns.createdAt') }}</span>
            <span class="detail-item__value">{{ formatTimestamp(detailUser.created_at) }}</span>
          </div>
          <div class="detail-item">
            <span class="detail-item__label">{{ t('user.userList.columns.updatedAt') }}</span>
            <span class="detail-item__value">{{ formatTimestamp(detailUser.updated_at) }}</span>
          </div>
        </div>
      </div>
    </t-drawer>

    <t-dialog
      v-model:visible="resetPasswordDialogVisible"
      :header="t('user.userList.resetPasswordDialog.title')"
      :confirm-btn="{ loading: submittingResetPassword, content: t('user.userList.resetPasswordDialog.confirm') }"
      :cancel-btn="t('user.userList.resetPasswordDialog.cancel')"
      @confirm="submitResetPassword"
      @close="closeResetPasswordDialog"
    >
      <div class="drawer-panel">
        <p class="table-head__description">
          {{
            t('user.userList.resetPasswordDialog.description', {
              user: resetPasswordTarget?.display || resetPasswordTarget?.username || '-',
            })
          }}
        </p>
        <t-form ref="resetPasswordFormRef" :data="resetPasswordForm" :rules="resetPasswordFormRules" label-align="top">
          <t-form-item :label="t('user.userList.resetPasswordDialog.password')" name="password">
            <t-input
              v-model="resetPasswordForm.password"
              type="password"
              :placeholder="t('user.userList.resetPasswordDialog.passwordPlaceholder')"
            />
          </t-form-item>
        </t-form>
      </div>
    </t-dialog>

    <assignment-drawer
      v-model:visible="userRoleDrawerVisible"
      :title="
        roleDialogMode === 'batch' ? t('user.userList.roleDialog.batchTitle') : t('user.userList.roleDialog.title')
      "
      size="760px"
      @close="requestCloseUserRoleDrawer"
    >
      <template #header>
        <div class="assignment-panel assignment-panel--compact" data-testid="user-role-drawer">
          <assignment-header
            :avatar-text="userAssignmentAvatar"
            :badges="userAssignmentBadges"
            :description="userAssignmentDescription"
            :eyebrow="t('user.userList.roleDialog.headerEyebrow')"
            :stats="userAssignmentStats"
            :subtitle="userAssignmentSubtitle"
            :title="userAssignmentTitle"
          />

          <assignment-toolbar
            v-model:mode-value="roleMutationMode"
            v-model:search-value="roleSearchKeyword"
            :disabled="submittingRoles || loadingRoleDialogData"
            :mode-label="t('user.userList.roleDialog.saveStrategyLabel')"
            :mode-options="roleMutationOptions"
            :search-placeholder="t('user.userList.roleDialog.searchPlaceholder')"
          />

          <assignment-summary
            :hint="userAssignmentHint"
            :hint-test-id="roleDialogMode === 'batch' ? 'batch-role-operation-hint' : ''"
            :items="userAssignmentSummaryItems"
            :warning="roleLoadWarning"
            :warning-action-label="roleLoadWarning ? t('user.userList.roleDialog.retry') : ''"
            :warning-action-loading="loadingRoleDialogData"
            @warning-action="retryUserRoleDrawerLoad"
          />
        </div>
      </template>

      <assignment-grid
        :empty="filteredAssignableRoles.length === 0 && !roleCatalogLoading"
        :empty-description="t('user.userList.roleDialog.empty')"
      >
        <t-checkbox-group
          v-model="selectedRoleIds"
          class="sr-only"
          :disabled="loadingRoleDialogData || !roleSelectionReady"
          data-testid="role-checkbox-group"
        />
        <div class="assignment-card-grid permission-card-grid">
          <assignment-card
            v-for="role in filteredAssignableRoles"
            :key="role.id"
            :assigned="currentRoleIds.includes(role.id)"
            :assigned-label="t('user.userList.roleDialog.assignedBadge')"
            :code="role.name"
            :description="role.description || t('user.userList.roleDialog.emptyDescription')"
            :disabled="loadingRoleDialogData || !roleSelectionReady"
            :selected="selectedRoleIds.includes(role.id)"
            :tags="[
              {
                label: role.builtin
                  ? t('user.userList.roleDialog.builtinYes')
                  : t('user.userList.roleDialog.builtinNo'),
                theme: role.builtin ? 'warning' : 'default',
              },
            ]"
            :title="role.display"
            @toggle="toggleUserRoleSelection(role.id)"
          />
        </div>
      </assignment-grid>

      <template #footer>
        <assignment-footer
          :cancel-label="t('user.userList.roleDialog.cancel')"
          cancel-test-id="user-role-cancel"
          :confirm-disabled="!canSubmitRoleAssignment"
          :confirm-label="t('user.userList.roleDialog.confirm')"
          :confirm-loading="submittingRoles"
          :details="userAssignmentFooterDetails"
          confirm-test-id="user-role-save"
          :summary="userAssignmentFooterSummary"
          @cancel="requestCloseUserRoleDrawer"
          @confirm="submitUserRoleAssignment"
        />
      </template>
    </assignment-drawer>

    <t-drawer
      v-model:visible="columnDrawerVisible"
      :header="t('user.userList.columnSettings')"
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
import { computed, nextTick, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { RBAC_PERMISSION_CODE } from '@/modules/rbac/contract/permissions';
import { localizedApiErrorMessage } from '@/modules/shared/localized-api-error';
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

import { getRoles, getUserRoleBindings, mutateBatchUserRoles, mutateUserRoles } from '../api/user-roles';
import { createUser, deleteUser, getUsers, resetUserPassword, updateUser, updateUserStatus } from '../api/users';
import { USER_PERMISSION_CODE } from '../contract/permissions';
import type { UserStatus } from '../contract/status';
import { USER_STATUS } from '../contract/status';
import { resolveResetPasswordFieldError, resolveUserFormFieldError } from '../error-adapter';
import { evaluateUserPasswordPolicy } from '../shared/password-policy';
import type { BatchUserRoleMutationPayload, RoleListItem, UserRoleMutation } from '../types/role';
import type {
  CreateUserPayload,
  ResetUserPasswordPayload,
  UpdateUserPayload,
  UserListItem,
  UserRoleSummary,
} from '../types/user';

defineOptions({
  name: 'UsersIndex',
});

const logger = createLogger('user.userList');

type UserFilters = {
  keyword: string;
  roleId: number | undefined;
  status: '' | UserStatus;
};

type UserRow = UserListItem & {
  // The current page still tolerates legacy read fields when the backend includes them.
  email?: string | null;
  last_login_at?: string | null;
};

type UserDrawerMode = 'create' | 'edit';

type UserFormState = {
  username: string;
  display: string;
  password: string;
};

type UserFormInstance = {
  clearValidate: (fields?: Array<keyof UserFormState>) => void;
  setValidateMessage: (message: FormValidateMessage<UserFormState>) => void;
};

type ResetPasswordFormInstance = {
  clearValidate: (fields?: Array<'password'>) => void;
  setValidateMessage: (message: FormValidateMessage<{ password: string }>) => void;
};

const INITIAL_USER_FORM: UserFormState = {
  username: '',
  display: '',
  password: '',
};

const DEFAULT_VISIBLE_COLUMNS = ['row-select', 'user', 'status', 'roles', 'last_login_at', 'updated_at', 'operation'];

const { t, locale } = useI18n();
const route = useRoute();
const router = useRouter();
const permissionStore = usePermissionStore();
const users = ref<UserRow[]>([]);
const roles = ref<RoleListItem[]>([]);
const loading = ref(false);
const listError = ref('');
const roleCatalogLoading = ref(false);
const userDrawerVisible = ref(false);
const userDrawerMode = ref<UserDrawerMode>('create');
const userDrawerTarget = ref<UserRow | null>(null);
const userFormRef = ref<UserFormInstance | null>(null);
const userForm = ref<UserFormState>({ ...INITIAL_USER_FORM });
const passwordFieldError = ref('');
const submittingUser = ref(false);
const resetPasswordDialogVisible = ref(false);
const resetPasswordTarget = ref<UserRow | null>(null);
const detailDrawerVisible = ref(false);
const detailUser = ref<UserRow | null>(null);
const resetPasswordFormRef = ref<ResetPasswordFormInstance | null>(null);
const resetPasswordForm = ref({
  password: '',
});
const submittingResetPassword = ref(false);
const filters = ref<UserFilters>({
  keyword: '',
  roleId: undefined,
  status: '',
});
const visibleColumnKeys = ref<string[]>([...DEFAULT_VISIBLE_COLUMNS]);
const columnDrawerVisible = ref(false);
const userRoleDrawerVisible = ref(false);
const selectedUser = ref<UserRow | null>(null);
const currentRoleIds = ref<number[]>([]);
const roleDialogMode = ref<'single' | 'batch'>('single');
const roleMutationMode = ref<UserRoleMutation>('replace');
const roleSearchKeyword = ref('');
const loadingRoleSelection = ref(false);
const submittingRoles = ref(false);
const roleSelectionReady = ref(false);
const roleLoadWarning = ref('');
const drawerSession = ref(0);
const selectedRowKeys = ref<Array<string | number>>([]);
const pagination = ref({
  current: 1,
  pageSize: 10,
});

const userPermissionCodes = USER_PERMISSION_CODE;
const rbacPermissionCodes = RBAC_PERMISSION_CODE;
const userRoleManagePermissionCodes = [rbacPermissionCodes.USER_ROLE_READ, rbacPermissionCodes.USER_ROLE_ASSIGN];
const loadingRoleDialogData = computed(() => roleCatalogLoading.value || loadingRoleSelection.value);
const roleMutationPayload = computed(() => {
  return {
    role_ids: [...selectedRoleIds.value].sort((left, right) => left - right),
  };
});
const hasUserRoleSelectionChanges = computed(() => {
  if (!roleSelectionReady.value) {
    return false;
  }

  if (roleDialogMode.value === 'batch') {
    return roleMutationPayload.value.role_ids.length > 0;
  }

  if (selectedUser.value === null) {
    return false;
  }

  switch (roleMutationMode.value) {
    case 'replace':
      return !arePermissionIDsEqual(currentRoleIds.value, selectedRoleIds.value);
    case 'add':
      return selectedRoleIds.value.some((id) => !currentRoleIds.value.includes(id));
    case 'remove':
      return selectedRoleIds.value.some((id) => currentRoleIds.value.includes(id));
    default:
      return false;
  }
});
const canSubmitRoleAssignment = computed(
  () =>
    canManageUserRoles() &&
    hasUserRoleSelectionChanges.value &&
    (roleDialogMode.value === 'batch' ? selectedRowKeys.value.length > 0 : selectedUser.value !== null) &&
    (roleMutationMode.value === 'replace' || roleMutationPayload.value.role_ids.length > 0),
);
const hasActiveFilters = computed(
  () => Boolean(filters.value.keyword.trim()) || Boolean(filters.value.status) || filters.value.roleId !== undefined,
);
const selectedBatchUserIds = computed(() =>
  selectedRowKeys.value.map((item) => Number(item)).filter((item) => Number.isInteger(item)),
);
const selectedBatchUsers = computed(() => {
  const ids = new Set(selectedBatchUserIds.value);
  return users.value.filter((item) => ids.has(item.id));
});

const statusOptions = computed(() => [
  { label: t('user.userList.toolbar.statusAll'), value: '' },
  { label: t('user.userList.status.enabled'), value: USER_STATUS.ENABLED },
  { label: t('user.userList.status.disabled'), value: USER_STATUS.DISABLED },
]);

const roleOptions = computed(() =>
  roles.value.map((role) => ({
    label: role.display,
    value: role.id,
  })),
);
const roleMutationOptions = computed(() => [
  { label: t('user.userList.roleActions.replace'), value: 'replace' },
  { label: t('user.userList.roleActions.add'), value: 'add' },
  { label: t('user.userList.roleActions.remove'), value: 'remove' },
]);
const batchRoleOperationHint = computed(() => {
  if (roleDialogMode.value !== 'batch') {
    return '';
  }

  if (roleMutationMode.value === 'add') {
    return t('user.userList.roleDialog.batchOperationHint.add');
  }

  if (roleMutationMode.value === 'remove') {
    return t('user.userList.roleDialog.batchOperationHint.remove');
  }

  if (selectedRoleIds.value.length === 0) {
    return t('user.userList.roleDialog.batchOperationHint.replaceEmpty', {
      count: selectedBatchUserIds.value.length,
    });
  }

  return t('user.userList.roleDialog.batchOperationHint.replace');
});

const columnSettingOptions = computed(() => [
  { label: t('user.userList.columns.user'), value: 'user' },
  { label: t('user.userList.columns.status'), value: 'status' },
  { label: t('user.userList.columns.roles'), value: 'roles' },
  { label: t('user.userList.columns.lastLoginAt'), value: 'last_login_at' },
  { label: t('user.userList.columns.createdAt'), value: 'created_at' },
  { label: t('user.userList.columns.updatedAt'), value: 'updated_at' },
  { label: t('components.commonTable.operation'), value: 'operation' },
]);

const filteredUsers = computed(() => {
  const keyword = filters.value.keyword.trim().toLowerCase();

  return users.value.filter((user) => {
    if (keyword) {
      const haystack = `${user.username} ${user.display} ${user.email ?? ''}`.toLowerCase();
      if (!haystack.includes(keyword)) {
        return false;
      }
    }

    if (filters.value.status && normalizeUserStatus(user.status) !== filters.value.status) {
      return false;
    }

    if (filters.value.roleId !== undefined) {
      const assignedRoleIds = user.roles.map((role) => role.id);
      if (!assignedRoleIds.includes(filters.value.roleId)) {
        return false;
      }
    }

    return true;
  });
});

const pagedUsers = computed(() => {
  const start = (pagination.value.current - 1) * pagination.value.pageSize;
  return filteredUsers.value.slice(start, start + pagination.value.pageSize);
});

const { selectedIds: selectedRoleIdsInternal, resetSelection: resetRoleSelection } = useAssignmentSelection({
  active: userRoleDrawerVisible,
  mode: roleMutationMode,
  originalIds: currentRoleIds,
});
const selectedRoleIds = selectedRoleIdsInternal;
const filteredAssignableRoles = computed(() => {
  const keyword = roleSearchKeyword.value.trim().toLowerCase();

  return roles.value.filter((role) => {
    if (!keyword) {
      return true;
    }

    return `${role.display} ${role.name} ${role.description ?? ''}`.toLowerCase().includes(keyword);
  });
});
const userAssignmentTitle = computed(() =>
  roleDialogMode.value === 'batch'
    ? t('user.userList.roleDialog.batchSummary', { count: selectedBatchUserIds.value.length })
    : selectedUser.value?.display || '-',
);
const userAssignmentSubtitle = computed(() =>
  roleDialogMode.value === 'batch'
    ? selectedBatchUsers.value.map((item) => `@${item.username}`).join(', ')
    : `@${selectedUser.value?.username || '-'}`,
);
const userAssignmentDescription = computed(() =>
  roleDialogMode.value === 'batch'
    ? t('user.userList.roleDialog.batchDescription')
    : t('user.userList.roleDialog.singleDescription'),
);
const userAssignmentAvatar = computed(() =>
  roleDialogMode.value === 'batch'
    ? String(selectedBatchUserIds.value.length)
    : userInitial(selectedUser.value?.display || selectedUser.value?.username),
);
const userAssignmentBadges = computed(() =>
  roleDialogMode.value === 'batch'
    ? [{ label: t('user.userList.roleDialog.batchBadge'), theme: 'primary' as const }]
    : [
        {
          label:
            normalizeUserStatus(selectedUser.value?.status) === USER_STATUS.DISABLED
              ? t('user.userList.status.disabled')
              : t('user.userList.status.enabled'),
          theme: statusTheme(selectedUser.value?.status) as 'danger' | 'success',
        },
      ],
);
const userAssignmentStats = computed(() => [
  {
    label: t('user.userList.roleDialog.stats.availableRoles'),
    value: roles.value.length,
  },
  {
    label: t('user.userList.roleDialog.stats.assignedRoles'),
    value: currentRoleIds.value.length,
  },
]);
const userAssignmentSummaryItems = computed(() => [
  {
    label:
      roleDialogMode.value === 'batch'
        ? t('user.userList.roleDialog.summary.selectedUsers')
        : t('user.userList.roleDialog.summary.createdAt'),
    value:
      roleDialogMode.value === 'batch'
        ? selectedBatchUsers.value.length
        : formatTimestamp(selectedUser.value?.created_at),
  },
  {
    label:
      roleDialogMode.value === 'batch'
        ? t('user.userList.roleDialog.summary.currentSelection')
        : t('user.userList.roleDialog.summary.updatedAt'),
    value:
      roleDialogMode.value === 'batch' ? selectedRoleIds.value.length : formatTimestamp(selectedUser.value?.updated_at),
  },
]);
const userAssignmentHint = computed(() =>
  roleDialogMode.value === 'batch'
    ? batchRoleOperationHint.value
    : t('user.userList.roleDialog.inlineHint', {
        assigned: currentRoleIds.value.length,
        total: roles.value.length,
      }),
);
const userRoleAddedCount = computed(() => {
  const current = new Set(currentRoleIds.value);
  return selectedRoleIds.value.filter((id) => !current.has(id)).length;
});
const userRoleRemovedCount = computed(() => {
  const selected = new Set(selectedRoleIds.value);
  return currentRoleIds.value.filter((id) => !selected.has(id)).length;
});
const userAssignmentFooterSummary = computed(() =>
  t('user.userList.roleDialog.selectionCount', {
    selected: selectedRoleIds.value.length,
    total: roles.value.length,
  }),
);
const userAssignmentFooterDetails = computed(() => {
  const details = [
    t('user.userList.roleDialog.modeSummary', {
      mode: t(`user.userList.roleDialog.modeValue.${roleMutationMode.value}`),
    }),
  ];

  if (roleDialogMode.value === 'batch') {
    details.push(
      t(`user.userList.roleDialog.${roleMutationMode.value}SelectionCount`, {
        count: roleMutationPayload.value.role_ids.length,
      }),
    );
    return details;
  }

  if (roleMutationMode.value === 'replace') {
    if (userRoleAddedCount.value > 0) {
      details.push(
        t('user.userList.roleDialog.addSelectionCount', {
          count: userRoleAddedCount.value,
        }),
      );
    }

    if (userRoleRemovedCount.value > 0) {
      details.push(
        t('user.userList.roleDialog.removeSelectionCount', {
          count: userRoleRemovedCount.value,
        }),
      );
    }

    return details;
  }

  details.push(
    t(`user.userList.roleDialog.${roleMutationMode.value}SelectionCount`, {
      count: roleMutationPayload.value.role_ids.length,
    }),
  );

  return details;
});

const userRowMoreOptions = (user: UserRow) => {
  const options: Array<{ content: string; fallbackLabel: string; value: string }> = [];

  if (permissionStore.hasPermission(userPermissionCodes.UPDATE)) {
    options.push({
      content: t('user.userList.edit'),
      fallbackLabel: '编辑',
      value: 'edit',
    });
  }

  if (permissionStore.hasPermission(userPermissionCodes.DISABLE)) {
    options.push({
      content:
        normalizeUserStatus(user.status) === USER_STATUS.DISABLED
          ? t('user.userList.moreActions.enable')
          : t('user.userList.moreActions.disable'),
      fallbackLabel: normalizeUserStatus(user.status) === USER_STATUS.DISABLED ? '启用用户' : '禁用用户',
      value: 'toggle-status',
    });
  }

  if (permissionStore.hasPermission(userPermissionCodes.UPDATE)) {
    options.push({
      content: t('user.userList.moreActions.resetPassword'),
      fallbackLabel: '重置密码',
      value: 'reset-password',
    });
  }

  if (permissionStore.hasPermission(userPermissionCodes.DISABLE)) {
    options.push({
      content: t('user.userList.moreActions.delete'),
      fallbackLabel: '删除用户',
      value: 'delete',
    });
  }

  return options;
};

const userFormRules = computed<Record<keyof UserFormState, FormRule[]>>(() => ({
  username: [{ required: true, message: t('user.userList.form.required.username'), type: 'error' }],
  display: [{ required: true, message: t('user.userList.form.required.display'), type: 'error' }],
  password:
    userDrawerMode.value === 'create'
      ? [
          {
            type: 'error',
            validator: (value) => {
              const errorMessage = resolveCreatePasswordError(typeof value === 'string' ? value : '', true);
              passwordFieldError.value = errorMessage;
              if (!errorMessage) {
                return true;
              }

              return {
                result: false,
                message: errorMessage,
                type: 'error',
              };
            },
          },
        ]
      : [],
}));

const resetPasswordFormRules = computed<Record<'password', FormRule[]>>(() => ({
  password: [{ required: true, message: t('user.userList.resetPasswordDialog.required'), type: 'error' }],
}));

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;

  const baseColumns = [
    {
      colKey: 'row-select',
      type: 'multiple',
      width: 48,
      fixed: 'left' as const,
      align: 'center' as const,
    },
    createTextColumn(t('user.userList.columns.user'), 'user', {
      width: 320,
      fixed: 'left',
    }),
    createStatusColumn(t('user.userList.columns.status'), 'status', 96),
    createTextColumn(t('user.userList.columns.roles'), 'roles', {
      align: 'center',
      width: 148,
    }),
    createTimeColumn(t('user.userList.columns.lastLoginAt'), 'last_login_at', 152),
    createTimeColumn(t('user.userList.columns.createdAt'), 'created_at', 160),
    createTimeColumn(t('user.userList.columns.updatedAt'), 'updated_at', 160),
  ];

  const allColumns = hasVisibleUserOperationActions()
    ? [...baseColumns, createActionColumn(t('components.commonTable.operation'), 160)]
    : baseColumns;

  return buildVisibleColumns(allColumns, visibleColumnKeys.value) as TdBaseTableProps['columns'];
});

const visibleColumns = computed(() => {
  if (hasVisibleUserOperationActions()) {
    return columns.value;
  }

  return (columns.value ?? []).filter((column) => column?.colKey !== 'operation');
});

const tableContentWidth = computed(() => calculateTableContentWidth(visibleColumns.value));

async function fetchUsers() {
  loading.value = true;
  listError.value = '';

  try {
    const response = await getUsers();
    users.value = response.items;
    selectedRowKeys.value = [];
    pagination.value.current = 1;
  } catch (error) {
    users.value = [];
    logger.error('failed to fetch users', error);
    listError.value = t('user.userList.loadFailed');
    MessagePlugin.error(listError.value);
  } finally {
    loading.value = false;
  }
}

async function loadRoleCatalog() {
  roleCatalogLoading.value = true;

  try {
    const response = await getRoles();
    roles.value = response?.items ?? [];
  } finally {
    roleCatalogLoading.value = false;
  }
}

function resetFilters() {
  filters.value = {
    keyword: '',
    roleId: undefined,
    status: '',
  };
  pagination.value.current = 1;
}

function formatTimestamp(value?: string | null) {
  return formatCompactDateTime(value);
}

function userInitial(value?: string | null) {
  if (!value) {
    return '?';
  }

  return value.trim().slice(0, 1).toUpperCase();
}

function roleSummaryText(user: UserRow) {
  if ((user.roles ?? []).length === 0) {
    return t('user.userList.roleSummary.empty');
  }

  return user.roles.map((role) => role.display).join(' / ');
}

function normalizeUserStatus(status?: string | null) {
  return status === USER_STATUS.DISABLED ? USER_STATUS.DISABLED : USER_STATUS.ENABLED;
}

function statusLabel(status?: string | null) {
  return normalizeUserStatus(status) === USER_STATUS.DISABLED
    ? t('user.userList.status.disabled')
    : t('user.userList.status.enabled');
}

function statusTheme(status?: string | null) {
  return normalizeUserStatus(status) === USER_STATUS.DISABLED ? 'danger' : 'success';
}

function canManageUserRoles() {
  return permissionStore.hasAllPermissions(userRoleManagePermissionCodes);
}

function hasVisibleUserOperationActions() {
  return (
    canManageUserRoles() ||
    permissionStore.hasPermission(userPermissionCodes.UPDATE) ||
    permissionStore.hasPermission(userPermissionCodes.DISABLE)
  );
}

function userRowActions(user: UserRow) {
  const actions: Array<{ disabled?: boolean; fallbackLabel?: string; label: string; testId?: string; value: string }> =
    [];

  if (canManageUserRoles()) {
    actions.push({
      label: t('user.userList.assignRoles'),
      testId: 'user-manage-roles',
      value: 'manage-roles',
    });
  }

  actions.push({
    fallbackLabel: '详情',
    label: t('components.commonTable.detail'),
    testId: 'user-detail',
    value: 'detail',
  });

  if (permissionStore.hasPermission('audit.read')) {
    actions.push({
      fallbackLabel: '查看审计',
      label: t('user.userList.viewAudit'),
      testId: 'user-view-audit',
      value: 'view-audit',
    });
  }

  userRowMoreOptions(user).forEach((option) => {
    actions.push({
      disabled: false,
      fallbackLabel: option.fallbackLabel,
      label: option.content,
      testId: option.value === 'edit' ? 'user-edit' : undefined,
      value: option.value,
    });
  });

  return actions;
}

function handleUserRowAction(action: string, user: UserRow) {
  if (action === 'detail') {
    detailUser.value = user;
    detailDrawerVisible.value = true;
    return;
  }

  if (action === 'manage-roles') {
    void handleOpenUserRoleDrawer(user);
    return;
  }

  if (action === 'view-audit') {
    void router.push({
      path: '/audit/logs',
      query: {
        resourceType: 'user',
        resourceId: String(user.id),
      },
    });
    return;
  }

  handleUserMoreAction({ value: action }, user);
}

function ensureUserPermission(allowed: boolean, message: string) {
  if (allowed) {
    return true;
  }

  MessagePlugin.warning(message);
  return false;
}

function openUserDrawer(mode: UserDrawerMode, user?: UserRow) {
  if (
    mode === 'create' &&
    !ensureUserPermission(
      permissionStore.hasPermission(userPermissionCodes.CREATE),
      t('user.userList.unavailable.create'),
    )
  ) {
    return;
  }

  if (
    mode === 'edit' &&
    !ensureUserPermission(
      permissionStore.hasPermission(userPermissionCodes.UPDATE),
      t('user.userList.unavailable.edit'),
    )
  ) {
    return;
  }

  userDrawerMode.value = mode;
  userDrawerTarget.value = user ?? null;
  passwordFieldError.value = '';
  userForm.value = {
    username: user?.username ?? '',
    display: user?.display ?? '',
    password: '',
  };
  userDrawerVisible.value = true;
}

function consumeCreateActionQuery() {
  if (route.query.action !== 'create') {
    return;
  }

  const nextQuery = { ...route.query };
  delete nextQuery.action;
  void router.replace({ query: nextQuery });
}

function closeUserDrawer() {
  userDrawerVisible.value = false;
  userDrawerTarget.value = null;
  passwordFieldError.value = '';
  userForm.value = { ...INITIAL_USER_FORM };
  userFormRef.value?.clearValidate();
  submittingUser.value = false;
}

function resolveCreatePasswordError(password: string, requireValue: boolean) {
  if (!password) {
    return requireValue ? t('user.userList.form.required.password') : '';
  }

  return evaluateUserPasswordPolicy(password).meetsMinimum ? '' : t('user.userList.form.passwordPolicy.error');
}

function setUserFormFieldError(field: keyof UserFormState, message: string) {
  userFormRef.value?.setValidateMessage({
    [field]: [{ type: 'error', message }],
  } as FormValidateMessage<UserFormState>);
}

function clearUserFormFieldError(field: keyof UserFormState) {
  userFormRef.value?.clearValidate([field]);
}

function setResetPasswordFieldError(message: string) {
  resetPasswordFormRef.value?.setValidateMessage({
    password: [{ type: 'error', message }],
  });
}

function clearResetPasswordFieldError() {
  resetPasswordFormRef.value?.clearValidate(['password']);
}

async function syncCreatePasswordFeedback() {
  if (userDrawerMode.value !== 'create' || !userDrawerVisible.value) {
    passwordFieldError.value = '';
    return;
  }

  const errorMessage = resolveCreatePasswordError(userForm.value.password, false);
  passwordFieldError.value = errorMessage;

  await nextTick();

  if (userDrawerMode.value !== 'create' || !userDrawerVisible.value) {
    return;
  }

  if (errorMessage) {
    setUserFormFieldError('password', errorMessage);
    return;
  }

  clearUserFormFieldError('password');
}

async function handleUserSubmit(ctx: SubmitContext) {
  if (ctx.validateResult !== true || submittingUser.value) {
    return;
  }

  submittingUser.value = true;
  try {
    if (userDrawerMode.value === 'create') {
      const payload: CreateUserPayload = {
        username: userForm.value.username.trim(),
        display: userForm.value.display.trim(),
        password: userForm.value.password,
      };
      const created = await createUser(payload);
      users.value = [{ ...created, roles: [] as UserRoleSummary[] }, ...users.value];
      MessagePlugin.success(formatHintedMessage(t, t('user.userList.createSuccess')));
    } else if (userDrawerTarget.value) {
      const payload: UpdateUserPayload = {
        username: userForm.value.username.trim(),
        display: userForm.value.display.trim(),
      };
      const updated = await updateUser(userDrawerTarget.value.id, payload);
      users.value = users.value.map((item) =>
        item.id === updated.id ? { ...item, ...updated, roles: item.roles } : item,
      );
      MessagePlugin.success(formatHintedMessage(t, t('user.userList.editSuccess')));
    }
    closeUserDrawer();
  } catch (error) {
    logger.error('failed to submit user form', error);
    if (isApiRequestError(error)) {
      const fallbackMessage =
        userDrawerMode.value === 'create' ? t('user.userList.createFailed') : t('user.userList.editFailed');
      const errorMessage = localizedApiErrorMessage(t, error.messageKey, error.message) || fallbackMessage;
      const field = resolveUserFormFieldError(error, userDrawerMode.value);

      if (field) {
        if (field === 'password') {
          passwordFieldError.value = errorMessage;
        }
        setUserFormFieldError(field, errorMessage);
        return;
      }

      MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, errorMessage));
      return;
    }

    MessagePlugin.error(
      formatHintedMessage(
        t,
        userDrawerMode.value === 'create' ? t('user.userList.createFailed') : t('user.userList.editFailed'),
      ),
    );
  } finally {
    submittingUser.value = false;
  }
}

function openResetPasswordDialog(user: UserRow) {
  if (
    !ensureUserPermission(
      permissionStore.hasPermission(userPermissionCodes.UPDATE),
      t('user.userList.unavailable.more'),
    )
  ) {
    return;
  }

  resetPasswordTarget.value = user;
  resetPasswordForm.value.password = '';
  clearResetPasswordFieldError();
  resetPasswordDialogVisible.value = true;
}

function closeResetPasswordDialog() {
  resetPasswordDialogVisible.value = false;
  resetPasswordTarget.value = null;
  resetPasswordForm.value.password = '';
  clearResetPasswordFieldError();
  submittingResetPassword.value = false;
}

async function submitResetPassword() {
  if (!resetPasswordTarget.value) {
    return;
  }
  if (!resetPasswordForm.value.password.trim()) {
    MessagePlugin.warning(t('user.userList.resetPasswordDialog.required'));
    return;
  }

  clearResetPasswordFieldError();
  submittingResetPassword.value = true;
  try {
    const payload: ResetUserPasswordPayload = {
      new_password: resetPasswordForm.value.password,
    };
    await resetUserPassword(resetPasswordTarget.value.id, payload);
    MessagePlugin.success(formatHintedMessage(t, t('user.userList.resetPasswordSuccess')));
    closeResetPasswordDialog();
  } catch (error) {
    logger.error('failed to reset password', error);
    if (isApiRequestError(error)) {
      const errorMessage =
        localizedApiErrorMessage(t, error.messageKey, error.message) || t('user.userList.resetPasswordFailed');
      const field = resolveResetPasswordFieldError(error);

      if (field === 'password') {
        setResetPasswordFieldError(errorMessage);
        return;
      }

      MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, errorMessage));
      return;
    }

    MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, t('user.userList.resetPasswordFailed')));
  } finally {
    submittingResetPassword.value = false;
  }
}

async function toggleUserStatus(user: UserRow) {
  if (
    !ensureUserPermission(
      permissionStore.hasPermission(userPermissionCodes.DISABLE),
      t('user.userList.unavailable.more'),
    )
  ) {
    return;
  }

  const nextStatus =
    normalizeUserStatus(user.status) === USER_STATUS.DISABLED ? USER_STATUS.ENABLED : USER_STATUS.DISABLED;
  const actionLabel =
    nextStatus === USER_STATUS.DISABLED
      ? t('user.userList.moreActions.disable')
      : t('user.userList.moreActions.enable');
  const confirmed = window.confirm(
    t('user.userList.statusConfirmDescription', { user: user.display || user.username, action: actionLabel }),
  );
  if (!confirmed) {
    return;
  }

  try {
    const updated = await updateUserStatus(user.id, { status: nextStatus });
    users.value = users.value.map((item) => (item.id === updated.id ? { ...item, ...updated } : item));
    MessagePlugin.success(formatHintedMessage(t, t('user.userList.statusUpdateSuccess')));
  } catch (error) {
    logger.error('failed to update status', error);
    if (isApiRequestError(error)) {
      MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, t('user.userList.statusUpdateFailed')));
      return;
    }

    MessagePlugin.error(formatHintedMessage(t, t('user.userList.statusUpdateFailed')));
  }
}

async function confirmDeleteUser(user: UserRow) {
  if (
    !ensureUserPermission(
      permissionStore.hasPermission(userPermissionCodes.DISABLE),
      t('user.userList.unavailable.more'),
    )
  ) {
    return;
  }

  const confirmed = window.confirm(
    t('user.userList.deleteConfirmDescription', { user: user.display || user.username }),
  );
  if (!confirmed) {
    return;
  }

  try {
    await deleteUser(user.id);
    users.value = users.value.filter((item) => item.id !== user.id);
    selectedRowKeys.value = selectedRowKeys.value.filter((item) => item !== user.id);
    MessagePlugin.success(formatHintedMessage(t, t('user.userList.deleteSuccess')));
  } catch (error) {
    logger.error('failed to delete user', error);
    MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, t('user.userList.deleteFailed')));
  }
}

function handleUserMoreAction(payload: { value?: string | number | Record<string, unknown> }, user: UserRow) {
  if (payload.value === 'edit') {
    openUserDrawer('edit', user);
    return;
  }
  if (payload.value === 'toggle-status') {
    void toggleUserStatus(user);
    return;
  }
  if (payload.value === 'reset-password') {
    openResetPasswordDialog(user);
    return;
  }
  if (payload.value === 'delete') {
    void confirmDeleteUser(user);
  }
}

function handleSelectChange(value: Array<string | number>) {
  selectedRowKeys.value = value;
}

function arePermissionIDsEqual(left: number[], right: number[]) {
  if (left.length !== right.length) {
    return false;
  }

  const sortedLeft = [...left].sort((a, b) => a - b);
  const sortedRight = [...right].sort((a, b) => a - b);
  return sortedLeft.every((id, index) => id === sortedRight[index]);
}

function closeUserRoleDrawer() {
  drawerSession.value += 1;
  userRoleDrawerVisible.value = false;
  currentRoleIds.value = [];
  roleDialogMode.value = 'single';
  roleMutationMode.value = 'replace';
  selectedUser.value = null;
  selectedRoleIds.value = [];
  loadingRoleSelection.value = false;
  roleSelectionReady.value = false;
  roleLoadWarning.value = '';
  submittingRoles.value = false;
}

function requestCloseUserRoleDrawer() {
  if (submittingRoles.value) {
    return;
  }

  if (!hasUserRoleSelectionChanges.value) {
    closeUserRoleDrawer();
    return;
  }

  const confirmed = window.confirm(t('user.userList.roleDialog.unsavedChangesConfirm'));
  if (confirmed) {
    closeUserRoleDrawer();
  }
}

function isActiveDrawerSession(session: number) {
  return userRoleDrawerVisible.value && drawerSession.value === session;
}

async function loadUserRoleSelection(user: UserRow, session: number) {
  roleDialogMode.value = 'single';
  selectedUser.value = user;
  currentRoleIds.value = [];
  selectedRoleIds.value = [];
  roleSelectionReady.value = false;
  roleLoadWarning.value = '';

  if (roles.value.length === 0) {
    try {
      await loadRoleCatalog();
    } catch (error) {
      if (isActiveDrawerSession(session)) {
        logger.error('failed to load role catalog', error);
        roleLoadWarning.value = t('user.userList.roleDialog.roleLoadFailed');
      }
      return;
    }
  }

  if (!isActiveDrawerSession(session)) {
    return;
  }

  loadingRoleSelection.value = true;
  try {
    const response = await getUserRoleBindings(user.id);

    if (!isActiveDrawerSession(session)) {
      return;
    }

    currentRoleIds.value = [...response.role_ids];
    selectedRoleIds.value = [...response.role_ids];
    roleSelectionReady.value = true;
  } catch (error) {
    if (isActiveDrawerSession(session)) {
      logger.error('failed to load user role selection', error);
      roleLoadWarning.value = t('user.userList.roleDialog.selectionLoadFailed');
    }
  } finally {
    if (isActiveDrawerSession(session)) {
      loadingRoleSelection.value = false;
    }
  }
}

async function handleOpenUserRoleDrawer(row: UserRow) {
  if (!ensureUserPermission(canManageUserRoles(), t('user.userList.unavailable.manageRoles'))) {
    return;
  }

  const session = drawerSession.value + 1;

  drawerSession.value = session;
  userRoleDrawerVisible.value = true;
  roleSearchKeyword.value = '';
  await loadUserRoleSelection(row, session);
}

async function openBatchUserRoleDrawer() {
  if (!ensureUserPermission(canManageUserRoles(), t('user.userList.unavailable.manageRoles'))) {
    return;
  }

  const session = drawerSession.value + 1;

  drawerSession.value = session;
  roleDialogMode.value = 'batch';
  userRoleDrawerVisible.value = true;
  roleSearchKeyword.value = '';
  currentRoleIds.value = [];
  selectedUser.value = null;
  selectedRoleIds.value = [];
  roleSelectionReady.value = false;
  roleLoadWarning.value = '';

  if (roles.value.length === 0) {
    try {
      await loadRoleCatalog();
    } catch (error) {
      if (isActiveDrawerSession(session)) {
        logger.error('failed to load role catalog for batch role operation', error);
        roleLoadWarning.value = t('user.userList.roleDialog.roleLoadFailed');
      }
      return;
    }
  }

  if (!isActiveDrawerSession(session)) {
    return;
  }

  roleSelectionReady.value = true;
}

function toggleUserRoleSelection(roleId: number) {
  if (loadingRoleDialogData.value || !roleSelectionReady.value || !canManageUserRoles()) {
    return;
  }

  selectedRoleIds.value = selectedRoleIds.value.includes(roleId)
    ? selectedRoleIds.value.filter((item) => item !== roleId)
    : [...selectedRoleIds.value, roleId].sort((left, right) => left - right);
}

async function retryUserRoleDrawerLoad() {
  if (roleDialogMode.value === 'batch') {
    await openBatchUserRoleDrawer();
    return;
  }

  if (!selectedUser.value) {
    return;
  }

  await loadUserRoleSelection(selectedUser.value, drawerSession.value);
}

async function submitUserRoleAssignment() {
  if (!ensureUserPermission(canManageUserRoles(), t('user.userList.unavailable.manageRoles'))) {
    return;
  }

  if (!canSubmitRoleAssignment.value) {
    return;
  }

  const session = drawerSession.value;
  submittingRoles.value = true;

  try {
    if (roleDialogMode.value === 'batch') {
      const payload: BatchUserRoleMutationPayload = {
        user_ids: selectedBatchUserIds.value,
        role_ids: roleMutationPayload.value.role_ids,
      };
      await mutateBatchUserRoles(roleMutationMode.value, payload);
    } else if (selectedUser.value) {
      await mutateUserRoles(selectedUser.value.id, roleMutationMode.value, roleMutationPayload.value);
    }

    if (!isActiveDrawerSession(session)) {
      return;
    }

    const mutationRoleIDs = new Set(roleMutationPayload.value.role_ids);
    const mutationRoles = roles.value.filter((role) => mutationRoleIDs.has(role.id));
    const applyRoleMutation = (currentRoles: UserRoleSummary[]) => {
      if (roleMutationMode.value === 'replace') {
        return mutationRoles;
      }

      if (roleMutationMode.value === 'add') {
        const merged = [...currentRoles];
        mutationRoles.forEach((role) => {
          if (!merged.some((existing) => existing.id === role.id)) {
            merged.push(role);
          }
        });
        return merged;
      }

      return currentRoles.filter((role) => !mutationRoleIDs.has(role.id));
    };

    if (roleDialogMode.value === 'batch') {
      const targetIds = new Set(selectedBatchUserIds.value);
      users.value = users.value.map((item) =>
        targetIds.has(item.id) ? { ...item, roles: applyRoleMutation(item.roles) } : item,
      );
      MessagePlugin.success(formatHintedMessage(t, t('user.userList.batchRoleUpdateSuccess')));
    } else {
      users.value = users.value.map((item) =>
        item.id === selectedUser.value?.id ? { ...item, roles: applyRoleMutation(item.roles) } : item,
      );
      MessagePlugin.success(formatHintedMessage(t, t('user.userList.roleUpdateSuccess')));
    }
    closeUserRoleDrawer();
  } catch (error) {
    if (isActiveDrawerSession(session)) {
      logger.error('failed to mutate user roles', error);
      MessagePlugin.error(
        resolveErrorMessageWithCorrelation(
          t,
          error,
          resolveRoleMutationErrorMessage(error, roleDialogMode.value === 'batch'),
        ),
      );
    }
  } finally {
    if (drawerSession.value === session) {
      submittingRoles.value = false;
    }
  }
}

function resolveRoleMutationErrorMessage(error: unknown, isBatch: boolean) {
  const fallbackMessage = isBatch ? t('user.userList.batchRoleUpdateFailed') : t('user.userList.roleUpdateFailed');
  if (!isApiRequestError(error)) {
    return fallbackMessage;
  }

  return localizedApiErrorMessage(t, error.messageKey, error.message) || fallbackMessage;
}

watch(
  () => roleDialogMode.value,
  (dialogMode) => {
    if (!userRoleDrawerVisible.value) {
      return;
    }

    if (dialogMode === 'batch') {
      selectedRoleIds.value = [];
      return;
    }

    resetRoleSelection();
  },
);

defineExpose({
  handleSelectChange,
});

onMounted(() => {
  fetchUsers();
  void loadRoleCatalog();
});

watch(
  () => [filters.value.keyword, filters.value.status, filters.value.roleId] as const,
  () => {
    pagination.value.current = 1;
  },
);

watch(
  () =>
    [route.query.action, permissionStore.hasPermission(userPermissionCodes.CREATE), userDrawerVisible.value] as const,
  ([action, allowed, visible]) => {
    if (action !== 'create' || !allowed || visible) {
      return;
    }

    openUserDrawer('create');
    consumeCreateActionQuery();
  },
  { immediate: true },
);

watch(
  () => [userDrawerVisible.value, userDrawerMode.value, userForm.value.password] as const,
  () => {
    void syncCreatePasswordFeedback();
  },
);
</script>
<style lang="less" scoped>
@import './index.less';
</style>
