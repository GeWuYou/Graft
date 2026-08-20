<template>
  <div class="project-import-page" data-page-type="workflow">
    <management-page-content>
      <management-page-header
        title-key="project.route.createImport.title"
        description-key="project.import.description"
        :source="{ labelKey: 'project.import.eyebrow', fallback: t('project.import.eyebrow') }"
      >
        <template #meta>
          <t-space break-line size="small">
            <t-tag v-if="selectedCandidateLabel" theme="default" variant="light-outline">
              {{ t('project.import.meta.selectedCandidate', { name: selectedCandidateLabel }) }}
            </t-tag>
          </t-space>
        </template>
        <template #actions>
          <t-button variant="text" @click="goToSource">
            <template #icon><chevron-left-icon /></template>
            {{ t('project.create.actions.backToSource') }}
          </t-button>
        </template>
      </management-page-header>

      <div class="project-import-surface">
        <t-card class="project-import-workflow" :bordered="true">
          <div class="project-import-workflow__header">
            <div class="project-import-workflow__copy">
              <p class="project-import-workflow__eyebrow">
                {{ t('project.import.steps.current', { current: currentStepIndex + 1, total: wizardSteps.length }) }}
              </p>
              <h2 class="project-import-workflow__title">
                {{ t(currentStepDefinition.titleKey) }}
              </h2>
              <p class="project-import-workflow__description">
                {{ t(currentStepDefinition.descriptionKey) }}
              </p>
            </div>
          </div>

          <t-steps :current="currentStepIndex" :options="wizardStepOptions" readonly separator="line" />
        </t-card>

        <section v-if="currentStep === 'select'" class="project-import-step">
          <management-toolbar>
            <template #filters>
              <t-input
                v-model="candidateSearchKeyword"
                class="management-list-search"
                clearable
                :placeholder="t('project.import.candidates.searchPlaceholder')"
              >
                <template #prefix-icon><search-icon /></template>
              </t-input>
              <t-select
                v-model="candidateStatusFilter"
                class="management-toolbar__select"
                data-testid="candidate-status-filter"
                :placeholder="t('project.import.candidates.statusFilter')"
              >
                <t-option
                  v-for="option in candidateStatusFilterOptions"
                  :key="option.value"
                  :value="option.value"
                  :label="option.label"
                />
              </t-select>
            </template>
            <template #actions>
              <t-button
                theme="default"
                variant="text"
                :disabled="!hasActiveCandidateFilters"
                @click="resetCandidateFilters"
              >
                {{ t('project.import.actions.clearFilters') }}
              </t-button>
            </template>
          </management-toolbar>

          <management-empty-state
            v-if="candidatesError && !candidatesLoading"
            tone="error"
            :title="t('project.import.candidates.title')"
            :description="candidatesError"
          >
            <template #actions>
              <t-button theme="primary" variant="outline" @click="loadCandidates">
                {{ t('project.import.actions.refreshCandidates') }}
              </t-button>
            </template>
          </management-empty-state>

          <advanced-query-paged-table
            v-else
            :key="candidateTableRenderKey"
            v-model:current="candidatePagination.current"
            v-model:page-size="candidatePagination.pageSize"
            :cell-slot-names="candidateCellSlotNames"
            :columns="visibleCandidateColumns"
            :description="candidateTableHint"
            :empty-description="candidateEmptyDescription"
            :empty-title="candidateEmptyTitle"
            :footer-summary="candidatePaginationSummary"
            :head-label="candidateHeadLabel"
            :loading="candidatesLoading"
            :row-class-name="candidateRowClassName"
            row-key="candidate_key"
            :rows="candidates"
            :summary="candidateTableSummary"
            :total="candidateListTotal"
          >
            <template #toolbar>
              <table-view-toolbar
                :column-settings-label="t('project.import.candidates.columnSettings')"
                :refresh-label="t('project.import.actions.refreshCandidates')"
                :refresh-loading="candidatesLoading"
                @column-settings="columnDrawerVisible = true"
                @refresh="loadCandidates"
              >
              </table-view-toolbar>
            </template>

            <template #project="{ row }">
              <div class="project-import-candidate-main">
                <div class="project-import-candidate-main__title">
                  <strong>{{ row.compose_project_name }}</strong>
                  <t-tag
                    v-if="row.candidate_key === selectedCandidateKey"
                    size="small"
                    theme="primary"
                    variant="light-outline"
                  >
                    {{ t('project.import.candidates.selectedMarker') }}
                  </t-tag>
                </div>
                <div class="project-import-candidate-main__meta">
                  <span>{{ formatContainerCounts(row.container_counts) }}</span>
                  <span>{{
                    t('project.import.candidates.configFileCountValue', { count: row.config_files.length })
                  }}</span>
                </div>
              </div>
            </template>

            <template #config_files="{ row }">
              <div class="project-import-candidate-code">
                <t-tooltip :content="formatListTooltip(row.config_files)" placement="top-left">
                  <code :title="formatListTooltip(row.config_files)">{{ firstListItem(row.config_files) }}</code>
                </t-tooltip>
                <span v-if="row.config_files.length > 1">
                  {{ t('project.import.candidates.additionalConfigFiles', { count: row.config_files.length - 1 }) }}
                </span>
              </div>
            </template>

            <template #workspace_path="{ row }">
              <div class="project-import-candidate-code">
                <t-tooltip :content="row.workspace_path || '-'" placement="top-left">
                  <code :title="row.workspace_path || '-'">{{ row.workspace_path || '-' }}</code>
                </t-tooltip>
              </div>
            </template>

            <template #runtime="{ row }">
              <span>{{ formatRuntimeLabel(row.runtime_type, row.runtime_version) }}</span>
            </template>

            <template #services="{ row }">
              <div class="project-import-candidate-services">
                <strong>{{
                  t('project.import.candidates.serviceCountValue', { count: row.service_names.length })
                }}</strong>
                <span>{{ formatServicePreview(row.service_names) }}</span>
              </div>
            </template>

            <template #status="{ row }">
              <t-tag :theme="candidateStatusTheme(row.status)" variant="light-outline">
                {{ t(`project.import.candidates.status.${row.status}`) }}
              </t-tag>
            </template>

            <template #reason="{ row }">
              <div class="project-import-candidate-reason">
                <template v-if="isApplicationImportRuntimeCandidateReady(row)">
                  <span>-</span>
                </template>
                <template v-else>
                  <strong>{{ candidateUnavailableReason(row) }}</strong>
                  <span v-if="candidateDiagnostics(row).length">
                    {{ candidateDiagnostics(row).join(' · ') }}
                  </span>
                </template>
              </div>
            </template>

            <template #operation="{ row }">
              <t-button
                theme="primary"
                variant="outline"
                size="small"
                :loading="inspectLoading && selectedCandidateKey === row.candidate_key"
                :disabled="
                  !isApplicationImportRuntimeCandidateReady(row) ||
                  (inspectLoading && selectedCandidateKey !== row.candidate_key)
                "
                @click="handleCandidateInspect(row)"
              >
                {{ t('project.import.actions.inspectCandidate') }}
              </t-button>
            </template>
          </advanced-query-paged-table>

          <advanced-query-column-drawer
            v-model:selected-keys="visibleCandidateColumnKeys"
            v-model:visible="columnDrawerVisible"
            :columns="candidateColumnSettingOptions"
            :default-selected-keys="DEFAULT_VISIBLE_COLUMNS"
            :disabled-keys="candidateColumnDisabledKeys"
            :reset-label="t('project.import.candidates.resetColumns')"
            :title="t('project.import.candidates.columnDrawerTitle')"
          />
        </section>

        <section v-else-if="currentStep === 'inspect'" class="project-import-step">
          <t-loading :loading="inspectLoading" size="small">
            <div v-if="inspectError && !hasPreview" class="project-import-feedback">
              <management-empty-state
                tone="error"
                :title="t('project.import.state.inspectErrorTitle')"
                :description="inspectError"
              >
                <template #actions>
                  <t-button theme="primary" type="button" @click="() => handleRefreshInspect()">
                    {{ t('project.import.actions.retryInspect') }}
                  </t-button>
                </template>
              </management-empty-state>
            </div>

            <template v-else-if="normalizedInspectResult">
              <project-import-inspection-session-alert
                :error-message="inspectError"
                :loading="inspectLoading"
                :valid="inspectionSessionValid"
                @refresh="handleRefreshInspect"
              />

              <project-import-inspect-overview
                :can-import="canImport"
                :resolved-workspace-path="resolvedWorkspacePath"
                :result="normalizedInspectResult"
              />

              <project-import-inspect-resources
                :inspect-loading="inspectLoading"
                :result="normalizedInspectResult"
                @refresh-requested="handleRefreshInspect"
              />

              <div class="project-import-step-actions">
                <t-button theme="default" variant="outline" @click="goToStep('select', true)">
                  {{ t('project.import.actions.backToCandidates') }}
                </t-button>
                <t-button
                  theme="default"
                  variant="outline"
                  :loading="inspectLoading"
                  @click="() => handleRefreshInspect()"
                >
                  {{ t('project.import.actions.refreshInspect') }}
                </t-button>
                <t-button theme="primary" :disabled="!canImport" @click="goToLifecycleStep">
                  {{ t('project.import.actions.continueToLifecycle') }}
                </t-button>
              </div>
            </template>
          </t-loading>
        </section>

        <section v-else-if="currentStep === 'lifecycle'" class="project-import-step">
          <project-import-inspection-session-alert
            :error-message="inspectError"
            :loading="inspectLoading"
            :valid="inspectionSessionValid"
            @refresh="handleRefreshInspect"
          />
          <project-import-lifecycle-review
            v-if="lifecycleDraft"
            v-model:draft="lifecycleDraft"
            :inspection-refresh-loading="inspectLoading"
            :service-options="inspectResult?.service_options ?? []"
            @back="goToStep('inspect', true)"
            @confirm="goToConfirmStep"
            @refresh="handleRefreshInspect"
          />
          <management-empty-state
            v-else
            tone="error"
            :title="t('project.import.lifecycle.unavailableTitle')"
            :description="lifecycleConfigError"
          >
            <template #actions>
              <t-button theme="default" variant="outline" @click="goToStep('inspect', true)">
                {{ t('project.import.actions.backToInspect') }}
              </t-button>
            </template>
          </management-empty-state>
        </section>

        <section v-else class="project-import-step">
          <project-import-inspection-session-alert
            :error-message="inspectError"
            :loading="inspectLoading"
            :valid="inspectionSessionValid"
            @refresh="handleRefreshInspect"
          />
          <project-import-confirm-review
            v-if="normalizedInspectResult"
            :can-import="canImport"
            :candidate="selectedCandidate"
            :compose-project-name-override="composeProjectNameOverride"
            :display-name="displayName"
            :form-data="formData"
            :form-rules="formRules"
            :import-error="importError"
            :import-loading="importLoading"
            :inspection-refresh-loading="inspectLoading"
            :resolved-workspace-path="resolvedWorkspacePath"
            :result="normalizedInspectResult"
            @back="goToStep('lifecycle', true)"
            @refresh="handleRefreshInspect"
            @reset="handleReset"
            @submit="handleSubmit"
            @update:compose-project-name-override="composeProjectNameOverride = $event"
            @update:display-name="displayName = $event"
          />
        </section>
      </div>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { ChevronLeftIcon, SearchIcon } from 'tdesign-icons-vue-next';
import type { FormProps, SubmitContext, TdBaseTableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import type { LocationQueryRaw } from 'vue-router';
import { useRoute } from 'vue-router';

import {
  createActionColumn,
  createMainTextColumn,
  createStatusColumn,
  createTechnicalColumn,
  ManagementEmptyState,
  ManagementPageContent,
  ManagementPageHeader,
  ManagementToolbar,
  resolveManagedColumns,
  TableViewToolbar,
} from '@/shared/components/management';
import { AdvancedQueryColumnDrawer, AdvancedQueryPagedTable } from '@/shared/components/query-list';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';

import { getApplicationImportRuntimeCandidates } from '../../api/import';
import ProjectImportConfirmReview from '../../components/ProjectImportConfirmReview.vue';
import ProjectImportInspectionSessionAlert from '../../components/ProjectImportInspectionSessionAlert.vue';
import ProjectImportInspectOverview from '../../components/ProjectImportInspectOverview.vue';
import ProjectImportInspectResources from '../../components/ProjectImportInspectResources.vue';
import ProjectImportLifecycleReview from '../../components/ProjectImportLifecycleReview.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import { isValidApplicationCanonicalName } from '../../shared/canonical-name';
import {
  isApplicationImportRuntimeCandidateReady,
  normalizeApplicationImportInspectResponse,
  normalizeStringArray,
  resolveApplicationImportRuntimeCandidateReasonKey,
} from '../../shared/import';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../../shared/navigation';
import { useApplicationPageContext } from '../../shared/page-context';
import { isApplicationImportInspectionExpiredError, useApplicationImportFlow } from '../../shared/useProjectImportFlow';
import type {
  ApplicationImportExecuteResponse,
  ApplicationImportRuntimeCandidate,
  ApplicationImportRuntimeCandidateFilterCounts,
  ApplicationImportRuntimeCandidatesQuery,
} from '../../types/import';

// 导入页编排分步 inspection 流程：服务端候选与检查结果是外部事实，步骤、筛选和列偏好属于页面交互状态。

defineOptions({
  name: 'ApplicationImportIndex',
});

type ImportWizardStep = 'select' | 'inspect' | 'lifecycle' | 'confirm';
type CandidateStatusFilter = 'all' | 'ready' | 'imported' | 'unavailable';
type PaginationState = {
  current: number;
  pageSize: number;
};

type CandidateListState = {
  items: ApplicationImportRuntimeCandidate[];
  total: number;
  filterCounts: ApplicationImportRuntimeCandidateFilterCounts;
};

const IMPORT_STEP_QUERY_KEY = 'step';
const IMPORT_CANDIDATE_QUERY_KEY = 'candidate';
const CANDIDATE_PAGE_SIZE = 10;
const ROUTE_RECOVERY_PAGE_SIZE = 50;
const CANDIDATE_COLUMN_STORAGE_KEY = 'graft.project.import.visibleColumns.v2';
const DEFAULT_VISIBLE_COLUMNS = [
  'project',
  'config_files',
  'workspace_path',
  'runtime',
  'services',
  'status',
  'reason',
  'operation',
];
const ALL_COLUMN_KEYS = [
  'project',
  'config_files',
  'workspace_path',
  'runtime',
  'services',
  'status',
  'reason',
  'operation',
];
const CANDIDATE_ALWAYS_VISIBLE_COLUMNS = ['reason', 'operation'];

const wizardSteps = [
  {
    key: 'select',
    shortTitleKey: 'project.import.steps.select.shortTitle',
    titleKey: 'project.import.steps.select.title',
    descriptionKey: 'project.import.steps.select.description',
  },
  {
    key: 'inspect',
    shortTitleKey: 'project.import.steps.inspect.shortTitle',
    titleKey: 'project.import.steps.inspect.title',
    descriptionKey: 'project.import.steps.inspect.description',
  },
  {
    key: 'lifecycle',
    shortTitleKey: 'project.import.steps.lifecycle.shortTitle',
    titleKey: 'project.import.steps.lifecycle.title',
    descriptionKey: 'project.import.steps.lifecycle.description',
  },
  {
    key: 'confirm',
    shortTitleKey: 'project.import.steps.confirm.shortTitle',
    titleKey: 'project.import.steps.confirm.title',
    descriptionKey: 'project.import.steps.confirm.description',
  },
] as const satisfies ReadonlyArray<{
  key: ImportWizardStep;
  shortTitleKey: string;
  titleKey: string;
  descriptionKey: string;
}>;

const { router, tabsRouterStore, t } = useApplicationPageContext();
const route = useRoute();
const candidatesLoading = ref(true);
const candidatesError = ref('');
const candidates = ref<ApplicationImportRuntimeCandidate[]>([]);
const candidateListTotal = ref(0);
const candidateFilterCounts = ref<ApplicationImportRuntimeCandidateFilterCounts>({
  all: 0,
  ready: 0,
  imported: 0,
  unavailable: 0,
});
const currentStep = ref<ImportWizardStep>('select');
const candidatesLoaded = ref(false);
const candidateSearchKeyword = ref('');
const candidateStatusFilter = ref<CandidateStatusFilter>('ready');
const candidatePagination = reactive<PaginationState>({
  current: 1,
  pageSize: CANDIDATE_PAGE_SIZE,
});
const columnDrawerVisible = ref(false);
const visibleCandidateColumnKeys = ref<string[]>(
  loadVisibleColumnKeys(
    CANDIDATE_COLUMN_STORAGE_KEY,
    DEFAULT_VISIBLE_COLUMNS,
    ALL_COLUMN_KEYS,
    CANDIDATE_ALWAYS_VISIBLE_COLUMNS,
  ),
);

const {
  canImport,
  composeProjectNameOverride,
  displayName,
  hasPreview,
  importError,
  importLoading,
  invalidateInspectionSession,
  inspectionSessionValid,
  lifecycleDraft,
  lifecycleConfigError,
  inspectCandidate,
  inspectError,
  inspectLoading,
  inspectResult,
  prepareLifecycleConfiguration,
  refreshInspect,
  reset,
  selectedCandidateKey,
  submitImport,
} = useApplicationImportFlow(t);

let inspectionExpiryTimer: ReturnType<typeof setTimeout> | undefined;

const formData = reactive({
  display_name: displayName,
  compose_project_name_override: composeProjectNameOverride,
});

const formRules: FormProps['rules'] = {
  display_name: [{ required: true, message: t('project.import.validation.displayNameRequired') }],
  compose_project_name_override: [
    {
      validator: (value) => {
        const normalized = String(value ?? '').trim();
        return !normalized || isValidApplicationCanonicalName(normalized);
      },
      message: t('project.import.validation.composeProjectNameOverridePattern'),
    },
  ],
};

const readyCandidates = computed(() =>
  candidates.value.filter((item) => isApplicationImportRuntimeCandidateReady(item)),
);
const selectedCandidate = computed(
  () => candidates.value.find((item) => item.candidate_key === selectedCandidateKey.value) ?? null,
);
const normalizedInspectResult = computed(() => normalizeApplicationImportInspectResponse(inspectResult.value));
const selectedCandidateLabel = computed(
  () => normalizedInspectResult.value?.compose_project_name || selectedCandidate.value?.compose_project_name || '',
);
const resolvedWorkspacePath = computed(
  () => normalizedInspectResult.value?.resolved_workspace_path || selectedCandidate.value?.workspace_path || '',
);

const normalizedCandidateSearch = computed(() => candidateSearchKeyword.value.trim());

watch(
  () => inspectResult.value?.expires_at,
  (expiresAt) => {
    if (inspectionExpiryTimer !== undefined) {
      clearTimeout(inspectionExpiryTimer);
      inspectionExpiryTimer = undefined;
    }
    const expiresAtTimestamp = expiresAt ? Date.parse(expiresAt) : Number.NaN;
    if (!Number.isFinite(expiresAtTimestamp)) {
      return;
    }
    inspectionExpiryTimer = setTimeout(
      () => {
        invalidateInspectionSession();
        void handleRefreshInspect(true);
      },
      Math.max(0, expiresAtTimestamp - Date.now()),
    );
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  if (inspectionExpiryTimer !== undefined) {
    clearTimeout(inspectionExpiryTimer);
  }
});
const hasActiveCandidateFilters = computed(
  () => Boolean(candidateSearchKeyword.value.trim()) || candidateStatusFilter.value !== 'ready',
);
const candidateStatusFilterOptions = computed(() => [
  {
    value: 'all' as const,
    label: t('project.import.candidates.allFilterLabel', { count: candidateFilterCounts.value.all }),
  },
  {
    value: 'ready' as const,
    label: t('project.import.candidates.readyFilterLabel', { count: candidateFilterCounts.value.ready }),
  },
  {
    value: 'imported' as const,
    label: t('project.import.candidates.importedFilterLabel', { count: candidateFilterCounts.value.imported }),
  },
  {
    value: 'unavailable' as const,
    label: t('project.import.candidates.unavailableFilterLabel', { count: candidateFilterCounts.value.unavailable }),
  },
]);
const candidateCellSlotNames = [
  'project',
  'config_files',
  'workspace_path',
  'runtime',
  'services',
  'status',
  'reason',
  'operation',
];
const candidateColumns = computed<TdBaseTableProps['columns']>(() => [
  createMainTextColumn(t('project.import.candidates.columnApplication'), 'project', 280),
  createTechnicalColumn(t('project.import.candidates.columnConfigFiles'), 'config_files', 260),
  createTechnicalColumn(t('project.import.candidates.columnWorkspacePath'), 'workspace_path', 240),
  createTechnicalColumn(t('project.import.candidates.columnRuntime'), 'runtime', 150),
  createTechnicalColumn(t('project.import.candidates.columnServices'), 'services', 180),
  createStatusColumn(t('project.import.candidates.columnStatus'), 'status', 120),
  createMainTextColumn(t('project.import.candidates.columnReason'), 'reason', 320),
  createActionColumn(t('project.import.candidates.columnOperation'), 132),
]);
const candidateColumnSettingOptions = computed(() =>
  ALL_COLUMN_KEYS.map((key) => ({
    label: t(`project.import.candidates.columnOptionLabels.${key}`),
    value: key,
  })),
);
const visibleCandidateColumns = computed(() =>
  resolveManagedColumns(candidateColumns.value, visibleCandidateColumnKeys.value, CANDIDATE_ALWAYS_VISIBLE_COLUMNS),
);
const candidateColumnDisabledKeys = computed(() => [...CANDIDATE_ALWAYS_VISIBLE_COLUMNS]);
const candidateTableSummary = computed(() =>
  t('project.import.candidates.tableSummary', { count: candidateListTotal.value }),
);
const candidateTableHint = computed(() => t('project.import.candidates.tableHint'));
const candidateHeadLabel = computed(() => 'project-import-candidates-table');
const candidateTableRenderKey = computed(() =>
  [
    candidateStatusFilter.value,
    candidatePagination.current,
    candidatePagination.pageSize,
    candidateListTotal.value,
    visibleCandidateColumnKeys.value.join('|'),
  ].join(':'),
);
const candidatePaginationSummary = computed(() =>
  buildPaginationSummary(candidateListTotal.value, candidatePagination),
);
const candidateEmptyTitle = computed(() =>
  hasActiveCandidateFilters.value
    ? t('project.import.candidates.filteredEmptyTitle')
    : t('project.import.candidates.emptyTitle'),
);
const candidateEmptyDescription = computed(() =>
  hasActiveCandidateFilters.value
    ? t('project.import.candidates.filteredEmptyDescription')
    : t('project.import.candidates.emptyDescription'),
);

const currentStepIndex = computed(() => wizardSteps.findIndex((step) => step.key === currentStep.value));
const currentStepDefinition = computed(() => wizardSteps[currentStepIndex.value] ?? wizardSteps[0]);
const wizardStepOptions = computed(() =>
  wizardSteps.map((step) => ({
    title: t(step.shortTitleKey),
    content: t(step.descriptionKey),
  })),
);
let latestCandidateRequestId = 0;
let latestRouteSyncRequestId = 0;

watch(
  () => [route.query[IMPORT_STEP_QUERY_KEY], route.query[IMPORT_CANDIDATE_QUERY_KEY]],
  () => {
    if (!candidatesLoaded.value) {
      return;
    }

    void syncWizardFromRoute();
  },
);

onMounted(() => {
  void initializePage();
});

watch([normalizedCandidateSearch, candidateStatusFilter], () => {
  candidatePagination.current = 1;
  columnDrawerVisible.value = false;
  if (candidatesLoaded.value) {
    void loadCandidates();
  }
});

watch(
  () => [candidatePagination.current, candidatePagination.pageSize],
  () => {
    if (candidatesLoaded.value) {
      void loadCandidates();
    }
  },
);

watch(
  () => candidateListTotal.value,
  (total) => {
    clampPagination(total, candidatePagination);
  },
);

watch(candidateStatusFilter, () => {
  columnDrawerVisible.value = false;
});

watch(
  visibleCandidateColumnKeys,
  (keys) => {
    const normalizedKeys = normalizeVisibleColumnKeys(keys, ALL_COLUMN_KEYS, CANDIDATE_ALWAYS_VISIBLE_COLUMNS);
    if (normalizedKeys.join('|') !== keys.join('|')) {
      visibleCandidateColumnKeys.value = normalizedKeys;
      return;
    }

    persistVisibleColumnKeys(CANDIDATE_COLUMN_STORAGE_KEY, normalizedKeys);
  },
  { deep: true },
);

async function initializePage() {
  await loadCandidates();
  candidatesLoaded.value = true;
  await syncWizardFromRoute();
}

function normalizeWizardStep(value: unknown): ImportWizardStep {
  if (value === 'inspect' || value === 'lifecycle' || value === 'confirm') {
    return value;
  }

  return 'select';
}

function queryString(value: unknown) {
  if (typeof value === 'string') {
    return value;
  }

  if (Array.isArray(value) && typeof value[0] === 'string') {
    return value[0];
  }

  return '';
}

async function syncWizardFromRoute() {
  const syncRequestId = ++latestRouteSyncRequestId;
  const desiredStep = normalizeWizardStep(route.query[IMPORT_STEP_QUERY_KEY]);
  const candidateKey = queryString(route.query[IMPORT_CANDIDATE_QUERY_KEY]);

  if (desiredStep === 'select') {
    currentStep.value = 'select';
    return;
  }

  if (!candidateKey) {
    currentStep.value = 'select';
    await updateWizardRoute('select', { replace: true });
    return;
  }

  let candidate = readyCandidates.value.find((item) => item.candidate_key === candidateKey) ?? null;
  if (!candidate) {
    try {
      candidate = await findReadyCandidateByKey(candidateKey);
    } catch (error) {
      if (syncRequestId !== latestRouteSyncRequestId) {
        return;
      }
      reset();
      candidatesError.value = resolveLocalizedErrorMessage(t, error, t('project.import.messages.candidateLoadFailed'));
      currentStep.value = 'select';
      await updateWizardRoute('select', { replace: true });
      return;
    }
  }
  if (syncRequestId !== latestRouteSyncRequestId) {
    return;
  }
  if (!candidate) {
    currentStep.value = 'select';
    await updateWizardRoute('select', { replace: true });
    return;
  }

  if (inspectResult.value?.candidate_key !== candidateKey || !hasPreview.value) {
    try {
      const result = await inspectCandidate(candidate);
      if (syncRequestId !== latestRouteSyncRequestId) {
        return;
      }
      if (result !== 'applied') {
        return;
      }
    } catch {
      if (syncRequestId !== latestRouteSyncRequestId) {
        return;
      }
      currentStep.value = 'inspect';
      if (desiredStep === 'lifecycle' || desiredStep === 'confirm') {
        await updateWizardRoute('inspect', { candidateKey, replace: true });
      }
      return;
    }
  }

  if ((desiredStep === 'lifecycle' || desiredStep === 'confirm') && !canImport.value) {
    currentStep.value = 'inspect';
    await updateWizardRoute('inspect', { candidateKey, replace: true });
    return;
  }

  if (
    (desiredStep === 'lifecycle' || desiredStep === 'confirm') &&
    !lifecycleDraft.value &&
    !prepareLifecycleConfiguration()
  ) {
    currentStep.value = 'lifecycle';
    if (desiredStep === 'confirm') {
      await updateWizardRoute('lifecycle', { candidateKey, replace: true });
    }
    return;
  }

  currentStep.value = desiredStep;
}

async function findReadyCandidateByKey(candidateKey: string) {
  let offset = 0;
  let total = Number.POSITIVE_INFINITY;

  while (offset < total) {
    const response = await getApplicationImportRuntimeCandidates({
      availability: 'ready',
      limit: ROUTE_RECOVERY_PAGE_SIZE,
      offset,
    });
    const nextState = normalizeCandidateListState(response);
    const candidate = nextState.items.find((item) => item.candidate_key === candidateKey);
    if (candidate && isApplicationImportRuntimeCandidateReady(candidate)) {
      return candidate;
    }

    total = nextState.total;
    offset += ROUTE_RECOVERY_PAGE_SIZE;

    if (!nextState.items.length) {
      break;
    }
  }

  return null;
}

function candidateRowClassName(params: { row: ApplicationImportRuntimeCandidate }) {
  return params.row.candidate_key === selectedCandidateKey.value ? 'project-import-candidate-row--active' : '';
}

function clampPagination(total: number, pagination: PaginationState) {
  const maxPage = Math.max(1, Math.ceil(total / pagination.pageSize));
  if (pagination.current > maxPage) {
    pagination.current = maxPage;
  }
}

function buildPaginationSummary(total: number, pagination: PaginationState) {
  return t('project.import.candidates.paginationSummary', buildPaginationRange(total, pagination));
}

function buildPaginationRange(total: number, pagination: PaginationState) {
  if (!total) {
    return {
      start: 0,
      end: 0,
      total: 0,
    };
  }

  const start = (pagination.current - 1) * pagination.pageSize + 1;
  const end = Math.min(pagination.current * pagination.pageSize, total);
  return { start, end, total };
}

function firstListItem(items: string[]) {
  return items[0] || '-';
}

function formatListTooltip(items: string[]) {
  return items.length ? items.join('\n') : '-';
}

function formatServicePreview(items: string[]) {
  if (!items.length) {
    return t('project.import.preview.none');
  }

  if (items.length <= 2) {
    return items.join(', ');
  }

  return `${items.slice(0, 2).join(', ')} ${t('project.import.candidates.additionalServices', {
    count: items.length - 2,
  })}`;
}

function formatContainerCounts(counts: ApplicationImportRuntimeCandidate['container_counts']) {
  return t('project.import.candidates.containerCountsValue', counts);
}

function formatRuntimeLabel(runtimeType: string, runtimeVersion?: string | null) {
  return runtimeVersion?.trim() ? `${runtimeType} ${runtimeVersion.trim()}` : runtimeType;
}

function formatRuntimeCandidateReason(reasonCode: string) {
  const translationKey = `project.import.candidates.reason.${reasonCode}`;
  const translated = t(translationKey);
  return translated === translationKey ? reasonCode : translated;
}

function formatRuntimeCandidateWarning(warningCode: string) {
  const translationKey = `project.import.candidates.warning.${warningCode}`;
  const translated = t(translationKey);
  return translated === translationKey ? warningCode : translated;
}

function candidateStatusTheme(status: ApplicationImportRuntimeCandidate['status']) {
  if (status === 'ready') return 'success';
  if (status === 'broken_compose') return 'danger';
  if (status === 'incomplete_metadata') return 'warning';
  if (status === 'unsupported_runtime') return 'default';
  return 'default';
}

function candidateUnavailableReason(candidate: ApplicationImportRuntimeCandidate) {
  const reasonKey = resolveApplicationImportRuntimeCandidateReasonKey(candidate);
  const translated = t(`project.import.candidates.reason.${reasonKey}`);
  if (translated === `project.import.candidates.reason.${reasonKey}`) {
    return t('project.import.candidates.reason.unavailable');
  }
  return translated;
}

function candidateDiagnostics(candidate: ApplicationImportRuntimeCandidate) {
  const diagnostics = [
    ...normalizeStringArray(candidate.status_reason_codes).map((code) => formatRuntimeCandidateReason(code)),
    ...normalizeStringArray(candidate.warnings).map((code) => formatRuntimeCandidateWarning(code)),
  ];
  const primaryReason = candidateUnavailableReason(candidate);
  return Array.from(new Set(diagnostics)).filter((item) => item && item !== primaryReason);
}

async function loadCandidates() {
  const requestId = ++latestCandidateRequestId;
  candidatesLoading.value = true;
  candidatesError.value = '';
  try {
    const response = await getApplicationImportRuntimeCandidates(buildCandidateQuery());
    if (requestId !== latestCandidateRequestId) {
      return;
    }
    const nextState = normalizeCandidateListState(response);
    candidates.value = nextState.items;
    candidateListTotal.value = nextState.total;
    candidateFilterCounts.value = nextState.filterCounts;

    if (
      selectedCandidateKey.value &&
      !readyCandidates.value.some((item) => item.candidate_key === selectedCandidateKey.value)
    ) {
      reset();
      currentStep.value = 'select';
      await updateWizardRoute('select', { replace: true });
    }
  } catch (error) {
    if (requestId !== latestCandidateRequestId) {
      return;
    }
    candidates.value = [];
    candidateListTotal.value = 0;
    candidateFilterCounts.value = {
      all: 0,
      ready: 0,
      imported: 0,
      unavailable: 0,
    };
    candidatesError.value = resolveLocalizedErrorMessage(t, error, t('project.import.messages.candidateLoadFailed'));
  } finally {
    if (requestId === latestCandidateRequestId) {
      candidatesLoading.value = false;
    }
  }
}

function buildCandidateQuery(): ApplicationImportRuntimeCandidatesQuery {
  return {
    keyword: normalizedCandidateSearch.value || undefined,
    availability: resolveCandidateAvailability(candidateStatusFilter.value),
    limit: candidatePagination.pageSize,
    offset: (candidatePagination.current - 1) * candidatePagination.pageSize,
  };
}

function resolveCandidateAvailability(value: CandidateStatusFilter) {
  if (value === 'all') {
    return undefined;
  }
  return value;
}

function normalizeCandidateListState(
  response?: Partial<CandidateListState> & Record<string, unknown>,
): CandidateListState {
  const items = Array.isArray(response?.items)
    ? response.items.map((candidate) => normalizeCandidate(candidate as ApplicationImportRuntimeCandidate))
    : [];
  const rawFilterCounts = response?.filter_counts ?? response?.filterCounts;
  const filterCounts = isCandidateFilterCounts(rawFilterCounts) ? rawFilterCounts : undefined;
  return {
    items,
    total: typeof response?.total === 'number' ? response.total : items.length,
    filterCounts: {
      all: typeof filterCounts?.all === 'number' ? filterCounts.all : items.length,
      ready: typeof filterCounts?.ready === 'number' ? filterCounts.ready : 0,
      imported: typeof filterCounts?.imported === 'number' ? filterCounts.imported : 0,
      unavailable: typeof filterCounts?.unavailable === 'number' ? filterCounts.unavailable : 0,
    },
  };
}

function isCandidateFilterCounts(value: unknown): value is ApplicationImportRuntimeCandidateFilterCounts {
  if (!value || typeof value !== 'object') {
    return false;
  }
  const candidate = value as Partial<ApplicationImportRuntimeCandidateFilterCounts>;
  return (
    typeof candidate.all === 'number' &&
    typeof candidate.ready === 'number' &&
    typeof candidate.imported === 'number' &&
    typeof candidate.unavailable === 'number'
  );
}

function normalizeCandidate(candidate: ApplicationImportRuntimeCandidate): ApplicationImportRuntimeCandidate {
  return {
    ...candidate,
    config_files: normalizeStringArray(candidate.config_files),
    service_names: normalizeStringArray(candidate.service_names),
    status_reason_codes: normalizeStringArray(candidate.status_reason_codes),
    warnings: normalizeStringArray(candidate.warnings),
  };
}

async function updateWizardRoute(
  step: ImportWizardStep,
  options: {
    candidateKey?: string;
    replace?: boolean;
  } = {},
) {
  const candidateKey = options.candidateKey ?? selectedCandidateKey.value;
  const nextQuery: LocationQueryRaw = { ...route.query };

  if (step === 'select') {
    delete nextQuery[IMPORT_STEP_QUERY_KEY];
    delete nextQuery[IMPORT_CANDIDATE_QUERY_KEY];
  } else {
    nextQuery[IMPORT_STEP_QUERY_KEY] = step;
    nextQuery[IMPORT_CANDIDATE_QUERY_KEY] = candidateKey;
  }

  const currentStepQuery = queryString(route.query[IMPORT_STEP_QUERY_KEY]);
  const currentCandidateQuery = queryString(route.query[IMPORT_CANDIDATE_QUERY_KEY]);
  if (
    currentStepQuery === queryString(nextQuery[IMPORT_STEP_QUERY_KEY]) &&
    currentCandidateQuery === queryString(nextQuery[IMPORT_CANDIDATE_QUERY_KEY])
  ) {
    return;
  }

  const navigate = options.replace ? router.replace : router.push;
  await navigate({
    name: route.name || PROJECT_BOOTSTRAP_ROUTE.CREATE_IMPORT.pageRouteName,
    params: route.params,
    query: nextQuery,
  });
}

async function goToStep(step: ImportWizardStep, replace = false) {
  currentStep.value = step;
  await updateWizardRoute(step, { replace });
}

async function goToLifecycleStep() {
  if (!canImport.value || !selectedCandidateKey.value) {
    return;
  }

  prepareLifecycleConfiguration();
  await goToStep('lifecycle');
}

async function goToConfirmStep() {
  if (!canImport.value || !lifecycleDraft.value) {
    return;
  }

  await goToStep('confirm');
}

async function handleRefreshInspect(automatic = false) {
  try {
    const result = await refreshInspect();
    if (result === 'applied' && inspectResult.value) {
      if (!automatic) {
        MessagePlugin.success(t('project.import.messages.inspectSuccess'));
      }
      if ((currentStep.value === 'lifecycle' || currentStep.value === 'confirm') && !canImport.value) {
        await goToStep('inspect', true);
      }
    }
  } catch (error) {
    if (isInspectionRefreshBlockingError(error)) {
      invalidateInspectionSession();
      await goToStep('inspect', true);
    }
    if (!automatic) {
      MessagePlugin.error(inspectError.value || t('project.import.messages.inspectFailed'));
    }
  }
}

function isInspectionRefreshBlockingError(error: unknown) {
  return (
    error &&
    typeof error === 'object' &&
    'isApiRequestError' in error &&
    (error as { isApiRequestError?: unknown }).isApiRequestError === true &&
    'status' in error &&
    ((error as { status?: unknown }).status === 400 || (error as { status?: unknown }).status === 409)
  );
}

async function handleCandidateInspect(candidate: ApplicationImportRuntimeCandidate) {
  if (!isApplicationImportRuntimeCandidateReady(candidate)) {
    return;
  }

  try {
    const result = await inspectCandidate(candidate);
    if (result === 'applied') {
      currentStep.value = 'inspect';
      MessagePlugin.success(t('project.import.messages.inspectSuccess'));
      await updateWizardRoute('inspect', { candidateKey: candidate.candidate_key });
    }
  } catch {
    MessagePlugin.error(inspectError.value || t('project.import.messages.inspectFailed'));
  }
}

async function handleSubmit(context?: SubmitContext) {
  if (!context || context.validateResult !== true) {
    return;
  }

  try {
    const response = await submitImport();
    MessagePlugin.success(t('project.import.messages.importSuccess'));
    await openDetail(response);
  } catch (error) {
    if (isApplicationImportInspectionExpiredError(error)) {
      invalidateInspectionSession();
      await handleRefreshInspect(true);
      return;
    }
    MessagePlugin.error(importError.value || t('project.import.messages.importFailed'));
  }
}

async function openDetail(response: ApplicationImportExecuteResponse) {
  const application = response.application;
  const currentTabPath = route.path;
  const currentTabFullPath = route.fullPath;
  const target = {
    name: PROJECT_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    params: { applicationId: application.application_id },
    query: { tab: 'lifecycle' },
  };
  const resolved = router.resolve(target);
  appendResolvedTab(
    tabsRouterStore,
    resolved,
    buildDetailTitleWithFallback('project.route.detail.title', application.display_name),
  );
  await router.push(target);
  tabsRouterStore.closeTabsByPredicate((tab) => tab.path === currentTabPath || tab.fullPath === currentTabFullPath);
}

function goToSource() {
  void router.push({
    name: PROJECT_BOOTSTRAP_ROUTE.CREATE_SOURCE.pageRouteName,
    query: route.query,
  });
}

function resetCandidateFilters() {
  candidateSearchKeyword.value = '';
  candidateStatusFilter.value = 'ready';
}

function handleReset() {
  reset();
  currentStep.value = 'select';
  void updateWizardRoute('select', { replace: true });
}

function loadVisibleColumnKeys(
  storageKey: string,
  defaultKeys: string[],
  allKeys: string[],
  alwaysVisibleKeys: string[],
) {
  if (typeof window === 'undefined') {
    return [...defaultKeys];
  }

  try {
    const stored = window.localStorage.getItem(storageKey);
    if (!stored) {
      return [...defaultKeys];
    }

    const parsed = JSON.parse(stored);
    if (!Array.isArray(parsed)) {
      return [...defaultKeys];
    }

    const normalizedKeys = normalizeVisibleColumnKeys(parsed, allKeys, alwaysVisibleKeys);
    persistVisibleColumnKeys(storageKey, normalizedKeys);
    return normalizedKeys;
  } catch {
    return [...defaultKeys];
  }
}

function persistVisibleColumnKeys(storageKey: string, keys: string[]) {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    window.localStorage.setItem(storageKey, JSON.stringify(keys));
  } catch {
    // 列设置只是可丢失的 UI 偏好；存储不可用时仍必须正常展示导入候选。
  }
}

function normalizeVisibleColumnKeys(keys: unknown[], allKeys: string[], alwaysVisibleKeys: string[]) {
  const availableKeySet = new Set(allKeys);
  const nextKeys = new Set<string>();

  for (const key of keys) {
    if (typeof key === 'string' && availableKeySet.has(key)) {
      nextKeys.add(key);
    }
  }

  for (const key of alwaysVisibleKeys) {
    nextKeys.add(key);
  }

  return allKeys.filter((key) => nextKeys.has(key));
}
</script>
<style scoped lang="less">
.project-import-page,
.project-import-surface,
.project-import-step,
.project-import-workflow,
.project-import-workflow__header,
.project-import-workflow__copy,
.project-import-unavailable,
.project-import-unavailable-card,
.project-import-feedback,
.project-import-preview__alerts,
.project-import-step-grid,
.project-import-preview-list,
.project-import-runtime-members {
  display: grid;
  gap: var(--graft-density-gap-16);
}

.project-import-surface {
  margin-top: var(--graft-density-gap-16);
}

.project-import-workflow {
  padding: 0;
}

.project-import-workflow__eyebrow {
  color: var(--td-brand-color);
  font: var(--td-font-body-small);
  font-weight: 600;
  margin: 0;
}

.project-import-workflow__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
}

.project-import-workflow__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  margin: 0;
}

.project-import-section-description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.project-import-empty {
  min-width: 0;
}

.project-import-candidate-main,
.project-import-candidate-code,
.project-import-candidate-services,
.project-import-candidate-reason,
.project-import-unavailable-drawer {
  display: grid;
  gap: var(--graft-density-gap-4);
}

.project-import-candidate-main__title {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.project-import-candidate-main__title strong,
.project-import-candidate-services strong {
  color: var(--td-text-color-primary);
  font-weight: 600;
}

.project-import-candidate-main__meta,
.project-import-candidate-code span,
.project-import-candidate-services span,
.project-import-candidate-reason span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.project-import-candidate-main__meta {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.project-import-candidate-code code {
  color: var(--td-text-color-primary);
  font-family: var(--td-font-family);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-import-diagnostics {
  display: grid;
  gap: var(--graft-density-gap-8);
}

.project-import-diagnostics__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
}

.project-import-diagnostics__list {
  color: var(--td-text-color-secondary);
  display: grid;
  gap: var(--graft-density-gap-6);
  margin: 0;
  padding-left: var(--graft-density-gap-20);
}

.project-import-step-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-import-step-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
}

.project-import-preview-list__item {
  min-width: 0;
}

.project-import-preview-technical {
  color: var(--td-text-color-primary);
  display: inline-block;
  font-family: var(
    --td-font-family-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    monospace
  );
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.project-import-runtime-members__summary,
.project-import-runtime-members__pagination-text {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.project-import-runtime-members__pagination {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.project-import-step :deep(.project-import-candidate-row--active > td) {
  background: color-mix(in srgb, var(--td-brand-color) 8%, var(--td-bg-color-container));
}

@media (width <= 1080px) {
  .project-import-step-grid {
    grid-template-columns: 1fr;
  }
}
</style>
