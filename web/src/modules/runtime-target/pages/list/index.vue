<template>
  <div class="runtime-target-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        :source="{ labelKey: 'runtimeTarget.list.eyebrow', fallback: t('runtimeTarget.list.eyebrow') }"
        :title="t('runtimeTarget.list.title')"
        :description="t('runtimeTarget.list.description')"
      >
        <template #actions>
          <t-button
            theme="default"
            variant="outline"
            :loading="discovering"
            data-testid="runtime-target-discover-local"
            @click="discoverLocal"
          >
            <template #icon><search-icon /></template>{{ t('runtimeTarget.list.discoverLocal') }}
          </t-button>
          <t-button theme="primary" variant="outline" :loading="loading" @click="load">
            <template #icon><refresh-icon /></template>{{ t('runtimeTarget.list.reload') }}
          </t-button>
        </template>
      </management-page-header>
      <management-table-card :description="t('runtimeTarget.list.summary', { count: total })">
        <template #toolbar>
          <t-button theme="default" variant="text" :loading="loading" @click="load"
            ><template #icon><refresh-icon /></template>{{ t('runtimeTarget.list.reload') }}</t-button
          >
        </template>
        <t-table row-key="id" :data="items" :columns="columns" :loading="loading">
          <template #empty>
            <t-empty
              :title="t('runtimeTarget.list.emptyTitle')"
              :description="t('runtimeTarget.list.emptyDescription')"
            >
              <template #action
                ><t-button theme="primary" :loading="discovering" @click="discoverLocal">{{
                  t('runtimeTarget.list.discoverLocal')
                }}</t-button></template
              >
            </t-empty>
          </template>
        </t-table>
        <template #footer>
          <management-table-pagination :summary="t('runtimeTarget.list.summary', { count: total })">
            <t-pagination
              v-model:current="pagination.current"
              v-model:page-size="pagination.pageSize"
              :total="total"
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
import { RefreshIcon, SearchIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, h, onActivated, onDeactivated, onMounted, onUnmounted, reactive, ref, resolveComponent } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  ManagementPageContent,
  ManagementPageHeader,
  ManagementTableCard,
  ManagementTablePagination,
} from '@/shared/components/management';
import { RealtimeResourceMetricCell } from '@/shared/components/metrics';
import { formatBytes } from '@/shared/observability';
import { openRealtimeTopicSocket, type RealtimeTopicSocketController } from '@/shared/realtime';

import {
  discoverLocalDocker,
  listRuntimeTargetPage,
  type RuntimeTarget,
  type RuntimeTargetMetric,
} from '../../api/runtime-target';
import { parseRuntimeTargetSummaryPayload, RUNTIME_TARGET_REALTIME_TOPIC } from '../../contract/realtime';

type Change = 'up' | 'down' | 'none';
type MetricChanges = Record<'cpu' | 'memory' | 'disk', Change>;
const { t } = useI18n();
const loading = ref(false);
const discovering = ref(false);
const items = ref<RuntimeTarget[]>([]);
const total = ref(0);
const pagination = reactive({ current: 1, pageSize: 10 });
const changes = ref<Record<number, MetricChanges>>({});
const active = ref(false);
let realtimeController: RealtimeTopicSocketController | null = null;
const changeTimers = new Map<number, number>();

function metricText(metric: RuntimeTargetMetric) {
  if (!metric.available) return t('runtimeTarget.metrics.unavailable');
  const percent = `${metric.usagePercent.toFixed(1)}%`;
  return metric.totalBytes > 0
    ? `${percent} · ${formatBytes(metric.usedBytes)} / ${formatBytes(metric.totalBytes)}`
    : percent;
}
function changeFor(id: number, metric: keyof MetricChanges) {
  return changes.value[id]?.[metric] ?? 'none';
}
function metricCell(id: number, metric: keyof MetricChanges, value: RuntimeTargetMetric) {
  const percentage = Math.max(0, Math.min(100, value.usagePercent));
  return h(RealtimeResourceMetricCell, {
    available: value.available,
    change: changeFor(id, metric),
    percentage,
    tooltip: value.available
      ? metricText(value)
      : value.unavailableReason || t('runtimeTarget.metrics.unavailableHint'),
    value: value.available ? `${percentage.toFixed(1)}%` : t('runtimeTarget.metrics.unavailable'),
  });
}
function countCell(totalValue: number, details: Array<[string, number]>) {
  return h('div', { class: 'runtime-target-counts' }, [
    h('strong', totalValue),
    ...details.map(([label, value]) => h('span', [h('small', label), h('b', value)])),
  ]);
}
const columns = computed<any[]>(() => [
  {
    colKey: 'displayName',
    title: t('runtimeTarget.columns.name'),
    minWidth: 230,
    cell: (_: unknown, { row }: { row: RuntimeTarget }) =>
      h('div', { class: 'runtime-target-identity' }, [h('strong', row.displayName), h('small', row.endpointLabel)]),
  },
  {
    colKey: 'availability',
    title: t('runtimeTarget.columns.status'),
    width: 110,
    cell: (_: unknown, { row }: { row: RuntimeTarget }) =>
      h(resolveComponent('t-tag'), { theme: row.availability ? 'success' : 'danger', variant: 'light' }, () =>
        row.availability ? t('runtimeTarget.status.available') : t('runtimeTarget.status.unavailable'),
      ),
  },
  {
    colKey: 'containers',
    title: t('runtimeTarget.metrics.containers'),
    width: 150,
    cell: (_: unknown, { row }: { row: RuntimeTarget }) =>
      countCell(row.summary.containers.total, [
        [t('runtimeTarget.metrics.running'), row.summary.containers.running],
        [t('runtimeTarget.metrics.stopped'), row.summary.containers.stopped],
      ]),
  },
  {
    colKey: 'images',
    title: t('runtimeTarget.metrics.images'),
    width: 150,
    cell: (_: unknown, { row }: { row: RuntimeTarget }) =>
      countCell(row.summary.images.total, [
        [t('runtimeTarget.metrics.used'), row.summary.images.used],
        [t('runtimeTarget.metrics.unused'), row.summary.images.unused],
      ]),
  },
  {
    colKey: 'cpu',
    title: t('runtimeTarget.metrics.cpu'),
    width: 142,
    cell: (_: unknown, { row }: { row: RuntimeTarget }) => metricCell(row.id, 'cpu', row.summary.cpu),
  },
  {
    colKey: 'memory',
    title: t('runtimeTarget.metrics.memory'),
    width: 142,
    cell: (_: unknown, { row }: { row: RuntimeTarget }) => metricCell(row.id, 'memory', row.summary.memory),
  },
  {
    colKey: 'disk',
    title: t('runtimeTarget.metrics.disk'),
    width: 142,
    cell: (_: unknown, { row }: { row: RuntimeTarget }) => metricCell(row.id, 'disk', row.summary.disk),
  },
]);

function compare(previous: number, next: number): Change {
  return next > previous ? 'up' : next < previous ? 'down' : 'none';
}
function applyRealtime(itemsUpdate: RuntimeTarget[]) {
  const byID = new Map(itemsUpdate.map((item) => [item.id, item]));
  items.value = items.value.map((current) => {
    const next = byID.get(current.id);
    if (!next) return current;
    const nextChanges: MetricChanges = {
      cpu: compare(current.summary.cpu.usagePercent, next.summary.cpu.usagePercent),
      memory: compare(current.summary.memory.usagePercent, next.summary.memory.usagePercent),
      disk: compare(current.summary.disk.usagePercent, next.summary.disk.usagePercent),
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
}
function startRealtime() {
  if (!active.value || realtimeController || items.value.length === 0) return;
  realtimeController = openRealtimeTopicSocket({
    topic: RUNTIME_TARGET_REALTIME_TOPIC,
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
  try {
    const page = await listRuntimeTargetPage({
      limit: pagination.pageSize,
      offset: (pagination.current - 1) * pagination.pageSize,
    });
    items.value = page.items;
    total.value = page.total;
    startRealtime();
  } finally {
    loading.value = false;
  }
}
async function discoverLocal() {
  discovering.value = true;
  try {
    await discoverLocalDocker();
    pagination.current = 1;
    await load();
    MessagePlugin.success(t('runtimeTarget.list.discoverSuccess'));
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
:deep(.runtime-target-identity),
:deep(.runtime-target-counts) {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
}

:deep(.runtime-target-identity strong),
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
</style>
