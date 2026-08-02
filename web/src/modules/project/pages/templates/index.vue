<template>
  <advanced-query-list-page
    root-class="application-template-list"
    title-key="project.route.templates.title"
    description-key="project.templates.description"
    :error-message="errorMessage"
    :error-title="t('project.templates.loadFailed')"
    :loading="loading"
    :reload-label="t('project.templates.refresh')"
    :retry-label="t('project.templates.retry')"
    :source="{ labelKey: 'project.templates.eyebrow', fallback: t('project.templates.eyebrow') }"
    @reload="loadTemplates"
  >
    <template #actions>
      <t-tooltip v-if="isCompact" :content="t('project.templates.create')" placement="bottom">
        <t-button :aria-label="t('project.templates.create')" shape="square" theme="primary" @click="openCreateDialog">
          <template #icon><add-icon /></template>
        </t-button>
      </t-tooltip>
      <t-button v-else theme="primary" @click="openCreateDialog">{{ t('project.templates.create') }}</t-button>
    </template>
    <template #feedback-extra>
      <management-statistics-bar
        :items="[
          { label: t('project.templates.total'), value: catalogTotal },
        ]"
        :label="t('project.templates.summaryLabel')"
        layout="summary"
      />
    </template>
    <template #filters>
      <advanced-query-filter-builder
        :active-preset="activePreset"
        :add-filter-label="`+ ${t('project.templates.filters.add')}`"
        :add-sorter-label="t('project.templates.filters.addSort')"
        :builder-hint="t('project.templates.description')"
        :builder-title="t('project.templates.filters.title')"
        :compact-mode="isCompact"
        :compact-toggle-label="t('project.templates.filters.filter')"
        :field-values="fieldValues"
        :fields="filterFields"
        :filters-group-label="t('project.templates.filters.title')"
        :keyword="filters.keyword"
        :keyword-placeholder="t('project.templates.searchPlaceholder')"
        :loading="loading"
        :move-down-label="t('project.templates.filters.moveDown')"
        :move-up-label="t('project.templates.filters.moveUp')"
        :preset-label="t('project.templates.filters.presets')"
        :presets="presets"
        :remove-sorter-label="t('project.templates.filters.removeSort')"
        :reset-label="t('project.templates.filters.reset')"
        :search-label="t('project.templates.filters.search')"
        :selected-field-key="selectedFieldKey"
        :show-sorter-builder="true"
        :sort-add-disabled="true"
        :sort-direction-options="sortDirectionOptions"
        :sort-direction-placeholder="t('project.templates.filters.sortDirection')"
        :sort-field-options-by-index="[sortOptions]"
        sort-field-key="sort"
        :sort-field-placeholder="t('project.templates.filters.sortField')"
        :sort-move-down-disabled="[true]"
        :sort-move-up-disabled="[true]"
        :sorters="[filters.sorter]"
        :tags="filterTags"
        time-field-key="updatedRange"
        :time-fields="timeFields"
        @apply-preset="applyPreset"
        @reset="resetFilters"
        @search="applyQuery"
        @remove-sorter="resetSorter"
        @update:field="updateFilterField"
        @update:keyword="filters.keyword = $event"
        @update:selected-field-key="selectedFieldKey = $event"
        @update:sort-direction="updateSortDirection"
        @update:sort-field="updateSortField"
        @update:time-field="updateTimeField"
      >
        <template #saved-query-views><saved-query-view-control :controller="savedViews" /></template>
      </advanced-query-filter-builder>
    </template>
    <template #table>
      <advanced-query-paged-table
        v-if="isTablePresentation"
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :cell-slot-names="['displayName', 'description', 'status', 'version', 'updatedAt', 'operation']"
        :columns="visibleColumns"
        :empty-description="t('project.templates.emptyDescription')"
        :empty-title="t('project.templates.emptyTitle')"
        :footer-summary="paginationSummary"
        :head-label="t('project.templates.total')"
        :loading="loading"
        row-key="template_id"
        :rows="templates"
        :total="pagination.total"
        @page-change="handlePageChange"
      >
        <template #toolbar>
          <table-view-toolbar
            :column-settings-label="t('project.templates.columnSettings')"
            :refresh-label="t('project.templates.refresh')"
            :refresh-loading="loading"
            @column-settings="columnDrawerVisible = true"
            @refresh="loadTemplates"
          />
        </template>
        <template #displayName="{ row }">
          <button class="application-template-list__name" type="button" @click="openTemplate(row)">{{ row.display_name }}</button>
        </template>
        <template #description="{ row }">{{ row.description || t('project.templates.noDescription') }}</template>
        <template #status="{ row }"><t-tag :theme="statusTheme(row)" variant="light-outline">{{ statusLabel(row) }}</t-tag></template>
        <template #version="{ row }">{{ t('project.templates.versionValue', { version: row.version.version_number }) }}</template>
        <template #updatedAt="{ row }">{{ updatedAtLabel(row) }}</template>
        <template #operation="{ row }"><table-action-menu :actions="templateActions(row)" @action="handleTemplateAction($event, row)" /></template>
      </advanced-query-paged-table>
      <template-card-list
        v-else
        :actions-for="templateActions"
        :adapter-label="t('project.templates.adapterCompose')"
        :empty-description="t('project.templates.emptyDescription')"
        :empty-title="t('project.templates.emptyTitle')"
        :loading="loading"
        :no-description-label="t('project.templates.noDescription')"
        :status-label="statusLabel"
        :status-theme="statusTheme"
        :templates="templates"
        :updated-at-label="updatedAtLabel"
        @action="handleTemplateAction"
        @open="openTemplate"
      />
    </template>
    <template #detail>
      <advanced-query-column-drawer
        v-model:visible="columnDrawerVisible"
        v-model:selected-keys="visibleColumnKeys"
        :columns="columnOptions"
        :default-selected-keys="DEFAULT_COLUMNS"
        :reset-label="t('project.templates.resetColumns')"
        :title="t('project.templates.columnSettings')"
      />
      <t-dialog v-model:visible="cloneVisible" :header="t('project.templates.cloneTitle')" :confirm-btn="t('project.templates.cloneConfirm')" :cancel-btn="t('project.templates.cancel')" :confirm-loading="cloning" @confirm="cloneTemplate">
        <t-form label-align="top"><t-form-item :label="t('project.templates.name')"><t-input v-model="cloneDisplayName" /></t-form-item></t-form>
      </t-dialog>
      <t-dialog v-model:visible="deleteVisible" theme="danger" :header="t('project.templates.deleteTitle')" :body="t('project.templates.deleteBody')" :confirm-btn="t('project.templates.deleteConfirm')" :cancel-btn="t('project.templates.cancel')" :confirm-loading="deleting" @confirm="deleteTemplate" />
    </template>
  </advanced-query-list-page>
</template>
<script setup lang="ts">
import { AddIcon } from 'tdesign-icons-vue-next';
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { isNavigationFailure, NavigationFailureType, useRoute, useRouter } from 'vue-router';

import { ManagementStatisticsBar, TableActionMenu, TableViewToolbar } from '@/shared/components/management';
import {
  AdvancedQueryColumnDrawer,
  AdvancedQueryFilterBuilder,
  type AdvancedQueryFilterFieldDefinition,
  type AdvancedQueryFilterTag,
  AdvancedQueryListPage,
  AdvancedQueryPagedTable,
  applySavedQueryViewPresentation,
  normalizeSavedQueryView,
  SavedQueryViewControl,
  useSavedQueryViews,
} from '@/shared/components/query-list';
import { useViewportResponsiveVariant } from '@/shared/composables/useViewportResponsiveVariant';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability/time';
import { localDateTimeToUtcIso, normalizeRouteRangeForPageState } from '@/shared/observability/time-range';

import {
  deleteApplicationTemplate,
  deleteApplicationTemplateSavedView,
  getApplicationManagedTemplates,
  getApplicationTemplateSavedViews,
  postApplicationTemplateArchive,
  postApplicationTemplateClone,
  postApplicationTemplatePublish,
  postApplicationTemplateSavedView,
  postApplicationTemplateWithdraw,
  putApplicationTemplateSavedView,
} from '../../api/project';
import { PROJECT_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import type { ApplicationTemplate, ApplicationTemplateSavedViewRequest } from '../../types/project';
import TemplateCardList from './TemplateCardList.vue';

defineOptions({ name: 'ApplicationTemplateListIndex' });

// URL 是可分享查询状态的优先来源；保存筛选只在 URL 未指定查询条件时作为初始偏好生效。
type TemplateStatus = 'draft' | 'published' | 'archived';
type TemplateSortField = 'updated_at' | 'display_name' | 'status' | 'version_number';
type TemplateSort = `${TemplateSortField}:asc` | `${TemplateSortField}:desc`;
type TemplateFilters = { keyword: string; status: TemplateStatus | ''; updatedAfter: string; updatedBefore: string; sorter: { field: TemplateSortField; direction: 'asc' | 'desc' } };
type TemplateSavedQueryState = { keyword?: string; status?: TemplateStatus; updated_after?: string; updated_before?: string; sort?: TemplateSort };
type TemplateSavedViewState = { pageSize: number; queryState: TemplateSavedQueryState; visibleColumns: string[] };
type TemplateManagedQuery = NonNullable<Parameters<typeof getApplicationManagedTemplates>[0]>;

const DEFAULT_COLUMNS = ['displayName', 'description', 'status', 'version', 'updatedAt', 'operation'];
const { t, locale } = useI18n();
const route = useRoute();
const router = useRouter();
const viewportVariant = useViewportResponsiveVariant();
const templates = ref<ApplicationTemplate[]>([]);
const loading = ref(false);
const cloning = ref(false);
const deleting = ref(false);
const errorMessage = ref('');
const catalogTotal = ref(0);
const pagination = ref({ current: 1, pageSize: 20, total: 0 });
const filters = ref<TemplateFilters>(createDefaultFilters());
const selectedFieldKey = ref('status');
const visibleColumnKeys = ref([...DEFAULT_COLUMNS]);
const columnDrawerVisible = ref(false);
const cloneVisible = ref(false);
const deleteVisible = ref(false);
const selectedTemplate = ref<ApplicationTemplate | null>(null);
const cloneDisplayName = ref('');
const applyingRoute = ref(false);
let routeSyncTimer: ReturnType<typeof setTimeout> | undefined;

const isCompact = computed(() => viewportVariant.value.density === 'compact');
const isTablePresentation = computed(() => viewportVariant.value.density === 'spacious');
const activePreset = computed(() => (filters.value.status || 'all'));
const sortOptions = computed(() => [
  { label: t('project.templates.sort.updatedAt'), value: 'updated_at' as const },
  { label: t('project.templates.sort.name'), value: 'display_name' as const },
  { label: t('project.templates.sort.status'), value: 'status' as const },
  { label: t('project.templates.sort.version'), value: 'version_number' as const },
]);
const sortDirectionOptions = computed(() => [{ label: t('project.templates.sort.desc'), value: 'desc' }, { label: t('project.templates.sort.asc'), value: 'asc' }]);
const filterFields = computed<AdvancedQueryFilterFieldDefinition[]>(() => [
  { key: 'status', label: t('project.templates.status'), kind: 'select', options: statusOptions.value },
  { key: 'updatedRange', label: t('project.templates.filters.updatedRange'), kind: 'special' },
  { key: 'sort', label: t('project.templates.filters.sort'), kind: 'special' },
]);
const statusOptions = computed(() => [
  { label: t('project.templates.statusDraft'), value: 'draft' },
  { label: t('project.templates.statusPublished'), value: 'published' },
  { label: t('project.templates.statusArchived'), value: 'archived' },
]);
const fieldValues = computed(() => ({ status: filters.value.status }));
const timeFields = computed(() => [{
  key: 'updatedRange',
  label: t('project.templates.filters.updatedRange'),
  value: [filters.value.updatedAfter, filters.value.updatedBefore].filter(Boolean),
  placeholder: [t('project.templates.filters.updatedAfter'), t('project.templates.filters.updatedBefore')] as [string, string],
}]);
const presets = computed(() => [
  { key: 'all', title: t('project.templates.presets.all') },
  ...statusOptions.value.map((option) => ({ key: option.value, title: option.label })),
]);
const filterTags = computed<AdvancedQueryFilterTag[]>(() => [
  ...(filters.value.status ? [{ key: 'status', label: `${t('project.templates.status')}=${statusLabelValue(filters.value.status)}` }] : []),
  ...(filters.value.updatedAfter || filters.value.updatedBefore ? [{ key: 'updatedRange', label: t('project.templates.filters.updatedRangeActive') }] : []),
  { key: 'sort', label: `${t('project.templates.filters.sort')}=${filters.value.sorter.field}:${filters.value.sorter.direction}` },
]);
const columnOptions = computed(() => [
  { label: t('project.templates.name'), value: 'displayName' }, { label: t('project.templates.descriptionField'), value: 'description' },
  { label: t('project.templates.status'), value: 'status' }, { label: t('project.templates.version'), value: 'version' },
  { label: t('project.templates.updatedAtColumn'), value: 'updatedAt' }, { label: t('project.templates.operation'), value: 'operation' },
]);
const columns = computed<NonNullable<TableProps['columns']>>(() => [
  { colKey: 'displayName', title: t('project.templates.name'), width: 260 }, { colKey: 'description', title: t('project.templates.descriptionField'), width: 260 },
  { colKey: 'status', title: t('project.templates.status'), width: 130 }, { colKey: 'version', title: t('project.templates.version'), width: 100 },
  { colKey: 'updatedAt', title: t('project.templates.updatedAtColumn'), width: 180 }, { colKey: 'operation', title: t('project.templates.operation'), width: 140, fixed: 'right' },
]);
const visibleColumns = computed(() => columns.value.filter((column) => visibleColumnKeys.value.includes(String(column.colKey))));
const paginationSummary = computed(() => t('project.templates.paginationSummary', { total: pagination.value.total }));

const savedViews = useSavedQueryViews<TemplateSavedViewState, number>({
  adapter: {
    list: async () => (await getApplicationTemplateSavedViews()).map((view) => normalizeSavedQueryView<TemplateSavedQueryState, number>(view)),
    create: async (input) => normalizeSavedQueryView<TemplateSavedQueryState, number>(await postApplicationTemplateSavedView(toSavedViewRequest(input))),
    update: async (id, input) => normalizeSavedQueryView<TemplateSavedQueryState, number>(await putApplicationTemplateSavedView(id, toSavedViewRequest(input))),
    remove: async (id) => { await deleteApplicationTemplateSavedView(id); },
  },
  applyView: async (view) => { applySavedState(view.state); await replaceRoute(); await loadTemplates(); },
  onError: (error, operation) => MessagePlugin.error(resolveLocalizedErrorMessage(t, error, operation === 'delete' ? t('project.templates.savedViews.deleteFailed') : t('project.templates.savedViews.conflict'))),
  serializeCurrentState: () => ({ pageSize: pagination.value.pageSize, queryState: currentQueryState(), visibleColumns: [...visibleColumnKeys.value] }),
});

onMounted(async () => {
  hydrateFromRoute();
  await savedViews.load({ hasExplicitState: hasExplicitRouteState() });
  await loadTemplates();
});
onBeforeUnmount(() => { if (routeSyncTimer) clearTimeout(routeSyncTimer); });
watch([filters, () => pagination.value.current, () => pagination.value.pageSize], () => { if (!applyingRoute.value) scheduleRouteReplace(); }, { deep: true });
watch(visibleColumnKeys, () => { if (!applyingRoute.value) scheduleRouteReplace(); }, { deep: true });

function createDefaultFilters(): TemplateFilters { return { keyword: '', status: '', updatedAfter: '', updatedBefore: '', sorter: { field: 'updated_at', direction: 'desc' } }; }
function currentQueryState(): TemplateSavedQueryState { return { ...(filters.value.keyword.trim() ? { keyword: filters.value.keyword.trim() } : {}), ...(filters.value.status ? { status: filters.value.status } : {}), ...(filters.value.updatedAfter ? { updated_after: localDateTimeToUtcIso(filters.value.updatedAfter) } : {}), ...(filters.value.updatedBefore ? { updated_before: localDateTimeToUtcIso(filters.value.updatedBefore) } : {}), sort: `${filters.value.sorter.field}:${filters.value.sorter.direction}` as TemplateSort }; }
function requestQuery(): TemplateManagedQuery { return { ...currentQueryState(), limit: pagination.value.pageSize as TemplateManagedQuery['limit'], offset: (pagination.value.current - 1) * pagination.value.pageSize }; }
function toSavedViewRequest(input: { name: string; isDefault: boolean; state: TemplateSavedViewState }): ApplicationTemplateSavedViewRequest { return { name: input.name, page_size: input.state.pageSize, query_state: input.state.queryState, visible_columns: input.state.visibleColumns as ApplicationTemplateSavedViewRequest['visible_columns'], is_default: input.isDefault }; }
function applySavedState(state: TemplateSavedViewState) { const query = state.queryState; const range = normalizeRouteRangeForPageState([query.updated_after ?? '', query.updated_before ?? '']); filters.value = { keyword: query.keyword ?? '', status: query.status ?? '', updatedAfter: range[0] ?? '', updatedBefore: range[1] ?? '', sorter: parseSorter(query.sort) }; applySavedQueryViewPresentation(state, { pagination: pagination.value, supportedColumns: DEFAULT_COLUMNS, visibleColumnKeys }); }
function applyPreset(preset: string) { filters.value.status = preset === 'all' ? '' : (preset as TemplateStatus); applyQuery(); }
function applyQuery() { pagination.value.current = 1; void loadTemplates(); }
function resetFilters() { filters.value = createDefaultFilters(); applyQuery(); }
function resetSorter() { filters.value.sorter = createDefaultFilters().sorter; applyQuery(); }
function updateFilterField(payload: { key: string; value: string | string[] }) { if (payload.key === 'status') { const value = Array.isArray(payload.value) ? payload.value[0] : payload.value; filters.value.status = statusOptions.value.some((option) => option.value === value) ? (value as TemplateStatus) : ''; } }
function updateSortField(payload: { value: string | number | Array<string | number> | undefined }) { const field = Array.isArray(payload.value) ? payload.value[0] : payload.value; if (sortOptions.value.some((option) => option.value === field)) filters.value.sorter.field = field as TemplateSortField; applyQuery(); }
function updateSortDirection(payload: { value: string | number | Array<string | number> | undefined }) { filters.value.sorter.direction = (Array.isArray(payload.value) ? payload.value[0] : payload.value) === 'asc' ? 'asc' : 'desc'; applyQuery(); }
function updateTimeField(payload: { key: string; value: string[] }) { if (payload.key === 'updatedRange') { filters.value.updatedAfter = payload.value[0] ?? ''; filters.value.updatedBefore = payload.value[1] ?? ''; } }
function handlePageChange() { void loadTemplates(); }
async function loadTemplates() { loading.value = true; errorMessage.value = ''; try { const response = await getApplicationManagedTemplates(requestQuery()); templates.value = response.items; pagination.value.total = response.total; pagination.value.pageSize = response.limit; pagination.value.current = Math.floor(response.offset / response.limit) + 1; catalogTotal.value = response.total; } catch (error) { errorMessage.value = resolveLocalizedErrorMessage(t, error, t('project.templates.loadFailed')); } finally { loading.value = false; } }
function hasExplicitRouteState() { return ['keyword', 'status', 'updated_after', 'updated_before', 'sort', 'page', 'page_size', 'columns'].some((key) => route.query[key] !== undefined); }
function hydrateFromRoute() { applyingRoute.value = true; const query = route.query; const range = normalizeRouteRangeForPageState([stringQuery(query.updated_after), stringQuery(query.updated_before)]); filters.value = { keyword: stringQuery(query.keyword), status: isStatus(stringQuery(query.status)) ? (stringQuery(query.status) as TemplateStatus) : '', updatedAfter: range[0] ?? '', updatedBefore: range[1] ?? '', sorter: parseSorter(stringQuery(query.sort)) }; pagination.value.current = Math.max(1, Number(stringQuery(query.page)) || 1); pagination.value.pageSize = [10, 20, 50, 100].includes(Number(stringQuery(query.page_size))) ? Number(stringQuery(query.page_size)) : 20; const columns = stringQuery(query.columns).split(',').filter((key) => DEFAULT_COLUMNS.includes(key)); visibleColumnKeys.value = columns.length ? columns : [...DEFAULT_COLUMNS]; applyingRoute.value = false; }
async function replaceRoute() { await router.replace({ query: { ...(filters.value.keyword.trim() ? { keyword: filters.value.keyword.trim() } : {}), ...(filters.value.status ? { status: filters.value.status } : {}), ...(filters.value.updatedAfter ? { updated_after: localDateTimeToUtcIso(filters.value.updatedAfter) } : {}), ...(filters.value.updatedBefore ? { updated_before: localDateTimeToUtcIso(filters.value.updatedBefore) } : {}), sort: `${filters.value.sorter.field}:${filters.value.sorter.direction}`, page: String(pagination.value.current), page_size: String(pagination.value.pageSize), columns: visibleColumnKeys.value.join(',') } }); }
function scheduleRouteReplace() {
  if (routeSyncTimer) clearTimeout(routeSyncTimer);
  routeSyncTimer = setTimeout(() => { routeSyncTimer = undefined; void replaceRoute(); }, 200);
}
function stringQuery(value: unknown) { return Array.isArray(value) ? String(value[0] ?? '') : typeof value === 'string' ? value : ''; }
function isStatus(value: string): value is TemplateStatus { return ['draft', 'published', 'archived'].includes(value); }
function parseSorter(value?: string): TemplateFilters['sorter'] { const [field, direction] = (value || 'updated_at:desc').split(':'); return { field: sortOptions.value.some((option) => option.value === field) ? (field as TemplateSortField) : 'updated_at', direction: direction === 'asc' ? 'asc' : 'desc' }; }
function statusLabelValue(status: TemplateStatus) { return status === 'draft' ? t('project.templates.statusDraft') : status === 'published' ? t('project.templates.statusPublished') : t('project.templates.statusArchived'); }
function isArchived(template: ApplicationTemplate) { return Boolean(template.archived_at); }
function isDraft(template: ApplicationTemplate) { return !isArchived(template) && template.version.status === 'draft'; }
function isPublished(template: ApplicationTemplate) { return !isArchived(template) && template.version.status === 'published'; }
function statusLabel(template: ApplicationTemplate) { return isArchived(template) ? t('project.templates.statusArchived') : isDraft(template) ? t('project.templates.statusDraft') : t('project.templates.statusPublished'); }
function statusTheme(template: ApplicationTemplate) { return isArchived(template) ? 'default' : isDraft(template) ? 'warning' : 'success'; }
function updatedAtLabel(template: ApplicationTemplate) { return t('project.templates.updatedAt', { value: formatLocaleDateTime(template.updated_at, locale.value, { dateStyle: 'medium' }) }); }
function templateActions(template: ApplicationTemplate) { return [{ value: 'detail', label: isDraft(template) ? t('project.templates.edit') : t('project.templates.open') }, ...(isDraft(template) ? [{ value: 'publish', label: t('project.templates.publish') }] : []), { value: 'clone', label: t('project.templates.clone') }, ...(isPublished(template) ? [{ value: 'withdraw', label: t('project.templates.withdraw') }] : []), ...(!isArchived(template) ? [{ value: 'archive', label: t('project.templates.archive') }] : []), { value: 'delete', label: t('project.templates.delete') }]; }
function handleTemplateAction(action: string, template: ApplicationTemplate) { if (action === 'detail') void openTemplate(template); else if (action === 'publish') void publishTemplate(template); else if (action === 'clone') openCloneDialog(template); else if (action === 'withdraw') void withdrawTemplate(template); else if (action === 'archive') void archiveTemplate(template); else if (action === 'delete') openDeleteDialog(template); }
function openCreateDialog() { void router.push({ name: PROJECT_BOOTSTRAP_ROUTE.TEMPLATE_CREATE.pageRouteName }); }
async function openTemplate(template: ApplicationTemplate) { try { const failure = await router.push({ name: PROJECT_BOOTSTRAP_ROUTE.TEMPLATE_DETAIL.pageRouteName, params: { templateId: template.template_id } }); if (failure && !isNavigationFailure(failure, NavigationFailureType.duplicated)) MessagePlugin.error(t('project.templates.detailNavigationFailed')); } catch { MessagePlugin.error(t('project.templates.detailNavigationFailed')); } }
function openCloneDialog(template: ApplicationTemplate) { selectedTemplate.value = template; cloneDisplayName.value = `${template.display_name} ${t('project.templates.cloneSuffix')}`; cloneVisible.value = true; }
function openDeleteDialog(template: ApplicationTemplate) { selectedTemplate.value = template; deleteVisible.value = true; }
async function cloneTemplate() { if (!selectedTemplate.value || !cloneDisplayName.value.trim()) return; cloning.value = true; try { const template = await postApplicationTemplateClone(selectedTemplate.value.template_id, cloneDisplayName.value.trim()); cloneVisible.value = false; await openTemplate(template); } catch (error) { MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.cloneFailed'))); } finally { cloning.value = false; } }
async function publishTemplate(template: ApplicationTemplate) { try { await postApplicationTemplatePublish(template.template_id); await loadTemplates(); MessagePlugin.success(t('project.templates.publishSuccess')); } catch (error) { MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.publishFailed'))); } }
async function withdrawTemplate(template: ApplicationTemplate) { try { await postApplicationTemplateWithdraw(template.template_id); await loadTemplates(); MessagePlugin.success(t('project.templates.withdrawSuccess')); } catch (error) { MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.withdrawFailed'))); } }
async function archiveTemplate(template: ApplicationTemplate) { try { await postApplicationTemplateArchive(template.template_id); await loadTemplates(); MessagePlugin.success(t('project.templates.archiveSuccess')); } catch (error) { MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.archiveFailed'))); } }
async function deleteTemplate() { if (!selectedTemplate.value) return; deleting.value = true; try { await deleteApplicationTemplate(selectedTemplate.value.template_id); deleteVisible.value = false; await loadTemplates(); MessagePlugin.success(t('project.templates.deleteSuccess')); } catch (error) { MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('project.templates.deleteFailed'))); } finally { deleting.value = false; } }
</script>
<style scoped>
.application-template-list__name {
  background: none;
  border: 0;
  color: var(--td-brand-color);
  cursor: pointer;
  font: inherit;
  font-weight: 600;
  padding: 0;
  text-align: left;
}

.application-template-list__name + p {
  color: var(--td-text-color-secondary);
  margin: var(--graft-density-gap-4) 0 0;
  overflow-wrap: anywhere;
}
</style>
