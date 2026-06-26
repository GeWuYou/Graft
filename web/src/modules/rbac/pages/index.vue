<template>
  <div class="role-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="rbac.roleList.listTitle"
        description-key="rbac.roleList.hint"
        :source="{ labelKey: 'menu.access_control.title', fallback: t('menu.access_control.title') }"
      >
        <template #actions>
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
            class="management-list-search"
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
        <template #toolbar>
          <table-view-toolbar
            :column-settings-label="t('rbac.roleList.columnSettings')"
            :refresh-label="t('rbac.roleList.refresh')"
            :refresh-loading="loading"
            @column-settings="columnDrawerVisible = true"
            @refresh="() => fetchRolePageData()"
          />
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
            <t-button theme="primary" variant="outline" @click="() => fetchRolePageData()">
              {{ t('rbac.roleList.retry') }}
            </t-button>
          </template>
        </management-empty-state>

        <div v-else ref="tableHostRef" class="table-host graft-scrollbar" :data-table-mode="tableWidthPolicy.mode">
          <t-table
            row-key="id"
            :data="pagedRoles"
            :columns="visibleColumns"
            :loading="loading"
            table-layout="fixed"
            :table-content-width="tableWidthPolicy.tableContentWidth"
            cell-empty-content="-"
          >
            <template #role="{ row }">
              <div class="role-identity">
                <span class="role-identity__display">{{ row.display }}</span>
                <span class="role-identity__code">{{ row.name }}</span>
              </div>
            </template>

            <template #builtin="{ row }">
              <t-tag class="role-type-tag" :theme="row.builtin ? 'primary' : 'default'" variant="light">
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
                :more-label-fallback="t('rbac.roleList.more')"
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
        </div>

        <template #footer>
          <management-table-pagination :summary="t('rbac.roleList.footerTotal', { count: filteredRoles.length })">
            <t-pagination
              v-model:current="pagination.current"
              v-model:page-size="pagination.pageSize"
              :total="filteredRoles.length"
              :page-size-options="[10, 20, 50, 100]"
              :show-page-number="true"
            />
          </management-table-pagination>
        </template>
      </management-table-card>
    </management-page-content>

    <t-drawer
      v-model:visible="roleDrawerVisible"
      :header="roleDrawerTitle"
      :footer="false"
      size="520px"
      placement="right"
      destroy-on-close
    >
      <div class="drawer-panel">
        <t-card v-if="roleDrawerRole" class="role-drawer-overview" size="small" :bordered="true">
          <div class="role-drawer-overview__head" data-testid="role-overview">
            <div class="role-drawer-overview__identity">
              <strong class="role-drawer-overview__name">{{ roleDrawerRole.display }}</strong>
              <span class="role-drawer-overview__code">{{ roleDrawerRole.name }}</span>
            </div>
            <div class="role-drawer-overview__tags">
              <t-tag :theme="roleDrawerRole.builtin ? 'primary' : 'default'" variant="light">
                {{ roleTypeLabel(roleDrawerRole) }}
              </t-tag>
              <t-tag :theme="roleStatusTagTheme(roleDrawerRole)" variant="light">
                {{ roleStatusLabel(roleDrawerRole) }}
              </t-tag>
            </div>
          </div>
          <div class="role-drawer-overview__stats">
            <div class="role-drawer-stat">
              <span>{{ t('rbac.roleList.columns.permissionCount') }}</span>
              <strong>{{ countLabel(roleDrawerRole.permission_count, 'rbac.roleList.permissionCount') }}</strong>
            </div>
            <div class="role-drawer-stat">
              <span>{{ t('rbac.roleList.columns.userCount') }}</span>
              <strong>{{ countLabel(roleDrawerRole.user_count, 'rbac.roleList.userCount') }}</strong>
            </div>
            <div class="role-drawer-stat role-drawer-stat--wide">
              <span>{{ t('rbac.roleList.columns.updatedAt') }}</span>
              <strong>{{ formatTimestamp(roleDrawerRole.updated_at) }}</strong>
            </div>
          </div>
        </t-card>

        <t-alert
          v-if="roleDrawerRole?.builtin"
          theme="info"
          :title="t('rbac.roleList.form.systemProtectionTitle')"
          class="role-protection-alert"
          data-testid="role-system-rules"
        >
          <template #message>
            <div class="role-protection-alert__content">
              <p>{{ t('rbac.roleList.form.systemProtectionBody') }}</p>
              <p>{{ t('rbac.roleList.form.systemProtectionNormal') }}</p>
              <p>{{ t('rbac.roleList.form.systemProtectionCopyHint') }}</p>
            </div>
          </template>
        </t-alert>

        <t-card
          v-if="roleDrawerMode === 'detail' && roleDrawerRole"
          class="role-drawer-section"
          size="small"
          :bordered="true"
          :title="t('rbac.roleList.form.basicInfoTitle')"
          data-testid="role-readonly-content"
        >
          <t-descriptions size="small" :column="1" table-layout="auto">
            <t-descriptions-item :label="t('rbac.roleList.form.description')">
              {{ roleRemark(roleDrawerRole) }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('rbac.roleList.form.meta.createdAt')">
              {{ formatTimestamp(roleDrawerRole.created_at) }}
            </t-descriptions-item>
          </t-descriptions>
        </t-card>
        <t-card
          v-if="roleDrawerMode === 'detail' && roleDrawerRole"
          class="role-drawer-section"
          size="small"
          :bordered="true"
          :title="t('rbac.roleList.form.permissionOverviewTitle')"
          data-testid="role-permission-overview"
        >
          <t-descriptions size="small" :column="1" table-layout="auto">
            <t-descriptions-item :label="t('rbac.roleList.form.permissionOverviewCount')">
              {{ countLabel(roleDrawerRole.permission_count, 'rbac.roleList.permissionCount') }}
            </t-descriptions-item>
            <t-descriptions-item :label="t('rbac.roleList.form.permissionOverviewAction')">
              <t-button
                v-if="canReadPermissions"
                size="small"
                theme="default"
                variant="outline"
                data-testid="role-drawer-view-permissions"
                @click="openRoleDrawerPermissionDrawer"
              >
                {{ t('rbac.roleList.viewPermissions') }}
              </t-button>
              <span v-else>{{ t('rbac.roleList.permissionDialog.readPermissionRequired') }}</span>
            </t-descriptions-item>
          </t-descriptions>
        </t-card>
        <t-card
          v-if="roleDrawerMode === 'detail' && roleDrawerRole"
          class="role-drawer-section"
          size="small"
          :bordered="true"
          :title="t('rbac.roleList.lifecycle.deleteRequirement')"
          data-testid="role-delete-condition"
        >
          <span data-testid="role-lifecycle-summary">{{ roleDeleteLifecycleHint(roleDrawerRole) }}</span>
        </t-card>
        <div v-if="roleDrawerMode === 'detail'" class="drawer-actions">
          <t-button variant="outline" data-testid="role-drawer-cancel" @click="closeRoleDrawer">
            {{ t('rbac.roleList.form.cancel') }}
          </t-button>
        </div>

        <t-card
          v-else
          class="role-drawer-section"
          size="small"
          :bordered="true"
          :title="t('rbac.roleList.form.editableContentTitle')"
        >
          <t-form
            ref="roleFormRef"
            :data="roleForm"
            :rules="roleFormRules"
            label-align="top"
            @submit="handleRoleSubmit"
          >
            <t-form-item v-if="canEditRoleCode" :label="t('rbac.roleList.form.name')" name="name">
              <t-input v-model="roleForm.name" :placeholder="t('rbac.roleList.form.namePlaceholder')" />
            </t-form-item>
            <t-form-item :label="t('rbac.roleList.form.display')" name="display">
              <t-input v-model="roleForm.display" :placeholder="t('rbac.roleList.form.displayPlaceholder')" />
            </t-form-item>
            <t-form-item :label="t('rbac.roleList.form.description')" name="description">
              <t-textarea
                v-model="roleForm.description"
                :maxlength="200"
                :placeholder="t('rbac.roleList.form.descriptionPlaceholder')"
              />
            </t-form-item>
            <div class="drawer-actions">
              <t-button variant="outline" data-testid="role-drawer-cancel" @click="closeRoleDrawer">
                {{ t('rbac.roleList.form.cancel') }}
              </t-button>
              <t-button
                v-if="canDeleteRoleFromDrawer"
                theme="danger"
                variant="outline"
                data-testid="role-drawer-delete"
                @click="() => removeRoleFromDrawer()"
              >
                {{ t('rbac.roleList.moreActions.delete') }}
              </t-button>
              <t-button theme="primary" type="submit" data-testid="role-drawer-save" :loading="submittingRole">
                {{ t('rbac.roleList.form.confirm') }}
              </t-button>
            </div>
          </t-form>
        </t-card>
      </div>
    </t-drawer>

    <assignment-drawer
      v-model:visible="permissionDrawerVisible"
      :title="permissionDrawerTitle"
      size="860px"
      @close="requestClosePermissionDrawer"
    >
      <template #header>
        <div class="assignment-panel assignment-panel--compact" data-testid="permission-drawer">
          <assignment-header
            :avatar-text="roleAssignmentAvatar"
            :badges="roleAssignmentBadges"
            :description="roleAssignmentDescription"
            :eyebrow="permissionDrawerEyebrow"
            :stats="roleAssignmentStats"
            :subtitle="roleAssignmentSubtitle"
            :title="roleAssignmentTitle"
          />

          <t-alert
            v-if="permissionDrawerReadonly"
            theme="info"
            :title="t('rbac.roleList.permissionDialog.readonlyProtectionTitle')"
            data-testid="permission-readonly-protection"
          >
            <template #message>
              {{ t('rbac.roleList.permissionDialog.readonlyProtectionBody') }}
            </template>
          </t-alert>

          <div class="assignment-toolbar-stack">
            <assignment-toolbar
              v-model:mode-value="permissionMutationMode"
              v-model:search-value="permissionKeyword"
              :disabled="
                loadingRolePermissions || submittingPermissions || permissionDrawerReadonly || !canAssignPermissions
              "
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
            :hint="permissionDialogHelp"
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
          :disabled="
            loadingRolePermissions || !permissionSelectionReady || permissionDrawerReadonly || !canAssignPermissions
          "
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
              permissionDrawerReadonly ||
              !canAssignPermissions ||
              isPermissionCardDisabled(item)
            "
            :selected="selectedPermissionIds.includes(item.id)"
            :tags="[
              {
                label: t('rbac.roleList.permissionDialog.categoryBadge', {
                  category: item.category || t('rbac.roleList.permissionDialog.defaultCategory'),
                }),
              },
            ]"
            :title="localizedPermissionDisplay(item)"
            @toggle="toggleRolePermissionSelection(item.id)"
          />
        </div>
      </assignment-grid>

      <template #footer>
        <assignment-footer
          :cancel-label="permissionDrawerCancelLabel"
          cancel-test-id="permission-drawer-cancel"
          :confirm-disabled="!canSubmitPermissionAssignment"
          :confirm-label="t('rbac.roleList.permissionDialog.confirm')"
          :confirm-loading="submittingPermissions"
          :details="permissionFooterDetails"
          confirm-test-id="permission-drawer-save"
          :show-confirm="!permissionDrawerReadonly"
          :summary="permissionFooterSummary"
          @cancel="requestClosePermissionDrawer"
          @confirm="submitPermissionAssignment"
        />
      </template>
    </assignment-drawer>

    <t-dialog
      v-model:visible="showDiscardConfirm"
      :header="t('rbac.roleList.permissionDialog.unsavedChangesTitle')"
      :body="t('rbac.roleList.permissionDialog.unsavedChangesConfirm')"
      theme="warning"
      :cancel-btn="t('rbac.roleList.permissionDialog.continueEditing')"
      :confirm-btn="{ content: t('rbac.roleList.permissionDialog.discardChanges'), theme: 'danger' }"
      @cancel="continueEditingPermissionDrawer"
      @close="continueEditingPermissionDrawer"
      @confirm="discardPermissionDrawerChanges"
    />

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
import type { FormRule, FormValidateMessage, SubmitContext, TdBaseTableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { buildAuditResourceLocation } from '@/modules/audit/contract/deep-link';
import { AUDIT_PERMISSION_CODE } from '@/modules/audit/contract/permissions';
import { openCorrelationErrorNotification, requestIdFromError } from '@/modules/audit/shared/correlation-actions';
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
  resolveTableWidthPolicy,
  TableActionMenu,
  TableViewToolbar,
  useTableHostWidth,
} from '@/shared/components/management';
import { useAssignmentSelection, useTabPageSnapshot } from '@/shared/composables';
import { formatHintedMessage, resolveErrorMessageWithCorrelation } from '@/shared/correlation';
import { localizedApiErrorMessage, resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
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
  created_at?: string | null;
  description?: string | null;
  disabled?: boolean;
  enabled?: boolean;
  status?: string | null;
  deleted_at?: string | null;
};

type RolePageSnapshot = {
  columnDrawerVisible: boolean;
  filters: RoleFilters;
  pagination: {
    current: number;
    pageSize: number;
  };
  roleDrawer: {
    form: RoleFormState;
    mode: RoleDrawerMode;
    visible: boolean;
  };
  visibleColumnKeys: string[];
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
const roleDrawerRole = ref<RoleStatusCompat | null>(null);
const roleFormRef = ref<RoleFormInstance | null>(null);
const roleForm = ref<RoleFormState>({ ...INITIAL_ROLE_FORM });
const submittingRole = ref(false);
const permissionDrawerVisible = ref(false);
const selectedRole = ref<RoleListItem | null>(null);
const copiedRolePermissionIds = ref<number[] | null>(null);
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
const showDiscardConfirm = ref(false);
const columnDrawerVisible = ref(false);
const pagination = ref({
  current: 1,
  pageSize: 10,
});

useTabPageSnapshot<RolePageSnapshot>({
  apply(snapshot) {
    filters.value = { ...snapshot.filters };
    visibleColumnKeys.value = [...snapshot.visibleColumnKeys];
    pagination.value = { ...snapshot.pagination };
    columnDrawerVisible.value = snapshot.columnDrawerVisible;
    if (snapshot.roleDrawer.visible && snapshot.roleDrawer.mode === 'create') {
      roleDrawerVisible.value = true;
      roleDrawerMode.value = 'create';
      roleDrawerRole.value = null;
      roleForm.value = { ...snapshot.roleDrawer.form };
    }
  },
  read() {
    return {
      columnDrawerVisible: columnDrawerVisible.value,
      filters: { ...filters.value },
      pagination: { ...pagination.value },
      roleDrawer: {
        form: { ...roleForm.value },
        mode: roleDrawerMode.value,
        visible: roleDrawerVisible.value && roleDrawerMode.value === 'create',
      },
      visibleColumnKeys: [...visibleColumnKeys.value],
    };
  },
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
const canCopyRoles = computed(
  () => canCreateRoles.value && canAssignPermissions.value && canOpenPermissionDrawer.value,
);
const canShowOperationColumn = computed(() =>
  permissionStore.hasAnyPermission([
    AUDIT_PERMISSION_CODE.READ,
    permissionCodes.ROLE_UPDATE,
    permissionCodes.ROLE_DELETE,
    permissionCodes.ROLE_STATUS_UPDATE,
    permissionCodes.PERMISSION_READ,
    permissionCodes.ROLE_PERMISSION_ASSIGN,
  ]),
);
const permissionDrawerReadonly = computed(() => selectedRole.value?.builtin === true);
const currentPermissionIds = computed(() => {
  switch (permissionMutationMode.value) {
    case 'add':
      return sortStableIDs([...new Set([...originalPermissionIds.value, ...selectedPermissionIds.value])]);
    case 'remove': {
      const removed = new Set(selectedPermissionIds.value);
      return sortStableIDs(originalPermissionIds.value.filter((id) => !removed.has(id)));
    }
    default:
      return sortStableIDs(selectedPermissionIds.value);
  }
});
const isPermissionDirty = computed(() => {
  if (!permissionSelectionReady.value || selectedRole.value === null) {
    return false;
  }

  return !arePermissionIDsEqual(originalPermissionIds.value, currentPermissionIds.value);
});
const canSubmitPermissionAssignment = computed(() => {
  return !permissionDrawerReadonly.value && canAssignPermissions.value && isPermissionDirty.value;
});
const hasActiveFilters = computed(() => Boolean(filters.value.keyword.trim()) || Boolean(filters.value.type));
const permissionDialogStatusMessage = computed(() =>
  loadingRolePermissions.value ? t('rbac.roleList.permissionDialog.loadingSelection') : permissionLoadWarning.value,
);
const permissionDrawerTitle = computed(() =>
  permissionDrawerReadonly.value
    ? t('rbac.roleList.permissionDialog.readonlyTitle')
    : t('rbac.roleList.permissionDialog.title'),
);
const permissionDrawerEyebrow = computed(() =>
  permissionDrawerReadonly.value
    ? t('rbac.roleList.permissionDialog.readonlyHeaderEyebrow')
    : t('rbac.roleList.permissionDialog.headerEyebrow'),
);
const permissionDialogHelp = computed(() =>
  permissionDrawerReadonly.value
    ? t('rbac.roleList.permissionDialog.readonlyHelp')
    : t('rbac.roleList.permissionDialog.operationHelp'),
);
const permissionDrawerCancelLabel = computed(() =>
  permissionDrawerReadonly.value ? t('rbac.roleList.permissionDialog.close') : t('rbac.roleList.form.cancel'),
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
const canEditRoleCode = computed(() => roleDrawerMode.value === 'create' || !roleDrawerRole.value?.builtin);
const canDeleteRoleFromDrawer = computed(
  () =>
    roleDrawerMode.value === 'update' &&
    roleDrawerRole.value !== null &&
    canDeleteRoles.value &&
    !roleDrawerRole.value.builtin,
);
const permissionFooterSummary = computed(() =>
  t('rbac.roleList.permissionDialog.selectionCount', {
    selected: selectedPermissionIds.value.length,
    total: permissions.value.length,
  }),
);
const permissionFooterDetails = computed(() => {
  if (permissionDrawerReadonly.value) {
    return [t('rbac.roleList.permissionDialog.readonlyFooterDetail')];
  }

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
    fallbackLabel: t('rbac.roleList.detail'),
    value: 'detail',
  });

  if (canCopyRoles.value) {
    options.push({
      content: t('rbac.roleList.copyAsCustom'),
      fallbackLabel: t('rbac.roleList.copyAsCustom'),
      value: 'copy-role',
    });
  }

  if (role.builtin) {
    return options;
  }

  if (permissionStore.hasPermission(permissionCodes.ROLE_UPDATE)) {
    options.push({
      content: t('rbac.roleList.edit'),
      fallbackLabel: t('rbac.roleList.edit'),
      value: 'edit',
    });
  }

  if (canToggleRoleStatus.value && !role.builtin) {
    options.push({
      content: isRoleEnabled(role) ? t('rbac.roleList.moreActions.disable') : t('rbac.roleList.moreActions.enable'),
      fallbackLabel: isRoleEnabled(role)
        ? t('rbac.roleList.moreActions.disable')
        : t('rbac.roleList.moreActions.enable'),
      value: 'toggle-status',
    });
  }

  if (canDeleteRoles.value && !role.builtin) {
    options.push({
      content: t('rbac.roleList.moreActions.delete'),
      fallbackLabel: t('rbac.roleList.moreActions.delete'),
      value: 'delete',
    });
  }

  return options;
};

function roleRowActions(role: RoleListItem) {
  const actions: Array<{ disabled?: boolean; label: string; testId?: string; value: string }> = [];

  if (role.builtin) {
    if (canReadPermissions.value) {
      actions.push({
        disabled: !canOpenPermissionDrawer.value,
        label: t('rbac.roleList.viewPermissions'),
        testId: 'role-view-permissions',
        value: 'view-permissions',
      });
    }
  } else if (canAssignPermissions.value) {
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
      testId:
        option.value === 'detail'
          ? 'role-detail'
          : option.value === 'edit'
            ? 'role-edit'
            : option.value === 'copy-role'
              ? 'role-copy'
              : undefined,
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
    theme: selectedRole.value?.builtin ? ('primary' as const) : ('default' as const),
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

const { tableHostRef, tableHostWidth } = useTableHostWidth(() => visibleColumns.value);
const tableWidthPolicy = computed(() => resolveTableWidthPolicy(visibleColumns.value, tableHostWidth.value));

async function fetchRolePageData(preservePagination = false) {
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
    if (!preservePagination) {
      pagination.value.current = 1;
    }

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

function roleStatusTagTheme(role: RoleStatusCompat) {
  return isRoleEnabled(role) ? ('success' as const) : ('default' as const);
}

function roleTypeLabel(role: Pick<RoleListItem, 'builtin'>) {
  return role.builtin ? t('rbac.roleList.form.type.system') : t('rbac.roleList.form.type.custom');
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
  const normalizedLeft = sortStableIDs(left);
  const normalizedRight = sortStableIDs(right);

  if (normalizedLeft.length !== normalizedRight.length) {
    return false;
  }

  return normalizedLeft.every((value, index) => value === normalizedRight[index]);
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
  copiedRolePermissionIds.value = null;
  roleDrawerVisible.value = true;
}

async function openCopyRoleDrawer(role: RoleListItem) {
  if (!canCopyRoles.value) {
    MessagePlugin.warning(permissionCatalogError.value || t('rbac.roleList.copyUnavailable'));
    return;
  }

  let permissionIds: number[] = [];
  try {
    const response = await getRolePermissionBindings(role.id);
    const normalized = normalizeRolePermissionIDs(extractPermissionIDs(response));
    if (normalized === null) {
      MessagePlugin.warning(t('rbac.roleList.permissionDialog.selectionUnavailable'));
      return;
    }
    permissionIds = normalized;
  } catch (error) {
    logger.warn('failed to load source role permissions for copy', error);
    MessagePlugin.warning(resolveLocalizedErrorMessage(t, error, t('rbac.roleList.copyLoadFailed')));
    return;
  }

  roleDrawerMode.value = 'create';
  roleDrawerRole.value = null;
  roleForm.value = {
    name: '',
    display: t('rbac.roleList.copyDisplayTemplate', { display: role.display }),
    description: resolveRoleRemark(role),
  };
  copiedRolePermissionIds.value = permissionIds;
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
  const action = typeof payload === 'object' ? payload.value : payload;
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

  if (action === 'copy-role') {
    void openCopyRoleDrawer(role);
    return;
  }

  if (action === 'detail') {
    void openDetailDrawer(role);
    return;
  }

  void handleMoreAction(role);
}

function handleRoleRowAction(action: string, role: RoleListItem) {
  if (action === 'assign-permissions' || action === 'view-permissions') {
    void openPermissionDrawer(role);
    return;
  }

  if (action === 'view-audit') {
    void router.push(buildAuditResourceLocation('role', String(role.id), role.display || role.name));
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
  copiedRolePermissionIds.value = null;
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
      if (copiedRolePermissionIds.value?.length) {
        try {
          await replaceRolePermissions(created.id, toReplaceRolePermissionsPayload(copiedRolePermissionIds.value));
        } catch (error) {
          logger.error('failed to copy role permissions after role creation', error);
          MessagePlugin.warning(
            resolveErrorMessageWithCorrelation(t, error, t('rbac.roleList.copyPermissionsPartialSuccess')),
          );
          closeRoleDrawer();
          return;
        }
      }
      MessagePlugin.success(
        formatHintedMessage(
          copiedRolePermissionIds.value ? t('rbac.roleList.copySuccess') : t('rbac.roleList.createSuccess'),
        ),
      );
    } else if (roleDrawerRole.value) {
      const updated = await updateRole(roleDrawerRole.value.id, toUpdateRolePayload(roleForm.value));
      roles.value = roles.value.map((item) => (item.id === updated.id ? updated : item));
      roleDrawerRole.value = updated;
      MessagePlugin.success(formatHintedMessage(t('rbac.roleList.updateSuccess')));
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

      const message = resolveErrorMessageWithCorrelation(t, error, errorMessage);
      MessagePlugin.error(message);
      openCorrelationErrorNotification({
        router,
        title: t('audit.correlation.errorTitle'),
        message,
        requestId: requestIdFromError(error),
        translate: t,
      });
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

function openRoleDrawerPermissionDrawer() {
  if (!roleDrawerRole.value) {
    return;
  }

  void openPermissionDrawer(roleDrawerRole.value);
}

function closePermissionDrawer() {
  permissionDrawerSession.value += 1;
  permissionDrawerVisible.value = false;
  selectedRole.value = null;
  permissionSelectionReady.value = false;
  loadingRolePermissions.value = false;
  permissionLoadWarning.value = '';
  permissionLoadRetryable.value = false;
  submittingPermissions.value = false;
  showDiscardConfirm.value = false;
  resetPermissionDraft();
}

function resetPermissionDraft() {
  originalPermissionIds.value = [];
  selectedPermissionIds.value = [];
  permissionMutationMode.value = 'replace';
  permissionKeyword.value = '';
  permissionOnlySelected.value = false;
}

function requestClosePermissionDrawer() {
  if (submittingPermissions.value) {
    return;
  }

  if (permissionDrawerReadonly.value || !isPermissionDirty.value) {
    closePermissionDrawer();
    return;
  }

  showDiscardConfirm.value = true;
}

function continueEditingPermissionDrawer() {
  showDiscardConfirm.value = false;
}

function discardPermissionDrawerChanges() {
  showDiscardConfirm.value = false;
  resetPermissionDraft();
  closePermissionDrawer();
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
  if (
    loadingRolePermissions.value ||
    !permissionSelectionReady.value ||
    permissionDrawerReadonly.value ||
    !canAssignPermissions.value
  ) {
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
  if (
    !selectedRole.value ||
    permissionDrawerReadonly.value ||
    !canSubmitPermissionAssignment.value ||
    loadingRolePermissions.value
  ) {
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

    MessagePlugin.success(formatHintedMessage(t('rbac.roleList.assignSuccess')));
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

        const message = resolveErrorMessageWithCorrelation(t, error, errorMessage);
        MessagePlugin.error(message);
        openCorrelationErrorNotification({
          router,
          title: t('audit.correlation.errorTitle'),
          message,
          requestId: requestIdFromError(error),
          translate: t,
        });
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
        isRoleEnabled(updated) ? t('rbac.roleList.statusEnabledSuccess') : t('rbac.roleList.statusDisabledSuccess'),
      ),
    );
  } catch (error) {
    logger.error('failed to update role status', error);
    if (isApiRequestError(error)) {
      const message = resolveErrorMessageWithCorrelation(t, error, t('rbac.roleList.statusUpdateFailed'));
      MessagePlugin.error(message);
      openCorrelationErrorNotification({
        router,
        title: t('audit.correlation.errorTitle'),
        message,
        requestId: requestIdFromError(error),
        translate: t,
      });
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
    MessagePlugin.success(formatHintedMessage(t('rbac.roleList.deleteSuccess')));
  } catch (error) {
    logger.error('failed to delete role', error);
    if (isApiRequestError(error)) {
      const message = resolveErrorMessageWithCorrelation(t, error, t('rbac.roleList.deleteFailed'));
      MessagePlugin.error(message);
      openCorrelationErrorNotification({
        router,
        title: t('audit.correlation.errorTitle'),
        message,
        requestId: requestIdFromError(error),
        translate: t,
      });
      return;
    }

    MessagePlugin.error(resolveErrorMessageWithCorrelation(t, error, t('rbac.roleList.deleteFailed')));
  }
}

async function removeRoleFromDrawer() {
  if (!roleDrawerRole.value) {
    return;
  }

  const roleId = roleDrawerRole.value.id;
  await removeRole(roleDrawerRole.value);
  if (!roles.value.some((item) => item.id === roleId)) {
    closeRoleDrawer();
  }
}

onMounted(() => {
  fetchRolePageData(true);
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
