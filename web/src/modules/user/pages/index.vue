<template>
  <div class="user-page" data-page-type="list-form-detail">
    <header class="user-page__header">
      <div class="user-page__header-copy">
        <p class="user-page__section">{{ t('user.userList.sectionTitle') }}</p>
        <h1 class="user-page__title">{{ t('user.userList.listTitle') }}</h1>
        <p class="user-page__hint">{{ t('user.userList.hint') }}</p>
      </div>
      <div class="user-page__metrics">
        <article class="user-page__metric">
          <span class="user-page__metric-label">{{ t('user.userList.countLabel') }}</span>
          <strong class="user-page__metric-value">{{ users.length }}</strong>
        </article>
        <article class="user-page__metric">
          <span class="user-page__metric-label">{{ t('user.userList.feedback.roleManagementLabel') }}</span>
          <strong class="user-page__metric-value">{{ roleManagementStateLabel }}</strong>
        </article>
      </div>
    </header>

    <section class="user-page__body-grid">
      <t-card class="user-page__action-card" :bordered="false" :title="t('user.userList.actionTitle')">
        <div class="user-page__action-content">
          <p class="user-page__action-hint">{{ t('user.userList.actionHint') }}</p>
          <div class="user-page__action-buttons">
            <t-button theme="primary" variant="outline" :loading="loading" @click="fetchUsers">
              {{ t('user.userList.refresh') }}
            </t-button>
          </div>
        </div>
      </t-card>

      <section class="user-page__feedback-grid">
        <article class="user-page__feedback-item" :data-tone="rowActionTone">
          <span class="user-page__feedback-label">{{ t('user.userList.feedback.rowActionsLabel') }}</span>
          <div class="user-page__feedback-head">
            <strong class="user-page__feedback-value">{{ rowActionStateLabel }}</strong>
            <t-tag :theme="rowActionTagTheme" variant="light">
              {{ rowActionStateLabel }}
            </t-tag>
          </div>
          <p class="user-page__feedback-hint">{{ rowActionStateHint }}</p>
        </article>
        <article class="user-page__feedback-item" :data-tone="roleManagementTone">
          <div class="user-page__feedback-head">
            <span class="user-page__feedback-label">{{ t('user.userList.feedback.roleManagementLabel') }}</span>
            <t-tag :theme="roleManagementTagTheme" variant="light">
              {{ roleManagementStateLabel }}
            </t-tag>
          </div>
          <p class="user-page__feedback-hint">{{ roleManagementStateHint }}</p>
        </article>
      </section>
    </section>

    <t-card class="user-page__table-card" :bordered="false" :title="t('user.userList.dataTitle')">
      <div class="user-page__table-head">
        <p class="user-page__table-hint">{{ t('user.userList.tableHint') }}</p>
      </div>

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
            <t-button
              v-permission="rbacPermissionCodes.USER_ROLE_READ"
              variant="text"
              theme="primary"
              @click="handleOpenUserRoleDialog(row)"
            >
              {{ t('user.userList.assignRoles') }}
            </t-button>
          </div>
        </template>
        <template #empty>
          <t-empty :description="t('user.userList.empty')" />
        </template>
      </t-table>
    </t-card>

    <t-dialog
      v-model:visible="userRoleDialogVisible"
      :header="t('user.userList.roleDialog.title')"
      :width="720"
      :footer="false"
    >
      <template #body>
        <div class="user-roles-panel">
          <div class="permission-summary">
            {{
              selectedUser
                ? t('user.userList.roleDialog.currentUser', { name: selectedUser.display || selectedUser.username })
                : t('user.userList.roleDialog.currentUserEmpty')
            }}
          </div>
          <div class="permission-summary">
            {{ t('user.userList.roleDialog.roleSummary', { count: roles.length }) }}
          </div>
          <div v-if="roleLoadWarning" class="permission-load-warning">
            <span>{{ roleLoadWarning }}</span>
            <t-button variant="text" theme="primary" :loading="loadingRoleDialogData" @click="retryUserRoleDialogLoad">
              {{ t('user.userList.roleDialog.retry') }}
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
                        {{
                          role.builtin
                            ? t('user.userList.roleDialog.builtinYes')
                            : t('user.userList.roleDialog.builtinNo')
                        }}
                      </t-tag>
                    </div>
                    <span class="role-option__code">{{ role.name }}</span>
                    <span class="role-option__description">
                      {{ role.description || t('user.userList.roleDialog.emptyDescription') }}
                    </span>
                  </div>
                </t-checkbox>
              </label>
            </div>
          </t-checkbox-group>

          <t-empty v-if="roles.length === 0 && !loadingRoles" :description="t('user.userList.roleDialog.empty')" />

          <div class="dialog-actions">
            <t-button variant="outline" @click="closeUserRoleDialog">
              {{ t('user.userList.roleDialog.cancel') }}
            </t-button>
            <t-button
              theme="primary"
              :disabled="!canSubmitRoleAssignment"
              :loading="submittingRoles"
              @click="submitUserRoleAssignment"
            >
              {{ t('user.userList.roleDialog.confirm') }}
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
import type { RoleListItem } from '@/modules/rbac/contract/role';
import { usePermissionStore } from '@/store';

import { assignUserRoles, getRoles, getUserRoleBindings } from '../api/user-roles';
import { getUsers } from '../api/users';
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
const rbacPermissionCodes = RBAC_PERMISSION_CODE;
const canReadUserRoles = computed(() => permissionStore.hasPermission(rbacPermissionCodes.USER_ROLE_READ));
const showOperationColumn = computed(() => canReadUserRoles.value);

const loadingRoleDialogData = computed(() => loadingRoles.value || loadingRoleSelection.value);
const canAssignUserRoles = computed(() => permissionStore.hasPermission(rbacPermissionCodes.USER_ROLE_ASSIGN));
const canSubmitRoleAssignment = computed(
  () => canAssignUserRoles.value && roleSelectionReady.value && roleOptionsReady.value && selectedUser.value !== null,
);
const roleManagementTone = computed(() => {
  if (canAssignUserRoles.value) {
    return 'primary';
  }

  if (canReadUserRoles.value) {
    return 'warning';
  }

  return 'default';
});
const roleManagementTagTheme = computed(() => {
  if (canAssignUserRoles.value) {
    return 'primary';
  }

  if (canReadUserRoles.value) {
    return 'warning';
  }

  return 'default';
});
const roleManagementStateLabel = computed(() => {
  if (canAssignUserRoles.value) {
    return t('user.userList.feedback.roleManagementReady');
  }

  if (canReadUserRoles.value) {
    return t('user.userList.feedback.roleManagementReadOnly');
  }

  return t('user.userList.feedback.roleManagementUnavailable');
});
const roleManagementStateHint = computed(() => {
  if (canAssignUserRoles.value) {
    return t('user.userList.feedback.roleManagementReadyHint');
  }

  if (canReadUserRoles.value) {
    return t('user.userList.feedback.roleManagementReadOnlyHint');
  }

  return t('user.userList.feedback.roleManagementUnavailableHint');
});
const rowActionStateLabel = computed(() =>
  showOperationColumn.value
    ? t('user.userList.feedback.rowActionsAvailable')
    : t('user.userList.feedback.rowActionsUnavailable'),
);
const rowActionTone = computed(() => (showOperationColumn.value ? 'primary' : 'warning'));
const rowActionTagTheme = computed(() => (showOperationColumn.value ? 'primary' : 'warning'));
const rowActionStateHint = computed(() =>
  showOperationColumn.value
    ? t('user.userList.feedback.rowActionsAvailableHint')
    : t('user.userList.feedback.rowActionsUnavailableHint'),
);

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;
  void showOperationColumn.value;

  const baseColumns: TdBaseTableProps['columns'] = [
    {
      title: t('user.userList.columns.id'),
      colKey: 'id',
      width: 100,
    },
    {
      title: t('user.userList.columns.username'),
      colKey: 'username',
      minWidth: 180,
    },
    {
      title: t('user.userList.columns.display'),
      colKey: 'display',
      minWidth: 180,
    },
    {
      title: t('user.userList.columns.createdAt'),
      colKey: 'created_at',
      minWidth: 220,
    },
    {
      title: t('user.userList.columns.updatedAt'),
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
    MessagePlugin.error(error instanceof Error ? error.message : t('user.userList.loadFailed'));
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
      roleLoadWarning.value = error instanceof Error ? error.message : t('user.userList.roleDialog.roleLoadFailed');
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
        error instanceof Error ? error.message : t('user.userList.roleDialog.selectionLoadFailed');
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

    MessagePlugin.success(t('user.userList.assignSuccess'));
    closeUserRoleDialog();
  } catch (error) {
    if (isActiveUserRoleDialogSession(session)) {
      MessagePlugin.error(error instanceof Error ? error.message : t('user.userList.assignFailed'));
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
</style>
