<template>
  <server-status-page-shell
    :eyebrow="t('monitor.sectionTitle')"
    :title="t('monitor.runtimePage.title')"
    :description="t('monitor.runtimePage.subtitle')"
    compact-header
  >
    <template #toolbar>
      <monitor-toolbar
        :auto-refresh-enabled="autoRefreshEnabled"
        :loading="loading"
        :pause-auto-refresh-label="t('monitor.serverStatus.pauseRefresh')"
        :refresh-interval-label="t('monitor.serverStatus.refreshIntervalLabel')"
        :refresh-interval-options="refreshIntervalOptions"
        :refresh-interval-value="selectedRefreshInterval"
        :refresh-now-label="t('monitor.serverStatus.refreshNow')"
        :resume-auto-refresh-label="t('monitor.serverStatus.resumeRefresh')"
        :show-trend-range="false"
        :status="headerStatus"
        :status-label="headerStatusLabel"
        :trend-range-label-placeholder="t('monitor.serverStatus.trendWindowLabel')"
        @refresh="refreshSnapshot"
        @toggle-auto-refresh="toggleAutoRefresh"
        @update:refresh-interval-value="handleRefreshIntervalChange"
      />
    </template>

    <template #headerHint>
      <div class="server-status-runtime-scope-line">
        {{ t('monitor.runtimePage.memoryBoundaryNotice') }}
      </div>
    </template>

    <template #summary>
      <summary-metric-card
        v-for="metric in summaryMetrics"
        :key="metric.key"
        :title="metric.label"
        :value="metric.value"
        :description="metric.description"
      />
    </template>

    <template #feedback>
      <section-card v-if="errorMessage" :title="t('monitor.shared.errorTitle')" :description="errorMessage" />
    </template>

    <div class="server-status-runtime-grid">
      <section-card
        class="server-status-runtime-grid__card"
        :title="t('monitor.runtimePage.runtimeMemoryTitle')"
        :description="t('monitor.runtimePage.runtimeMemoryDescription')"
        :min-height="360"
      >
        <div class="server-status-runtime-memory-hero">
          <div v-for="field in runtimePrimaryFields" :key="field.key" class="server-status-runtime-memory-hero__item">
            <span class="server-status-runtime-memory-hero__label">{{ field.label }}</span>
            <strong class="server-status-runtime-memory-hero__value">{{ field.value }}</strong>
          </div>
        </div>

        <div class="server-status-kv-list">
          <key-value-row
            v-for="field in runtimeSecondaryFields"
            :key="field.key"
            :label="field.label"
            :value="field.value"
            :description="field.description"
          />
        </div>
      </section-card>

      <section-card
        class="server-status-runtime-grid__card"
        :title="t('monitor.runtimePage.processBuildTitle')"
        :description="t('monitor.runtimePage.processBuildDescription')"
        :min-height="360"
      >
        <div class="server-status-kv-list">
          <key-value-row
            v-for="field in processAndBuildFields"
            :key="field.key"
            :label="field.label"
            :value="field.value"
          />
        </div>
      </section-card>

      <section-card
        class="server-status-runtime-grid__card"
        :title="t('monitor.runtimePage.hostEnvironmentTitle')"
        :description="t('monitor.runtimePage.serverEnvironmentDescription')"
        :min-height="360"
      >
        <div class="server-status-kv-list">
          <key-value-row
            v-for="field in hostEnvironmentFields"
            :key="field.key"
            :label="field.label"
            :value="field.value"
          />
        </div>
      </section-card>
    </div>

    <t-empty v-if="initialized && !serverStatus && !loading" :description="t('monitor.shared.empty')" />
  </server-status-page-shell>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import KeyValueRow from '../../components/KeyValueRow.vue';
import MonitorToolbar from '../../components/MonitorToolbar.vue';
import SectionCard from '../../components/SectionCard.vue';
import { resolveServerStatusTone } from '../../components/server-status-ui';
import ServerStatusPageShell from '../../components/ServerStatusPageShell.vue';
import SummaryMetricCard from '../../components/SummaryMetricCard.vue';
import type { MonitorRefreshInterval } from '../../contract/refresh';
import {
  displayText,
  formatBytes,
  formatTimestamp,
  formatUptime,
  useServerStatusSnapshot,
} from '../../shared/server-status-snapshot';

const { t } = useI18n();
const {
  autoRefreshEnabled,
  errorMessage,
  initialized,
  loading,
  observedAt,
  refreshIntervalOptions,
  refreshSnapshot,
  selectedRefreshInterval,
  serverStatus,
  toggleAutoRefresh,
} = useServerStatusSnapshot();

const headerStatus = computed(() => resolveServerStatusTone(serverStatus.value?.status));
const headerStatusLabel = computed(() =>
  serverStatus.value ? t('monitor.runtimePage.snapshotReady') : t('monitor.runtimePage.snapshotPending'),
);
const notReportedLabel = computed(() => t('monitor.shared.notReported'));

const summaryMetrics = computed(() => {
  const response = serverStatus.value;

  return [
    {
      key: 'uptime',
      label: t('monitor.runtimePage.summary.uptime'),
      value: formatSnapshotUptime(response?.server.uptime_seconds),
      description: t('monitor.runtimePage.summary.uptimeDescription'),
    },
    {
      key: 'goroutines',
      label: t('monitor.runtimePage.summary.goroutines'),
      value: formatCount(response?.runtime.goroutines),
      description: t('monitor.runtimePage.summary.goroutinesDescription'),
    },
    {
      key: 'runtimeAlloc',
      label: t('monitor.runtimePage.summary.runtimeAlloc'),
      value: formatSnapshotBytes(response?.runtime.runtime_alloc_bytes),
      description: t('monitor.runtimePage.summary.runtimeAllocDescription'),
    },
    {
      key: 'gcCycles',
      label: t('monitor.runtimePage.summary.gcCycles'),
      value: formatCount(response?.runtime.runtime_gc_cycles),
      description: t('monitor.runtimePage.summary.gcCyclesDescription'),
    },
  ];
});

const runtimePrimaryFields = computed(() => {
  const runtime = serverStatus.value?.runtime;

  return [
    {
      key: 'alloc',
      label: t('monitor.runtimePage.fields.runtimeAlloc'),
      value: formatSnapshotBytes(runtime?.runtime_alloc_bytes),
    },
    {
      key: 'heap',
      label: t('monitor.runtimePage.fields.runtimeHeap'),
      value: formatSnapshotBytes(runtime?.runtime_heap_in_use_bytes),
    },
  ];
});

const runtimeSecondaryFields = computed(() => {
  const runtime = serverStatus.value?.runtime;

  return [
    {
      key: 'sys',
      label: t('monitor.runtimePage.fields.runtimeSys'),
      value: formatSnapshotBytes(runtime?.runtime_sys_bytes),
      description: t('monitor.runtimePage.fieldDescriptions.runtimeSys'),
    },
    {
      key: 'gcCycles',
      label: t('monitor.runtimePage.fields.gcCycles'),
      value: formatCount(runtime?.runtime_gc_cycles),
      description: t('monitor.runtimePage.fieldDescriptions.gcCycles'),
    },
    {
      key: 'lastGc',
      label: t('monitor.runtimePage.fields.lastGc'),
      value: notReportedLabel.value,
      description: t('monitor.runtimePage.fieldDescriptions.lastGc'),
    },
    {
      key: 'observedAt',
      label: t('monitor.runtimePage.fields.observedAt'),
      value: formatSnapshotTimestamp(observedAt.value),
      description: t('monitor.runtimePage.fieldDescriptions.observedAt'),
    },
    {
      key: 'loadAverage',
      label: t('monitor.runtimePage.fields.loadAverage'),
      value: formatLoadAverage(serverStatus.value?.runtime.load_average),
      description: t('monitor.runtimePage.fieldDescriptions.loadAverage'),
    },
  ];
});

const processAndBuildFields = computed(() => {
  const server = serverStatus.value?.server;
  const runtime = serverStatus.value?.runtime;

  return [
    {
      key: 'appName',
      label: t('monitor.runtimePage.fields.appName'),
      value: displaySnapshotText(server?.app_name),
    },
    {
      key: 'appEnv',
      label: t('monitor.runtimePage.fields.appEnv'),
      value: displaySnapshotText(server?.app_env),
    },
    {
      key: 'buildVersion',
      label: t('monitor.runtimePage.fields.buildVersion'),
      value: displaySnapshotText(server?.version),
    },
    {
      key: 'gitCommit',
      label: t('monitor.runtimePage.fields.gitCommit'),
      value: notReportedLabel.value,
    },
    {
      key: 'startedAt',
      label: t('monitor.runtimePage.fields.startedAt'),
      value: formatSnapshotTimestamp(server?.started_at),
    },
    {
      key: 'goVersion',
      label: t('monitor.runtimePage.fields.goVersion'),
      value: displaySnapshotText(runtime?.go_version ?? server?.go_version),
    },
  ];
});

const hostEnvironmentFields = computed(() => {
  const runtime = serverStatus.value?.runtime;

  return [
    {
      key: 'hostName',
      label: t('monitor.runtimePage.fields.hostName'),
      value: displaySnapshotText(runtime?.host_name),
    },
    {
      key: 'platform',
      label: t('monitor.runtimePage.fields.platform'),
      value:
        runtime?.operating_system && runtime?.architecture
          ? `${runtime.operating_system} / ${runtime.architecture}`
          : notReportedLabel.value,
    },
    {
      key: 'cpuCores',
      label: t('monitor.runtimePage.fields.cpuCores'),
      value: formatCount(runtime?.cpu_cores),
    },
    {
      key: 'hostMemory',
      label: t('monitor.runtimePage.fields.hostMemory'),
      value: formatHostMemoryValue(runtime),
    },
    {
      key: 'hostMemoryUsage',
      label: t('monitor.runtimePage.fields.hostMemoryUsage'),
      value: formatHostMemoryPercent(runtime),
    },
  ];
});

function displaySnapshotText(value?: string | null) {
  const formatted = displayText(value);
  return formatted === '--' ? notReportedLabel.value : formatted;
}

function formatSnapshotBytes(value?: number | null) {
  const formatted = formatBytes(value);
  return formatted === '--' ? notReportedLabel.value : formatted;
}

function formatSnapshotTimestamp(value?: string | null) {
  const formatted = formatTimestamp(value);
  return formatted === '--' ? notReportedLabel.value : formatted;
}

function formatSnapshotUptime(value?: number | null) {
  const formatted = formatUptime(value);
  return formatted === '--' ? notReportedLabel.value : formatted;
}

function formatCount(value?: number | null) {
  return Number.isFinite(value) ? String(value) : notReportedLabel.value;
}

function handleRefreshIntervalChange(value: number | string) {
  selectedRefreshInterval.value = value as MonitorRefreshInterval;
}

function formatLoadAverage(
  value?: {
    one_minute?: number;
    five_minutes?: number;
    fifteen_minutes?: number;
  } | null,
) {
  const oneMinute = value?.one_minute;
  const fiveMinutes = value?.five_minutes;
  const fifteenMinutes = value?.fifteen_minutes;

  if (!Number.isFinite(oneMinute) || !Number.isFinite(fiveMinutes) || !Number.isFinite(fifteenMinutes)) {
    return notReportedLabel.value;
  }

  return `${Number(oneMinute).toFixed(2)} / ${Number(fiveMinutes).toFixed(2)} / ${Number(fifteenMinutes).toFixed(2)}`;
}

function formatHostMemoryValue(
  value?: {
    host_memory_total_bytes?: number;
    host_memory_used_bytes?: number;
  } | null,
) {
  if (!value) {
    return notReportedLabel.value;
  }

  const used = formatSnapshotBytes(value.host_memory_used_bytes);
  const total = formatSnapshotBytes(value.host_memory_total_bytes);

  if (used === notReportedLabel.value || total === notReportedLabel.value) {
    return notReportedLabel.value;
  }

  return `${used} / ${total}`;
}

function formatHostMemoryPercent(
  value?: {
    host_memory_used_percent?: number;
  } | null,
) {
  if (!value) {
    return notReportedLabel.value;
  }

  return typeof value.host_memory_used_percent === 'number' && Number.isFinite(value.host_memory_used_percent)
    ? `${value.host_memory_used_percent.toFixed(0)}%`
    : notReportedLabel.value;
}
</script>
<style scoped lang="less">
.server-status-runtime-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(12, minmax(0, 1fr));
}

.server-status-runtime-grid__card {
  grid-column: span 4;
}

.server-status-runtime-scope-line {
  background: color-mix(in srgb, var(--td-bg-color-container-hover) 72%, transparent);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
  padding: 6px 10px;
}

.server-status-runtime-memory-hero {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-bottom: 16px;
}

.server-status-runtime-memory-hero__item {
  background: var(--td-bg-color-container-select);
  border: 1px solid var(--td-component-stroke);
  border-radius: calc(var(--td-radius-large) - 2px);
  min-height: 92px;
  padding: 16px;
}

.server-status-runtime-memory-hero__label {
  color: var(--td-text-color-secondary);
  display: block;
  font-size: 13px;
  line-height: 20px;
  margin-bottom: 8px;
}

.server-status-runtime-memory-hero__value {
  color: var(--td-text-color-primary);
  display: block;
  font-size: 22px;
  font-variant-numeric: tabular-nums;
  font-weight: 700;
  line-height: 28px;
}

.server-status-kv-list {
  display: flex;
  flex-direction: column;
}

@media (width <= 991px) {
  .server-status-runtime-grid__card {
    grid-column: span 12;
  }
}

@media (width <= 575px) {
  .server-status-runtime-memory-hero {
    grid-template-columns: 1fr;
  }
}
</style>
