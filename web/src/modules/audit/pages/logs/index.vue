<template>
  <advanced-query-list-page
    page-type="log-audit"
    root-class="audit-page"
    title-key="audit.logList.title"
    description-key="audit.logList.description"
    :error-message="listError"
    :error-title="t('audit.logList.errorTitle')"
    :loading="loading"
    compact-header
    :reload-label="t('audit.logList.refresh')"
    :retry-label="t('audit.logList.retry')"
    :show-header-reload="false"
    :source="{ labelKey: 'menu.audit.title', fallback: t('menu.audit.title'), color: 'var(--td-brand-color-6)' }"
    @reload="fetchAuditLogs"
  >
    <template #actions>
      <t-button v-if="canManageAuditPolicy" theme="default" variant="outline" @click="openPolicyDrawer">
        {{ t('audit.logList.policy.manage') }}
      </t-button>
      <t-button v-if="monitorReturnLocation" theme="primary" variant="outline" @click="returnToMonitor">
        {{ t('audit.logList.actions.backToMonitor') }}
      </t-button>
    </template>
    <template #feedback-extra>
      <section v-if="scopeState" class="audit-scope-banner">
        <div class="audit-scope-banner__main">
          <div class="audit-scope-banner__summary">
            <t-tag theme="primary" variant="light-outline" size="small">
              {{ t('audit.logList.scope.drilldownTag', { name: scopeState.appliedScope.name }) }}
            </t-tag>
            <span v-if="primaryScopeCondition" class="audit-scope-banner__condition">
              {{ t('audit.logList.scope.conditionInline', { condition: primaryScopeCondition }) }}
            </span>
          </div>
        </div>
        <div class="audit-scope-banner__actions">
          <t-button theme="primary" variant="outline" size="small" @click="convertScopeToFilters">
            {{ t('audit.logList.scope.convertAction') }}
          </t-button>
          <t-button theme="default" variant="text" size="small" @click="exitDrilldown">
            {{ t('audit.logList.scope.exitAction') }}
          </t-button>
        </div>
      </section>
    </template>
    <template #filters>
      <audit-filters
        v-model="filters"
        :active-preset="activePreset"
        :can-manage-visibility="canManageAuditPolicy"
        :locked-fields="scopeOwnedFilterKeys"
        :loading="loading"
        :presets="presetViews"
        @apply-preset="applyPreset"
        @reset="resetFilters"
        @search="handleSearch"
      >
        <template #saved-query-views>
          <saved-query-view-control :controller="auditSavedViews" />
        </template>
      </audit-filters>
    </template>
    <template #table>
      <audit-table
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :footer-summary="footerSummary"
        :loading="loading"
        :local-filter-active="hasClientOnlyFilters"
        :rows="displayRows"
        :can-delete="canDeleteAuditLogs"
        :selected-row-keys="selectedRowKeys"
        :total="tableTotal"
        :visible-column-keys="visibleColumnKeys"
        @detail="openDetailDrawer"
        @page-change="fetchAuditLogs"
        @select-change="handleSelectChange"
        @view-access-log="openAccessLog"
        @view-app-log="openAppLog"
        @view-security-event="openSecurityEvent"
      >
        <template #toolbar>
          <table-view-toolbar
            :column-settings-label="t('audit.logList.columnSettings')"
            :refresh-label="t('audit.logList.refresh')"
            :refresh-loading="loading"
            @column-settings="columnDrawerVisible = true"
            @refresh="fetchAuditLogs"
          />
        </template>
        <template v-if="selectedRowKeys.length > 0" #batch>
          <management-batch-bar
            :clear-label="t('audit.logList.batch.clear')"
            :compact-action-label="t('audit.logList.batch.actions')"
            :invert-current-page-label="t('audit.logList.batch.invertCurrentPage')"
            :select-current-page-label="t('audit.logList.batch.selectCurrentPage')"
            :selected-label="t('audit.logList.batch.selected', { count: selectedRowKeys.length })"
            @clear="clearSelection"
            @invert-current-page="invertCurrentPage"
            @select-current-page="selectCurrentPage"
          >
            <t-button theme="danger" variant="outline" @click="confirmBatchDelete">
              {{ t('audit.logList.batch.delete') }}
            </t-button>
          </management-batch-bar>
        </template>
      </audit-table>
    </template>
    <template #detail>
      <advanced-query-column-drawer
        v-model:visible="columnDrawerVisible"
        v-model:selected-keys="visibleColumnKeys"
        :columns="columnSettingOptions"
        :default-selected-keys="DEFAULT_VISIBLE_COLUMNS"
        :presets-label="t('audit.logList.columnViews.label')"
        :reset-label="t('audit.logList.columnViews.resetDefault')"
        :title="t('audit.logList.columnSettings')"
        :view-presets="columnViewPresets"
      />
      <audit-detail-drawer
        v-model:visible="detailDrawerVisible"
        :initial-tab="detailInitialTab"
        :record="detailRecord"
        :rows="rows"
        :monitor-origin="navigationContext.monitorOrigin"
      />
      <t-drawer
        v-model:visible="policyDrawerVisible"
        :footer="false"
        :header="t('audit.logList.policy.drawerTitle')"
        size="720px"
      >
        <div class="audit-policy-drawer">
          <div class="audit-policy-drawer__section">
            <div class="audit-policy-drawer__title-row">
              <label class="audit-policy-drawer__label">{{ t('audit.logList.policy.defaultStrategy') }}</label>
              <t-tag v-if="policyDefaultDirty" theme="primary" variant="light-outline" size="small">
                {{ t('audit.logList.policy.unsavedTag') }}
              </t-tag>
            </div>
            <p class="audit-policy-drawer__hint">{{ t('audit.logList.policy.defaultHint') }}</p>
            <t-select
              :disabled="defaultPolicySaving || bulkPolicySaving"
              :model-value="policyDefaultStrategy"
              :options="visibilityStrategyOptions"
              @update:model-value="handlePolicyDefaultChange"
            />
            <t-button
              theme="primary"
              class="audit-policy-drawer__action"
              :disabled="!policyDefaultDirty || bulkPolicySaving"
              :loading="defaultPolicySaving"
              @click="requestSavePolicyDefault"
            >
              {{ t('audit.logList.policy.saveDefault') }}
            </t-button>
          </div>
          <div class="audit-policy-drawer__section">
            <div class="audit-policy-drawer__section-header">
              <div>
                <div class="audit-policy-drawer__label">{{ t('audit.logList.policy.overrideTitle') }}</div>
                <p class="audit-policy-drawer__hint">
                  {{ t('audit.logList.policy.overrideHint') }}
                </p>
              </div>
              <t-button
                theme="primary"
                variant="outline"
                :disabled="!overrideDirtyCount || defaultPolicySaving"
                :loading="bulkPolicySaving"
                @click="saveAllPolicyOverrides"
              >
                {{ t('audit.logList.policy.saveAllOverrides') }}
              </t-button>
            </div>
            <div class="audit-policy-drawer__catalog">
              <div
                v-for="item in policyCatalog"
                :key="`${item.source}:${item.action_key}`"
                class="audit-policy-drawer__catalog-item"
              >
                <div class="audit-policy-drawer__catalog-meta">
                  <div class="audit-policy-drawer__catalog-title">
                    <span>{{ resolvePolicyCatalogDisplayName(item) }}</span>
                    <t-tag v-if="item.overridden" theme="warning" variant="light-outline" size="small">
                      {{ t('audit.logList.policy.overriddenTag') }}
                    </t-tag>
                    <t-tag
                      v-if="isOverrideDirty(item.source, item.action_key)"
                      theme="primary"
                      variant="light-outline"
                      size="small"
                    >
                      {{ t('audit.logList.policy.unsavedTag') }}
                    </t-tag>
                  </div>
                  <div class="audit-policy-drawer__catalog-key">{{ item.source }} / {{ item.action_key }}</div>
                  <p class="audit-policy-drawer__catalog-description">
                    {{ resolvePolicyCatalogDescription(item) }}
                  </p>
                  <div class="audit-policy-drawer__catalog-state">
                    <span>{{
                      t('audit.logList.policy.defaultState', { value: visibilityStrategyLabel(item.default_strategy) })
                    }}</span>
                    <span>{{
                      t('audit.logList.policy.effectiveState', {
                        value: visibilityStrategyLabel(item.effective_strategy),
                      })
                    }}</span>
                  </div>
                </div>
                <div class="audit-policy-drawer__catalog-actions">
                  <t-select
                    :disabled="isOverrideBusy(item.source, item.action_key)"
                    :model-value="overrideDrafts[item.source]?.[item.action_key] ?? item.effective_strategy"
                    :options="visibilityStrategyOptions"
                    @update:model-value="handleOverrideDraftChange(item.source, item.action_key, $event)"
                  />
                  <div class="audit-policy-drawer__catalog-buttons">
                    <t-button
                      theme="primary"
                      variant="outline"
                      :disabled="!isOverrideDirty(item.source, item.action_key) || bulkPolicySaving"
                      :loading="isOverrideSaving(item.source, item.action_key)"
                      @click="savePolicyOverride(item.source, item.action_key)"
                    >
                      {{ t('audit.logList.policy.saveOverride') }}
                    </t-button>
                    <t-button
                      theme="default"
                      variant="text"
                      :disabled="
                        (!item.overridden && !isOverrideDirty(item.source, item.action_key)) ||
                        isOverrideBusy(item.source, item.action_key)
                      "
                      :loading="isOverrideResetting(item.source, item.action_key)"
                      @click="resetPolicyOverride(item.source, item.action_key)"
                    >
                      {{ t('audit.logList.policy.resetOverride') }}
                    </t-button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </t-drawer>
      <t-dialog
        v-model:visible="ignoreDefaultConfirmVisible"
        theme="warning"
        :header="t('audit.logList.policy.ignoreConfirmTitle')"
        :body="t('audit.logList.policy.ignoreConfirmBody')"
        :cancel-btn="t('audit.logList.policy.ignoreConfirmCancel')"
        :confirm-btn="{ content: t('audit.logList.policy.ignoreConfirmAction'), theme: 'danger' }"
        :confirm-loading="defaultPolicySaving"
        @cancel="cancelIgnoreDefaultSave"
        @close="cancelIgnoreDefaultSave"
        @confirm="confirmIgnoreDefaultSave"
      />
    </template>
  </advanced-query-list-page>
</template>
<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next';
import { computed, nextTick, onActivated, onDeactivated, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import type { LocationQueryValue } from 'vue-router';
import { useRoute, useRouter } from 'vue-router';

import { buildAccessLogRequestLocation } from '@/modules/access-log/contract/deep-link';
import { buildAppLogLocation } from '@/modules/app-log/contract/deep-link';
import { ManagementBatchBar, TableViewToolbar } from '@/shared/components/management';
import {
  AdvancedQueryColumnDrawer,
  AdvancedQueryListPage,
  applySavedQueryViewPresentation,
  normalizeSavedQueryView,
  SavedQueryViewControl,
  useSavedQueryViews,
} from '@/shared/components/query-list';
import { describeCorrelationId, formatMessageWithCorrelation } from '@/shared/correlation';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import {
  buildRecentHoursLocalRange,
  createSingleSorter,
  decodeSorters,
  encodeSorters,
  localDateTimeToUtcIso,
  normalizePageStateRangeForRoute,
  normalizeRouteRangeForPageState,
  normalizeSorters,
  openLogDetailRow,
} from '@/shared/observability';
import { queryClient } from '@/shared/query';
import { getPermissionStore } from '@/store/modules/permission';
import { createLogger } from '@/utils/logger';

import {
  deleteAuditLogs,
  deleteAuditSavedView,
  deleteAuditVisibilityOverride,
  getAuditLogDetail,
  getAuditLogs,
  getAuditSavedViews,
  getAuditVisibilityPolicy,
  postAuditSavedView,
  putAuditSavedView,
  updateAuditVisibilityDefault,
  upsertAuditVisibilityOverride,
  upsertAuditVisibilityOverridesBatch,
} from '../../api/audit';
import AuditDetailDrawer from '../../components/AuditDetailDrawer.vue';
import AuditFilters from '../../components/AuditFilters.vue';
import AuditTable from '../../components/AuditTable.vue';
import { AUDIT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { buildAuditLogsLocation, parseAuditLogsRouteQuery } from '../../contract/deep-link';
import {
  buildAuditRelatedRecordLocation,
  buildMonitorReturnLocation,
  resolveAuditNavigationContext,
  withMonitorOrigin,
} from '../../contract/navigation';
import { AUDIT_PERMISSION_CODE } from '../../contract/permissions';
import {
  AUDIT_BUSINESS_CATEGORY,
  AUDIT_DRILLDOWN_SCOPE,
  type AuditQuickPresetKey,
  listAuditPresets,
} from '../../contract/presets';
import { AUDIT_TIME_PRESET, type AuditTimePreset } from '../../contract/time-presets';
import type { AuditFilterKey } from '../../shared/filter-definitions';
import type { AuditClientFilterState } from '../../shared/presentation';
import type {
  AppliedDrilldownScope,
  AuditDrilldownScope,
  AuditEventCatalogItem,
  AuditLogConvertibleFilters,
  AuditLogListItem,
  AuditLogQuery,
  AuditResult,
  AuditSavedViewRequest,
  AuditSortBy,
  AuditSource,
  AuditVisibilityOverrideResponse,
  AuditVisibilityScope,
  AuditVisibilityStrategy,
  DrilldownScopeProjection,
} from '../../types/audit';

// 审计日志页把服务端日志快照交给 Query cache，把筛选/排序/详情抽屉与返回监控上下文留在路由和页面状态中。

defineOptions({
  name: 'AuditLogListIndex',
});

const logger = createLogger('audit.logs');
type AuditSavedQueryState = Omit<AuditLogQuery, 'actor_user_id' | 'page' | 'page_size' | 'scope'>;
type AuditSavedQueryViewState = {
  pageSize: number;
  queryState: AuditSavedQueryState;
  visibleColumns: string[];
};
const securityEventPresetResults: AuditResult[] = ['DENIED', 'FAILED', 'ERROR'];
const DEFAULT_VISIBLE_COLUMNS = ['action', 'actor', 'resource', 'correlation', 'result', 'risk', 'created_at'];
const TROUBLESHOOTING_VISIBLE_COLUMNS = [
  'action',
  'actor',
  'resource',
  'correlation',
  'session_id',
  'result',
  'risk',
  'created_at',
];
const TECHNICAL_VISIBLE_COLUMNS = [
  'action',
  'actor',
  'resource',
  'correlation',
  'session_id',
  'ip',
  'result',
  'risk',
  'created_at',
];
const { t } = useI18n();
const route = useRoute();
const router = useRouter();

const listError = ref('');
const rows = ref<AuditLogListItem[]>([]);
const selectedRowKeys = ref<Array<string | number>>([]);
const total = ref(0);
const detailDrawerVisible = ref(false);
const detailRecord = ref<AuditLogListItem | null>(null);
const detailInitialTab = ref<'context' | 'metadata' | 'raw'>('context');
const columnDrawerVisible = ref(false);
const visibleColumnKeys = ref([...DEFAULT_VISIBLE_COLUMNS]);
const pagination = ref({
  current: 1,
  pageSize: 10,
});
const filters = ref<AuditClientFilterState>({
  ...createDefaultFilters(),
});
const routePreset = ref<AuditTimePreset | ''>('');
const routeScope = ref<AuditDrilldownScope | ''>('');
const policyDrawerVisible = ref(false);
const policyDefaultBaseline = ref<AuditVisibilityStrategy>('visible');
const policyDefaultStrategy = ref<AuditVisibilityStrategy>('visible');
const policyDefaultDirty = ref(false);
const defaultPolicySaving = ref(false);
const bulkPolicySaving = ref(false);
const ignoreDefaultConfirmVisible = ref(false);
const pendingDefaultStrategy = ref<AuditVisibilityStrategy | null>(null);
const policyCatalog = ref<AuditEventCatalogItem[]>([]);
const policyOverrides = ref<AuditVisibilityOverrideResponse[]>([]);
const overrideDrafts = ref<Record<string, Record<string, AuditVisibilityStrategy>>>({});
// dirty 集合记录用户创建显式覆盖的意图，避免默认策略变化或局部保存后的快照刷新吞掉其他草稿。
const overrideDirtyKeys = ref<Set<string>>(new Set());
const savingOverrideKeys = ref<Set<string>>(new Set());
const resettingOverrideKeys = ref<Set<string>>(new Set());
const appliedScope = ref<AppliedDrilldownScope | null>(null);
const scopeProjection = ref<DrilldownScopeProjection | null>(null);
const convertibleFilters = ref<AuditLogConvertibleFilters | null>(null);
const applyingRoute = ref(false);
const isRouteSyncActive = ref(true);
const routeHydrated = ref(false);
const navigationContext = computed(() => resolveAuditNavigationContext(route.query));
const routeAuditLogId = computed(() => firstRouteQueryValue(route.query.audit_log_id));
const monitorReturnLocation = computed(() => buildMonitorReturnLocation(route.query));
const activePreset = computed(() => inferPresetFromState(filters.value, routeScope.value));
const scopeState = computed(() =>
  appliedScope.value && scopeProjection.value
    ? {
        appliedScope: appliedScope.value,
        projection: scopeProjection.value,
        convertibleFilters: convertibleFilters.value,
      }
    : null,
);
const scopeOwnedFilterKeys = computed(() => mapOwnedFieldsToFilterKeys(appliedScope.value?.owned_fields ?? []));

const presetViews = computed(() =>
  listAuditPresets().map((preset) => ({
    key: preset.key,
    title: t(preset.titleKey),
  })),
);
const sortOptions = computed(() => [{ label: t('audit.logList.sort.createdAt'), value: 'created_at' as const }]);
const localizedScopeProjectionItems = computed(() =>
  (scopeState.value?.projection.items ?? []).map((item) => ({
    ...item,
    localizedValues: (item.values ?? [])
      .map((value) => formatScopeProjectionValue(item.key, value))
      .filter((value, index, values) => Boolean(value) && values.indexOf(value) === index),
  })),
);
const scopeConditionTags = computed(() =>
  localizedScopeProjectionItems.value.flatMap((item) => {
    const values = item.localizedValues.filter(Boolean);
    if (item.key === 'business_category' && values.length === 1) {
      return [];
    }
    return values.map((value) => `${t(item.label_key)}=${value}`);
  }),
);
const primaryScopeCondition = computed(() => scopeConditionTags.value[0] ?? '');
const columnSettingOptions = computed(() => [
  { label: t('audit.logList.columns.action'), value: 'action' },
  { label: t('audit.logList.columns.actor'), value: 'actor' },
  { label: t('audit.logList.columns.resource'), value: 'resource' },
  { label: t('audit.logList.columns.correlation'), value: 'correlation' },
  { label: t('audit.logList.columns.sessionId'), value: 'session_id' },
  { label: t('audit.logList.columns.ip'), value: 'ip' },
  { label: t('audit.logList.columns.result'), value: 'result' },
  { label: t('audit.logList.columns.risk'), value: 'risk' },
  { label: t('audit.logList.columns.createdAt'), value: 'created_at' },
]);
const columnViewPresets = computed(() => [
  { value: 'default', label: t('audit.logList.columnViews.default'), keys: DEFAULT_VISIBLE_COLUMNS },
  {
    value: 'troubleshooting',
    label: t('audit.logList.columnViews.troubleshooting'),
    keys: TROUBLESHOOTING_VISIBLE_COLUMNS,
  },
  { value: 'technical', label: t('audit.logList.columnViews.technical'), keys: TECHNICAL_VISIBLE_COLUMNS },
]);

const hasClientOnlyFilters = computed(() => false);
const canManageAuditPolicy = computed(() => getPermissionStore().hasPermission(AUDIT_PERMISSION_CODE.MANAGE));
const canDeleteAuditLogs = computed(() => getPermissionStore().hasPermission(AUDIT_PERMISSION_CODE.DELETE));
const visibilityStrategyOptions = computed(() => [
  { label: t('audit.logList.policy.strategy.visible'), value: 'visible' },
  { label: t('audit.logList.policy.strategy.hidden'), value: 'hidden' },
  { label: t('audit.logList.policy.strategy.ignore'), value: 'ignore' },
]);
const overrideDirtyCount = computed(() => overrideDirtyKeys.value.size);

const displayRows = computed(() => rows.value);
const tableTotal = computed(() => total.value);
const footerSummary = computed(() =>
  hasClientOnlyFilters.value
    ? t('audit.logList.footerFiltered', { count: displayRows.value.length })
    : t('audit.logList.footerTotal', { count: total.value }),
);
const reportDetailLoadError = (error: unknown) => {
  MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('audit.logList.loadFailed')));
};
const auditSavedViews = useSavedQueryViews<AuditSavedQueryViewState, number>({
  adapter: {
    list: async () =>
      (
        await queryClient.fetchQuery({
          queryKey: ['audit', 'saved-views'],
          queryFn: getAuditSavedViews,
        })
      ).map((view) => normalizeSavedQueryView<AuditSavedQueryState, number>(view)),
    create: async (input) => {
      const view = await postAuditSavedView(toAuditSavedViewRequest(input));
      await queryClient.invalidateQueries({ queryKey: ['audit', 'saved-views'] });
      return normalizeSavedQueryView<AuditSavedQueryState, number>(view);
    },
    update: async (id, input) => {
      const view = await putAuditSavedView(id, toAuditSavedViewRequest(input));
      await queryClient.invalidateQueries({ queryKey: ['audit', 'saved-views'] });
      return normalizeSavedQueryView<AuditSavedQueryState, number>(view);
    },
    remove: async (id) => {
      await deleteAuditSavedView(id);
      await queryClient.invalidateQueries({ queryKey: ['audit', 'saved-views'] });
    },
  },
  applyView: async (view) => {
    applyAuditSavedQueryView(view.state);
    await restoreAuditSavedQueryRoute(view.state.queryState);
  },
  onError: (error) => {
    logger.error('failed to manage audit saved view', error);
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('audit.logList.loadFailed')));
  },
  serializeCurrentState: () => ({
    pageSize: pagination.value.pageSize,
    queryState: currentAuditSavedViewQueryState(),
    visibleColumns: [...visibleColumnKeys.value],
  }),
});

const isCurrentAuditLogsRoute = computed(
  () => route.path === buildAuditLogsLocation({}).path || route.name === AUDIT_BOOTSTRAP_ROUTE.LOG_LIST.routeName,
);

function serializeRouteQuery(query: Record<string, unknown> | undefined) {
  return JSON.stringify(query ?? {});
}

function firstRouteQueryValue(value: LocationQueryValue | LocationQueryValue[] | undefined) {
  return Array.isArray(value) ? (value[0] ?? '') : (value ?? '');
}

function canSyncAuditRoute(reason: string) {
  const allowed = isRouteSyncActive.value && isCurrentAuditLogsRoute.value;

  if (!allowed) {
    logger.debug('skip audit route sync while page is inactive or route changed', {
      reason,
      routePath: route.path,
      routeName: route.name,
      isRouteSyncActive: isRouteSyncActive.value,
      isCurrentAuditLogsRoute: isCurrentAuditLogsRoute.value,
      query: route.query,
    });
  }

  return allowed;
}

function buildQuery(): AuditLogQuery {
  const normalizedSorters = normalizeSorters(filters.value.sorters, sortOptions.value);
  const query: AuditLogQuery = {
    page: pagination.value.current,
    page_size: pagination.value.pageSize,
  };
  if (routePreset.value) {
    query.preset = routePreset.value;
  }
  if (routeScope.value) {
    query.scope = routeScope.value;
  }
  query.visibility_scope = filters.value.visibilityScope;
  if (filters.value.keyword) {
    query.keyword = filters.value.keyword;
  }
  if (filters.value.actor) {
    query.actor = filters.value.actor;
  }
  if (filters.value.action) {
    query.action = filters.value.action;
  }
  if (filters.value.actionPrefix) {
    query.action_prefix = filters.value.actionPrefix;
  }
  if (filters.value.actionPrefixes.length) {
    query.action_prefixes = [...filters.value.actionPrefixes];
  }
  if (filters.value.actionKeywords.length) {
    query.action_keywords = [...filters.value.actionKeywords];
  }
  if (filters.value.source) {
    query.source = filters.value.source as AuditLogQuery['source'];
  }
  if (filters.value.businessCategory) {
    query.business_category = filters.value.businessCategory;
  }
  if (filters.value.resourceType) {
    query.resource_type = filters.value.resourceType;
  }
  if (filters.value.resourceTypes.length) {
    query.resource_types = [...filters.value.resourceTypes];
  }
  if (filters.value.resourceName) {
    query.resource_name = filters.value.resourceName;
  }
  if (filters.value.requestId) {
    query.request_id = filters.value.requestId;
  }
  if (filters.value.resourceId) {
    query.resource_id = filters.value.resourceId;
  }
  if (filters.value.result !== 'all') {
    query.result = filters.value.result;
  }
  if (filters.value.results.length) {
    query.results = [...filters.value.results];
  }
  if (filters.value.riskLevel !== 'all') {
    query.risk_level = filters.value.riskLevel;
  }
  if (filters.value.riskLevels.length) {
    query.risk_levels = [...filters.value.riskLevels];
  }
  if (filters.value.success !== 'all') {
    query.success = filters.value.success === 'true';
  }
  if (filters.value.session) {
    query.session_id = filters.value.session;
  }
  if (filters.value.requestPathPrefixes.length) {
    query.request_path_prefixes = [...filters.value.requestPathPrefixes];
  }
  const explicitCreatedRange = filters.value.createdRange;
  if (explicitCreatedRange[0]) {
    query.created_from = localDateTimeToUtcIso(explicitCreatedRange[0]);
  }
  if (explicitCreatedRange[1]) {
    query.created_to = localDateTimeToUtcIso(explicitCreatedRange[1]);
  }
  const encodedSorters = encodeSorters(normalizedSorters, sortOptions.value);
  if (encodedSorters.length) {
    query.sort = encodedSorters;
  }

  return query;
}

const auditLogListQuery = useQuery(
  {
    queryKey: computed(() => ['audit', 'log-list', buildQuery()]),
    queryFn: () => getAuditLogs(buildQuery()),
    enabled: computed(() => routeHydrated.value && canSyncAuditRoute('query-enabled')),
  },
  queryClient,
);
const loading = computed(() => auditLogListQuery.isFetching.value);

watch(auditLogListQuery.data, async (response) => {
  if (!response) return;
  listError.value = '';
  rows.value = response.items;
  total.value = response.total;
  appliedScope.value = response.applied_scope ?? null;
  scopeProjection.value = response.scope_projection ?? null;
  convertibleFilters.value = response.convertible_filters ?? null;
  await openRouteAuditLog();
});

watch(auditLogListQuery.error, (error) => {
  if (!error) return;
  logger.error('failed to fetch audit logs', error);
  listError.value = resolveLocalizedErrorMessage(t, error, t('audit.logList.loadFailed'));
  const correlationId = filters.value.requestId;
  MessagePlugin.error(
    correlationId
      ? formatMessageWithCorrelation(listError.value, describeCorrelationId(t, correlationId))
      : listError.value,
  );
});

async function fetchAuditLogs() {
  listError.value = '';
  await nextTick();
  await auditLogListQuery.refetch();
}

function handleSelectChange(rowKeys: Array<string | number>) {
  const pageIds = new Set(rows.value.map((row) => row.id));
  const preserved = selectedRowKeys.value.filter((key) => !pageIds.has(Number(key)));
  selectedRowKeys.value = [...preserved, ...rowKeys];
}

function clearSelection() {
  selectedRowKeys.value = [];
}

function selectCurrentPage() {
  handleSelectChange([...selectedRowKeys.value, ...rows.value.map((row) => row.id)]);
}

function invertCurrentPage() {
  const pageIds = new Set(rows.value.map((row) => row.id));
  const selected = new Set(selectedRowKeys.value);
  const next = selectedRowKeys.value.filter((key) => !pageIds.has(Number(key)));
  rows.value.forEach((row) => {
    if (!selected.has(row.id)) next.push(row.id);
  });
  selectedRowKeys.value = next;
}

function createAuditDeleteIdempotencyKey() {
  return globalThis.crypto?.randomUUID?.() ?? `audit-delete-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

function confirmBatchDelete() {
  if (!canDeleteAuditLogs.value || selectedRowKeys.value.length === 0) return;
  const dialog = DialogPlugin.confirm({
    header: t('audit.logList.batch.confirmTitle'),
    body: t('audit.logList.batch.confirmBody', { count: selectedRowKeys.value.length }),
    theme: 'danger',
    confirmBtn: t('audit.logList.batch.confirm'),
    cancelBtn: t('audit.logList.batch.cancel'),
    onConfirm: async () => {
      dialog.setConfirmLoading(true);
      try {
        await deleteAuditLogs({ ids: selectedRowKeys.value.map(Number) }, createAuditDeleteIdempotencyKey());
        clearSelection();
        await queryClient.invalidateQueries({ queryKey: ['audit', 'log-list'] });
        await fetchAuditLogs();
        dialog.hide();
        MessagePlugin.success(t('audit.logList.batch.success'));
      } catch (error) {
        MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('audit.logList.batch.failed')));
      } finally {
        dialog.setConfirmLoading(false);
      }
    },
  });
}

async function openPolicyDrawer() {
  if (!canManageAuditPolicy.value) {
    return;
  }

  try {
    await loadPolicySnapshot();
    policyDrawerVisible.value = true;
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('audit.logList.loadFailed')));
  }
}

async function invalidateAuditPolicyQueries() {
  await queryClient.invalidateQueries({ queryKey: ['audit', 'visibility-policy'] });
  await queryClient.invalidateQueries({ queryKey: ['audit', 'log-list'] });
}

function handlePolicyDefaultChange(value: string | number | undefined) {
  const strategy = normalizeOverrideStrategy(value);
  if (!strategy) {
    return;
  }
  policyDefaultStrategy.value = strategy;
  policyDefaultDirty.value = strategy !== policyDefaultBaseline.value;
}

function requestSavePolicyDefault() {
  if (!policyDefaultDirty.value || defaultPolicySaving.value || bulkPolicySaving.value) {
    return;
  }
  const strategy = policyDefaultStrategy.value;
  if (strategy === 'ignore') {
    pendingDefaultStrategy.value = strategy;
    ignoreDefaultConfirmVisible.value = true;
    return;
  }
  void savePolicyDefault(strategy);
}

function cancelIgnoreDefaultSave() {
  if (defaultPolicySaving.value) {
    return;
  }
  ignoreDefaultConfirmVisible.value = false;
  pendingDefaultStrategy.value = null;
}

async function confirmIgnoreDefaultSave() {
  const strategy = pendingDefaultStrategy.value;
  if (strategy !== 'ignore') {
    cancelIgnoreDefaultSave();
    return;
  }
  await savePolicyDefault(strategy);
}

async function savePolicyDefault(strategy: AuditVisibilityStrategy) {
  if (defaultPolicySaving.value) {
    return;
  }
  defaultPolicySaving.value = true;
  try {
    await updateAuditVisibilityDefault({ strategy });
    policyDefaultDirty.value = false;
    policyDefaultBaseline.value = strategy;
    ignoreDefaultConfirmVisible.value = false;
    pendingDefaultStrategy.value = null;
    await invalidateAuditPolicyQueries();
    await loadPolicySnapshot();
    MessagePlugin.success(t('audit.logList.policy.saveSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('audit.logList.policy.saveFailed')));
  } finally {
    defaultPolicySaving.value = false;
  }
}

async function loadPolicySnapshot() {
  const response = await queryClient.fetchQuery({
    queryKey: ['audit', 'visibility-policy'],
    queryFn: getAuditVisibilityPolicy,
    staleTime: 0,
  });
  policyDefaultBaseline.value = response.default.strategy;
  if (!policyDefaultDirty.value) {
    policyDefaultStrategy.value = response.default.strategy;
  }
  policyCatalog.value = response.catalog;
  policyOverrides.value = response.overrides;
  overrideDrafts.value = mergeOverrideDrafts(response.catalog, response.overrides);
}

function mergeOverrideDrafts(
  catalog: AuditEventCatalogItem[],
  overrides: AuditVisibilityOverrideResponse[],
): Record<string, Record<string, AuditVisibilityStrategy>> {
  const drafts: Record<string, Record<string, AuditVisibilityStrategy>> = {};
  const overrideIndex = new Map(overrides.map((item) => [`${item.source}:${item.action_key}`, item.strategy]));

  catalog.forEach((item) => {
    const sourceDrafts = (drafts[item.source] ??= {});
    const key = policyOverrideKey(item.source, item.action_key);
    sourceDrafts[item.action_key] =
      overrideDirtyKeys.value.has(key) && overrideDrafts.value[item.source]?.[item.action_key]
        ? overrideDrafts.value[item.source][item.action_key]
        : (overrideIndex.get(key) ?? item.effective_strategy);
  });

  return drafts;
}

function normalizePolicyCatalogKeyFragment(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '');
}

function buildPolicyCatalogLocaleKey(item: AuditEventCatalogItem, field: 'displayName' | 'description') {
  const source = normalizePolicyCatalogKeyFragment(item.source);
  const actionKey = normalizePolicyCatalogKeyFragment(item.action_key);
  if (!source || !actionKey) {
    return '';
  }
  return `audit.logList.policy.catalog.${source}.${actionKey}.${field}`;
}

function resolvePolicyCatalogDisplayName(item: AuditEventCatalogItem) {
  const localeKey = buildPolicyCatalogLocaleKey(item, 'displayName');
  if (localeKey) {
    const localized = t(localeKey);
    if (localized !== localeKey) {
      return localized;
    }
  }
  return item.display_name || item.action_key;
}

function resolvePolicyCatalogDescription(item: AuditEventCatalogItem) {
  const localeKey = buildPolicyCatalogLocaleKey(item, 'description');
  if (localeKey) {
    const localized = t(localeKey);
    if (localized !== localeKey) {
      return localized;
    }
  }
  return item.description || t('audit.logList.policy.descriptionFallback');
}

function handleOverrideDraftChange(source: string, actionKey: string, value: string | number | undefined) {
  const next = normalizeOverrideStrategy(value);
  if (!next) {
    return;
  }
  overrideDrafts.value = {
    ...overrideDrafts.value,
    [source]: {
      ...(overrideDrafts.value[source] ?? {}),
      [actionKey]: next,
    },
  };
  overrideDirtyKeys.value = new Set(overrideDirtyKeys.value).add(policyOverrideKey(source, actionKey));
}

function normalizeOverrideStrategy(value: string | number | undefined): AuditVisibilityStrategy | '' {
  if (value === 'visible' || value === 'hidden' || value === 'ignore') {
    return value;
  }
  return '';
}

async function savePolicyOverride(source: AuditSource, actionKey: string) {
  const key = policyOverrideKey(source, actionKey);
  const strategy = overrideDrafts.value[source]?.[actionKey];
  const catalogItem = policyCatalog.value.find((item) => item.source === source && item.action_key === actionKey);
  if (!strategy || !catalogItem || !overrideDirtyKeys.value.has(key) || isOverrideBusy(source, actionKey)) {
    return;
  }

  savingOverrideKeys.value = new Set(savingOverrideKeys.value).add(key);
  try {
    await upsertAuditVisibilityOverride({
      source,
      action_key: actionKey,
      strategy,
      description: catalogItem.description,
    });
    clearOverrideDirtyKey(key);
    await invalidateAuditPolicyQueries();
    await loadPolicySnapshot();
    MessagePlugin.success(t('audit.logList.policy.saveOverrideSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('audit.logList.policy.saveOverrideFailed')));
  } finally {
    const next = new Set(savingOverrideKeys.value);
    next.delete(key);
    savingOverrideKeys.value = next;
  }
}

async function saveAllPolicyOverrides() {
  if (!overrideDirtyKeys.value.size || bulkPolicySaving.value || defaultPolicySaving.value) {
    return;
  }

  const submittedKeys = new Set(overrideDirtyKeys.value);
  const items = policyCatalog.value.flatMap((item) => {
    const key = policyOverrideKey(item.source, item.action_key);
    const strategy = overrideDrafts.value[item.source]?.[item.action_key];
    if (!submittedKeys.has(key) || !strategy) {
      return [];
    }
    return [
      {
        source: item.source,
        action_key: item.action_key,
        strategy,
        description: item.description,
      },
    ];
  });
  if (!items.length) {
    return;
  }
  const submittedItemKeys = new Set(items.map((item) => policyOverrideKey(item.source, item.action_key)));

  bulkPolicySaving.value = true;
  try {
    await upsertAuditVisibilityOverridesBatch({ items });
    submittedItemKeys.forEach(clearOverrideDirtyKey);
    await invalidateAuditPolicyQueries();
    await loadPolicySnapshot();
    MessagePlugin.success(t('audit.logList.policy.saveAllSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('audit.logList.policy.saveAllFailed')));
  } finally {
    bulkPolicySaving.value = false;
  }
}

async function resetPolicyOverride(source: AuditSource, actionKey: string) {
  const key = policyOverrideKey(source, actionKey);
  const persisted = policyOverrides.value.some((item) => item.source === source && item.action_key === actionKey);
  if (!persisted) {
    clearOverrideDirtyKey(key);
    const catalogItem = policyCatalog.value.find((item) => item.source === source && item.action_key === actionKey);
    if (catalogItem) {
      overrideDrafts.value = {
        ...overrideDrafts.value,
        [source]: {
          ...(overrideDrafts.value[source] ?? {}),
          [actionKey]: catalogItem.effective_strategy,
        },
      };
    }
    return;
  }
  if (isOverrideBusy(source, actionKey)) {
    return;
  }

  resettingOverrideKeys.value = new Set(resettingOverrideKeys.value).add(key);
  try {
    await deleteAuditVisibilityOverride(source, actionKey);
    clearOverrideDirtyKey(key);
    await invalidateAuditPolicyQueries();
    await loadPolicySnapshot();
    MessagePlugin.success(t('audit.logList.policy.resetOverrideSuccess'));
  } catch (error) {
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('audit.logList.policy.resetOverrideFailed')));
  } finally {
    const next = new Set(resettingOverrideKeys.value);
    next.delete(key);
    resettingOverrideKeys.value = next;
  }
}

function policyOverrideKey(source: string, actionKey: string) {
  return `${source}:${actionKey}`;
}

function clearOverrideDirtyKey(key: string) {
  const next = new Set(overrideDirtyKeys.value);
  next.delete(key);
  overrideDirtyKeys.value = next;
}

function isOverrideDirty(source: string, actionKey: string) {
  return overrideDirtyKeys.value.has(policyOverrideKey(source, actionKey));
}

function isOverrideSaving(source: string, actionKey: string) {
  return savingOverrideKeys.value.has(policyOverrideKey(source, actionKey));
}

function isOverrideResetting(source: string, actionKey: string) {
  return resettingOverrideKeys.value.has(policyOverrideKey(source, actionKey));
}

function isOverrideBusy(source: string, actionKey: string) {
  return bulkPolicySaving.value || isOverrideSaving(source, actionKey) || isOverrideResetting(source, actionKey);
}

function visibilityStrategyLabel(strategy: AuditVisibilityStrategy) {
  return t(`audit.logList.policy.strategy.${strategy}`);
}

function applyPreset(preset: AuditQuickPresetKey) {
  filters.value = createDefaultFilters();
  routePreset.value = resolvePresetTimeWindow(preset);
  routeScope.value = '';
  applyQuickPresetFilters(preset);
  pagination.value.current = 1;
  updateRouteQuery();
}

function handleSearch() {
  pagination.value.current = 1;
  updateRouteQuery();
}

function resetFilters() {
  filters.value = createDefaultFilters();
  auditSavedViews.selectedId.value = undefined;
  routePreset.value = '';
  routeScope.value = scopeState.value ? routeScope.value : '';
  pagination.value.current = 1;
  updateRouteQuery();
}

function exitDrilldown() {
  routeScope.value = '';
  pagination.value.current = 1;
  updateRouteQuery();
}

function convertScopeToFilters() {
  if (!convertibleFilters.value) {
    return;
  }

  routeScope.value = '';
  routePreset.value = convertibleFilters.value.preset ?? routePreset.value;
  applyConvertibleFilters(convertibleFilters.value);
  pagination.value.current = 1;
  updateRouteQuery();
}

function createDefaultFilters(): AuditClientFilterState {
  return {
    keyword: '',
    visibilityScope: 'default',
    actor: '',
    success: 'all',
    action: '',
    actionPrefix: '',
    actionPrefixes: [],
    actionKeywords: [],
    requestPathPrefixes: [],
    source: '',
    businessCategory: '',
    createdRange: [],
    resourceType: '',
    resourceTypes: [],
    resourceName: '',
    resourceId: '',
    result: 'all',
    results: [],
    riskLevel: 'all',
    riskLevels: [],
    session: '',
    requestId: '',
    sorters: createSingleSorter('created_at', 'desc'),
  };
}

async function openRouteAuditLog() {
  const rawAuditLogID = String(routeAuditLogId.value ?? '').trim();
  if (!/^[1-9]\d*$/.test(rawAuditLogID)) {
    return;
  }
  const auditLogId = Number(rawAuditLogID);

  const row = rows.value.find((item) => item.id === auditLogId);
  if (row) {
    await openDetailDrawer(row);
    return;
  }

  await openLogDetailRow(
    { id: auditLogId },
    (id) => queryClient.fetchQuery({ queryKey: ['audit', 'log-detail', id], queryFn: () => getAuditLogDetail(id) }),
    detailRecord,
    detailDrawerVisible,
    reportDetailLoadError,
  );
}

async function openDetailDrawer(row: AuditLogListItem) {
  detailInitialTab.value = 'context';
  await openLogDetailRow(
    row,
    (id) => queryClient.fetchQuery({ queryKey: ['audit', 'log-detail', id], queryFn: () => getAuditLogDetail(id) }),
    detailRecord,
    detailDrawerVisible,
    reportDetailLoadError,
  );
}

function auditRequestId(row: AuditLogListItem) {
  return row.request_id || '';
}

function openAccessLog(row: AuditLogListItem) {
  const requestId = auditRequestId(row);
  if (!requestId) {
    return;
  }

  void router.push(withMonitorOrigin(buildAccessLogRequestLocation(requestId), navigationContext.value.monitorOrigin));
}

function openAppLog(row: AuditLogListItem) {
  const requestId = auditRequestId(row);
  if (!requestId) {
    return;
  }

  void router.push(
    withMonitorOrigin(buildAppLogLocation({ request_id: requestId }), navigationContext.value.monitorOrigin),
  );
}

function openSecurityEvent(row: AuditLogListItem) {
  void router.push(buildAuditRelatedRecordLocation(row, navigationContext.value.monitorOrigin));
}

function applyRouteFilters() {
  const query = parseAuditLogsRouteQuery(route.query);
  routePreset.value = normalizePreset(query.preset);
  routeScope.value = normalizeScope(query.scope);
  const nextFilters: AuditClientFilterState = {
    ...createDefaultFilters(),
    visibilityScope: normalizeVisibilityScope(query.visibility_scope),
    keyword: query.keyword ?? '',
    actor: query.actor ?? '',
    success: query.success === 'true' ? 'true' : query.success === 'false' ? 'false' : 'all',
    action: query.action || '',
    actionPrefix: query.action_prefix || '',
    actionPrefixes: splitRouteList(query.action_prefixes),
    actionKeywords: splitRouteList(query.action_keywords),
    requestPathPrefixes: splitRouteList(query.request_path_prefixes),
    source: query.source || '',
    businessCategory: normalizeBusinessCategory(query.business_category),
    createdRange: normalizeRouteRangeForPageState([query.created_from ?? '', query.created_to ?? '']),
    resourceType: query.resource_type || '',
    resourceTypes: splitRouteList(query.resource_types),
    resourceName: query.resource_name ?? '',
    resourceId: query.resource_id ?? '',
    result: (query.result as AuditClientFilterState['result']) || 'all',
    results: splitRouteList(query.results) as AuditClientFilterState['results'],
    riskLevel: (query.risk_level as AuditClientFilterState['riskLevel']) || 'all',
    riskLevels: splitRouteList(query.risk_levels) as AuditClientFilterState['riskLevels'],
    session: query.session ?? '',
    requestId: query.request_id ?? '',
    sorters: (() => {
      const parsed = normalizeSorters(
        decodeSorters(query.sort, normalizeSortField, normalizeSortOrder),
        sortOptions.value,
      );
      return parsed.length ? parsed : createSingleSorter('created_at', 'desc');
    })(),
  };
  filters.value = nextFilters;
  routeHydrated.value = true;
}

function buildRouteQuery() {
  const normalizedSorters = normalizeSorters(filters.value.sorters, sortOptions.value);
  const explicitCreatedRange = filters.value.createdRange;
  const [createdFrom = '', createdTo = ''] = normalizePageStateRangeForRoute(explicitCreatedRange);

  return {
    audit_log_id: routeAuditLogId.value,
    preset: routePreset.value,
    scope: routeScope.value,
    visibility_scope: filters.value.visibilityScope,
    keyword: filters.value.keyword,
    actor: filters.value.actor,
    success: filters.value.success === 'all' ? '' : filters.value.success,
    action: filters.value.action,
    action_prefix: filters.value.actionPrefix,
    action_prefixes: joinRouteList(filters.value.actionPrefixes),
    action_keywords: joinRouteList(filters.value.actionKeywords),
    request_path_prefixes: joinRouteList(filters.value.requestPathPrefixes),
    source: filters.value.source,
    business_category: filters.value.businessCategory,
    created_from: createdFrom,
    created_to: createdTo,
    resource_type: filters.value.resourceType,
    resource_types: joinRouteList(filters.value.resourceTypes),
    resource_name: filters.value.resourceName,
    resource_id: filters.value.resourceId,
    result: filters.value.result === 'all' ? '' : filters.value.result,
    results: joinRouteList(filters.value.results),
    risk_level: filters.value.riskLevel === 'all' ? '' : filters.value.riskLevel,
    risk_levels: joinRouteList(filters.value.riskLevels),
    session: filters.value.session,
    request_id: filters.value.requestId,
    sort: encodeSorters(normalizedSorters, sortOptions.value),
  };
}

function currentAuditSavedViewQueryState(): AuditSavedQueryState {
  const { actor_user_id: _actorUserID, page: _page, page_size: _pageSize, scope: _scope, ...query } = buildQuery();
  return query;
}

function toAuditSavedViewRequest(input: {
  name: string;
  isDefault: boolean;
  state: AuditSavedQueryViewState;
}): AuditSavedViewRequest {
  return {
    name: input.name,
    page_size: input.state.pageSize,
    query_state: input.state.queryState as unknown as Record<string, unknown>,
    visible_columns: input.state.visibleColumns,
    is_default: input.isDefault,
  };
}

function applyAuditSavedQueryView(savedState: AuditSavedQueryViewState) {
  applySavedQueryViewPresentation(savedState, {
    pagination: pagination.value,
    supportedColumns: columnSettingOptions.value.map((column) => column.value),
    visibleColumnKeys,
  });
}

async function restoreAuditSavedQueryRoute(query: AuditSavedQueryState) {
  const location = withMonitorOrigin(
    buildAuditLogsLocation({
      preset: query.preset,
      visibility_scope: query.visibility_scope,
      keyword: query.keyword,
      actor: query.actor,
      success: query.success === undefined ? '' : String(query.success),
      action: query.action,
      action_prefix: query.action_prefix,
      action_prefixes: joinRouteList(query.action_prefixes ?? []),
      action_keywords: joinRouteList(query.action_keywords ?? []),
      request_path_prefixes: joinRouteList(query.request_path_prefixes ?? []),
      source: query.source,
      business_category: query.business_category,
      created_from: query.created_from,
      created_to: query.created_to,
      resource_type: query.resource_type,
      resource_types: joinRouteList(query.resource_types ?? []),
      resource_name: query.resource_name,
      resource_id: query.resource_id,
      result: query.result,
      results: joinRouteList(query.results ?? []),
      risk_level: query.risk_level,
      risk_levels: joinRouteList(query.risk_levels ?? []),
      session: query.session_id,
      request_id: query.request_id,
      sort: query.sort,
    }),
    navigationContext.value.monitorOrigin,
  );
  const currentLocation = withMonitorOrigin(buildAuditLogsLocation(route.query), navigationContext.value.monitorOrigin);
  if (serializeRouteQuery(location.query) === serializeRouteQuery(currentLocation.query)) {
    await syncFromCurrentRoute('saved-view-apply');
    return;
  }
  await router.replace(location);
}

async function updateRouteQuery() {
  if (applyingRoute.value) {
    return;
  }
  if (!canSyncAuditRoute('interactive-filter-sync')) {
    return;
  }

  const nextLocation = withMonitorOrigin(
    buildAuditLogsLocation(buildRouteQuery()),
    navigationContext.value.monitorOrigin,
  );
  const currentLocation = withMonitorOrigin(buildAuditLogsLocation(route.query), navigationContext.value.monitorOrigin);

  if (serializeRouteQuery(nextLocation.query) === serializeRouteQuery(currentLocation.query)) {
    await fetchAuditLogs();
    return;
  }

  logger.debug('replace audit route query from interactive filters', {
    reason: 'interactive-filter-sync',
    routePath: route.path,
    routeName: route.name,
    currentQuery: currentLocation.query,
    nextQuery: nextLocation.query,
  });
  await router.replace(nextLocation);
}

async function syncFromCurrentRoute(reason: string) {
  logger.debug('observe route query change for audit logs', {
    reason,
    routePath: route.path,
    routeName: route.name,
    isRouteSyncActive: isRouteSyncActive.value,
    isCurrentAuditLogsRoute: isCurrentAuditLogsRoute.value,
    applyingRoute: applyingRoute.value,
    query: route.query,
  });
  if (!canSyncAuditRoute(reason)) {
    return;
  }

  applyingRoute.value = true;
  try {
    applyRouteFilters();
  } finally {
    applyingRoute.value = false;
  }
  pagination.value.current = 1;
  const canonicalLocation = withMonitorOrigin(
    buildAuditLogsLocation(buildRouteQuery()),
    navigationContext.value.monitorOrigin,
  );
  const currentLocation = withMonitorOrigin(buildAuditLogsLocation(route.query), navigationContext.value.monitorOrigin);
  if (serializeRouteQuery(canonicalLocation.query) !== serializeRouteQuery(currentLocation.query)) {
    logger.debug('canonicalize audit route query after route change', {
      reason,
      routePath: route.path,
      routeName: route.name,
      currentQuery: currentLocation.query,
      canonicalQuery: canonicalLocation.query,
    });
    await router.replace(canonicalLocation);
    return;
  }
}

watch(
  () => route.query,
  async () => {
    await syncFromCurrentRoute('route-query-watch');
  },
  { immediate: true },
);

onMounted(() => {
  const routeQuery = parseAuditLogsRouteQuery(route.query);
  const hasExplicitState = Object.values(routeQuery).some((value) =>
    Array.isArray(value) ? value.length > 0 : Boolean(value),
  );
  void auditSavedViews.load({ hasExplicitState });
});

onActivated(() => {
  isRouteSyncActive.value = true;
  void syncFromCurrentRoute('route-activated');
});

onDeactivated(() => {
  isRouteSyncActive.value = false;
});

function returnToMonitor() {
  if (!monitorReturnLocation.value) {
    return;
  }

  void router.push(monitorReturnLocation.value);
}

function normalizeSortOrder(value: string) {
  return value === 'asc' ? 'asc' : 'desc';
}

function normalizeSortField(value: string): AuditSortBy | '' {
  return value === 'created_at' ? 'created_at' : '';
}

function normalizePreset(value?: string) {
  return value === AUDIT_TIME_PRESET.LAST_24H ||
    value === AUDIT_TIME_PRESET.LAST_7D ||
    value === AUDIT_TIME_PRESET.LAST_30D
    ? value
    : '';
}

function normalizeScope(value?: string): AuditDrilldownScope | '' {
  switch (value) {
    case AUDIT_DRILLDOWN_SCOPE.FAILED_OPERATIONS:
    case AUDIT_DRILLDOWN_SCOPE.HIGH_RISK_OPERATIONS:
    case AUDIT_DRILLDOWN_SCOPE.SENSITIVE_OPERATIONS:
    case AUDIT_DRILLDOWN_SCOPE.AUTH_FAILURES:
    case AUDIT_DRILLDOWN_SCOPE.PERMISSION_DENIALS:
    case AUDIT_DRILLDOWN_SCOPE.RBAC_CHANGES:
    case AUDIT_DRILLDOWN_SCOPE.CRITICAL_SECURITY:
      return value;
    default:
      return '';
  }
}

function normalizeBusinessCategory(value?: string): AuditClientFilterState['businessCategory'] {
  switch (value) {
    case AUDIT_BUSINESS_CATEGORY.FAILED_OPERATIONS:
    case AUDIT_BUSINESS_CATEGORY.HIGH_RISK_OPERATIONS:
    case AUDIT_BUSINESS_CATEGORY.SENSITIVE_OPERATIONS:
    case AUDIT_BUSINESS_CATEGORY.AUTH_FAILURES:
    case AUDIT_BUSINESS_CATEGORY.PERMISSION_DENIALS:
    case AUDIT_BUSINESS_CATEGORY.RBAC_CHANGES:
    case AUDIT_BUSINESS_CATEGORY.CRITICAL_SECURITY:
      return value;
    default:
      return '';
  }
}

function normalizeVisibilityScope(value?: string): AuditVisibilityScope {
  if (!canManageAuditPolicy.value) {
    return 'default';
  }
  switch (value) {
    case 'all':
    case 'hidden_only':
      return value;
    default:
      return 'default';
  }
}

function applyConvertibleFilters(next: AuditLogConvertibleFilters) {
  filters.value = {
    ...filters.value,
    source: next.source ?? '',
    businessCategory: normalizeBusinessCategory(next.business_category),
    success: next.success === true ? 'true' : next.success === false ? 'false' : 'all',
    actionPrefixes: next.action_prefixes ? [...next.action_prefixes] : [],
    actionKeywords: next.action_keywords ? [...next.action_keywords] : [],
    resourceTypes: next.resource_types ? [...next.resource_types] : [],
    requestPathPrefixes: next.request_path_prefixes ? [...next.request_path_prefixes] : [],
    results: next.results ? [...next.results] : [],
    riskLevels: next.risk_levels ? [...next.risk_levels] : [],
  };
}

function mapOwnedFieldsToFilterKeys(fields: string[]) {
  const mapped: AuditFilterKey[] = [];

  fields.forEach((field) => {
    switch (field) {
      case 'business_category':
        mapped.push('businessCategory');
        break;
      case 'action_keywords':
        mapped.push('actionKeywords');
        break;
      case 'action_prefixes':
        mapped.push('actionPrefixes');
        break;
      case 'resource_types':
        mapped.push('resourceTypes');
        break;
      case 'request_path_prefixes':
        mapped.push('requestPathPrefixes');
        break;
      case 'results':
        mapped.push('results');
        break;
      case 'risk_levels':
        mapped.push('riskLevels');
        break;
      case 'source':
        mapped.push('source');
        break;
      case 'success':
        mapped.push('success');
        break;
      default:
        break;
    }
  });

  return mapped;
}

function splitRouteList(value: string | undefined) {
  if (!value) {
    return [];
  }

  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean);
}

function joinRouteList(values: string[]) {
  return values.length ? values.join(',') : '';
}

function inferPresetFromState(value: AuditClientFilterState, scope: string): AuditQuickPresetKey {
  const isSecurityEventPreset =
    !scope &&
    value.source === 'SECURITY_EVENT' &&
    !value.businessCategory &&
    value.results.length === securityEventPresetResults.length &&
    securityEventPresetResults.every((result) => value.results.includes(result));
  if (isSecurityEventPreset) {
    return 'security-events';
  }
  if (scope === AUDIT_DRILLDOWN_SCOPE.FAILED_OPERATIONS) {
    return 'failed-operations';
  }
  if (scope === AUDIT_DRILLDOWN_SCOPE.RBAC_CHANGES) {
    return 'rbac-changes';
  }
  if (scope === AUDIT_DRILLDOWN_SCOPE.PERMISSION_DENIALS) {
    return 'permission-denied';
  }
  if (scope === AUDIT_DRILLDOWN_SCOPE.SENSITIVE_OPERATIONS) {
    return 'sensitive-ops';
  }
  if (scope === AUDIT_DRILLDOWN_SCOPE.AUTH_FAILURES) {
    return 'auth-failed';
  }
  if (scope === AUDIT_DRILLDOWN_SCOPE.HIGH_RISK_OPERATIONS || scope === AUDIT_DRILLDOWN_SCOPE.CRITICAL_SECURITY) {
    return 'high-risk';
  }
  if (value.businessCategory === AUDIT_BUSINESS_CATEGORY.FAILED_OPERATIONS) {
    return 'failed-operations';
  }
  if (value.businessCategory === AUDIT_BUSINESS_CATEGORY.RBAC_CHANGES) {
    return 'rbac-changes';
  }
  if (value.businessCategory === AUDIT_BUSINESS_CATEGORY.PERMISSION_DENIALS) {
    return 'permission-denied';
  }
  if (value.businessCategory === AUDIT_BUSINESS_CATEGORY.SENSITIVE_OPERATIONS) {
    return 'sensitive-ops';
  }
  if (value.businessCategory === AUDIT_BUSINESS_CATEGORY.AUTH_FAILURES) {
    return 'auth-failed';
  }
  if (
    value.businessCategory === AUDIT_BUSINESS_CATEGORY.HIGH_RISK_OPERATIONS &&
    value.actionPrefix === 'ops.container.action.' &&
    !value.resourceType &&
    value.resourceTypes.includes('container') &&
    value.resourceTypes.includes('container_batch')
  ) {
    return 'container-dangerous-ops';
  }
  if (
    value.businessCategory === AUDIT_BUSINESS_CATEGORY.HIGH_RISK_OPERATIONS ||
    value.businessCategory === AUDIT_BUSINESS_CATEGORY.CRITICAL_SECURITY
  ) {
    return 'high-risk';
  }
  return 'all';
}

function applyQuickPresetFilters(preset: AuditQuickPresetKey) {
  const createdRange = buildPresetCreatedRange(routePreset.value);
  filters.value.createdRange = createdRange;

  switch (preset) {
    case 'security-events':
      filters.value.source = 'SECURITY_EVENT';
      filters.value.results = ['DENIED', 'FAILED', 'ERROR'];
      return;
    case 'failed-operations':
      filters.value.result = 'FAILED';
      filters.value.businessCategory = AUDIT_BUSINESS_CATEGORY.FAILED_OPERATIONS;
      return;
    case 'rbac-changes':
      filters.value.businessCategory = AUDIT_BUSINESS_CATEGORY.RBAC_CHANGES;
      return;
    case 'permission-denied':
      filters.value.result = 'DENIED';
      filters.value.businessCategory = AUDIT_BUSINESS_CATEGORY.PERMISSION_DENIALS;
      return;
    case 'sensitive-ops':
      filters.value.businessCategory = AUDIT_BUSINESS_CATEGORY.SENSITIVE_OPERATIONS;
      return;
    case 'auth-failed':
      filters.value.businessCategory = AUDIT_BUSINESS_CATEGORY.AUTH_FAILURES;
      return;
    case 'container-dangerous-ops':
      filters.value.actionPrefix = 'ops.container.action.';
      filters.value.businessCategory = AUDIT_BUSINESS_CATEGORY.HIGH_RISK_OPERATIONS;
      filters.value.resourceTypes = ['container', 'container_batch'];
      filters.value.riskLevels = ['HIGH'];
      return;
    case 'high-risk':
      filters.value.riskLevels = ['HIGH', 'CRITICAL'];
      filters.value.businessCategory = AUDIT_BUSINESS_CATEGORY.HIGH_RISK_OPERATIONS;
      return;
    default:
      return;
  }
}

function buildPresetCreatedRange(preset: AuditTimePreset | '') {
  const now = new Date();
  switch (preset) {
    case 'last_24h':
      return buildRecentHoursLocalRange(now, 24);
    case 'last_7d':
      return buildRecentHoursLocalRange(now, 24 * 7);
    case 'last_30d':
      return buildRecentHoursLocalRange(now, 24 * 30);
    default:
      return [];
  }
}

function resolvePresetTimeWindow(preset: AuditQuickPresetKey): AuditTimePreset | '' {
  return preset === 'all' ? '' : AUDIT_TIME_PRESET.LAST_24H;
}

function formatScopeProjectionValue(key: string, value: string) {
  const normalized = value.trim();
  if (!normalized) {
    return '';
  }

  if (key === 'business_category') {
    switch (normalized) {
      case AUDIT_BUSINESS_CATEGORY.FAILED_OPERATIONS:
        return resolveNonRedundantScopeValue(t('audit.logList.businessCategory.failedOperations'), 'business_category');
      case AUDIT_BUSINESS_CATEGORY.HIGH_RISK_OPERATIONS:
        return resolveNonRedundantScopeValue(
          t('audit.logList.businessCategory.highRiskOperations'),
          'business_category',
        );
      case AUDIT_BUSINESS_CATEGORY.SENSITIVE_OPERATIONS:
        return resolveNonRedundantScopeValue(
          t('audit.logList.businessCategory.sensitiveOperations'),
          'business_category',
        );
      case AUDIT_BUSINESS_CATEGORY.AUTH_FAILURES:
        return resolveNonRedundantScopeValue(t('audit.logList.businessCategory.authFailures'), 'business_category');
      case AUDIT_BUSINESS_CATEGORY.PERMISSION_DENIALS:
        return resolveNonRedundantScopeValue(
          t('audit.logList.businessCategory.permissionDenials'),
          'business_category',
        );
      case AUDIT_BUSINESS_CATEGORY.RBAC_CHANGES:
        return resolveNonRedundantScopeValue(t('audit.logList.businessCategory.rbacChanges'), 'business_category');
      case AUDIT_BUSINESS_CATEGORY.CRITICAL_SECURITY:
        return resolveNonRedundantScopeValue(t('audit.logList.businessCategory.criticalSecurity'), 'business_category');
      default:
        return t('audit.logList.scope.unknownValue');
    }
  }

  return normalized;
}

function resolveNonRedundantScopeValue(localizedValue: string, key: string) {
  const localizedLabel = key === 'business_category' ? t('audit.logList.builder.fields.businessCategory') : '';
  if (localizedLabel && localizedLabel === localizedValue) {
    return '';
  }
  return localizedValue;
}
</script>
<style scoped lang="less">
@import '../../../rbac/shared/list-page.less';

.audit-page {
  .management-list-header();
  .management-list-toolbar();
  .management-list-table-empty();
  .management-list-table-shell();
  .management-list-mobile();

  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.audit-scope-banner {
  align-items: center;
  background: color-mix(in srgb, var(--td-brand-color-light) 22%, var(--td-bg-color-container) 78%);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-height: 48px;
  padding: var(--graft-density-gap-8) var(--graft-density-gap-12);
}

.audit-scope-banner__main {
  flex: 1;
  min-width: 0;
}

.audit-scope-banner__summary,
.audit-scope-banner__actions {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.audit-scope-banner__condition {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.audit-scope-banner__actions {
  flex-shrink: 0;
  justify-content: flex-end;
}

.audit-policy-drawer {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.audit-policy-drawer__section {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  padding: var(--graft-density-gap-12);
}

.audit-policy-drawer__section-header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.audit-policy-drawer__title-row {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.audit-policy-drawer__label {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.audit-policy-drawer__hint {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-4) 0 0;
}

.audit-policy-drawer__action {
  align-self: flex-start;
}

.audit-policy-drawer__catalog {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
}

.audit-policy-drawer__catalog-item {
  align-items: flex-start;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-default);
  display: flex;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
  padding: var(--graft-density-gap-12);
}

.audit-policy-drawer__catalog-meta {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.audit-policy-drawer__catalog-title {
  align-items: center;
  color: var(--td-text-color-primary);
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  overflow-wrap: anywhere;
}

.audit-policy-drawer__catalog-key {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  word-break: break-all;
}

.audit-policy-drawer__catalog-description,
.audit-policy-drawer__catalog-state {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.audit-policy-drawer__catalog-state {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}

.audit-policy-drawer__catalog-actions {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-width: 220px;
}

.audit-policy-drawer__catalog-buttons {
  display: flex;
  gap: var(--graft-density-gap-8);
  justify-content: flex-end;
}

@media (width <= 768px) {
  .audit-scope-banner {
    flex-direction: column;
  }

  .audit-scope-banner__actions {
    width: 100%;
  }

  .audit-policy-drawer__catalog-item {
    flex-direction: column;
  }

  .audit-policy-drawer__section-header {
    align-items: stretch;
    flex-direction: column;
  }

  .audit-policy-drawer__catalog-actions {
    min-width: 0;
    width: 100%;
  }

  .audit-policy-drawer__catalog-buttons {
    justify-content: flex-start;
  }
}
</style>
