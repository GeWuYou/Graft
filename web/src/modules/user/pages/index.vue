<template>
  <div class="user-page">
    <t-row :gutter="[24, 24]">
      <t-col :span="12">
        <t-card class="summary-card" :bordered="false" :title="t('pages.userList.listTitle')">
          <div class="summary-metric">
            <span class="summary-metric__label">{{ t('pages.userList.countLabel') }}</span>
            <span class="summary-metric__value">{{ users.length }}</span>
          </div>
          <div class="summary-hint">{{ t('pages.userList.hint') }}</div>
        </t-card>
      </t-col>
      <t-col :span="12">
        <t-card class="summary-card" :bordered="false" :title="t('pages.userList.apiTitle')">
          <div class="summary-meta">
            <span
              >{{ t('pages.userList.endpointLabel') }}<code>{{ t('pages.userList.endpointValue') }}</code></span
            >
            <span
              >{{ t('pages.userList.fieldsLabel') }}<code>{{ t('pages.userList.fieldsValue') }}</code></span
            >
          </div>
          <div class="summary-actions">
            <t-button theme="primary" variant="outline" :loading="loading" @click="fetchUsers">
              {{ t('pages.userList.refresh') }}
            </t-button>
            <t-button v-permission="permissionCodes.CREATE" theme="default" variant="base" disabled>
              {{ t('pages.listBase.create') }}
            </t-button>
          </div>
        </t-card>
      </t-col>
    </t-row>

    <t-card class="table-card" :bordered="false" :title="t('pages.userList.dataTitle')">
      <t-table
        row-key="id"
        :data="users"
        :columns="columns"
        :loading="loading"
        size="medium"
        :table-layout="showOperationColumn ? 'fixed' : 'auto'"
      >
        <template #operation="{ row }">
          <div class="operation-cell">
            <t-button v-permission="permissionCodes.UPDATE" variant="text" theme="primary" disabled>
              {{ t('components.commonTable.detail') }}
            </t-button>
            <t-button
              v-permission="rbacPermissionCodes.USER_ROLE_READ"
              variant="text"
              theme="primary"
              @click="handleOpenUserRoleDialog(row)"
            >
              {{ t('pages.userList.assignRoles') }}
            </t-button>
            <t-button v-permission="permissionCodes.DISABLE" variant="text" theme="danger" disabled>
              {{ t('components.manage') }}
            </t-button>
          </div>
        </template>
        <template #empty>
          <t-empty :description="t('pages.userList.empty')" />
        </template>
      </t-table>
    </t-card>

    <t-dialog
      v-model:visible="userRoleDialogVisible"
      :header="t('pages.userList.roleDialog.title')"
      :width="720"
      :footer="false"
    >
      <template #body>
        <div class="user-roles-panel">
          <div class="permission-summary">
            {{
              selectedUser
                ? t('pages.userList.roleDialog.currentUser', { name: selectedUser.display || selectedUser.username })
                : t('pages.userList.roleDialog.currentUserEmpty')
            }}
          </div>
          <div class="permission-summary">
            {{ t('pages.userList.roleDialog.roleSummary', { count: roles.length }) }}
          </div>
          <div v-if="roleLoadWarning" class="permission-load-warning">
            <span>{{ roleLoadWarning }}</span>
            <t-button variant="text" theme="primary" :loading="loadingRoleDialogData" @click="retryUserRoleDialogLoad">
              {{ t('pages.userList.roleDialog.retry') }}
            </t-button>
          </div>

          <t-checkbox-group
            v-model="selectedRoleIds"
            :disabled="loadingRoleDialogData || !roleSelectionReady || !canAssignUserRoles"
          >
            <div class="role-grid">
              <label v-for="role in roles" :key="role.id" class="role-option">
                <t-checkbox :value="role.id">
                  <div class="role-option__content">
                    <div class="role-option__header">
                      <span class="role-option__label">{{ role.display }}</span>
                      <t-tag :theme="role.builtin ? 'success' : 'default'" variant="light" size="small">
                        {{ role.builtin ? t('pages.roleList.builtinYes') : t('pages.roleList.builtinNo') }}
                      </t-tag>
                    </div>
                    <span class="role-option__code">{{ role.name }}</span>
                    <span class="role-option__description">
                      {{ role.description || t('pages.roleList.emptyDescription') }}
                    </span>
                  </div>
                </t-checkbox>
              </label>
            </div>
          </t-checkbox-group>

          <t-empty v-if="roles.length === 0 && !loadingRoles" :description="t('pages.userList.roleDialog.empty')" />

          <div class="dialog-actions">
            <t-button variant="outline" @click="closeUserRoleDialog">
              {{ t('pages.roleList.form.cancel') }}
            </t-button>
            <t-button
              theme="primary"
              :disabled="!canSubmitRoleAssignment"
              :loading="submittingRoles"
              @click="submitUserRoleAssignment"
            >
              {{ t('pages.userList.roleDialog.confirm') }}
            </t-button>
          </div>
        </div>
      </template>
    </t-dialog>
  </div>
</template>
<script setup lang="ts">
import { MessagePlugin, type TableRowData, type TdBaseTableProps } from 'tdesign-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { RBAC_PERMISSION_CODE } from '@/modules/rbac/contract/permissions';
import type { RoleListItem } from '@/modules/rbac/types/rbac';
import { usePermissionStore } from '@/store';

import { assignUserRoles, getRoles, getUserRoleBindings } from '../api/user-roles';
import { getUsers } from '../api/users';
import { USER_PERMISSION_CODE } from '../contract/permissions';
import type { UserListItem } from '../types/user';

defineOptions({
  name: 'UsersIndex',
});

const { t, locale } = useI18n();
const permissionStore = usePermissionStore();
const users = ref<UserListItem[]>([]);
const loading = ref(false);
const loadingRoles = ref(false);
const loadingRoleSelection = ref(false);
const submittingRoles = ref(false);
const userRoleDialogVisible = ref(false);
const selectedUser = ref<UserListItem | null>(null);
const roles = ref<RoleListItem[]>([]);
const selectedRoleIds = ref<number[]>([]);
const userRoleDialogSession = ref(0);
const roleSelectionReady = ref(false);
const roleOptionsReady = ref(false);
const roleLoadWarning = ref('');
const permissionCodes = USER_PERMISSION_CODE;
const rbacPermissionCodes = RBAC_PERMISSION_CODE;

const showOperationColumn = computed(() =>
  permissionStore.hasAnyPermission([
    permissionCodes.UPDATE,
    permissionCodes.DISABLE,
    rbacPermissionCodes.USER_ROLE_READ,
  ]),
);

const loadingRoleDialogData = computed(() => loadingRoles.value || loadingRoleSelection.value);
const canAssignUserRoles = computed(() => permissionStore.hasPermission(rbacPermissionCodes.USER_ROLE_ASSIGN));
const canSubmitRoleAssignment = computed(
  () => canAssignUserRoles.value && roleSelectionReady.value && roleOptionsReady.value && selectedUser.value !== null,
);

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;
  void showOperationColumn.value;

  const baseColumns: TdBaseTableProps['columns'] = [
    {
      title: t('pages.userList.columns.id'),
      colKey: 'id',
      width: 100,
    },
    {
      title: t('pages.userList.columns.username'),
      colKey: 'username',
      minWidth: 180,
    },
    {
      title: t('pages.userList.columns.display'),
      colKey: 'display',
      minWidth: 180,
    },
    {
      title: t('pages.userList.columns.createdAt'),
      colKey: 'created_at',
      minWidth: 220,
    },
    {
      title: t('pages.userList.columns.updatedAt'),
      colKey: 'updated_at',
      minWidth: 220,
    },
  ];

  if (showOperationColumn.value) {
    baseColumns.push({
      title: t('components.commonTable.operation'),
      colKey: 'operation',
      width: 320,
      fixed: 'right',
    });
  }

  return baseColumns;
});

async function fetchUsers() {
  loading.value = true;
  try {
    const response = await getUsers();
    users.value = response.items;
  } catch (error) {
    users.value = [];
    MessagePlugin.error(error instanceof Error ? error.message : t('pages.userList.loadFailed'));
  } finally {
    loading.value = false;
  }
}

function isActiveUserRoleDialogSession(session: number) {
  return userRoleDialogVisible.value && userRoleDialogSession.value === session;
}

async function ensureRoleOptionsLoaded(session: number) {
  if (roles.value.length > 0) {
    if (isActiveUserRoleDialogSession(session)) {
      roleOptionsReady.value = true;
    }
    return isActiveUserRoleDialogSession(session);
  }

  if (isActiveUserRoleDialogSession(session)) {
    loadingRoles.value = true;
  }

  try {
    const response = await getRoles();

    if (!isActiveUserRoleDialogSession(session)) {
      return false;
    }

    roles.value = response.items;
    roleOptionsReady.value = true;
    return true;
  } catch (error) {
    if (isActiveUserRoleDialogSession(session)) {
      roles.value = [];
      roleOptionsReady.value = false;
    }
    throw error;
  } finally {
    if (isActiveUserRoleDialogSession(session)) {
      loadingRoles.value = false;
    }
  }
}

function closeUserRoleDialog() {
  userRoleDialogSession.value += 1;
  userRoleDialogVisible.value = false;
  submittingRoles.value = false;
  selectedUser.value = null;
  selectedRoleIds.value = [];
  loadingRoles.value = false;
  loadingRoleSelection.value = false;
  roleSelectionReady.value = false;
  roleOptionsReady.value = roles.value.length > 0;
  roleLoadWarning.value = '';
}

async function loadUserRoleDialog(user: UserListItem, session: number) {
  selectedUser.value = user;
  selectedRoleIds.value = [];
  loadingRoleSelection.value = false;
  roleSelectionReady.value = false;
  roleLoadWarning.value = '';

  try {
    const roleOptionsLoaded = await ensureRoleOptionsLoaded(session);

    if (!roleOptionsLoaded || !isActiveUserRoleDialogSession(session)) {
      return;
    }
  } catch (error) {
    if (isActiveUserRoleDialogSession(session)) {
      roleLoadWarning.value = error instanceof Error ? error.message : t('pages.userList.roleDialog.roleLoadFailed');
    }
  }

  if (!roleOptionsReady.value) {
    return;
  }

  loadingRoleSelection.value = true;
  try {
    const response = await getUserRoleBindings(user.id);

    if (!isActiveUserRoleDialogSession(session)) {
      return;
    }

    selectedRoleIds.value = response.role_ids;
    roleSelectionReady.value = true;
  } catch (error) {
    if (isActiveUserRoleDialogSession(session)) {
      roleSelectionReady.value = false;
      roleLoadWarning.value =
        error instanceof Error ? error.message : t('pages.userList.roleDialog.selectionLoadFailed');
    }
  } finally {
    if (isActiveUserRoleDialogSession(session)) {
      loadingRoleSelection.value = false;
    }
  }
}

async function handleOpenUserRoleDialog(row: TableRowData) {
  const user = row as UserListItem;
  const session = userRoleDialogSession.value + 1;

  userRoleDialogSession.value = session;
  roleOptionsReady.value = false;
  userRoleDialogVisible.value = true;
  await loadUserRoleDialog(user, session);
}

async function retryUserRoleDialogLoad() {
  if (!selectedUser.value) {
    return;
  }

  const session = userRoleDialogSession.value;

  await loadUserRoleDialog(selectedUser.value, session);
}

async function submitUserRoleAssignment() {
  if (!selectedUser.value || !canSubmitRoleAssignment.value) {
    return;
  }

  const session = userRoleDialogSession.value;
  const userId = selectedUser.value.id;
  const roleIds = [...selectedRoleIds.value];

  submittingRoles.value = true;
  try {
    await assignUserRoles(userId, {
      role_ids: roleIds,
    });
    if (!isActiveUserRoleDialogSession(session)) {
      return;
    }

    MessagePlugin.success(t('pages.userList.assignSuccess'));
    closeUserRoleDialog();
  } catch (error) {
    if (isActiveUserRoleDialogSession(session)) {
      MessagePlugin.error(error instanceof Error ? error.message : t('pages.userList.assignFailed'));
    }
  } finally {
    if (userRoleDialogSession.value === session) {
      submittingRoles.value = false;
    }
  }
}

onMounted(() => {
  fetchUsers();
});
</script>
<style lang="less" scoped>
@import './index.less';

.summary-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--td-comp-margin-s);
}

.operation-cell {
  display: flex;
  gap: var(--td-comp-margin-xs);
  justify-content: flex-start;
}

.dialog-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--td-comp-margin-s);
  justify-content: flex-end;
}

.permission-summary {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
}

.user-roles-panel {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-l);
}

.role-grid {
  display: grid;
  gap: var(--td-comp-margin-l);
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
}

.role-option {
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
}

.role-option__content {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.role-option__header {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: space-between;
}

.role-option__label {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.role-option__code {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
}

.role-option__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.permission-load-warning {
  align-items: center;
  color: var(--td-error-color);
  display: flex;
  font: var(--td-font-body-small);
  gap: var(--td-comp-margin-xs);
  justify-content: space-between;
}
</style>
