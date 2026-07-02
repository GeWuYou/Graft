<template>
  <div class="project-import-page" data-page-type="workflow">
    <management-page-content>
      <management-page-header
        title-key="project.route.import.title"
        description-key="project.import.description"
        :source="{ labelKey: 'project.import.eyebrow', fallback: t('project.import.eyebrow') }"
      >
        <template #meta>
          <t-space break-line size="small">
            <t-tag theme="primary" variant="light-outline">
              {{ t('project.import.meta.step', { current: currentStepIndex + 1, total: wizardSteps.length }) }}
            </t-tag>
            <t-tag theme="success" variant="light-outline">
              {{ t('project.import.meta.readyCount', { count: candidateFilterCounts.ready }) }}
            </t-tag>
            <t-tag theme="warning" variant="light-outline">
              {{ t('project.import.meta.unavailableCount', { count: candidateFilterCounts.unavailable }) }}
            </t-tag>
            <t-tag v-if="selectedCandidateLabel" theme="default" variant="light-outline">
              {{ t('project.import.meta.selectedCandidate', { name: selectedCandidateLabel }) }}
            </t-tag>
          </t-space>
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
                <template #before>
                  <span class="project-import-section-description">{{
                    t('project.import.candidates.summary', {
                      ready: candidateFilterCounts.ready,
                      unavailable: candidateFilterCounts.unavailable,
                    })
                  }}</span>
                </template>
              </table-view-toolbar>
            </template>

            <template #project="{ row }">
              <div class="project-import-candidate-main">
                <div class="project-import-candidate-main__title">
                  <strong>{{ row.canonical_project_name }}</strong>
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
                <code>{{ firstListItem(row.config_files) }}</code>
                <span v-if="row.config_files.length > 1">
                  {{ t('project.import.candidates.additionalConfigFiles', { count: row.config_files.length - 1 }) }}
                </span>
              </div>
            </template>

            <template #working_directory="{ row }">
              <div class="project-import-candidate-code">
                <code>{{ row.working_directory || '-' }}</code>
                <span>
                  {{ t(`project.import.candidates.workingDirectorySourceValues.${row.working_directory_source}`) }}
                </span>
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
                <template v-if="isProjectImportRuntimeCandidateReady(row)">
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
                  !isProjectImportRuntimeCandidateReady(row) ||
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
                  <t-button theme="primary" type="button" @click="handleRefreshInspect">
                    {{ t('project.import.actions.retryInspect') }}
                  </t-button>
                </template>
              </management-empty-state>
            </div>

            <template v-else-if="normalizedInspectResult">
              <project-import-inspect-overview
                :can-import="canImport"
                :resolved-working-directory="resolvedWorkingDirectory"
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
                <t-button theme="default" variant="outline" :loading="inspectLoading" @click="handleRefreshInspect">
                  {{ t('project.import.actions.refreshInspect') }}
                </t-button>
                <t-button theme="primary" :disabled="!canImport" @click="goToConfirmStep">
                  {{ t('project.import.actions.continueToConfirm') }}
                </t-button>
              </div>
            </template>
          </t-loading>
        </section>

        <section v-else class="project-import-step">
          <t-card :bordered="true" :title="t('project.import.confirm.title')">
            <t-form
              ref="formRef"
              :data="formData"
              :rules="formRules"
              label-align="top"
              scroll-to-first-error="smooth"
              @submit="handleSubmit"
            >
              <div class="project-import-authority">
                <t-alert
                  theme="info"
                  :message="
                    t('project.import.confirm.hint', {
                      name: inspectResult?.canonical_project_name || '-',
                    })
                  "
                />
                <t-alert v-if="importError" theme="error" :message="importError" />
              </div>

              <div class="project-import-form-grid">
                <t-form-item :label="t('project.import.form.displayName')" name="display_name">
                  <t-input v-model="displayName" :placeholder="t('project.import.form.displayNamePlaceholder')" />
                </t-form-item>
                <t-form-item
                  :label="t('project.import.form.canonicalProjectNameOverride')"
                  name="canonical_project_name_override"
                >
                  <t-input
                    v-model="canonicalProjectNameOverride"
                    :placeholder="t('project.import.form.canonicalProjectNameOverridePlaceholder')"
                  />
                </t-form-item>
              </div>

              <div class="project-import-form-actions">
                <t-button theme="default" variant="outline" type="button" @click="goToStep('inspect', true)">
                  {{ t('project.import.actions.backToInspect') }}
                </t-button>
                <t-button theme="primary" type="submit" :disabled="!canImport" :loading="importLoading">
                  {{ t('project.import.actions.import') }}
                </t-button>
                <t-button theme="default" variant="text" type="button" @click="handleReset">
                  {{ t('project.import.actions.reset') }}
                </t-button>
              </div>
            </t-form>
          </t-card>
        </section>
      </div>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { SearchIcon } from 'tdesign-icons-vue-next';
import type { FormInstanceFunctions, FormProps, SubmitContext, TdBaseTableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, reactive, ref, watch } from 'vue';
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

import { getProjectImportRuntimeCandidates } from '../../api/import';
import ProjectImportInspectOverview from '../../components/ProjectImportInspectOverview.vue';
import ProjectImportInspectResources from '../../components/ProjectImportInspectResources.vue';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import {
  isProjectImportRuntimeCandidateReady,
  normalizeProjectImportInspectResponse,
  normalizeStringArray,
  resolveProjectImportRuntimeCandidateReasonKey,
} from '../../shared/import';
import { appendResolvedTab, buildDetailTitleWithFallback } from '../../shared/navigation';
import { useProjectPageContext } from '../../shared/page-context';
import { useProjectImportFlow } from '../../shared/useProjectImportFlow';
import type {
  ProjectImportExecuteResponse,
  ProjectImportRuntimeCandidate,
  ProjectImportRuntimeCandidateFilterCounts,
  ProjectImportRuntimeCandidatesQuery,
} from '../../types/import';

defineOptions({
  name: 'ProjectImportIndex',
});

type ImportWizardStep = 'select' | 'inspect' | 'confirm';
type CandidateStatusFilter = 'all' | 'ready' | 'unavailable';
type PaginationState = {
  current: number;
  pageSize: number;
};

type CandidateListState = {
  items: ProjectImportRuntimeCandidate[];
  total: number;
  filterCounts: ProjectImportRuntimeCandidateFilterCounts;
};

const IMPORT_STEP_QUERY_KEY = 'step';
const IMPORT_CANDIDATE_QUERY_KEY = 'candidate';
const CANDIDATE_PAGE_SIZE = 10;
const ROUTE_RECOVERY_PAGE_SIZE = 50;
const CANDIDATE_COLUMN_STORAGE_KEY = 'graft.project.import.visibleColumns.v2';
const DEFAULT_VISIBLE_COLUMNS = [
  'project',
  'config_files',
  'working_directory',
  'runtime',
  'services',
  'status',
  'reason',
  'operation',
];
const ALL_COLUMN_KEYS = [
  'project',
  'config_files',
  'working_directory',
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

const { router, tabsRouterStore, t } = useProjectPageContext();
const route = useRoute();
const formRef = ref<FormInstanceFunctions | null>(null);
const candidatesLoading = ref(true);
const candidatesError = ref('');
const candidates = ref<ProjectImportRuntimeCandidate[]>([]);
const candidateListTotal = ref(0);
const candidateFilterCounts = ref<ProjectImportRuntimeCandidateFilterCounts>({
  all: 0,
  ready: 0,
  unavailable: 0,
});
const currentStep = ref<ImportWizardStep>('select');
const candidatesLoaded = ref(false);
const candidateSearchKeyword = ref('');
const candidateStatusFilter = ref<CandidateStatusFilter>('all');
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
  canonicalProjectNameOverride,
  displayName,
  hasPreview,
  importError,
  importLoading,
  inspectCandidate,
  inspectError,
  inspectLoading,
  inspectResult,
  refreshInspect,
  reset,
  selectedCandidateKey,
  submitImport,
} = useProjectImportFlow(t);

const formData = reactive({
  display_name: displayName,
  canonical_project_name_override: canonicalProjectNameOverride,
});

const formRules: FormProps['rules'] = {
  display_name: [{ required: true, message: t('project.import.validation.displayNameRequired') }],
};

const readyCandidates = computed(() => candidates.value.filter((item) => isProjectImportRuntimeCandidateReady(item)));
const selectedCandidate = computed(
  () => candidates.value.find((item) => item.candidate_key === selectedCandidateKey.value) ?? null,
);
const normalizedInspectResult = computed(() => normalizeProjectImportInspectResponse(inspectResult.value));
const selectedCandidateLabel = computed(
  () => normalizedInspectResult.value?.canonical_project_name || selectedCandidate.value?.canonical_project_name || '',
);
const resolvedWorkingDirectory = computed(
  () => normalizedInspectResult.value?.resolved_working_directory || selectedCandidate.value?.working_directory || '',
);

const normalizedCandidateSearch = computed(() => candidateSearchKeyword.value.trim());
const hasActiveCandidateFilters = computed(
  () => Boolean(candidateSearchKeyword.value.trim()) || candidateStatusFilter.value !== 'all',
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
    value: 'unavailable' as const,
    label: t('project.import.candidates.unavailableFilterLabel', { count: candidateFilterCounts.value.unavailable }),
  },
]);
const candidateCellSlotNames = [
  'project',
  'config_files',
  'working_directory',
  'runtime',
  'services',
  'status',
  'reason',
  'operation',
];
const candidateColumns = computed<TdBaseTableProps['columns']>(() => [
  createMainTextColumn(t('project.import.candidates.columnProject'), 'project', 280),
  createTechnicalColumn(t('project.import.candidates.columnConfigFiles'), 'config_files', 260),
  createTechnicalColumn(t('project.import.candidates.columnWorkingDirectory'), 'working_directory', 240),
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
  if (value === 'inspect' || value === 'confirm') {
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
      if (desiredStep === 'confirm') {
        await updateWizardRoute('inspect', { candidateKey, replace: true });
      }
      return;
    }
  }

  if (desiredStep === 'confirm' && !canImport.value) {
    currentStep.value = 'inspect';
    await updateWizardRoute('inspect', { candidateKey, replace: true });
    return;
  }

  currentStep.value = desiredStep;
}

async function findReadyCandidateByKey(candidateKey: string) {
  let offset = 0;
  let total = Number.POSITIVE_INFINITY;

  while (offset < total) {
    const response = await getProjectImportRuntimeCandidates({
      availability: 'ready',
      limit: ROUTE_RECOVERY_PAGE_SIZE,
      offset,
    });
    const nextState = normalizeCandidateListState(response);
    const candidate = nextState.items.find((item) => item.candidate_key === candidateKey);
    if (candidate && isProjectImportRuntimeCandidateReady(candidate)) {
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

function candidateRowClassName(params: { row: ProjectImportRuntimeCandidate }) {
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

function formatContainerCounts(counts: ProjectImportRuntimeCandidate['container_counts']) {
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

function candidateStatusTheme(status: ProjectImportRuntimeCandidate['status']) {
  if (status === 'ready') return 'success';
  if (status === 'broken_compose') return 'danger';
  if (status === 'incomplete_metadata') return 'warning';
  if (status === 'unsupported_runtime') return 'default';
  return 'default';
}

function candidateUnavailableReason(candidate: ProjectImportRuntimeCandidate) {
  const reasonKey = resolveProjectImportRuntimeCandidateReasonKey(candidate);
  const translated = t(`project.import.candidates.reason.${reasonKey}`);
  if (translated === `project.import.candidates.reason.${reasonKey}`) {
    return t('project.import.candidates.reason.unavailable');
  }
  return translated;
}

function candidateDiagnostics(candidate: ProjectImportRuntimeCandidate) {
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
    const response = await getProjectImportRuntimeCandidates(buildCandidateQuery());
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
      unavailable: 0,
    };
    candidatesError.value = resolveLocalizedErrorMessage(t, error, t('project.import.messages.candidateLoadFailed'));
  } finally {
    if (requestId === latestCandidateRequestId) {
      candidatesLoading.value = false;
    }
  }
}

function buildCandidateQuery(): ProjectImportRuntimeCandidatesQuery {
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
    ? response.items.map((candidate) => normalizeCandidate(candidate as ProjectImportRuntimeCandidate))
    : [];
  const rawFilterCounts = response?.filter_counts ?? response?.filterCounts;
  const filterCounts = isCandidateFilterCounts(rawFilterCounts) ? rawFilterCounts : undefined;
  return {
    items,
    total: typeof response?.total === 'number' ? response.total : items.length,
    filterCounts: {
      all: typeof filterCounts?.all === 'number' ? filterCounts.all : items.length,
      ready: typeof filterCounts?.ready === 'number' ? filterCounts.ready : 0,
      unavailable: typeof filterCounts?.unavailable === 'number' ? filterCounts.unavailable : 0,
    },
  };
}

function isCandidateFilterCounts(value: unknown): value is ProjectImportRuntimeCandidateFilterCounts {
  if (!value || typeof value !== 'object') {
    return false;
  }
  const candidate = value as Partial<ProjectImportRuntimeCandidateFilterCounts>;
  return (
    typeof candidate.all === 'number' &&
    typeof candidate.ready === 'number' &&
    typeof candidate.unavailable === 'number'
  );
}

function normalizeCandidate(candidate: ProjectImportRuntimeCandidate): ProjectImportRuntimeCandidate {
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
    name: route.name || PROJECT_BOOTSTRAP_ROUTE.IMPORT.pageRouteName,
    params: route.params,
    query: nextQuery,
  });
}

async function goToStep(step: ImportWizardStep, replace = false) {
  currentStep.value = step;
  await updateWizardRoute(step, { replace });
}

async function goToConfirmStep() {
  if (!canImport.value || !selectedCandidateKey.value) {
    return;
  }

  await goToStep('confirm');
}

async function handleRefreshInspect() {
  try {
    const result = await refreshInspect();
    if (result === 'applied' && inspectResult.value) {
      MessagePlugin.success(t('project.import.messages.inspectSuccess'));
      if (currentStep.value === 'confirm' && !canImport.value) {
        await goToStep('inspect', true);
      }
    }
  } catch {
    MessagePlugin.error(inspectError.value || t('project.import.messages.inspectFailed'));
  }
}

async function handleCandidateInspect(candidate: ProjectImportRuntimeCandidate) {
  if (!isProjectImportRuntimeCandidateReady(candidate)) {
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

async function handleSubmit(context: SubmitContext) {
  if (context.validateResult !== true) {
    return;
  }

  try {
    const response = await submitImport();
    MessagePlugin.success(t('project.import.messages.importSuccess'));
    openDetail(response);
  } catch {
    MessagePlugin.error(importError.value || t('project.import.messages.importFailed'));
  }
}

function openDetail(response: ProjectImportExecuteResponse) {
  const project = response.project;
  const target = {
    name: PROJECT_BOOTSTRAP_ROUTE.DETAIL.pageRouteName,
    params: { id: project.id },
    query: { tab: 'overview' },
  };
  const resolved = router.resolve(target);
  appendResolvedTab(
    tabsRouterStore,
    resolved,
    buildDetailTitleWithFallback('project.route.detail.title', project.display_name),
  );
  void router.push(target);
}

function resetCandidateFilters() {
  candidateSearchKeyword.value = '';
  candidateStatusFilter.value = 'all';
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
    // Column settings are a convenience preference; candidate rendering must not depend on storage availability.
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
.project-import-authority,
.project-import-form-grid,
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
  padding: var(--graft-density-gap-4);
}

.project-import-workflow__eyebrow {
  color: var(--td-brand-color);
  font: var(--td-font-body-small);
  font-weight: 600;
  margin: 0;
}

.project-import-workflow__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
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

.project-import-step-actions,
.project-import-form-actions {
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

.project-import-form-grid {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.project-import-step :deep(.project-import-candidate-row--active > td) {
  background: color-mix(in srgb, var(--td-brand-color) 8%, var(--td-bg-color-container));
}

@media (width <= 1080px) {
  .project-import-step-grid,
  .project-import-form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
