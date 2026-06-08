<template>
  <monitor-status-page-frame
    v-bind="frameProps"
    @refresh="refreshSnapshot"
    @toggle-auto-refresh="toggleAutoRefresh"
    @update:refresh-interval-value="handleRefreshIntervalChange"
  >
    <template #headerHint>
      <div class="server-status-runtime-scope-line">
        {{ t('monitor.runtimePage.memoryBoundaryNotice') }}
      </div>
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
  </monitor-status-page-frame>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import KeyValueRow from '../../components/KeyValueRow.vue';
import MonitorStatusPageFrame from '../../components/MonitorStatusPageFrame.vue';
import SectionCard from '../../components/SectionCard.vue';
import { resolveServerStatusTone } from '../../components/server-status-ui';
import type { MonitorRefreshInterval } from '../../contract/refresh';
import { buildStandardMonitorStatusFrameProps } from '../../shared/frame-props';
import {
  displayText,
  formatBytes,
  formatTimestamp,
  formatUptime,
  useServerStatusSnapshot,
} from '../../shared/server-status-snapshot';

const { t } = useI18n();
/* jscpd:ignore-start */
// 这里保留页面本地 snapshot 解构，避免为压低重复率再抽一层“万能页面上下文”。
// 若未来删除或改造该代码，必须同步移除对应 jscpd ignore，重新评估是否仍需保留本地解构。
const snapshot = useServerStatusSnapshot();
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
} = snapshot;
/* jscpd:ignore-end */

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

/* jscpd:ignore-start */
// 这里保留页面级 frame 配置，页面标题、摘要和状态语义直接贴近页面实现更易维护。
// 若未来删除或改造该代码，必须同步移除对应 jscpd ignore，重新评估是否仍需保留页面本地配置。
const frameProps = computed(() =>
  buildStandardMonitorStatusFrameProps({
    t,
    page: {
      eyebrow: t('monitor.sectionTitle'),
      titleKey: 'monitor.runtimePage.title',
      title: t('monitor.runtimePage.title'),
      descriptionKey: 'monitor.runtimePage.subtitle',
      description: t('monitor.runtimePage.subtitle'),
      compactHeader: true,
      status: headerStatus.value,
      statusLabel: headerStatusLabel.value,
      summaryItems: summaryMetrics.value,
    },
    snapshot: {
      autoRefreshEnabled: autoRefreshEnabled.value,
      loading: loading.value,
      refreshIntervalOptions: refreshIntervalOptions.value,
      refreshIntervalValue: selectedRefreshInterval.value,
      errorMessage: errorMessage.value,
      initialized: initialized.value,
      hasServerStatus: Boolean(serverStatus.value),
    },
  }),
);
/* jscpd:ignore-end */

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
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(12, minmax(0, 1fr));
}

.server-status-runtime-grid__card {
  grid-column: span 4;
}

.server-status-runtime-scope-line {
  background: var(--server-status-card-background-subtle, var(--td-bg-color-container-hover));
  border: 1px solid var(--server-status-card-border, var(--td-component-stroke));
  border-radius: var(--td-radius-medium);
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  padding: var(--graft-density-gap-6) var(--graft-density-gap-10);
}

.server-status-runtime-memory-hero {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-bottom: var(--graft-density-gap-16);
}

.server-status-runtime-memory-hero__item {
  background: var(--server-status-card-background-subtle, var(--td-bg-color-container-hover));
  border: 1px solid var(--server-status-card-border, var(--td-component-stroke));
  border-radius: calc(var(--td-radius-large) - 2px);
  min-height: 92px;
  padding: var(--graft-density-gap-16);
}

.server-status-runtime-memory-hero__label {
  color: var(--td-text-color-secondary);
  display: block;
  font: var(--td-font-body-small);
  margin-bottom: var(--graft-density-gap-8);
}

.server-status-runtime-memory-hero__value {
  color: var(--td-text-color-primary);
  display: block;
  font: var(--td-font-title-large);
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
