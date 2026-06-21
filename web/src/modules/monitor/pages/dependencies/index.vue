<template>
  <monitor-status-page-frame
    v-bind="frameProps"
    @refresh="refreshSnapshot"
    @toggle-auto-refresh="toggleAutoRefresh"
    @update:refresh-interval-value="handleRefreshIntervalChange"
  >
    <div class="server-status-dependencies-layout">
      <section-card
        class="server-status-dependencies-layout__main"
        :title="t('monitor.dependenciesPage.serviceListTitle')"
        :description="t('monitor.dependenciesPage.noteDescription')"
        :min-height="380"
      >
        <div class="server-status-dependency-grid">
          <dependency-health-card
            v-for="service in serviceCards"
            :key="service.key"
            :service-key="service.key"
            :title="service.name"
            :description="service.description"
            :status="service.status"
            :status-label="service.statusLabel"
            :primary-metric="service.primaryMetric"
            :pool="service.pool"
            :diagnostics-title="service.diagnostics.title"
            @show-diagnostics="showDiagnostics(service)"
          />
        </div>
      </section-card>

      <section-card
        class="server-status-dependencies-layout__side"
        :title="t('monitor.dependenciesPage.futureEntryTitle')"
        :description="t('monitor.dependenciesPage.futureEntryHint')"
        :min-height="380"
      >
        <div class="server-status-module-entry">
          <status-tag
            :label="t('monitor.dependenciesPage.statusNotConfigured')"
            status="disabled"
            class="server-status-module-entry__tag"
          />
          <p class="server-status-module-entry__title">{{ t('monitor.dependenciesPage.futureEntrySubtitle') }}</p>
          <p class="server-status-module-entry__description">
            {{ t('monitor.dependenciesPage.futureEntryDescription') }}
          </p>
        </div>
      </section-card>
    </div>

    <dependency-diagnostic-drawer
      v-model:visible="diagnosticDrawerVisible"
      :title="diagnosticDrawerTitle"
      :diagnostics="selectedDependency?.diagnostics ?? null"
    />
  </monitor-status-page-frame>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import DependencyDiagnosticDrawer from '../../components/DependencyDiagnosticDrawer.vue';
import DependencyHealthCard, {
  type DependencyHealthDiagnostics,
  type DependencyHealthMetric,
  type DependencyHealthPool,
} from '../../components/DependencyHealthCard.vue';
import MonitorStatusPageFrame from '../../components/MonitorStatusPageFrame.vue';
import SectionCard from '../../components/SectionCard.vue';
import { type ServerStatusTone } from '../../components/server-status-ui';
import StatusTag from '../../components/StatusTag.vue';
import type { MonitorRefreshInterval } from '../../contract/refresh';
import { buildStandardMonitorStatusFrameProps } from '../../shared/frame-props';
import {
  formatDependencyPoolUsage,
  formatPoolCount,
  poolUsagePercent,
  poolUsageStatus,
} from '../../shared/pool-metrics';
import {
  displayText,
  formatLatency,
  formatPoolWait,
  formatTimestamp,
  normalizeDependencyStatus,
  useServerStatusSnapshot,
} from '../../shared/server-status-snapshot';
import { formatDateOnly, formatTimeOnly } from '../../shared/time-display';
import type { ServerStatusConnectionPool, ServerStatusDependency } from '../../types/server-status';

type DependencyCard = {
  key: string;
  name: string;
  description: string;
  status: ServerStatusTone;
  statusLabel: string;
  primaryMetric: DependencyHealthMetric;
  pool: DependencyHealthPool;
  diagnostics: DependencyHealthDiagnostics;
};

const { locale, t } = useI18n();
const diagnosticDrawerVisible = ref(false);
const selectedDependencyKey = ref<string | null>(null);
/* jscpd:ignore-start */
// 这里保留页面本地 snapshot 解构，避免为压低重复率再抽一层“万能页面上下文”。
// 若未来删除或改造该代码，必须同步移除对应 jscpd ignore，重新评估是否仍需保留本地解构。
const snapshot = useServerStatusSnapshot();
const {
  errorMessage,
  initialized,
  loading,
  observedAt,
  remainingRefreshSeconds,
  refreshControlStatus,
  refreshIntervalOptions,
  refreshSnapshot,
  selectedRefreshInterval,
  serverStatus,
  toggleAutoRefresh,
} = snapshot;
/* jscpd:ignore-end */

const headerStatus = computed(() => overallDependencyStatus.value);
const headerStatusLabel = computed(() => {
  switch (overallDependencyStatus.value) {
    case 'healthy':
      return t('monitor.dependenciesPage.statusHealthy');
    case 'error':
      return t('monitor.dependenciesPage.statusAbnormal');
    case 'disabled':
      return t('monitor.dependenciesPage.statusNotConfigured');
    default:
      return t('monitor.dependenciesPage.statusUnknown');
  }
});

const summaryMetrics = computed(() => {
  const summary = serverStatus.value?.summary;

  return [
    {
      key: 'healthy',
      label: t('monitor.dependenciesPage.summary.healthy'),
      value: summary?.healthy_dependencies !== undefined ? String(summary.healthy_dependencies) : '--',
      description: t('monitor.dependenciesPage.summary.healthyDescription'),
    },
    {
      key: 'abnormal',
      label: t('monitor.dependenciesPage.summary.abnormal'),
      value: summary?.degraded_dependencies !== undefined ? String(summary.degraded_dependencies) : '--',
      description: t('monitor.dependenciesPage.summary.abnormalDescription'),
    },
    {
      key: 'notConfigured',
      label: t('monitor.dependenciesPage.summary.notConfigured'),
      value: summary?.disabled_dependencies !== undefined ? String(summary.disabled_dependencies) : '--',
      description: t('monitor.dependenciesPage.summary.notConfiguredDescription'),
    },
    {
      key: 'lastCheck',
      label: t('monitor.dependenciesPage.summary.lastCheck'),
      value: formatTimeOnly(observedAt.value, locale),
      description:
        formatDateOnly(observedAt.value, locale) || t('monitor.dependenciesPage.summary.lastCheckDescription'),
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
      titleKey: 'monitor.dependenciesPage.title',
      title: t('monitor.dependenciesPage.title'),
      descriptionKey: 'monitor.dependenciesPage.subtitle',
      description: t('monitor.dependenciesPage.subtitle'),
      status: headerStatus.value,
      statusLabel: headerStatusLabel.value,
      summaryItems: summaryMetrics.value,
    },
    snapshot: {
      refreshControlStatus: refreshControlStatus.value,
      remainingRefreshSeconds: remainingRefreshSeconds.value,
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

const serviceCards = computed<DependencyCard[]>(() => {
  const response = serverStatus.value;
  const observedLabel = formatTimestamp(response?.observed_at, locale);
  const database = response?.dependencies.database;
  const redis = response?.dependencies.redis;

  return [
    buildServiceCard({
      key: 'postgresql',
      name: t('monitor.serverStatus.postgresqlLabel'),
      description: t('monitor.dependenciesPage.postgresqlSubtitle'),
      status: toServerStatusTone(normalizeDependencyStatus(database?.status)),
      latency: database?.latency_ms,
      pool: database?.pool,
      checkedAt: observedLabel,
      detail: database?.detail,
    }),
    buildServiceCard({
      key: 'redis',
      name: t('monitor.serverStatus.redisLabel'),
      description: t('monitor.dependenciesPage.redisSubtitle'),
      status: toServerStatusTone(normalizeDependencyStatus(redis?.status)),
      latency: redis?.latency_ms,
      pool: redis?.pool,
      checkedAt: observedLabel,
      detail: redis?.detail,
    }),
  ];
});

const diagnosticDrawerTitle = computed(() => {
  if (!selectedDependency.value) {
    return t('monitor.dependenciesPage.diagnostics.title');
  }

  return `${selectedDependency.value.name} ${selectedDependency.value.diagnostics.title}`;
});

const selectedDependency = computed(
  () => serviceCards.value.find((service) => service.key === selectedDependencyKey.value) ?? null,
);

const overallDependencyStatus = computed<ServerStatusTone>(() => {
  const statuses = serviceCards.value.map((service) => service.status);

  if (statuses.includes('error')) {
    return 'error';
  }

  if (statuses.includes('unknown')) {
    return 'unknown';
  }

  if (statuses.length > 0 && statuses.every((status) => status === 'disabled')) {
    return 'disabled';
  }

  if (statuses.length > 0 && statuses.every((status) => status === 'healthy' || status === 'disabled')) {
    return 'healthy';
  }

  return 'unknown';
});

function buildServiceCard(options: {
  key: string;
  name: string;
  description: string;
  status: ServerStatusTone;
  latency?: number | null;
  pool?: ServerStatusDependency['pool'] | null;
  checkedAt: string;
  detail?: string;
}): DependencyCard {
  return {
    key: options.key,
    name: options.name,
    description: options.description,
    status: options.status,
    statusLabel: dependencyStatusLabel(options.status),
    primaryMetric: {
      label: t('monitor.dependenciesPage.fields.latency'),
      value: formatLatency(options.latency),
      description: t('monitor.dependenciesPage.fieldDescriptions.latency'),
    },
    pool: buildPoolView(options.name, options.pool),
    diagnostics: buildDiagnosticsView(options.status, options.pool, options.checkedAt, options.detail),
  };
}

function buildPoolView(label: string, pool?: ServerStatusConnectionPool | null): DependencyHealthPool {
  const usagePercent = pool ? poolUsagePercent(pool) : null;
  const usageText = pool ? formatDependencyPoolUsage(pool, emptyMetricText()) : emptyMetricText();
  const usagePercentText = formatPoolPercent(usagePercent);

  return {
    title: t('monitor.dependenciesPage.pool.title'),
    stateTitle: t('monitor.dependenciesPage.pool.stateTitle'),
    usageText,
    usagePercent,
    usagePercentText,
    usageStatus: poolUsageStatus(usagePercent),
    usageLabel: t('monitor.dependenciesPage.pool.usageLabel', { label }),
    usageTooltip: t('monitor.dependenciesPage.pool.usageTooltip', {
      label,
      value: usageText,
      percent: usagePercentText,
    }),
    summary: poolUsageSummary(usagePercent),
    emptyText: emptyMetricText(),
    items: [
      {
        key: 'inUse',
        label: t('monitor.dependenciesPage.pool.inUse'),
        value: formatPoolCount(pool?.in_use_connections, emptyMetricText()),
      },
      {
        key: 'idle',
        label: t('monitor.dependenciesPage.pool.idle'),
        value: formatPoolCount(pool?.idle_connections, emptyMetricText()),
      },
      {
        key: 'open',
        label: t('monitor.dependenciesPage.pool.open'),
        value: formatPoolCount(pool?.open_connections, emptyMetricText()),
      },
      {
        key: 'capacity',
        label: t('monitor.dependenciesPage.pool.capacity'),
        value: formatPoolCount(pool?.capacity, emptyMetricText()),
      },
    ],
  };
}

function buildDiagnosticsView(
  status: ServerStatusTone,
  pool: ServerStatusConnectionPool | null | undefined,
  checkedAt: string,
  detail?: string,
): DependencyHealthDiagnostics {
  return {
    title: t('monitor.dependenciesPage.diagnostics.title'),
    items: [
      {
        key: 'poolWait',
        label: t('monitor.dependenciesPage.fields.poolWait'),
        value: formatPoolWait(pool),
      },
      {
        key: 'timeoutCount',
        label: t('monitor.dependenciesPage.fields.timeoutCount'),
        value: formatPoolCount(pool?.timeout_count, emptyMetricText()),
      },
      {
        key: 'staleCount',
        label: t('monitor.dependenciesPage.fields.staleCount'),
        value: formatPoolCount(pool?.stale_count, emptyMetricText()),
      },
      {
        key: 'checkedAt',
        label: t('monitor.dependenciesPage.fields.checkedAt'),
        value: checkedAt,
      },
      {
        key: 'errorInfo',
        label: t('monitor.dependenciesPage.fields.errorInfo'),
        value: status === 'error' || status === 'unknown' ? displayText(detail) : t('monitor.dependenciesPage.noError'),
      },
      {
        key: 'detail',
        label: t('monitor.dependenciesPage.fields.detail'),
        value: displayText(detail),
      },
    ],
  };
}

function poolUsageSummary(percent: number | null) {
  switch (poolUsageStatus(percent)) {
    case 'danger':
      return t('monitor.dependenciesPage.pool.riskCritical');
    case 'warning':
      return t('monitor.dependenciesPage.pool.riskWarning');
    case 'healthy':
      return t('monitor.dependenciesPage.pool.riskHealthy');
    default:
      return t('monitor.dependenciesPage.pool.riskUnknown');
  }
}

function formatPoolPercent(percent: number | null) {
  if (percent === null || Number.isNaN(percent)) {
    return emptyMetricText();
  }

  return `${percent.toFixed(0)}%`;
}

function emptyMetricText() {
  return t('monitor.serverStatus.metricUsageNoData');
}

function dependencyStatusLabel(status: ServerStatusTone) {
  switch (status) {
    case 'healthy':
      return t('monitor.dependenciesPage.statusHealthy');
    case 'error':
      return t('monitor.dependenciesPage.statusAbnormal');
    case 'disabled':
      return t('monitor.dependenciesPage.statusNotConfigured');
    default:
      return t('monitor.dependenciesPage.statusUnknown');
  }
}

function toServerStatusTone(status: ReturnType<typeof normalizeDependencyStatus>): ServerStatusTone {
  switch (status) {
    case 'healthy':
      return 'healthy';
    case 'abnormal':
      return 'error';
    case 'notConfigured':
      return 'disabled';
    default:
      return 'unknown';
  }
}

function handleRefreshIntervalChange(value: number | string) {
  selectedRefreshInterval.value = value as MonitorRefreshInterval;
}

function showDiagnostics(service: DependencyCard) {
  selectedDependencyKey.value = service.key;
  diagnosticDrawerVisible.value = true;
}
</script>
<style scoped lang="less">
.server-status-dependencies-layout {
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(12, minmax(0, 1fr));
}

.server-status-dependencies-layout__main {
  grid-column: span 8;
}

.server-status-dependencies-layout__side {
  grid-column: span 4;
}

.server-status-dependency-grid {
  align-items: stretch;
  display: grid;
  gap: var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.server-status-module-entry {
  align-items: flex-start;
  background: var(--server-status-card-background-subtle, var(--td-bg-color-container-hover));
  border: 1px dashed var(--server-status-card-border, var(--td-component-stroke));
  border-radius: calc(var(--td-radius-large) - 2px);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  min-height: 100%;
  padding: var(--graft-density-gap-16);
}

.server-status-module-entry__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0;
}

.server-status-module-entry__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

@media (width <= 991px) {
  .server-status-dependencies-layout__main,
  .server-status-dependencies-layout__side {
    grid-column: span 12;
  }
}

@media (width <= 767px) {
  .server-status-dependency-grid {
    grid-template-columns: 1fr;
  }
}
</style>
