<template>
  <section class="diagnostics-page" data-page-type="form-detail">
    <page-header
      :source="{
        labelKey: 'network.outbound.connectivity.eyebrow',
        fallback: t('network.outbound.connectivity.eyebrow'),
      }"
      title-key="network.outbound.connectivity.diagnosticsTitle"
      :title-fallback="targetTitle"
      description-key="network.outbound.connectivity.diagnosticsDescription"
      :description-fallback="t('network.outbound.connectivity.diagnosticsDescription')"
    >
      <template #actions>
        <t-space>
          <t-button v-if="selectedCheckId && supports('export')" variant="outline" @click="exportReport">{{
            t('network.outbound.connectivity.export')
          }}</t-button>
          <t-button theme="primary" :loading="store.running" @click="run">{{
            t('network.outbound.connectivity.run')
          }}</t-button>
        </t-space>
      </template>
    </page-header>
    <t-alert v-if="error" theme="error" :message="error" close @close="error = ''" />
    <t-loading :loading="loading">
      <t-card :title="t('network.outbound.connectivity.latestResult')">
        <template v-if="report">
          <div class="diagnostics-page__summary">
            <t-tag :theme="statusTheme(report.status)" variant="light">{{ statusLabel(report.status) }}</t-tag>
            <span>{{ t('network.outbound.connectivity.totalLatency') }}: {{ report.total_latency_ms }} ms</span>
            <span>{{ t('network.outbound.connectivity.lastChecked') }}: {{ formatTime(report.checked_at) }}</span>
          </div>
          <t-table row-key="kind" :data="trace" :columns="probeColumns">
            <template #status="{ row }"
              ><t-tag :theme="probeTheme(row.status)" variant="light">{{
                probeStatusLabel(row.status)
              }}</t-tag></template
            >
            <template #duration="{ row }">{{ row.duration_ms }} ms</template>
          </t-table>
        </template>
        <t-empty v-else :description="t('network.outbound.connectivity.noReport')">
          <template #action
            ><t-button theme="primary" @click="run">{{ t('network.outbound.connectivity.run') }}</t-button></template
          >
        </t-empty>
      </t-card>

      <div class="diagnostics-page__details">
        <t-card v-if="supports('proxy_route')" :title="t('network.outbound.connectivity.route')">
          <t-descriptions v-if="report?.route" :column="1">
            <t-descriptions-item :label="t('network.outbound.connectivity.strategy')">{{
              report.route.matched_strategy
            }}</t-descriptions-item>
            <t-descriptions-item :label="t('network.outbound.connectivity.decision')">{{
              report.route.decision
            }}</t-descriptions-item>
            <t-descriptions-item :label="t('network.outbound.connectivity.reason')">{{
              report.route.reason
            }}</t-descriptions-item>
          </t-descriptions>
          <t-empty v-else :description="t('network.outbound.connectivity.routeUnavailable')" />
        </t-card>
        <t-card v-if="supports('exit_ip')" :title="t('network.outbound.connectivity.exitIp')">
          <t-descriptions v-if="report?.exit_ip?.available" :column="1">
            <t-descriptions-item :label="t('network.outbound.connectivity.exitIpMasked')">{{
              report.exit_ip.masked
            }}</t-descriptions-item>
          </t-descriptions>
          <t-empty v-else :description="t('network.outbound.connectivity.exitIpUnavailable')" />
        </t-card>
        <t-card v-if="supports('history')" :title="t('network.outbound.connectivity.history')">
          <t-table row-key="check_id" :data="history" :columns="historyColumns" @row-click="handleSelectReport">
            <template #checked="{ row }">
              <t-link
                theme="primary"
                hover="color"
                tabindex="0"
                @click.stop="selectReport({ row })"
                @keydown.enter.stop.prevent="selectReport({ row })"
                >{{ formatTime(row.checked_at) }}</t-link
              >
            </template>
            <template #status="{ row }"
              ><t-tag :theme="statusTheme(row.status)" variant="light">{{ statusLabel(row.status) }}</t-tag></template
            >
            <template #latency="{ row }">{{ row.latency_ms }} ms</template>
          </t-table>
          <t-empty v-if="!history.length" :description="t('network.outbound.connectivity.noHistory')" />
        </t-card>
      </div>
    </t-loading>
  </section>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import { PageHeader } from '@/shared/components/page';
import { formatLocaleDateTime } from '@/shared/observability';

import type {
  ConnectivityCheck,
  ConnectivityProbe,
  ConnectivityReport,
  ConnectivityTarget,
} from '../../api/connectivity';
import { useConnectivityStore } from '../../store/connectivity';

/** 单目标诊断页以 target 为稳定页面身份，报告 ID 只在页面内切换历史、Trace 与导出数据。 */
const { t, te, locale } = useI18n();
const route = useRoute();
const store = useConnectivityStore();
const loading = ref(false);
const error = ref('');
const report = ref<ConnectivityReport>();
const trace = ref<ConnectivityProbe[]>([]);
const history = ref<ConnectivityCheck[]>([]);
const selectedCheckId = ref<number>();
const targetId = computed(() => String(route.params.targetId ?? ''));
const isDiagnosticsRoute = computed(() => route.name === 'PlatformNetworkConnectivityDiagnosticsIndex');
const target = computed(() => store.targets.find((item) => item.id === targetId.value));
const targetTitle = computed(() => {
  const custom = store.customTargets.find((item) => item.id === targetId.value);
  if (custom) return custom.display_name;
  return target.value && te(target.value.title_key) ? t(target.value.title_key) : targetId.value;
});
const probeColumns = computed(() => [
  { colKey: 'kind', title: t('network.outbound.connectivity.probe') },
  { colKey: 'status', title: t('network.outbound.connectivity.status') },
  { colKey: 'duration', title: t('network.outbound.connectivity.latency') },
  { colKey: 'summary', title: t('network.outbound.connectivity.result') },
]);
const historyColumns = computed(() => [
  { colKey: 'checked', title: t('network.outbound.connectivity.lastChecked') },
  { colKey: 'status', title: t('network.outbound.connectivity.status') },
  { colKey: 'latency', title: t('network.outbound.connectivity.latency') },
]);

function supports(feature: ConnectivityTarget['features'][number]) {
  return target.value?.features.includes(feature) ?? false;
}
function formatTime(value?: string | null) {
  return formatLocaleDateTime(value, locale.value);
}
function statusTheme(status: ConnectivityReport['status']) {
  return status === 'healthy' ? 'success' : status === 'degraded' ? 'warning' : 'danger';
}
function statusLabel(status: ConnectivityReport['status']) {
  return t(`network.outbound.connectivity.statuses.${status}`);
}
function probeTheme(status: ConnectivityProbe['status']) {
  return status === 'succeeded' ? 'success' : status === 'skipped' ? 'default' : 'danger';
}
function probeStatusLabel(status: ConnectivityProbe['status']) {
  return t(`network.outbound.connectivity.probeStatuses.${status}`);
}

async function loadReport(checkId: number) {
  const [nextReport, nextTrace] = await Promise.all([
    store.loadReport(targetId.value, checkId),
    store.loadTrace(targetId.value, checkId),
  ]);
  report.value = nextReport;
  trace.value = nextTrace.probes;
  selectedCheckId.value = checkId;
}

async function load() {
  loading.value = true;
  error.value = '';
  try {
    await store.refresh();
    history.value = await store.loadHistory(targetId.value);
    if (history.value[0]) await loadReport(history.value[0].check_id);
    else {
      report.value = undefined;
      trace.value = [];
      selectedCheckId.value = undefined;
    }
  } catch (value) {
    error.value = String(value);
  } finally {
    loading.value = false;
  }
}

async function run() {
  error.value = '';
  try {
    const result = await store.runTarget(targetId.value);
    report.value = result.report;
    trace.value = result.report.probes;
    selectedCheckId.value = result.check.check_id;
    history.value = await store.loadHistory(targetId.value);
  } catch (value) {
    error.value = String(value);
  }
}

async function selectReport({ row }: { row: ConnectivityCheck }) {
  try {
    await loadReport(row.check_id);
  } catch (value) {
    error.value = String(value);
  }
}

function handleSelectReport({ row }: { row: unknown }) {
  void selectReport({ row: row as ConnectivityCheck });
}

async function exportReport() {
  if (!selectedCheckId.value) return;
  try {
    const exported = await store.exportReport(targetId.value, selectedCheckId.value);
    const blob = new Blob([JSON.stringify(exported, null, 2)], { type: 'application/json' });
    const link = document.createElement('a');
    link.href = URL.createObjectURL(blob);
    link.download = `connectivity-${targetId.value}-${selectedCheckId.value}.json`;
    link.click();
    URL.revokeObjectURL(link.href);
  } catch (value) {
    error.value = String(value);
  }
}

watch(targetId, () => {
  if (isDiagnosticsRoute.value) void load();
});
onMounted(() => void load());
</script>
<style scoped>
.diagnostics-page__summary {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  flex-wrap: wrap;
  gap: calc(12px * var(--graft-theme-density-scale)) calc(24px * var(--graft-theme-density-scale));
  margin-bottom: calc(16px * var(--graft-theme-density-scale));
}

.diagnostics-page__details {
  display: grid;
  gap: calc(16px * var(--graft-theme-density-scale));
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-top: calc(16px * var(--graft-theme-density-scale));
}

.diagnostics-page__details > :last-child:nth-child(odd) {
  grid-column: 1 / -1;
}

@media (width <= 800px) {
  .diagnostics-page__details {
    grid-template-columns: 1fr;
  }

  .diagnostics-page__details > :last-child:nth-child(odd) {
    grid-column: auto;
  }
}
</style>
