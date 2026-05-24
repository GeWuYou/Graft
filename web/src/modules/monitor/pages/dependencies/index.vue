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
          <dependency-status-card
            v-for="service in serviceCards"
            :key="service.key"
            :title="service.title"
            :description="service.subtitle"
            :status="service.status"
            :status-label="service.statusLabel"
            :items="service.fields"
          />
        </div>
      </section-card>

      <section-card
        class="server-status-dependencies-layout__side"
        :title="t('monitor.dependenciesPage.futureEntryTitle')"
        :description="t('monitor.dependenciesPage.futureEntryHint')"
        :min-height="380"
      >
        <div class="server-status-plugin-entry">
          <status-tag
            :label="t('monitor.dependenciesPage.statusNotConfigured')"
            status="disabled"
            class="server-status-plugin-entry__tag"
          />
          <p class="server-status-plugin-entry__title">{{ t('monitor.dependenciesPage.futureEntrySubtitle') }}</p>
          <p class="server-status-plugin-entry__description">
            {{ t('monitor.dependenciesPage.futureEntryDescription') }}
          </p>
        </div>
      </section-card>
    </div>
  </monitor-status-page-frame>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import DependencyStatusCard from '../../components/DependencyStatusCard.vue';
import MonitorStatusPageFrame from '../../components/MonitorStatusPageFrame.vue';
import SectionCard from '../../components/SectionCard.vue';
import { type ServerStatusTone } from '../../components/server-status-ui';
import StatusTag from '../../components/StatusTag.vue';
import type { MonitorRefreshInterval } from '../../contract/refresh';
import { buildStandardMonitorStatusFrameProps } from '../../shared/frame-props';
import {
  displayText,
  formatLatency,
  formatTimestamp,
  normalizeDependencyStatus,
  useServerStatusSnapshot,
} from '../../shared/server-status-snapshot';
import { formatDateOnly, formatTimeOnly } from '../../shared/time-display';

type DependencyCard = {
  key: string;
  title: string;
  subtitle: string;
  status: ServerStatusTone;
  statusLabel: string;
  fields: Array<{
    key: string;
    label: string;
    value: string;
    description: string;
  }>;
};

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
      value: formatTimeOnly(observedAt.value),
      description: formatDateOnly(observedAt.value) || t('monitor.dependenciesPage.summary.lastCheckDescription'),
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
      title: t('monitor.dependenciesPage.title'),
      description: t('monitor.dependenciesPage.subtitle'),
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

const serviceCards = computed<DependencyCard[]>(() => {
  const response = serverStatus.value;
  const observedLabel = formatTimestamp(response?.observed_at);
  const database = response?.dependencies.database;
  const redis = response?.dependencies.redis;

  return [
    buildServiceCard({
      key: 'postgresql',
      title: t('monitor.serverStatus.postgresqlLabel'),
      subtitle: t('monitor.dependenciesPage.postgresqlSubtitle'),
      status: toServerStatusTone(normalizeDependencyStatus(database?.status)),
      latency: database?.latency_ms,
      checkedAt: observedLabel,
      detail: database?.detail,
    }),
    buildServiceCard({
      key: 'redis',
      title: t('monitor.serverStatus.redisLabel'),
      subtitle: t('monitor.dependenciesPage.redisSubtitle'),
      status: toServerStatusTone(normalizeDependencyStatus(redis?.status)),
      latency: redis?.latency_ms,
      checkedAt: observedLabel,
      detail: redis?.detail,
    }),
  ];
});

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
  title: string;
  subtitle: string;
  status: ServerStatusTone;
  latency?: number | null;
  checkedAt: string;
  detail?: string;
}): DependencyCard {
  return {
    key: options.key,
    title: options.title,
    subtitle: options.subtitle,
    status: options.status,
    statusLabel: dependencyStatusLabel(options.status),
    fields: [
      {
        key: 'latency',
        label: t('monitor.dependenciesPage.fields.latency'),
        value: formatLatency(options.latency),
        description: t('monitor.dependenciesPage.fieldDescriptions.latency'),
      },
      {
        key: 'checkedAt',
        label: t('monitor.dependenciesPage.fields.checkedAt'),
        value: options.checkedAt,
        description: t('monitor.dependenciesPage.fieldDescriptions.checkedAt'),
      },
      {
        key: 'errorInfo',
        label: t('monitor.dependenciesPage.fields.errorInfo'),
        value:
          options.status === 'error' || options.status === 'unknown'
            ? displayText(options.detail)
            : t('monitor.dependenciesPage.noError'),
        description: t('monitor.dependenciesPage.fieldDescriptions.errorInfo'),
      },
      {
        key: 'detail',
        label: t('monitor.dependenciesPage.fields.detail'),
        value: displayText(options.detail),
        description: t('monitor.dependenciesPage.fieldDescriptions.detail'),
      },
    ],
  };
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
</script>
<style scoped lang="less">
.server-status-dependencies-layout {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(12, minmax(0, 1fr));
}

.server-status-dependencies-layout__main {
  grid-column: span 8;
}

.server-status-dependencies-layout__side {
  grid-column: span 4;
}

.server-status-dependency-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.server-status-plugin-entry {
  align-items: flex-start;
  background: var(--server-status-card-background-subtle, var(--td-bg-color-container-hover));
  border: 1px dashed var(--server-status-card-border, var(--td-component-stroke));
  border-radius: calc(var(--td-radius-large) - 2px);
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 100%;
  padding: 16px;
}

.server-status-plugin-entry__title {
  color: var(--td-text-color-primary);
  font-size: 15px;
  font-weight: 600;
  line-height: 24px;
  margin: 0;
}

.server-status-plugin-entry__description {
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 22px;
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
