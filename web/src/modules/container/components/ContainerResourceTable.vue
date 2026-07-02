<template>
  <management-paged-table
    v-model:current="current"
    v-model:page-size="pageSize"
    :cell-slot-names="cellSlotNames"
    :columns="columns"
    :description="props.headDescription"
    :empty-description="props.emptyDescription"
    :empty-title="props.emptyTitle"
    :footer-summary="props.footerSummary"
    :loading="props.loading"
    :pagination-props="props.paginationProps"
    :row-key="'id'"
    :rows="props.rows"
    :selected-row-keys="props.selectedRowKeys"
    :size="props.tableDensity"
    :sort="props.sort"
    :summary="props.headSummary"
    :total="props.total"
    @page-change="(pageInfo) => $emit('page-change', pageInfo)"
    @row-click="(row) => $emit('row-click', row as ContainerSummaryRecord)"
    @select-change="(rowKeys) => $emit('select-change', rowKeys)"
    @sort-change="(nextSort) => $emit('sort-change', nextSort)"
  >
    <template v-if="$slots.toolbar" #toolbar>
      <slot name="toolbar" />
    </template>
    <template v-if="$slots.batch" #batch>
      <slot name="batch" />
    </template>
    <template v-if="$slots.feedback" #feedback>
      <slot name="feedback" />
    </template>
    <template v-if="$slots['empty-action']" #empty-action>
      <slot name="empty-action" />
    </template>

    <template #state="{ row }">
      <t-tag :theme="stateTheme(row.state)" variant="light-outline">
        {{ t(`container.list.states.${row.state}`) }}
      </t-tag>
    </template>

    <template #name="{ row }">
      <div class="container-identity">
        <span class="container-identity__name">{{ displayContainerName(row) }}</span>
        <t-tooltip :content="row.id" placement="top-left">
          <span class="container-identity__id">{{ shortContainerId(row.id) }}</span>
        </t-tooltip>
      </div>
    </template>

    <template #image="{ row }">
      <div class="container-image">
        <span>{{ row.image }}</span>
        <span v-if="row.runtime" class="container-muted">{{ row.runtime }}</span>
      </div>
    </template>

    <template #ports="{ row }">
      <div v-if="visiblePortLabels(row).length" class="container-port-list">
        <t-tag v-for="port in visiblePortLabels(row)" :key="port" size="small" theme="default" variant="light">
          {{ port }}
        </t-tag>
        <t-tooltip v-if="hiddenPortLabels(row).length" :content="hiddenPortLabels(row).join(' / ')" placement="top">
          <t-tag size="small" theme="primary" variant="light">
            {{ t('container.list.morePorts', { count: hiddenPortLabels(row).length }) }}
          </t-tag>
        </t-tooltip>
      </div>
      <span v-else>-</span>
    </template>

    <template #runtime_status="{ row }">
      <div class="container-runtime-status">
        <span class="container-runtime-status__text">{{ row.status || '-' }}</span>
        <t-tag v-if="shouldShowHealthTag(row.health)" :theme="healthTheme(row.health)" size="small" variant="light">
          {{ healthLabel(row.health) }}
        </t-tag>
      </div>
    </template>

    <template #network="{ row }">
      <div class="container-runtime-status">
        <span>{{ row.primary_ip || '-' }}</span>
        <span>{{ row.network_summary || '-' }}</span>
      </div>
    </template>

    <template #source="{ row }">
      <div class="container-source-cell">
        <div class="container-source-cell__header">
          <t-tag :theme="orchestratorTheme(row)" size="small" variant="light-outline">
            {{ orchestratorLabel(readContainerOrchestratorType(row)) }}
          </t-tag>
        </div>
        <div v-if="sourceGroupFilter(row)" class="container-source-cell__line">
          <span class="container-source-cell__label">{{
            t(`container.list.sourceKinds.${sourceGroupFilter(row)?.kind}`)
          }}</span>
          <t-button
            data-testid="container-source-group-filter"
            size="small"
            theme="primary"
            variant="text"
            @click="handleSourceFilterClick($event, row, 'group')"
          >
            {{ sourceGroupFilter(row)?.value }}
          </t-button>
        </div>
        <div v-if="sourceMemberFilter(row)" class="container-source-cell__line">
          <span class="container-source-cell__label">{{
            t(`container.list.sourceKinds.${sourceMemberFilter(row)?.kind}`)
          }}</span>
          <t-button
            data-testid="container-source-member-filter"
            size="small"
            theme="default"
            variant="text"
            @click="handleSourceFilterClick($event, row, 'member')"
          >
            {{ sourceMemberFilter(row)?.value }}
          </t-button>
        </div>
        <span v-if="!sourceGroupFilter(row) && !sourceMemberFilter(row)" class="container-muted">
          {{ orchestratorSummary(row) }}
        </span>
      </div>
    </template>

    <template #cpu="{ row }">
      <container-resource-metric-cell :metric="cpuMetric(row)" test-id="container-cpu-meter" />
    </template>

    <template #memory="{ row }">
      <container-resource-metric-cell :metric="memoryMetric(row)" test-id="container-memory-meter" />
    </template>

    <template #resource="{ row }">
      <span>{{ resourceSummary(row) }}</span>
    </template>

    <template #image_id="{ row }">
      <span>{{ row.image_id || '-' }}</span>
    </template>

    <template #labels="{ row }">
      <span>{{ labelSummary(row) }}</span>
    </template>

    <template #created_at="{ row }">
      {{ formatLocaleDateTime(row.created_at, locale) }}
    </template>

    <template #started_at="{ row }">
      {{ formatLocaleDateTime(row.started_at, locale) }}
    </template>

    <template #restart_policy="{ row }">
      {{ row.restart_policy || '-' }}
    </template>

    <template #operation="{ row }">
      <slot name="operation" :row="row">
        <table-action-menu
          v-if="props.rowActions"
          :actions="props.rowActions(row)"
          :more-label="resolvedMoreActionsLabel"
          :more-label-fallback="resolvedMoreActionsLabel"
          @action="(action) => $emit('action', { action, row })"
        />
      </slot>
    </template>
  </management-paged-table>
</template>
<script setup lang="ts">
import type { PageInfo, PaginationProps, TableSort, TdBaseTableProps } from 'tdesign-vue-next';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { ManagementPagedTable, resolveManagedColumns, TableActionMenu } from '@/shared/components/management';
import { formatLocaleDateTime } from '@/shared/observability';

import {
  buildContainerResourceColumns,
  type ContainerResourceRowAction,
  type ContainerSourceQuickFilter,
  type ContainerSourceQuickFilterTarget,
  createContainerSourceQuickFilter,
  displayContainerName,
  readContainerOrchestratorType,
  shortContainerId,
} from '../shared/resource-table';
import { selectContainerStatsChangeState } from '../shared/stats-manager';
import { metricChangedClass, metricProgressStatus } from '../shared/stats-visual-state';
import type {
  ContainerHealth,
  ContainerOrchestratorType,
  ContainerState,
  ContainerSummaryRecord,
} from '../types/container';
import ContainerResourceMetricCell from './ContainerResourceMetricCell.vue';

type ResourceMetric = {
  available: boolean;
  changeClass: Record<string, boolean>;
  percentage: number;
  progressStatus: 'success' | 'warning' | undefined;
  tooltip: string;
  value: string;
};

const CONTAINER_PORT_VISIBLE_LIMIT = 2;
const BYTES_PER_MIB = 1024 * 1024;

const props = withDefaults(
  defineProps<{
    alwaysVisibleColumnKeys?: string[];
    emptyDescription: string;
    emptyTitle: string;
    footerSummary: string;
    headDescription?: string;
    headSummary?: string;
    loading?: boolean;
    moreActionsLabel?: string;
    paginationProps?: Partial<PaginationProps>;
    rowActions?: (row: ContainerSummaryRecord) => ContainerResourceRowAction[];
    rows: ContainerSummaryRecord[];
    selectedRowKeys?: Array<string | number>;
    sort?: TableSort;
    tableDensity?: TdBaseTableProps['size'];
    total: number;
    visibleColumnKeys?: string[];
  }>(),
  {
    alwaysVisibleColumnKeys: () => [],
    headDescription: '',
    headSummary: '',
    loading: false,
    moreActionsLabel: '',
    paginationProps: () => ({}),
    rowActions: undefined,
    selectedRowKeys: () => [],
    sort: undefined,
    tableDensity: undefined,
    visibleColumnKeys: undefined,
  },
);

const emit = defineEmits<{
  (e: 'action', payload: { action: string; row: ContainerSummaryRecord }): void;
  (e: 'page-change', pageInfo: PageInfo): void;
  (e: 'row-click', row: ContainerSummaryRecord): void;
  (e: 'select-change', rowKeys: Array<string | number>): void;
  (e: 'sort-change', sort: TableSort): void;
  (e: 'source-filter', filter: ContainerSourceQuickFilter): void;
}>();

const current = defineModel<number>('current', { required: true });
const pageSize = defineModel<number>('pageSize', { required: true });

const { locale, t, te } = useI18n();
const cellSlotNames = [
  'state',
  'name',
  'image',
  'ports',
  'runtime_status',
  'network',
  'source',
  'cpu',
  'memory',
  'resource',
  'image_id',
  'labels',
  'created_at',
  'started_at',
  'restart_policy',
  'operation',
];
const columns = computed<TdBaseTableProps['columns']>(() =>
  resolveManagedColumns(buildContainerResourceColumns(t), props.visibleColumnKeys, props.alwaysVisibleColumnKeys),
);
const resolvedMoreActionsLabel = computed(() => props.moreActionsLabel || t('container.list.actions.more'));

function stateTheme(state: ContainerState) {
  if (state === 'running') return 'success';
  if (state === 'created' || state === 'paused' || state === 'restarting') return 'warning';
  if (state === 'dead') return 'danger';
  return 'default';
}

function formatPorts(ports: ContainerSummaryRecord['ports']) {
  return ports.map((port) => {
    const target = `${port.private_port}/${port.type}`;
    if (port.public_port === undefined) {
      return target;
    }
    return `${port.ip ? `${port.ip}:` : ''}${port.public_port}->${target}`;
  });
}

function visiblePortLabels(row: ContainerSummaryRecord) {
  return formatPorts(row.ports).slice(0, CONTAINER_PORT_VISIBLE_LIMIT);
}

function hiddenPortLabels(row: ContainerSummaryRecord) {
  return formatPorts(row.ports).slice(CONTAINER_PORT_VISIBLE_LIMIT);
}

function healthLabel(health?: ContainerHealth | null) {
  return t(`container.list.health.${health || 'unavailable'}`);
}

function shouldShowHealthTag(health?: ContainerHealth | null) {
  return health === 'healthy' || health === 'unhealthy' || health === 'starting';
}

function healthTheme(health?: ContainerHealth | null) {
  if (health === 'healthy') return 'success';
  if (health === 'unhealthy') return 'danger';
  if (health === 'starting') return 'warning';
  return 'default';
}

function orchestratorLabel(type: ContainerOrchestratorType) {
  return t(`container.list.orchestrators.${type}`);
}

function orchestratorTheme(row: ContainerSummaryRecord) {
  const type = readContainerOrchestratorType(row);
  if (type === 'standalone') return 'success';
  if (type === 'compose') return 'warning';
  if (type === 'unknown') return 'danger';
  return 'default';
}

function sourceGroupFilter(row: ContainerSummaryRecord) {
  return createContainerSourceQuickFilter(row, 'group');
}

function sourceMemberFilter(row: ContainerSummaryRecord) {
  return createContainerSourceQuickFilter(row, 'member');
}

function orchestratorSummary(row: ContainerSummaryRecord) {
  const group = sourceGroupFilter(row);
  if (group) {
    return group.value;
  }

  const member = sourceMemberFilter(row);
  if (member) {
    return member.value;
  }

  return row.orchestrator?.display_name || t('container.list.sourceUnknownSummary');
}

function emitSourceFilter(row: ContainerSummaryRecord, target: ContainerSourceQuickFilterTarget) {
  const filter = createContainerSourceQuickFilter(row, target);
  if (filter) {
    emit('source-filter', filter);
  }
}

function handleSourceFilterClick(
  event: MouseEvent | undefined,
  row: ContainerSummaryRecord,
  target: ContainerSourceQuickFilterTarget,
) {
  event?.stopPropagation?.();
  emitSourceFilter(row, target);
}

function cpuMetric(row: ContainerSummaryRecord): ResourceMetric & { summaryValue: string } {
  const change = selectContainerStatsChangeState(row.id);
  if (row.resource?.cpu_percent === undefined) {
    return {
      available: false,
      changeClass: metricChangedClass(change, 'cpu'),
      percentage: 0,
      progressStatus: metricProgressStatus(change.cpu),
      summaryValue: t('container.list.stats.unavailable'),
      tooltip: resourceUnavailableSummary(row, 'cpu'),
      value: t('container.list.stats.unavailable'),
    };
  }

  const value = formatPercent(row.resource.cpu_percent);
  return {
    available: true,
    changeClass: metricChangedClass(change, 'cpu'),
    percentage: clampPercentage(row.resource.cpu_percent),
    progressStatus: metricProgressStatus(change.cpu),
    summaryValue: value,
    tooltip: t('container.list.stats.cpuTooltip', { percent: value }),
    value,
  };
}

function memoryMetric(row: ContainerSummaryRecord): ResourceMetric & { summaryValue: string } {
  const change = selectContainerStatsChangeState(row.id);
  if (row.resource?.memory_usage_bytes === undefined || row.resource?.memory_percent === undefined) {
    return {
      available: false,
      changeClass: metricChangedClass(change, 'memory'),
      percentage: 0,
      progressStatus: metricProgressStatus(change.memory),
      summaryValue: t('container.list.stats.unavailable'),
      tooltip: resourceUnavailableSummary(row, 'memory'),
      value: t('container.list.stats.unavailable'),
    };
  }

  const usage = formatBytes(row.resource.memory_usage_bytes);
  const limit = formatBytes(row.resource.memory_limit_bytes);
  const percent = formatPercent(row.resource.memory_percent);
  const value = usage || t('container.list.stats.unavailable');

  return {
    available: true,
    changeClass: metricChangedClass(change, 'memory'),
    percentage: clampPercentage(row.resource.memory_percent),
    progressStatus: metricProgressStatus(change.memory),
    summaryValue: percent,
    tooltip: t('container.list.stats.memoryTooltip', {
      limit: limit || t('container.list.stats.unavailable'),
      percent,
      usage: value,
    }),
    value,
  };
}

function resourceSummary(row: ContainerSummaryRecord) {
  return `${cpuMetric(row).value} / ${memoryMetric(row).summaryValue}`;
}

function labelSummary(row: ContainerSummaryRecord) {
  const count = Object.keys(row.labels ?? {}).length;
  return count ? t('container.list.labelCount', { count }) : '-';
}

function clampPercentage(value: number) {
  return Math.min(100, Math.max(0, Number.isFinite(value) ? value : 0));
}

function formatPercent(value?: number) {
  if (value === undefined || !Number.isFinite(value)) {
    return t('container.list.stats.unavailable');
  }

  return `${value.toFixed(2)}%`;
}

function formatBytes(value?: number) {
  if (value === undefined) {
    return '';
  }

  return `${(value / BYTES_PER_MIB).toFixed(2)} MiB`;
}

function resourceUnavailableSummary(row: ContainerSummaryRecord, metric: 'cpu' | 'memory') {
  const reason = localizeResourceUnavailableReason(
    (metric === 'memory' && row.resource?.memory_usage_bytes === undefined && row.resource?.stats_error_message) ||
      row.resource?.stats_error_message ||
      row.resource?.stats_error_key ||
      row.resource?.unavailable_reason,
  );
  return reason || t('container.list.stats.unavailable');
}

function localizeResourceUnavailableReason(reason?: string | null) {
  const normalizedReason = reason?.trim();
  if (!normalizedReason) {
    return '';
  }
  if (!normalizedReason.startsWith('ops.container.error.') && !normalizedReason.startsWith('container.stats:')) {
    return normalizedReason;
  }
  if (te(normalizedReason)) {
    return t(normalizedReason);
  }
  return t('container.list.stats.unavailableReasonFallback');
}
</script>
<style scoped lang="less">
.container-image,
.container-identity,
.container-source-cell {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-identity__name {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.container-identity__id,
.container-muted {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.container-identity__id {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-port-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6);
}

.container-runtime-status {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-runtime-status__text {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-runtime-status .t-tag {
  align-self: flex-start;
}

.container-source-cell {
  align-items: flex-start;
}

.container-source-cell__header {
  align-items: center;
  display: flex;
}

.container-source-cell__line {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-source-cell__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.container-source-cell .t-tag,
.container-source-cell :deep(.t-button) {
  align-self: flex-start;
}

.container-source-cell :deep(.t-button) {
  min-width: auto;
  padding: 0;
}

.container-resource-meter {
  align-items: center;
  border-radius: 999px;
  display: inline-flex;
  gap: var(--graft-density-gap-8);
  justify-content: center;
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-2) var(--graft-density-gap-8) var(--graft-density-gap-2) var(--graft-density-gap-2);
  position: relative;
  transform: translateZ(0);
}

.container-resource-meter[data-available='true'] {
  background: var(--td-bg-color-container-hover);
}

.container-resource-meter :deep(.t-progress) {
  flex: none;
}

.container-resource-meter__empty {
  background: var(--td-bg-color-component-disabled);
  border-radius: 50%;
  display: inline-flex;
  height: 36px;
  width: 36px;
}
</style>
