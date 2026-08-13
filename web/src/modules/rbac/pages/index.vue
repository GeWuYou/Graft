<template>
  <div class="role-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="rbac.roleList.listTitle"
        description-key="rbac.roleList.hint"
        :source="{ labelKey: 'menu.domain.security.title', fallback: t('menu.domain.security.title') }"
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

      <advanced-query-filter-builder
        active-preset="all"
        :add-filter-label="`+ ${t('rbac.roleList.toolbar.addFilter')}`"
        add-sorter-label=""
        :builder-hint="t('rbac.roleList.hint')"
        :builder-title="t('rbac.roleList.toolbar.filterPanelTitle')"
        :field-values="roleFilterFieldValues"
        :fields="roleFilterDefinitions"
        :filters-group-label="t('rbac.roleList.toolbar.filterPanelTitle')"
        :keyword="filters.keyword"
        :keyword-placeholder="t('rbac.roleList.toolbar.searchPlaceholder')"
        :loading="loading"
        move-down-label=""
        move-up-label=""
        preset-label=""
        :presets="[]"
        remove-sorter-label=""
        :reset-label="t('rbac.roleList.toolbar.clearFilters')"
        :search-label="t('rbac.roleList.toolbar.query')"
        selected-field-key="type"
        :sort-direction-options="[]"
        sort-direction-placeholder=""
        sort-field-key="sorter"
        :sort-field-options-by-index="[]"
        sort-field-placeholder=""
        :sorters="[]"
        :show-sorter-builder="false"
        :tags="roleFilterTags"
        time-field-key="timeRange"
        :time-fields="[]"
        @close-tag="clearRoleFilterTag"
        @reset="resetFilters"
        @search="applyRoleFilters"
        @update:field="updateRoleFilterField"
        @update:keyword="filters.keyword = $event"
      >
        <template #saved-query-views><saved-query-view-control :controller="savedViews" /></template>
      </advanced-query-filter-builder>

      <management-empty-state
        v-if="listError && !loading"
        tone="error"
        :title="t('rbac.roleList.errorTitle')"
        :description="listError"
      >
        <template #actions>
          <t-button theme="primary" variant="outline" @click="refreshRolePageData">
            {{ t('rbac.roleList.retry') }}
          </t-button>
        </template>
      </management-empty-state>

      <management-paged-table
        v-else
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :columns="visibleColumns"
        :cards-visible="true"
        :column-sets="roleColumnSets"
        density-scope="viewport"
        presentation="data"
        :rows="roles"
        :loading="loading"
        :total="rolesTotal"
        :footer-summary="t('rbac.roleList.footerTotal', { count: rolesTotal })"
        :empty-title="t('rbac.roleList.emptyTitle')"
        :empty-description="t('rbac.roleList.emptyDescription')"
        :pagination-props="{ showPageNumber: true }"
      >
        <template #head>
          <div class="table-head">
            <div>
              <p class="table-head__summary">{{ t('rbac.roleList.summary', { count: rolesTotal }) }}</p>
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
            @refresh="refreshRolePageData"
          />
        </template>

        <template #feedback>
          <div v-if="permissionCatalogError" class="inline-warning">
            <span>{{ permissionCatalogError }}</span>
          </div>
        </template>

        <template #role="{ row }">
          <div class="role-identity">
            <span class="role-identity__display">{{ row.display }}</span>
            <span class="role-identity__code">{{ row.name }}</span>
          </div>
        </template>

        <template #builtin="{ row }">
          <t-tag class="role-type-tag" :theme="isSystemRole(row) ? 'primary' : 'default'" variant="light">
            {{ roleTypeLabel(row) }}
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

        <template #cards>
          <t-loading :loading="loading">
            <responsive-card-list v-if="roles.length" class="role-mobile-list">
              <t-card v-for="row in roles" :key="row.id" class="role-mobile-card" size="small">
                <div class="role-mobile-card__head">
                  <div class="role-identity">
                    <span class="role-identity__display">{{ row.display }}</span>
                    <span class="role-identity__code">{{ row.name }}</span>
                  </div>
                  <t-tag class="role-type-tag" :theme="isSystemRole(row) ? 'primary' : 'default'" variant="light">
                    {{ roleTypeLabel(row) }}
                  </t-tag>
                </div>
                <dl class="role-mobile-card__details">
                  <div>
                    <dt>{{ t('rbac.roleList.columns.permissionCount') }}</dt>
                    <dd>{{ countLabel(row.permission_count, 'rbac.roleList.permissionCount') }}</dd>
                  </div>
                  <div>
                    <dt>{{ t('rbac.roleList.columns.userCount') }}</dt>
                    <dd>{{ countLabel(row.user_count, 'rbac.roleList.userCount') }}</dd>
                  </div>
                  <div class="role-mobile-card__updated-at">
                    <dt>{{ t('rbac.roleList.columns.updatedAt') }}</dt>
                    <dd>{{ formatTimestamp(row.updated_at) }}</dd>
                  </div>
                </dl>
                <div class="role-mobile-card__actions">
                  <table-action-menu
                    :actions="roleRowActions(row)"
                    :more-label="t('rbac.roleList.more')"
                    :more-label-fallback="t('rbac.roleList.more')"
                    @action="(action) => handleRoleRowAction(action, row)"
                  />
                </div>
              </t-card>
            </responsive-card-list>
            <management-empty-state
              v-else-if="!loading"
              :title="t('rbac.roleList.emptyTitle')"
              :description="t('rbac.roleList.emptyDescription')"
            >
              <template #actions>
                <t-button
                  v-if="hasActiveFilters"
                  theme="default"
                  variant="outline"
                  data-testid="role-mobile-empty-clear-filters"
                  @click="resetFilters"
                >
                  {{ t('rbac.roleList.toolbar.clearFilters') }}
                </t-button>
                <t-button
                  v-permission="permissionCodes.ROLE_CREATE"
                  theme="primary"
                  data-testid="role-mobile-empty-create"
                  @click="openCreateDrawer"
                >
                  {{ t('rbac.roleList.emptyCreate') }}
                </t-button>
              </template>
            </management-empty-state>
          </t-loading>
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
      </management-paged-table>
    </management-page-content>

    <responsive-dialog
      v-model:visible="roleDrawerVisible"
      :close-label="t('components.common.close')"
      :title="roleDrawerTitle"
      :purpose="roleDrawerMode === 'detail' ? 'detail' : 'form'"
      size="medium"
    >
      <div class="drawer-panel">
        <t-card v-if="roleDrawerRole" class="role-drawer-overview" size="small" :bordered="true">
          <div class="role-drawer-overview__head" data-testid="role-overview">
            <div class="role-drawer-overview__identity">
              <strong class="role-drawer-overview__name">{{ roleDrawerRole.display }}</strong>
              <span class="role-drawer-overview__code">{{ roleDrawerRole.name }}</span>
            </div>
            <div class="role-drawer-overview__tags">
              <t-tag :theme="isSystemRole(roleDrawerRole) ? 'primary' : 'default'" variant="light">
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
          v-if="roleDrawerRole && isSystemRole(roleDrawerRole)"
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
    </responsive-dialog>

    <assignment-drawer
      v-model:visible="permissionDrawerVisible"
      :close-label="t('components.common.close')"
      :title="permissionDrawerTitle"
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
        <t-collapse
          v-if="permissionDomains.length"
          borderless
          default-expand-all
          expand-icon-placement="right"
          class="permission-domains"
        >
          <t-collapse-panel v-for="domain in permissionDomains" :key="domain.key" :value="domain.key">
            <template #header>
              <div class="permission-domain-header">
                <span>{{ domain.label }}</span>
                <t-tag size="small" theme="default" variant="light">
                  {{ t('rbac.roleList.permissionDialog.domainCount', { count: domain.items.length }) }}
                </t-tag>
              </div>
            </template>
            <div class="assignment-card-grid permission-card-grid">
              <assignment-card
                v-for="item in domain.items"
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
                :tags="permissionTags(item)"
                :title="localizedPermissionDisplay(item)"
                @toggle="toggleRolePermissionSelection(item.id)"
              />
            </div>
          </t-collapse-panel>
        </t-collapse>
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

    <responsive-dialog
      v-model:visible="columnDrawerVisible"
      :close-label="t('components.common.close')"
      :title="t('rbac.roleList.columnSettings')"
      purpose="form"
      size="compact"
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
    </responsive-dialog>
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
  TableActionMenu,
  TableViewToolbar,
} from '@/shared/components/management';
import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';
import {
  AdvancedQueryFilterBuilder,
  type AdvancedQueryFilterFieldDefinition,
  type AdvancedQueryFilterTag,
  applySavedQueryViewPresentation,
  normalizeSavedQueryView,
  SavedQueryViewControl,
  serializeSavedQueryViewRequest,
  useSavedQueryViews,
} from '@/shared/components/query-list';
import ResponsiveCardList from '@/shared/components/responsive/ResponsiveCardList.vue';
import ResponsiveDialog from '@/shared/components/responsive/ResponsiveDialog.vue';
import { useAssignmentSelection } from '@/shared/composables';
import { useTabPageSnapshot } from '@/shared/composables/useTabPageSnapshot';
import { formatHintedMessage, resolveErrorMessageWithCorrelation } from '@/shared/correlation';
import { localizedApiErrorMessage, resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { usePermissionStore } from '@/store';
import { createLogger } from '@/utils/logger';
import { isApiRequestError } from '@/utils/request';

import {
  addRolePermissions,
  cloneRole,
  createRole,
  deleteRole,
  deleteRoleSavedView,
  getRoleDetail,
  getRolePermissionBindings,
  getRoleSavedViews,
  postRoleSavedView,
  putRoleSavedView,
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
import {
  invalidateRolesQuery,
  updateRoleListCache,
  usePermissionCatalogQuery,
  useRolesQuery,
} from '../shared/rbac-queries';
import type { PermissionListItem } from '../types/permission';
import type {
  CloneRolePayload,
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

/** 角色页消费 Query 中的角色与权限目录快照，筛选和抽屉编辑会话仍由页面局部拥有。 */
const logger = createLogger('rbac.roleList');

type RoleDrawerMode = 'create' | 'detail' | 'update';

type RoleFilters = {
  keyword: string;
  type: '' | 'builtin' | 'custom';
};

type RoleSavedViewState = {
  pageSize: number;
  queryState: RoleFilters;
  visibleColumns: string[];
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
  editable?: boolean;
  system?: boolean;
  type?: 'system' | 'custom';
};

type SystemRoleCompat = {
  builtin?: boolean;
  system?: boolean;
  type?: 'system' | 'custom';
};

type PermissionMetadataCompat = PermissionListItem & {
  action?: string | null;
  resource?: string | null;
  risk_level?: 'read' | 'write' | 'destructive' | 'security' | null;
  scope?: 'all' | 'owned' | null;
};

type PermissionDomain = {
  items: PermissionMetadataCompat[];
  key: string;
  label: string;
};

type RolePageSnapshot = {
  columnDrawerVisible: boolean;
  appliedFilters: RoleFilters;
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
const filters = ref<RoleFilters>({
  keyword: '',
  type: '',
});
const appliedFilters = ref<RoleFilters>({ ...filters.value });
const visibleColumnKeys = ref<string[]>([...DEFAULT_VISIBLE_COLUMNS]);
const roleDrawerVisible = ref(false);
const roleDrawerMode = ref<RoleDrawerMode>('create');
const roleDrawerRole = ref<RoleStatusCompat | null>(null);
const roleFormRef = ref<RoleFormInstance | null>(null);
const roleForm = ref<RoleFormState>({ ...INITIAL_ROLE_FORM });
const submittingRole = ref(false);
const permissionDrawerVisible = ref(false);
const selectedRole = ref<RoleListItem | null>(null);
const cloneSourceRoleID = ref<number | null>(null);
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
    appliedFilters.value = { ...(snapshot.appliedFilters ?? snapshot.filters) };
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
      appliedFilters: { ...appliedFilters.value },
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
const roleQueryInput = computed(() => ({
  keyword: appliedFilters.value.keyword,
  builtin: appliedFilters.value.type === 'builtin' ? true : appliedFilters.value.type === 'custom' ? false : undefined,
  limit: pagination.value.pageSize,
  offset: (pagination.value.current - 1) * pagination.value.pageSize,
}));
const rolesQuery = useRolesQuery(roleQueryInput);
const permissionCatalogQuery = usePermissionCatalogQuery(canReadPermissions);
const roles = computed(() => rolesQuery.data.value?.items ?? []);
const rolesTotal = computed(() => rolesQuery.data.value?.total ?? 0);
const permissions = computed(() => permissionCatalogQuery.data.value?.items ?? []);
const loading = computed(
  () => rolesQuery.isFetching.value || (canReadPermissions.value && permissionCatalogQuery.isFetching.value),
);
const listError = computed(() =>
  rolesQuery.isError.value
    ? resolveLocalizedErrorMessage(t, rolesQuery.error.value, t('rbac.roleList.loadFailed'))
    : '',
);
const permissionCatalogError = computed(() =>
  permissionCatalogQuery.isError.value
    ? resolveLocalizedErrorMessage(t, permissionCatalogQuery.error.value, t('rbac.roleList.permissionLoadFailed'))
    : '',
);
watch(permissionCatalogError, (message) => {
  if (message) {
    MessagePlugin.warning(message);
  }
});
const canAssignPermissions = computed(
  () => canReadPermissions.value && permissionStore.hasPermission(permissionCodes.ROLE_PERMISSION_ASSIGN),
);
const canOpenPermissionDrawer = computed(() => canReadPermissions.value && permissions.value.length > 0);
const canCopyRoles = computed(() => canCreateRoles.value);
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
const permissionDrawerReadonly = computed(() => selectedRole.value !== null && isSystemRole(selectedRole.value));
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
const hasActiveFilters = computed(
  () => Boolean(appliedFilters.value.keyword.trim()) || Boolean(appliedFilters.value.type),
);
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
const canEditRoleCode = computed(() => roleDrawerMode.value === 'create' || !isSystemRole(roleDrawerRole.value));
const canDeleteRoleFromDrawer = computed(
  () =>
    roleDrawerMode.value === 'update' &&
    roleDrawerRole.value !== null &&
    canDeleteRoles.value &&
    !isSystemRole(roleDrawerRole.value),
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
          count: selectedPermissionIds.value.length,
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
const roleFilterDefinitions = computed<AdvancedQueryFilterFieldDefinition[]>(() => [
  {
    key: 'type',
    kind: 'select',
    label: t('rbac.roleList.toolbar.typePlaceholder'),
    options: roleTypeOptions.value.map((option) => ({ label: String(option.label), value: String(option.value) })),
  },
]);
const roleFilterFieldValues = computed(() => ({ type: filters.value.type }));
const roleFilterTags = computed<AdvancedQueryFilterTag[]>(() => {
  const tags: AdvancedQueryFilterTag[] = [];
  if (appliedFilters.value.keyword.trim()) tags.push({ key: 'keyword', label: appliedFilters.value.keyword.trim() });
  if (appliedFilters.value.type) {
    const label = roleTypeOptions.value.find((option) => option.value === appliedFilters.value.type)?.label;
    tags.push({
      key: 'type',
      label: `${t('rbac.roleList.toolbar.typePlaceholder')}: ${label ?? appliedFilters.value.type}`,
    });
  }
  return tags;
});

const columnSettingOptions = computed(() => [
  { label: t('rbac.roleList.columns.role'), value: 'role' },
  { label: t('rbac.roleList.columns.type'), value: 'builtin' },
  { label: t('rbac.roleList.columns.permissionCount'), value: 'permission_count' },
  { label: t('rbac.roleList.columns.userCount'), value: 'user_count' },
  { label: t('rbac.roleList.columns.remark'), value: 'remark' },
  { label: t('rbac.roleList.columns.updatedAt'), value: 'updated_at' },
  { label: t('components.commonTable.operation'), value: 'operation' },
]);

const savedViews = useSavedQueryViews<RoleSavedViewState, number>({
  adapter: {
    list: async () =>
      (await getRoleSavedViews()).map((view) =>
        normalizeSavedQueryView<RoleSavedViewState['queryState'], number>(view),
      ),
    create: async (input) =>
      normalizeSavedQueryView<RoleSavedViewState['queryState'], number>(
        await postRoleSavedView(serializeSavedQueryViewRequest(input)),
      ),
    update: async (id, input) =>
      normalizeSavedQueryView<RoleSavedViewState['queryState'], number>(
        await putRoleSavedView(id, serializeSavedQueryViewRequest(input)),
      ),
    remove: deleteRoleSavedView,
  },
  applyView: (view) => {
    const savedFilters = view.state.queryState;
    filters.value = {
      keyword: savedFilters.keyword ?? '',
      type: savedFilters.type === 'builtin' || savedFilters.type === 'custom' ? savedFilters.type : '',
    };
    appliedFilters.value = { ...filters.value };
    applySavedQueryViewPresentation(view.state, {
      pagination: pagination.value,
      supportedColumns: columnSettingOptions.value.map((option) => option.value),
      visibleColumnKeys,
    });
  },
  onError: (error) => MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('rbac.roleList.loadFailed'))),
  serializeCurrentState: () => ({
    pageSize: pagination.value.pageSize,
    queryState: { ...appliedFilters.value },
    visibleColumns: [...visibleColumnKeys.value],
  }),
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

  if (isSystemRole(role)) {
    return options;
  }

  if (permissionStore.hasPermission(permissionCodes.ROLE_UPDATE)) {
    options.push({
      content: t('rbac.roleList.edit'),
      fallbackLabel: t('rbac.roleList.edit'),
      value: 'edit',
    });
  }

  if (canToggleRoleStatus.value && !isSystemRole(role)) {
    options.push({
      content: isRoleEnabled(role) ? t('rbac.roleList.moreActions.disable') : t('rbac.roleList.moreActions.enable'),
      fallbackLabel: isRoleEnabled(role)
        ? t('rbac.roleList.moreActions.disable')
        : t('rbac.roleList.moreActions.enable'),
      value: 'toggle-status',
    });
  }

  if (canDeleteRoles.value && !isSystemRole(role)) {
    options.push({
      content: t('rbac.roleList.moreActions.delete'),
      fallbackLabel: t('rbac.roleList.moreActions.delete'),
      value: 'delete',
    });
  }

  return options;
};

function roleRowActions(role: RoleListItem) {
  const actions: Array<{ disabled?: boolean; label: string; testId?: string; value: string }> = [
    {
      label: t('rbac.roleList.detail'),
      testId: 'role-detail',
      value: 'detail',
    },
  ];

  if (isSystemRole(role)) {
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
    ...roleRowMoreOptions(role)
      .filter((option) => option.value !== 'detail')
      .map((option) => ({
        disabled: option.disabled,
        fallbackLabel: option.fallbackLabel,
        label: option.content,
        testId: option.value === 'edit' ? 'role-edit' : option.value === 'copy-role' ? 'role-copy' : undefined,
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

const filteredPermissionItems = computed<PermissionMetadataCompat[]>(() => {
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

      return `${item.code} ${localizedPermissionDisplay(item)} ${localizedPermissionDescription(item)} ${item.module}`
        .toLowerCase()
        .includes(keyword);
    })
    .slice()
    .sort((left, right) => left.code.localeCompare(right.code));
});
const permissionDomains = computed<PermissionDomain[]>(() => {
  const domains = new Map<string, PermissionMetadataCompat[]>();
  for (const permission of filteredPermissionItems.value) {
    const key = permissionDomainKey(permission);
    domains.set(key, [...(domains.get(key) ?? []), permission]);
  }

  return Array.from(domains, ([key, items]) => ({ key, items, label: permissionDomainLabel(key) }));
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
    label: isSystemRole(selectedRole.value) ? t('rbac.roleList.builtinYes') : t('rbac.roleList.builtinNo'),
    theme: isSystemRole(selectedRole.value) ? ('primary' as const) : ('default' as const),
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
const roleColumnSets = {
  comfortable: ['role', 'builtin', 'permission_count', 'user_count', 'operation'],
  compact: ['role', 'builtin', 'operation'],
};

async function refreshRolePageData() {
  await Promise.all([
    rolesQuery.refetch(),
    canReadPermissions.value ? permissionCatalogQuery.refetch() : Promise.resolve(),
  ]);
}

function resetFilters() {
  filters.value = {
    keyword: '',
    type: '',
  };
  appliedFilters.value = { ...filters.value };
  pagination.value.current = 1;
}

function applyRoleFilters() {
  appliedFilters.value = { ...filters.value };
  pagination.value.current = 1;
}

function updateRoleFilterField(payload: { key: string; value: string | string[] }) {
  if (payload.key !== 'type') return;
  const value = Array.isArray(payload.value) ? (payload.value[0] ?? '') : payload.value;
  filters.value.type = value === 'builtin' || value === 'custom' ? value : '';
}

function clearRoleFilterTag(key: string) {
  if (key === 'keyword') filters.value.keyword = '';
  if (key === 'type') filters.value.type = '';
  applyRoleFilters();
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

function isSystemRole(role: SystemRoleCompat | null | undefined) {
  return role?.system === true || role?.type === 'system' || role?.builtin === true;
}

function roleTypeLabel(role: RoleStatusCompat | RoleListItem) {
  return isSystemRole(role) ? t('rbac.roleList.form.type.system') : t('rbac.roleList.form.type.custom');
}

function roleHasDeleteBlockingBindings(role: RoleStatusCompat) {
  return Number(role.permission_count ?? 0) > 0 || Number(role.user_count ?? 0) > 0;
}

function roleDeleteLifecycleHint(role: RoleStatusCompat) {
  if (isSystemRole(role)) {
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

function toCloneRolePayload(form: RoleFormState): CloneRolePayload {
  return toCreateRolePayload(form);
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
  return localizePermissionDisplay(t, permission, locale.value);
}

function localizedPermissionDescription(permission: PermissionListItem) {
  return localizePermissionDescription(t, permission, 'rbac.roleList.permissionDialog.emptyDescription', locale.value);
}

function permissionDomainKey(permission: PermissionMetadataCompat) {
  return permission.resource?.trim() || permission.module?.trim() || permission.code.split('.')[0] || 'general';
}

function permissionDomainLabel(value: string) {
  return value
    .split(/[._-]/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

function permissionTags(permission: PermissionMetadataCompat) {
  const tags = [{ label: permission.action?.trim() || permission.code.split('.').at(-1) || permission.code }];

  if (permission.risk_level) {
    tags.push({ label: t(`rbac.roleList.permissionDialog.risk.${permission.risk_level}`) });
  }
  if (permission.scope) {
    tags.push({ label: t(`rbac.roleList.permissionDialog.scope.${permission.scope}`) });
  }
  return tags;
}

function openCreateDrawer() {
  roleDrawerMode.value = 'create';
  roleDrawerRole.value = null;
  roleForm.value = { ...INITIAL_ROLE_FORM };
  cloneSourceRoleID.value = null;
  roleDrawerVisible.value = true;
}

async function openCopyRoleDrawer(role: RoleListItem) {
  if (!canCopyRoles.value) {
    MessagePlugin.warning(permissionCatalogError.value || t('rbac.roleList.copyUnavailable'));
    return;
  }

  roleDrawerMode.value = 'create';
  roleDrawerRole.value = null;
  roleForm.value = {
    name: '',
    display: t('rbac.roleList.copyDisplayTemplate', { display: role.display }),
    description: resolveRoleRemark(role),
  };
  cloneSourceRoleID.value = role.id;
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
  cloneSourceRoleID.value = null;
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
      const created =
        cloneSourceRoleID.value === null
          ? await createRole(toCreateRolePayload(roleForm.value))
          : await cloneRole(cloneSourceRoleID.value, toCloneRolePayload(roleForm.value));
      updateRoleListCache((items) => [...items, created].sort((left, right) => left.id - right.id));
      MessagePlugin.success(
        formatHintedMessage(
          cloneSourceRoleID.value !== null ? t('rbac.roleList.copySuccess') : t('rbac.roleList.createSuccess'),
        ),
      );
    } else if (roleDrawerRole.value) {
      const updated = await updateRole(roleDrawerRole.value.id, toUpdateRolePayload(roleForm.value));
      updateRoleListCache((items) => items.map((item) => (item.id === updated.id ? updated : item)));
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
    await invalidateRolesQuery();
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
  if (isSystemRole(role)) {
    MessagePlugin.warning(t('rbac.roleList.moreBuiltinHint'));
    return;
  }

  MessagePlugin.warning(t('rbac.roleList.moreCustomHint'));
}

async function toggleRoleStatus(role: RoleStatusCompat) {
  if (!canToggleRoleStatus.value || isSystemRole(role)) {
    return;
  }

  try {
    const updated = await updateRoleStatus(role.id, {
      status: isRoleEnabled(role) ? 'disabled' : 'enabled',
    });
    updateRoleListCache((items) => items.map((item) => (item.id === updated.id ? updated : item)));
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
  if (!canDeleteRoles.value || isSystemRole(role)) {
    return;
  }
  if (isRoleEnabled(role) || roleHasDeleteBlockingBindings(role)) {
    MessagePlugin.warning(roleDeleteLifecycleHint(role));
    return;
  }

  try {
    await deleteRole(role.id);
    updateRoleListCache((items) => items.filter((item) => item.id !== role.id));
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

watch(
  () => [appliedFilters.value.keyword, appliedFilters.value.type] as const,
  () => {
    pagination.value.current = 1;
  },
);

onMounted(() => void savedViews.load());

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
