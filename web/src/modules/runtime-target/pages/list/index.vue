<template>
  <div class="runtime-target-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header :title="t('runtimeTarget.list.title')" :description="t('runtimeTarget.list.description')">
        <template #actions>
          <t-button
            theme="default"
            variant="outline"
            :loading="discovering"
            data-testid="runtime-target-discover-local"
            @click="discoverLocal"
          >
            <template #icon><search-icon /></template>
            {{ t('runtimeTarget.list.discoverLocal') }}
          </t-button>
          <t-button theme="primary" variant="outline" :loading="loading" @click="load">
            <template #icon><refresh-icon /></template>
            {{ t('runtimeTarget.list.reload') }}
          </t-button>
        </template>
      </management-page-header>

      <management-table-card :description="t('runtimeTarget.list.summary', { count: total })">
        <template #toolbar>
          <t-button theme="default" variant="text" :loading="loading" @click="load">
            <template #icon><refresh-icon /></template>
            {{ t('runtimeTarget.list.reload') }}
          </t-button>
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
import { formatBytes } from '@/shared/observability';

import {
  discoverLocalDocker,
  listRuntimeTargetPage,
  refreshRuntimeTarget,
  type RuntimeTarget,
  type RuntimeTargetMetric,
} from '../../api/runtime-target';

const { t } = useI18n();
const loading = ref(false);
const discovering = ref(false);
const refreshingId = ref<number | null>(null);
const items = ref<RuntimeTarget[]>([]);
const total = ref(0);
const pagination = reactive({ current: 1, pageSize: 10 });

function metricText(metric: RuntimeTargetMetric) {
  if (!metric.available) return t('runtimeTarget.metrics.unavailable');
  const percent = `${metric.usagePercent.toFixed(1)}%`;
  return metric.totalBytes > 0
    ? `${percent} · ${formatBytes(metric.usedBytes)} / ${formatBytes(metric.totalBytes)}`
    : percent;
}

function metricCell(metric: RuntimeTargetMetric) {
  const percent = Math.max(0, Math.min(100, metric.usagePercent));
  const tooltip = metric.available
    ? metricText(metric)
    : metric.unavailableReason || t('runtimeTarget.metrics.unavailableHint');
  return h(resolveComponent('t-tooltip'), { content: tooltip }, () =>
    h('div', { class: 'runtime-target-meter', 'data-available': metric.available }, [
      metric.available
        ? h(resolveComponent('t-progress'), {
            theme: 'circle',
            label: false,
            percentage: percent,
            size: 36,
            strokeWidth: 4,
          })
        : h('span', { class: 'runtime-target-meter__empty' }),
      h('span', metric.available ? `${percent.toFixed(1)}%` : t('runtimeTarget.metrics.unavailable')),
    ]),
  );
}

const columns = computed<any[]>(() => [
  {
    colKey: 'displayName',
    title: t('runtimeTarget.columns.name'),
    cell: (_h: unknown, { row }: { row: RuntimeTarget }) =>
      h('div', [h('strong', row.displayName), h('small', row.endpointLabel)]),
  },
  {
    colKey: 'availability',
    title: t('runtimeTarget.columns.status'),
    cell: (_h: unknown, { row }: { row: RuntimeTarget }) =>
      h('t-tag', { theme: row.availability ? 'success' : 'danger', variant: 'light' }, () =>
        availabilityText(row.availability),
      ),
  },
  {
    colKey: 'containers',
    title: t('runtimeTarget.metrics.containers'),
    cell: (_h: unknown, { row }: { row: RuntimeTarget }) =>
      `${row.summary.containers.total} (${t('runtimeTarget.metrics.running')} ${row.summary.containers.running} · ${t('runtimeTarget.metrics.stopped')} ${row.summary.containers.stopped})`,
  },
  {
    colKey: 'images',
    title: t('runtimeTarget.metrics.images'),
    cell: (_h: unknown, { row }: { row: RuntimeTarget }) =>
      `${row.summary.images.total} (${t('runtimeTarget.metrics.used')} ${row.summary.images.used} · ${t('runtimeTarget.metrics.unused')} ${row.summary.images.unused})`,
  },
  {
    colKey: 'cpu',
    title: t('runtimeTarget.metrics.cpu'),
    cell: (_h: unknown, { row }: { row: RuntimeTarget }) => metricCell(row.summary.cpu),
  },
  {
    colKey: 'memory',
    title: t('runtimeTarget.metrics.memory'),
    cell: (_h: unknown, { row }: { row: RuntimeTarget }) => metricCell(row.summary.memory),
  },
  {
    colKey: 'disk',
    title: t('runtimeTarget.metrics.disk'),
    cell: (_h: unknown, { row }: { row: RuntimeTarget }) => metricCell(row.summary.disk),
  },
  {
    colKey: 'actions',
    title: t('runtimeTarget.list.refresh'),
    cell: (_h: unknown, { row }: { row: RuntimeTarget }) =>
      h(
        't-button',
        {
          theme: 'default',
          variant: 'text',
          loading: refreshingId.value === row.id,
          onClick: () => refreshTarget(row.id),
        },
        () => h(RefreshIcon),
      ),
  },
]);

function availabilityText(value: boolean) {
  return value ? t('runtimeTarget.status.available') : t('runtimeTarget.status.unavailable');
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
  } finally {
    loading.value = false;
  }
}

async function refreshTarget(id: number) {
  refreshingId.value = id;
  try {
    await refreshRuntimeTarget(id);
    await load();
  } finally {
    refreshingId.value = null;
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

const refreshIntervalMs = 5_000;
let refreshTimer: number | null = null;
function startRealtimeRefresh() {
  if (refreshTimer !== null) return;
  refreshTimer = window.setInterval(() => {
    if (document.visibilityState === 'visible' && !loading.value) void load();
  }, refreshIntervalMs);
}
function stopRealtimeRefresh() {
  if (refreshTimer !== null) window.clearInterval(refreshTimer);
  refreshTimer = null;
}
onMounted(() => {
  void load();
  startRealtimeRefresh();
});
onActivated(startRealtimeRefresh);
onDeactivated(stopRealtimeRefresh);
onUnmounted(stopRealtimeRefresh);
</script>
<style scoped lang="less">
.runtime-target-meter {
  align-items: center;
  background: var(--td-bg-color-container-hover);
  border-radius: 999px;
  display: inline-flex;
  gap: var(--graft-density-gap-8);
  padding: var(--graft-density-gap-2) var(--graft-density-gap-8) var(--graft-density-gap-2) var(--graft-density-gap-2);
  white-space: nowrap;
}

.runtime-target-meter > span:last-child {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.runtime-target-meter__empty {
  border: 1px dashed var(--td-component-stroke);
  border-radius: 50%;
  height: 36px;
  width: 36px;
}

.runtime-target-page__grid {
  margin-top: var(--graft-density-gap-4);
}

.runtime-target-card {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-18);
  min-height: 312px;
  padding: var(--graft-density-gap-20);
}

.runtime-target-card__header,
.runtime-target-card__title-row,
.runtime-target-metric__head {
  align-items: center;
  display: flex;
}

.runtime-target-card__header {
  align-items: flex-start;
  justify-content: space-between;
}

.runtime-target-card__identity {
  min-width: 0;
}

.runtime-target-card__title-row {
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.runtime-target-card h2,
.runtime-target-card p,
.runtime-target-card section,
.runtime-target-card small {
  margin: 0;
}

.runtime-target-card h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.runtime-target-card p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin-top: var(--graft-density-gap-6);
  overflow-wrap: anywhere;
}

.runtime-target-card__counts {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.runtime-target-card__counts section {
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-small);
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.runtime-target-card__counts span,
.runtime-target-metric__head span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.runtime-target-card__counts strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  line-height: 1.2;
  margin-top: var(--graft-density-gap-4);
}

.runtime-target-card__counts small,
.runtime-target-metric small {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  margin-top: var(--graft-density-gap-4);
}

.runtime-target-card__metrics {
  display: grid;
  gap: var(--graft-density-gap-12);
}

.runtime-target-metric__head {
  justify-content: space-between;
  margin-bottom: var(--graft-density-gap-6);
}

.runtime-target-metric__head strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
}

.runtime-target-metric__unavailable,
.runtime-target-metric__hint {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
}

.runtime-target-metric__hint {
  display: block;
  min-height: 18px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.runtime-target-page__empty {
  min-height: 280px;
}

@media (width <= 640px) {
  .runtime-target-card {
    min-height: 0;
    padding: var(--graft-density-gap-16);
  }
}
</style>
