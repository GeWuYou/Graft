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
            <template #icon><search-icon /></template>{{ t('runtimeTarget.list.discoverLocalDocker') }}
          </t-button>
          <t-button theme="primary" variant="outline" :loading="loading" @click="load">
            <template #icon><refresh-icon /></template>{{ t('runtimeTarget.list.reload') }}
          </t-button>
        </template>
      </management-page-header>
      <t-alert v-if="errorMessage" theme="error" :message="errorMessage" class="runtime-target-feedback" />
      <management-table-card :description="t('runtimeTarget.list.summary', { count: total })">
        <template #toolbar>
          <t-button theme="default" variant="text" :loading="loading" @click="load">
            <template #icon><refresh-icon /></template>{{ t('runtimeTarget.list.reload') }}
          </t-button>
        </template>
        <t-table row-key="id" :data="items" :columns="tableColumns" :loading="loading">
          <template #empty>
            <t-empty
              :title="t('runtimeTarget.list.emptyTitle')"
              :description="t('runtimeTarget.list.emptyDescription')"
            >
              <template #action>
                <t-button theme="primary" :loading="discovering" @click="discoverLocal">
                  {{ t('runtimeTarget.list.discoverLocalDocker') }}
                </t-button>
              </template>
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
const pagination = reactive({ current: 1, pageSize: 10 });
const changes = ref<Record<number, MetricChanges>>({});
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
function changeFor(id: number, metric: keyof MetricChanges) {
  return changes.value[id]?.[metric] ?? 'none';
}
function metricCell(id: number, metric: keyof MetricChanges, value: RuntimeTargetUsageMetric) {
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
        () =>
          row.health.status === 'healthy' ? t('runtimeTarget.status.healthy') : t('runtimeTarget.status.unavailable'),
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
</style>
