<template>
  <div class="runtime-target-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        action-layout="inline"
        compact
        :title="t('runtimeTarget.list.title')"
        :description="t('runtimeTarget.list.description')"
        :source="{ labelKey: 'runtimeTarget.list.eyebrow', fallback: t('runtimeTarget.list.eyebrow') }"
      >
        <template v-if="total > 0" #actions>
          <t-tooltip :content="t('runtimeTarget.list.discoverLocalDocker')" placement="bottom">
            <t-button
              class="runtime-target-discover-button"
              shape="square"
              theme="default"
              variant="outline"
              :aria-label="t('runtimeTarget.list.discoverLocalDocker')"
              :loading="discovering"
              data-testid="runtime-target-discover-local"
              @click="discoverLocal"
            >
              <template #icon><search-icon /></template>
            </t-button>
          </t-tooltip>
        </template>
      </management-page-header>
      <management-statistics-bar
        layout="summary"
        :items="statistics"
        :label="t('runtimeTarget.list.summary', { count: total })"
      />
      <t-alert v-if="errorMessage" theme="error" :message="errorMessage" class="runtime-target-feedback" />
      <management-table-card>
        <template #toolbar>
          <t-button theme="default" variant="text" :loading="loading" @click="load">
            <template #icon><refresh-icon /></template>{{ t('runtimeTarget.list.reload') }}
          </t-button>
        </template>
        <responsive-table entity-card-layout="adaptive" presentation="entity">
          <template #cards>
            <t-empty
              v-if="!items.length"
              :title="t('runtimeTarget.list.emptyTitle')"
              :description="t('runtimeTarget.list.emptyDescription')"
            >
              <template #action>
                <t-button
                  theme="primary"
                  :loading="discovering"
                  data-testid="runtime-target-discover-local-empty"
                  @click="discoverLocal"
                >
                  {{ t('runtimeTarget.list.discoverLocalDocker') }}
                </t-button>
              </template>
            </t-empty>
            <article
              v-for="row in items"
              :key="row.id"
              class="runtime-target-card"
              :data-testid="`runtime-target-card-${row.id}`"
            >
              <header class="runtime-target-card__header">
                <div class="runtime-target-card__identity">
                  <router-link :to="runtimeTargetDetailPath(row.id)">{{ row.displayName }}</router-link>
                  <span>{{ row.connection.endpoint }}</span>
                </div>
                <t-tag :theme="healthTheme(row)" variant="light">
                  {{ healthLabel(row) }}
                </t-tag>
              </header>
              <div class="runtime-target-card__provider">
                <span>{{ t('runtimeTarget.columns.provider') }}</span>
                <strong>{{ row.runtime.provider }}</strong>
              </div>
              <dl class="runtime-target-card__metrics">
                <div class="runtime-target-card__metric">
                  <dt>{{ t('runtimeTarget.metrics.workloads') }}</dt>
                  <dd>
                    <strong>{{ workloadValue(row) }}</strong>
                    <span v-if="row.resources.workloads.available">
                      {{ t('runtimeTarget.metrics.active') }} {{ row.resources.workloads.active }}
                    </span>
                  </dd>
                </div>
                <div v-for="metric in resourceMetrics(row)" :key="metric.key" class="runtime-target-card__metric">
                  <dt>{{ metric.label }}</dt>
                  <dd>
                    <realtime-resource-metric-cell
                      :available="metric.value.available"
                      :change="changeFor(row.id, metric.key)"
                      :percentage="metricPercentage(metric.value)"
                      :tooltip="metricText(metric.value)"
                      :value="metricValue(metric.value)"
                    />
                  </dd>
                </div>
              </dl>
              <router-link class="runtime-target-card__detail" :to="runtimeTargetDetailPath(row.id)">
                {{ t('runtimeTarget.list.viewDetail') }}
              </router-link>
            </article>
          </template>
          <t-table row-key="id" :data="items" :columns="tableColumns" :loading="loading">
            <template #empty>
              <t-empty
                :title="t('runtimeTarget.list.emptyTitle')"
                :description="t('runtimeTarget.list.emptyDescription')"
              >
                <template #action>
                  <t-button
                    theme="primary"
                    :loading="discovering"
                    data-testid="runtime-target-discover-local-empty"
                    @click="discoverLocal"
                  >
                    {{ t('runtimeTarget.list.discoverLocalDocker') }}
                  </t-button>
                </template>
              </t-empty>
            </template>
          </t-table>
        </responsive-table>
        <template #footer>
          <management-table-pagination :summary="t('runtimeTarget.list.summary', { count: total })">
            <t-pagination
              v-model:current="pagination.current"
              v-model:page-size="pagination.pageSize"
              :total="total"
              :total-content="false"
              :page-size-options="[10, 20, 50, 100]"
              :show-page-number="true"
              @change="load"
            />
          </management-table-pagination>
        </template>
      </management-table-card>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
// 列表页负责发现/刷新运行时目标并维护列表请求状态，详情数据由详情路由独立加载。
import { RefreshIcon, SearchIcon } from 'tdesign-icons-vue-next';
import type { PrimaryTableCol } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, h, onActivated, onDeactivated, onMounted, onUnmounted, reactive, ref, resolveComponent } from 'vue';
import { useI18n } from 'vue-i18n';

import { RUNTIME_TARGET_REALTIME_TOPIC } from '@/contracts/generated/modules/runtime-target';
import {
  ManagementPageContent,
  ManagementPageHeader,
  ManagementStatisticsBar,
  ManagementTableCard,
  ManagementTablePagination,
} from '@/shared/components/management';
import { RealtimeResourceMetricCell } from '@/shared/components/metrics';
import ResponsiveTable from '@/shared/components/responsive/ResponsiveTable.vue';
import { formatBytes } from '@/shared/observability';
import { openRealtimeTopicSocket, type RealtimeTopicSocketController } from '@/shared/realtime';

import {
  discoverLocalDocker,
  listRuntimeTargetPage,
  type RuntimeTarget,
  type RuntimeTargetUsageMetric,
} from '../../api/runtime-target';
import { runtimeTargetDetailPath } from '../../contract/paths';
import { parseRuntimeTargetSummaryPayload } from '../../contract/realtime';

type Change = 'up' | 'down' | 'none';
type MetricChanges = Record<'cpu' | 'memory' | 'storage', Change>;
const { t } = useI18n();
const loading = ref(false);
const discovering = ref(false);
const errorMessage = ref('');
const items = ref<RuntimeTarget[]>([]);
const total = ref(0);
const summary = ref({ total: 0, healthy: 0, unavailable: 0 });
const pagination = reactive({ current: 1, pageSize: 10 });
const changes = ref<Record<number, MetricChanges>>({});
const statistics = computed(() => [
  { label: t('runtimeTarget.list.targetCount'), value: summary.value.total },
  {
    label: t('runtimeTarget.status.healthy'),
    marker: '🟢',
    value: summary.value.healthy,
  },
  {
    label: t('runtimeTarget.status.unavailable'),
    marker: '🔴',
    value: summary.value.unavailable,
  },
]);
const active = ref(false);
let realtimeController: RealtimeTopicSocketController | null = null;
const changeTimers = new Map<number, number>();

function metricText(metric: RuntimeTargetUsageMetric) {
  if (!metric.available) return t('runtimeTarget.metrics.unavailable');
  const percent = `${metric.usagePercent.toFixed(1)}%`;
  return metric.totalBytes > 0
    ? `${percent} · ${formatBytes(metric.usedBytes)} / ${formatBytes(metric.totalBytes)}`
    : percent;
}
function metricPercentage(metric: RuntimeTargetUsageMetric) {
  return Math.max(0, Math.min(100, metric.usagePercent));
}
function metricValue(metric: RuntimeTargetUsageMetric) {
  return metric.available ? `${metricPercentage(metric).toFixed(1)}%` : t('runtimeTarget.metrics.unavailable');
}
function changeFor(id: number, metric: keyof MetricChanges) {
  return changes.value[id]?.[metric] ?? 'none';
}
function metricCell(id: number, metric: keyof MetricChanges, value: RuntimeTargetUsageMetric) {
  const percentage = metricPercentage(value);
  return h(RealtimeResourceMetricCell, {
    available: value.available,
    change: changeFor(id, metric),
    percentage,
    tooltip: value.available
      ? metricText(value)
      : value.unavailableReason || t('runtimeTarget.metrics.unavailableHint'),
    value: metricValue(value),
  });
}
function healthLabel(row: RuntimeTarget) {
  return row.health.status === 'healthy' ? t('runtimeTarget.status.healthy') : t('runtimeTarget.status.unavailable');
}
function healthTheme(row: RuntimeTarget) {
  return row.health.status === 'healthy' ? 'success' : 'danger';
}
function workloadValue(row: RuntimeTarget) {
  return row.resources.workloads.available ? row.resources.workloads.total : t('runtimeTarget.metrics.unavailable');
}
function resourceMetrics(row: RuntimeTarget) {
  return [
    { key: 'cpu' as const, label: t('runtimeTarget.metrics.cpu'), value: row.resources.cpu },
    { key: 'memory' as const, label: t('runtimeTarget.metrics.memory'), value: row.resources.memory },
    { key: 'storage' as const, label: t('runtimeTarget.metrics.storage'), value: row.resources.storage },
  ];
}
function workloadCell(row: RuntimeTarget) {
  const workloads = row.resources.workloads;
  if (!workloads.available) return t('runtimeTarget.metrics.unavailable');
  return h('div', { class: 'runtime-target-counts' }, [
    h('strong', workloads.total),
    h('span', [h('small', t('runtimeTarget.metrics.active')), h('b', workloads.active)]),
  ]);
}
const columns = computed<PrimaryTableCol<RuntimeTarget>[]>(() => [
  {
    colKey: 'displayName',
    title: t('runtimeTarget.columns.name'),
    minWidth: 230,
    cell: (_h, { row }) =>
      h('div', { class: 'runtime-target-identity' }, [
        h(resolveComponent('router-link'), { to: runtimeTargetDetailPath(row.id) }, () => row.displayName),
        h('small', row.connection.endpoint),
      ]),
  },
  {
    colKey: 'provider',
    title: t('runtimeTarget.columns.provider'),
    width: 140,
    cell: (_h, { row }) => row.runtime.provider,
  },
  {
    colKey: 'health',
    title: t('runtimeTarget.columns.health'),
    width: 120,
    cell: (_h, { row }) =>
      h(
        resolveComponent('t-tag'),
        { theme: row.health.status === 'healthy' ? 'success' : 'danger', variant: 'light' },
        () => healthLabel(row),
      ),
  },
  {
    colKey: 'workloads',
    title: t('runtimeTarget.metrics.workloads'),
    width: 130,
    cell: (_h, { row }) => workloadCell(row),
  },
  {
    colKey: 'cpu',
    title: t('runtimeTarget.metrics.cpu'),
    width: 142,
    cell: (_h, { row }) => metricCell(row.id, 'cpu', row.resources.cpu),
  },
  {
    colKey: 'memory',
    title: t('runtimeTarget.metrics.memory'),
    width: 142,
    cell: (_h, { row }) => metricCell(row.id, 'memory', row.resources.memory),
  },
  {
    colKey: 'storage',
    title: t('runtimeTarget.metrics.storage'),
    width: 142,
    cell: (_h, { row }) => metricCell(row.id, 'storage', row.resources.storage),
  },
]);
const tableColumns = columns as unknown as PrimaryTableCol[];

function compare(previous: number, next: number): Change {
  return next > previous ? 'up' : next < previous ? 'down' : 'none';
}
function reconcileRealtimePage(nextItems: RuntimeTarget[]) {
  const nextByID = new Map(nextItems.map((item) => [item.id, item]));
  items.value = nextItems.map((next) => {
    const current = items.value.find((item) => item.id === next.id);
    if (!current) return next;
    const nextChanges: MetricChanges = {
      cpu: compare(current.resources.cpu.usagePercent, next.resources.cpu.usagePercent),
      memory: compare(current.resources.memory.usagePercent, next.resources.memory.usagePercent),
      storage: compare(current.resources.storage.usagePercent, next.resources.storage.usagePercent),
    };
    if (Object.values(nextChanges).some((value) => value !== 'none')) {
      changes.value = { ...changes.value, [current.id]: nextChanges };
      const oldTimer = changeTimers.get(current.id);
      if (oldTimer) window.clearTimeout(oldTimer);
      changeTimers.set(
        current.id,
        window.setTimeout(() => {
          const { [current.id]: _, ...rest } = changes.value;
          changes.value = rest;
        }, 800),
      );
    }
    return next;
  });
  changes.value = Object.fromEntries(
    Object.entries(changes.value).filter(([id]) => nextByID.has(Number(id))),
  ) as Record<number, MetricChanges>;
}
function applyRealtime(itemsUpdate: RuntimeTarget[]) {
  const offset = (pagination.current - 1) * pagination.pageSize;
  if (offset >= itemsUpdate.length) return;
  total.value = Math.max(total.value, itemsUpdate.length);
  reconcileRealtimePage(itemsUpdate.slice(offset, offset + pagination.pageSize));
}
function startRealtime() {
  if (!active.value || realtimeController) return;
  realtimeController = openRealtimeTopicSocket({
    topic: RUNTIME_TARGET_REALTIME_TOPIC.SUMMARY,
    parseMessage: parseRuntimeTargetSummaryPayload,
    onMessage: (payload) => applyRealtime(payload.items),
  });
}
function stopRealtime() {
  realtimeController?.close();
  realtimeController = null;
}
async function load() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const page = await listRuntimeTargetPage({
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize,
    });
    items.value = page.items;
    total.value = page.total;
    summary.value = page.summary;
    startRealtime();
  } catch {
    errorMessage.value = t('runtimeTarget.list.loadError');
  } finally {
    loading.value = false;
  }
}
async function discoverLocal() {
  discovering.value = true;
  errorMessage.value = '';
  try {
    await discoverLocalDocker();
    pagination.current = 1;
    await load();
    MessagePlugin.success(t('runtimeTarget.list.discoverSuccess'));
  } catch {
    errorMessage.value = t('runtimeTarget.list.discoverError');
  } finally {
    discovering.value = false;
  }
}
onMounted(() => {
  active.value = true;
  void load();
});
onActivated(() => {
  active.value = true;
  startRealtime();
});
onDeactivated(() => {
  active.value = false;
  stopRealtime();
});
onUnmounted(() => {
  stopRealtime();
  changeTimers.forEach((timer) => window.clearTimeout(timer));
});
</script>
<style scoped lang="less">
@import '@/shared/components/card-surface.less';

.runtime-target-feedback {
  margin-bottom: var(--td-comp-margin-l);
}

:deep(.runtime-target-identity),
:deep(.runtime-target-counts) {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
}

:deep(.runtime-target-identity a),
:deep(.runtime-target-counts strong) {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

:deep(.runtime-target-identity small),
:deep(.runtime-target-counts small) {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  overflow-wrap: anywhere;
}

:deep(.runtime-target-counts span) {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-6);
  justify-content: space-between;
}

:deep(.runtime-target-counts b) {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
}

.runtime-target-discover-button {
  min-height: var(--td-comp-size-xxxl);
  min-width: var(--td-comp-size-xxxl);
}

.runtime-target-card {
  .graft-entity-card-surface();

  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-14);
  min-width: 0;
  padding: var(--graft-density-gap-16);
}

.runtime-target-card__header {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
}

.runtime-target-card__identity {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.runtime-target-card__identity a {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  overflow-wrap: anywhere;
}

.runtime-target-card__identity span,
.runtime-target-card__provider span,
.runtime-target-card__metric dt,
.runtime-target-card__metric dd span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  overflow-wrap: anywhere;
}

.runtime-target-card__provider {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-8);
}

.runtime-target-card__provider strong,
.runtime-target-card__metric dd strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.runtime-target-card__metrics {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
}

.runtime-target-card__metric {
  background: var(--td-bg-color-container-hover);
  border-radius: var(--td-radius-default);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-width: 0;
  padding: var(--graft-density-gap-10);
}

.runtime-target-card__metric dt,
.runtime-target-card__metric dd {
  margin: 0;
}

.runtime-target-card__metric dd {
  align-items: flex-start;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.runtime-target-card__detail {
  align-items: center;
  color: var(--td-brand-color);
  display: inline-flex;
  font: var(--td-font-body-medium);
  justify-content: center;
  min-height: var(--td-comp-size-xxxl);
  padding: 0 var(--graft-density-gap-12);
  text-decoration: none;
}

.runtime-target-card__detail:focus-visible {
  border-radius: var(--td-radius-default);
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}
</style>
