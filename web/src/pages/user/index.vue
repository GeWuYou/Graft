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
            {{ roleLoadWarning }}
          </div>

          <t-checkbox-group v-model="selectedRoleIds" :disabled="!roleSelectionReady || !canAssignUserRoles">
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

import type { RoleListItem } from '@/api/model/rbacModel';
import type { UserListItem } from '@/api/model/userModel';
import { assignUserRoles, getRoles, getUserRoleBindings } from '@/api/rbac';
import { getUsers } from '@/api/user';
import { RBAC_PERMISSION_CODE } from '@/contracts/rbac/permissions';
import { USER_PERMISSION_CODE } from '@/contracts/user/permissions';
import { usePermissionStore } from '@/store';

defineOptions({
  name: 'UsersIndex',
});

const { t, locale } = useI18n();
const permissionStore = usePermissionStore();
const users = ref<UserListItem[]>([]);
const loading = ref(false);
const loadingRoles = ref(false);
const submittingRoles = ref(false);
const userRoleDialogVisible = ref(false);
const selectedUser = ref<UserListItem | null>(null);
const roles = ref<RoleListItem[]>([]);
const selectedRoleIds = ref<number[]>([]);
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
    // 后端契约假设：GET /api/users 始终返回 { items: UserListItem[] }，且时间字段已可直接展示。
    const response = await getUsers();
    users.value = response.items;
  } catch (error) {
    users.value = [];
    MessagePlugin.error(error instanceof Error ? error.message : t('pages.userList.loadFailed'));
  } finally {
    loading.value = false;
  }
}

async function ensureRoleOptionsLoaded() {
  if (roles.value.length > 0) {
    roleOptionsReady.value = true;
    return;
  }

  loadingRoles.value = true;
  try {
    const response = await getRoles();
    roles.value = response.items;
    roleOptionsReady.value = true;
  } catch (error) {
    roles.value = [];
    roleOptionsReady.value = false;
    throw error;
  } finally {
    loadingRoles.value = false;
  }
}

function closeUserRoleDialog() {
  userRoleDialogVisible.value = false;
  selectedUser.value = null;
  selectedRoleIds.value = [];
  roleSelectionReady.value = false;
  roleOptionsReady.value = roles.value.length > 0;
  roleLoadWarning.value = '';
}

async function handleOpenUserRoleDialog(row: TableRowData) {
  const user = row as UserListItem;
  selectedUser.value = user;
  selectedRoleIds.value = [];
  roleSelectionReady.value = false;
  roleOptionsReady.value = false;
  roleLoadWarning.value = '';
  userRoleDialogVisible.value = true;

  try {
    await ensureRoleOptionsLoaded();
  } catch (error) {
    roleLoadWarning.value = error instanceof Error ? error.message : t('pages.userList.roleDialog.roleLoadFailed');
  }

  if (!roleOptionsReady.value) {
    return;
  }

  try {
    const response = await getUserRoleBindings(user.id);
    selectedRoleIds.value = response.role_ids;
    roleSelectionReady.value = true;
  } catch (error) {
    roleSelectionReady.value = false;
    roleLoadWarning.value = error instanceof Error ? error.message : t('pages.userList.roleDialog.selectionLoadFailed');
  }
}

async function submitUserRoleAssignment() {
  if (!selectedUser.value || !canSubmitRoleAssignment.value) {
    return;
  }

  submittingRoles.value = true;
  try {
    await assignUserRoles(selectedUser.value.id, {
      role_ids: selectedRoleIds.value,
    });
    MessagePlugin.success(t('pages.userList.assignSuccess'));
    closeUserRoleDialog();
  } catch (error) {
    MessagePlugin.error(error instanceof Error ? error.message : t('pages.userList.assignFailed'));
  } finally {
    submittingRoles.value = false;
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
  color: var(--td-error-color);
  font: var(--td-font-body-small);
}
</style>
