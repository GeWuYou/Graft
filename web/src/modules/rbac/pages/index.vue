<template>
  <div class="role-page">
    <t-row :gutter="[24, 24]">
      <t-col :span="12">
        <t-card class="summary-card" :bordered="false" :title="t('pages.roleList.listTitle')">
          <div class="summary-metric">
            <span class="summary-metric__label">{{ t('pages.roleList.countLabel') }}</span>
            <span class="summary-metric__value">{{ roles.length }}</span>
          </div>
          <div class="summary-hint">{{ t('pages.roleList.hint') }}</div>
        </t-card>
      </t-col>
      <t-col :span="12">
        <t-card class="summary-card" :bordered="false" :title="t('pages.roleList.apiTitle')">
          <div class="summary-meta">
            <span
              >{{ t('pages.roleList.endpointLabel') }}<code>{{ t('pages.roleList.endpointValue') }}</code></span
            >
            <span
              >{{ t('pages.roleList.fieldsLabel') }}<code>{{ t('pages.roleList.fieldsValue') }}</code></span
            >
          </div>
          <div class="summary-actions">
            <t-button theme="primary" variant="outline" :loading="loading" @click="fetchRolePageData">
              {{ t('pages.roleList.refresh') }}
            </t-button>
            <t-button v-permission="permissionCodes.ROLE_CREATE" theme="primary" @click="openCreateDialog">
              {{ t('pages.roleList.create') }}
            </t-button>
          </div>
        </t-card>
      </t-col>
    </t-row>

    <t-card class="table-card" :bordered="false" :title="t('pages.roleList.dataTitle')">
      <div class="toolbar-actions">
        <div class="permission-summary">
          {{ t('pages.roleList.permissionSummary', { count: permissions.length }) }}
        </div>
      </div>

      <t-table row-key="id" :data="roles" :columns="columns" :loading="loading" size="medium" table-layout="fixed">
        <template #builtin="{ row }">
          <t-tag :theme="row.builtin ? 'success' : 'default'" variant="light">
            {{ row.builtin ? t('pages.roleList.builtinYes') : t('pages.roleList.builtinNo') }}
          </t-tag>
        </template>
        <template #description="{ row }">
          <span>{{ row.description || t('pages.roleList.emptyDescription') }}</span>
        </template>
        <template #operation="{ row }">
          <div class="operation-cell">
            <t-button
              v-permission="permissionCodes.ROLE_UPDATE"
              variant="text"
              theme="primary"
              :disabled="row.builtin"
              @click="openEditDialog(row)"
            >
              {{ t('components.commonTable.detail') }}
            </t-button>
            <t-button
              v-permission="[permissionCodes.ROLE_PERMISSION_ASSIGN, permissionCodes.PERMISSION_READ]"
              variant="text"
              theme="primary"
              :disabled="!canAssignPermissions"
              @click="openPermissionDialog(row)"
            >
              {{ t('pages.roleList.assignPermissions') }}
            </t-button>
          </div>
        </template>
        <template #empty>
          <t-empty :description="t('pages.roleList.empty')" />
        </template>
      </t-table>
    </t-card>

    <t-dialog v-model:visible="roleDialogVisible" :header="roleDialogTitle" :width="640" :footer="false">
      <template #body>
        <t-form :data="roleForm" :rules="roleFormRules" label-align="top" @submit="handleRoleSubmit">
          <t-form-item :label="t('pages.roleList.form.name')" name="name">
            <t-input v-model="roleForm.name" :placeholder="t('pages.roleList.form.namePlaceholder')" />
          </t-form-item>
          <t-form-item :label="t('pages.roleList.form.display')" name="display">
            <t-input v-model="roleForm.display" :placeholder="t('pages.roleList.form.displayPlaceholder')" />
          </t-form-item>
          <t-form-item :label="t('pages.roleList.form.description')" name="description">
            <t-textarea
              v-model="roleForm.description"
              :placeholder="t('pages.roleList.form.descriptionPlaceholder')"
              :maxlength="200"
            />
          </t-form-item>
          <div class="dialog-actions">
            <t-button variant="outline" @click="closeRoleDialog">
              {{ t('pages.roleList.form.cancel') }}
            </t-button>
            <t-button theme="primary" type="submit" :loading="submittingRole">
              {{ t('pages.roleList.form.confirm') }}
            </t-button>
          </div>
        </t-form>
      </template>
    </t-dialog>

    <t-dialog
      v-model:visible="permissionDialogVisible"
      :header="t('pages.roleList.permissionDialog.title')"
      :width="760"
      :footer="false"
    >
      <template #body>
        <div class="permissions-panel">
          <div class="permission-summary">
            {{
              selectedRole
                ? t('pages.roleList.permissionDialog.currentRole', { name: selectedRole.display })
                : t('pages.roleList.permissionDialog.currentRoleEmpty')
            }}
          </div>

          <div class="permissions-grid">
            <section v-for="group in permissionGroups" :key="group.category" class="permission-group">
              <div class="permission-group__title">{{ group.title }}</div>
              <div class="permission-group__hint">
                {{ t('pages.roleList.permissionDialog.groupHint', { count: group.items.length }) }}
              </div>
              <div v-if="!permissionSelectionReady" class="permission-load-warning">
                {{ t('pages.roleList.permissionDialog.selectionUnavailable') }}
              </div>
              <t-checkbox-group v-model="selectedPermissionIds" class="permission-checkbox-group">
                <t-checkbox v-for="item in group.items" :key="item.id" class="permission-checkbox" :value="item.id">
                  <div class="permission-checkbox__content">
                    <span class="permission-checkbox__label">{{ item.display }}</span>
                    <span class="permission-checkbox__code">{{ item.code }}</span>
                  </div>
                </t-checkbox>
              </t-checkbox-group>
            </section>
          </div>

          <div class="dialog-actions">
            <t-button variant="outline" @click="closePermissionDialog">
              {{ t('pages.roleList.form.cancel') }}
            </t-button>
            <t-button
              theme="primary"
              :disabled="!permissionSelectionReady"
              :loading="submittingPermissions"
              @click="submitPermissionAssignment"
            >
              {{ t('pages.roleList.permissionDialog.confirm') }}
            </t-button>
          </div>
        </div>
      </template>
    </t-dialog>
  </div>
</template>
<script setup lang="ts">
import type { FormRule, SubmitContext, TdBaseTableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { usePermissionStore } from '@/store';

import {
  assignRolePermissions,
  createRole,
  getPermissions,
  getRolePermissionBindings,
  getRoles,
  updateRole,
} from '../api/rbac';
import { RBAC_PERMISSION_CODE } from '../contract/permissions';
import type { CreateRolePayload, PermissionListItem, RoleListItem } from '../types/rbac';

defineOptions({
  name: 'RolesIndex',
});

type RoleFormMode = 'create' | 'update';

type RoleFormState = {
  name: string;
  display: string;
  description: string;
};

type PermissionGroup = {
  category: string;
  title: string;
  items: PermissionListItem[];
};

const INITIAL_ROLE_FORM: RoleFormState = {
  name: '',
  display: '',
  description: '',
};

const { t, locale } = useI18n();
const permissionStore = usePermissionStore();
const roles = ref<RoleListItem[]>([]);
const permissions = ref<PermissionListItem[]>([]);
const loading = ref(false);
const submittingRole = ref(false);
const submittingPermissions = ref(false);
const roleDialogVisible = ref(false);
const permissionDialogVisible = ref(false);
const roleFormMode = ref<RoleFormMode>('create');
const selectedRole = ref<RoleListItem | null>(null);
const roleForm = ref<RoleFormState>({ ...INITIAL_ROLE_FORM });
const selectedPermissionIds = ref<number[]>([]);
const permissionSelectionReady = ref(false);
const loadingRolePermissions = ref(false);

const permissionCodes = RBAC_PERMISSION_CODE;
const canReadPermissions = computed(() => permissionStore.hasPermission(permissionCodes.PERMISSION_READ));
const canAssignPermissions = computed(() => canReadPermissions.value && permissions.value.length > 0);

const roleDialogTitle = computed(() =>
  roleFormMode.value === 'create' ? t('pages.roleList.form.createTitle') : t('pages.roleList.form.editTitle'),
);

const roleFormRules = computed<Record<keyof RoleFormState, FormRule[]>>(() => ({
  name: [{ required: true, message: t('pages.roleList.form.required.name'), type: 'error' }],
  display: [{ required: true, message: t('pages.roleList.form.required.display'), type: 'error' }],
  description: [],
}));

const permissionGroups = computed<PermissionGroup[]>(() => {
  const groups = new Map<string, PermissionListItem[]>();

  permissions.value.forEach((item) => {
    const category = item.category || 'general';
    const items = groups.get(category) ?? [];
    items.push(item);
    groups.set(category, items);
  });

  return Array.from(groups.entries())
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([category, items]) => ({
      category,
      title: t('pages.roleList.permissionDialog.category', { category }),
      items: items.slice().sort((left, right) => left.code.localeCompare(right.code)),
    }));
});

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;

  return [
    {
      title: t('pages.roleList.columns.id'),
      colKey: 'id',
      width: 100,
    },
    {
      title: t('pages.roleList.columns.name'),
      colKey: 'name',
      minWidth: 160,
    },
    {
      title: t('pages.roleList.columns.display'),
      colKey: 'display',
      minWidth: 180,
    },
    {
      title: t('pages.roleList.columns.description'),
      colKey: 'description',
      minWidth: 240,
    },
    {
      title: t('pages.roleList.columns.builtin'),
      colKey: 'builtin',
      width: 120,
    },
    {
      title: t('components.commonTable.operation'),
      colKey: 'operation',
      width: 220,
      fixed: 'right',
    },
  ];
});

function normalizeDescription(description: string) {
  const trimmed = description.trim();
  return trimmed ? trimmed : null;
}

function sortStableIDs(ids: number[]) {
  return ids.slice().sort((left, right) => left - right);
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

function updateRolePermissionSnapshot(roleID: number, permissionIDs: number[]) {
  const normalizedPermissionIDs = sortStableIDs(permissionIDs);
  roles.value = roles.value.map((item) => {
    if (item.id !== roleID) {
      return item;
    }

    return {
      ...item,
      permission_ids: normalizedPermissionIDs,
    } as RoleListItem;
  });
}

function applyRolePermissionSelection(permissionIDs: number[]) {
  const normalizedPermissionIDs = normalizeRolePermissionIDs(permissionIDs);
  if (normalizedPermissionIDs === null) {
    permissionSelectionReady.value = false;
    selectedPermissionIds.value = [];
    return false;
  }

  selectedPermissionIds.value = normalizedPermissionIDs;
  permissionSelectionReady.value = true;
  return true;
}

async function loadRolePermissionSelection(roleID: number) {
  loadingRolePermissions.value = true;
  permissionSelectionReady.value = false;
  selectedPermissionIds.value = [];

  try {
    const response = await getRolePermissionBindings(roleID);
    if (!applyRolePermissionSelection(response.permission_ids)) {
      MessagePlugin.error(t('pages.roleList.permissionDialog.selectionUnavailable'));
      return false;
    }

    return true;
  } catch (error) {
    MessagePlugin.error(
      error instanceof Error ? error.message : t('pages.roleList.permissionDialog.selectionLoadFailed'),
    );
    return false;
  } finally {
    loadingRolePermissions.value = false;
  }
}

async function fetchRolePageData() {
  loading.value = true;
  try {
    const results = await Promise.allSettled([
      getRoles(),
      canReadPermissions.value ? getPermissions() : Promise.resolve({ items: [] as PermissionListItem[] }),
    ]);

    if (results[0].status === 'fulfilled') {
      roles.value = results[0].value.items;
    } else {
      roles.value = [];
      throw results[0].reason;
    }

    if (results[1].status === 'fulfilled') {
      permissions.value = results[1].value.items;
    } else {
      permissions.value = [];
      MessagePlugin.warning(t('pages.roleList.permissionLoadFailed'));
    }
  } catch (error) {
    roles.value = [];
    MessagePlugin.error(error instanceof Error ? error.message : t('pages.roleList.loadFailed'));
  } finally {
    loading.value = false;
  }
}

function openCreateDialog() {
  roleFormMode.value = 'create';
  selectedRole.value = null;
  roleForm.value = { ...INITIAL_ROLE_FORM };
  roleDialogVisible.value = true;
}

function openEditDialog(role: RoleListItem) {
  roleFormMode.value = 'update';
  selectedRole.value = role;
  roleForm.value = {
    name: role.name,
    display: role.display,
    description: role.description ?? '',
  };
  roleDialogVisible.value = true;
}

function closeRoleDialog() {
  roleDialogVisible.value = false;
  roleForm.value = { ...INITIAL_ROLE_FORM };
}

async function handleRoleSubmit(ctx: SubmitContext) {
  if (ctx.validateResult !== true || submittingRole.value) {
    return;
  }

  submittingRole.value = true;
  try {
    const payload: CreateRolePayload = {
      name: roleForm.value.name.trim(),
      display: roleForm.value.display.trim(),
      description: normalizeDescription(roleForm.value.description),
    };

    if (roleFormMode.value === 'create') {
      const created = await createRole(payload);
      roles.value = [...roles.value, created].sort((left, right) => left.id - right.id);
      MessagePlugin.success(t('pages.roleList.createSuccess'));
    } else if (selectedRole.value) {
      const updated = await updateRole(selectedRole.value.id, payload);
      roles.value = roles.value.map((item) => (item.id === updated.id ? updated : item));
      selectedRole.value = updated;
      MessagePlugin.success(t('pages.roleList.updateSuccess'));
    }

    closeRoleDialog();
  } catch (error) {
    MessagePlugin.error(error instanceof Error ? error.message : t('pages.roleList.submitFailed'));
  } finally {
    submittingRole.value = false;
  }
}

async function openPermissionDialog(role: RoleListItem) {
  if (!canAssignPermissions.value) {
    MessagePlugin.warning(t('pages.roleList.permissionUnavailable'));
    return;
  }

  selectedRole.value = role;
  if (!(await loadRolePermissionSelection(role.id))) {
    return;
  }

  permissionDialogVisible.value = true;
}

function closePermissionDialog() {
  permissionDialogVisible.value = false;
  selectedRole.value = null;
  selectedPermissionIds.value = [];
  permissionSelectionReady.value = false;
}

async function submitPermissionAssignment() {
  if (!selectedRole.value || submittingPermissions.value || loadingRolePermissions.value) {
    return;
  }

  if (!permissionSelectionReady.value) {
    MessagePlugin.error(t('pages.roleList.permissionDialog.selectionUnavailable'));
    return;
  }

  submittingPermissions.value = true;
  try {
    const permissionIDs = sortStableIDs(selectedPermissionIds.value);
    await assignRolePermissions(selectedRole.value.id, {
      permission_ids: permissionIDs,
    });
    updateRolePermissionSnapshot(selectedRole.value.id, permissionIDs);
    selectedRole.value = roles.value.find((item) => item.id === selectedRole.value?.id) ?? selectedRole.value;
    MessagePlugin.success(t('pages.roleList.assignSuccess'));
    closePermissionDialog();
  } catch (error) {
    MessagePlugin.error(error instanceof Error ? error.message : t('pages.roleList.assignFailed'));
  } finally {
    submittingPermissions.value = false;
  }
}

onMounted(() => {
  fetchRolePageData();
});
</script>
<style lang="less" scoped>
@import './index.less';
</style>
