<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="container-detail-page" data-page-type="operations-detail">
    <management-page-header
      :breadcrumb="detailBreadcrumb"
      :title="pageTitle"
      :description="detail ? detail.image : t('container.detail.description')"
      :source="{ labelKey: 'container.list.eyebrow', fallback: t('container.list.eyebrow') }"
    >
      <template #meta>
        <t-space class="container-detail-header-meta" break-line size="small">
          <span v-if="detail" class="container-detail-header-id">{{ shortContainerId(detail) }}</span>
          <t-tag v-if="detail" :theme="stateTheme(detail.state)" variant="light-outline">
            {{ stateLabel(detail.state) }}
          </t-tag>
          <t-tag v-if="detail" :theme="healthTheme(detail.health)" variant="light-outline">
            {{ healthLabel(detail.health) }}
          </t-tag>
          <t-tag v-if="detail?.runtime" theme="default" variant="light-outline">
            {{ detail.runtime }}
          </t-tag>
          <t-tag v-if="detail?.inspect_updated_at" theme="default" variant="light-outline">
            {{ t('container.detail.inspectUpdatedAt') }}: {{ formatTime(detail.inspect_updated_at) }}
          </t-tag>
        </t-space>
      </template>
      <template #actions>
        <t-space break-line size="small">
          <t-button theme="default" variant="outline" @click="goBack">
            {{ t('container.detail.back') }}
          </t-button>
          <t-button theme="primary" :loading="loading" @click="loadDetail">
            {{ t('container.detail.refresh') }}
          </t-button>
        </t-space>
      </template>
    </management-page-header>

    <t-alert v-if="error" theme="error" :title="error">
      <template #operation>
        <t-button theme="danger" variant="text" @click="loadDetail">
          {{ t('container.list.retry') }}
        </t-button>
      </template>
    </t-alert>

    <t-loading :loading="loading">
      <template v-if="detail">
        <section class="container-detail-summary">
          <t-card
            class="container-detail-summary-card container-detail-summary-card--identity"
            size="small"
            :bordered="false"
            :title="t('container.detail.summary.identity')"
          >
            <div class="container-detail-summary-list">
              <div class="container-detail-kv">
                <span>{{ t('container.list.fields.name') }}</span>
                <strong>{{ displayName(detail) }}</strong>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.list.fields.image') }}</span>
                <copyable-detail-value
                  :copy-label="t('container.detail.copy')"
                  :value="detail.image"
                  :display-value="detail.image"
                  @copy="copyDetailText"
                />
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.list.fields.id') }}</span>
                <copyable-detail-value
                  :value="detail.id"
                  :display-value="shortContainerId(detail)"
                  :copy-label="t('container.detail.copy')"
                  code
                  data-testid="summary-container-id-copy"
                  @copy="copyDetailText"
                />
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.list.fields.runtime') }}</span>
                <strong>{{ runtimeLabel(detail) }}</strong>
              </div>
            </div>
          </t-card>
          <t-card
            class="container-detail-summary-card container-detail-summary-card--runtime"
            size="small"
            :bordered="false"
            :title="t('container.detail.summary.runtime')"
          >
            <div class="container-detail-summary-list">
              <div class="container-detail-kv container-detail-kv--inline">
                <span>{{ t('container.list.fields.status') }}</span>
                <t-tag :theme="stateTheme(detail.state)" variant="light-outline">
                  {{ stateLabel(detail.state) }}
                </t-tag>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.list.fields.state') }}</span>
                <code>{{ detail.state || '-' }}</code>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.list.fields.startedAt') }}</span>
                <strong>{{ formatTime(detail.started_at) }}</strong>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.detail.health.status') }}</span>
                <strong>{{ healthLabel(detail.health) }}</strong>
              </div>
            </div>
          </t-card>
          <t-card
            class="container-detail-summary-card container-detail-summary-card--resources"
            size="small"
            :bordered="false"
            :title="t('container.detail.summary.resources')"
          >
            <div class="container-detail-summary__resource">
              <div class="container-detail-resource-meter container-detail-resource-meter--cpu">
                <div class="container-detail-resource-meter__content">
                  <span>{{ t('container.detail.resources.cpu') }}</span>
                  <strong>{{ formatPercent(detail.resource?.cpu_percent) }}</strong>
                </div>
                <t-progress
                  theme="circle"
                  size="small"
                  :label="formatPercent(detail.resource?.cpu_percent)"
                  :percentage="toProgressPercent(detail.resource?.cpu_percent)"
                />
              </div>
              <div class="container-detail-resource-meter container-detail-resource-meter--memory">
                <div class="container-detail-resource-meter__content">
                  <span>{{ t('container.detail.resources.memory') }}</span>
                  <strong>{{ memorySummary(detail) }}</strong>
                  <em>{{ formatPercent(detail.resource?.memory_percent) }}</em>
                </div>
                <t-progress
                  theme="line"
                  size="small"
                  :label="false"
                  :percentage="toProgressPercent(detail.resource?.memory_percent)"
                />
              </div>
            </div>
          </t-card>
          <t-card
            class="container-detail-summary-card container-detail-summary-card--network"
            size="small"
            :bordered="false"
            :title="t('container.detail.summary.network')"
          >
            <div class="container-detail-summary-list">
              <div class="container-detail-kv">
                <span>{{ t('container.detail.network.primaryIp') }}</span>
                <strong>{{ detail.primary_ip || '-' }}</strong>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.detail.network.summary') }}</span>
                <strong>{{ networkSummary(detail) }}</strong>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.detail.network.name') }}</span>
                <strong>{{ primaryNetworkName(detail) }}</strong>
              </div>
              <div class="container-detail-kv">
                <span>{{ t('container.detail.network.ports') }}</span>
                <div v-if="detail.ports.length" class="container-detail-port-list">
                  <t-tag
                    v-for="port in detail.ports"
                    :key="portLabel(port)"
                    class="container-detail-port-chip"
                    theme="default"
                    variant="light-outline"
                  >
                    {{ portLabel(port) }}
                  </t-tag>
                </div>
                <strong v-else>{{ t('container.detail.network.noPublicPorts') }}</strong>
              </div>
            </div>
          </t-card>
        </section>

        <t-card class="container-detail-tabs-card" :bordered="true">
          <t-tabs v-model:value="activeTab" theme="card" @change="handleTabChange">
            <t-tab-panel value="overview" :label="t('container.detail.tabs.overview')" :destroy-on-hide="false">
              <section class="container-detail-section container-detail-section--overview">
                <container-overview-panel
                  :copy-label="t('container.detail.copy')"
                  :sections="overviewSections"
                  @copy="copyDetailText"
                />
              </section>
            </t-tab-panel>

            <t-tab-panel value="resources" :label="t('container.detail.tabs.resources')" :destroy-on-hide="false">
              <section class="container-detail-section">
                <div class="container-detail-resource-grid">
                  <metric-card
                    :title="t('container.detail.resources.cpu')"
                    :value="resourceMetrics.cpu.value"
                    :description="resourceMetrics.cpu.description"
                    :progress="resourceMetrics.cpu.progress"
                    :progress-label="resourceMetrics.cpu.progressLabel"
                  />
                  <metric-card
                    :title="t('container.detail.resources.memory')"
                    :value="resourceMetrics.memory.value"
                    :description="resourceMetrics.memory.description"
                    :progress="resourceMetrics.memory.progress"
                    :progress-label="resourceMetrics.memory.progressLabel"
                  />
                  <article class="container-detail-resource-status-card">
                    <div class="container-detail-resource-status-card__content">
                      <span class="container-detail-resource-status-card__title">
                        {{ t('container.detail.resources.status') }}
                      </span>
                      <strong>{{ resourceMetrics.status.value }}</strong>
                      <span>{{ resourceMetrics.status.description }}</span>
                    </div>
                    <t-tag :theme="resourceMetrics.status.theme" variant="light-outline">
                      {{ resourceMetrics.status.value }}
                    </t-tag>
                  </article>
                </div>

                <section class="container-resource-dashboard-section">
                  <div class="container-resource-dashboard-section__title">
                    {{ t('container.detail.resources.dashboard') }}
                  </div>
                  <div class="container-resource-dashboard-grid">
                    <article class="container-resource-dashboard-panel">
                      <div class="container-resource-dashboard-panel__heading">
                        <span>{{ t('container.detail.resources.cpuUsageRate') }}</span>
                        <strong>{{ resourceMetrics.cpu.value }}</strong>
                      </div>
                      <t-progress theme="line" size="small" :label="false" :percentage="resourceMetrics.cpu.progress" />
                      <div class="container-resource-dashboard-panel__meta">
                        <span>
                          {{ t('container.detail.resources.cpuLimit') }}
                          <strong>{{ notCollectedLabel() }}</strong>
                        </span>
                        <span>
                          {{ t('container.detail.resources.onlineCpus') }}
                          <strong>{{ readCpuCountText() }}</strong>
                        </span>
                        <span>
                          {{ t('container.detail.resources.systemCpuUsage') }}
                          <strong>{{ readCpuSystemTimeText() }}</strong>
                        </span>
                      </div>
                    </article>

                    <article class="container-resource-dashboard-panel">
                      <div class="container-resource-dashboard-panel__heading">
                        <span>{{ t('container.detail.resources.memoryUsageRate') }}</span>
                        <strong>{{ resourceMetrics.memory.description }}</strong>
                      </div>
                      <t-progress
                        theme="line"
                        size="small"
                        :label="false"
                        :percentage="resourceMetrics.memory.progress"
                      />
                      <div class="container-resource-dashboard-panel__usage">
                        {{ resourceMetrics.memory.value }}
                      </div>
                      <div class="container-resource-dashboard-panel__meta">
                        <span>
                          {{ t('container.detail.resources.memoryCache') }}
                          <strong>{{ readMetricText('memory_cache', 'bytes') }}</strong>
                        </span>
                        <span>
                          {{ t('container.detail.resources.memoryRss') }}
                          <strong>{{ readMetricText('memory_rss', 'bytes') }}</strong>
                        </span>
                        <span>
                          {{ t('container.detail.resources.memoryActiveFile') }}
                          <strong>{{ readMetricText('memory_active_file', 'bytes') }}</strong>
                        </span>
                        <span>
                          {{ t('container.detail.resources.memoryInactiveFile') }}
                          <strong>{{ readMetricText('memory_inactive_file', 'bytes') }}</strong>
                        </span>
                      </div>
                    </article>
                  </div>
                </section>

                <section class="container-resource-detail-section">
                  <div class="container-resource-detail-section__title">
                    {{ t('container.detail.resources.detailedMetrics') }}
                  </div>
                  <div class="container-resource-detail-grid">
                    <article
                      v-for="group in resourceDetailGroups"
                      :key="group.key"
                      class="container-resource-detail-card"
                      :class="{
                        'container-resource-detail-card--cpu': group.key === 'cpu',
                        'container-resource-detail-card--memory': group.key === 'memory',
                      }"
                    >
                      <h3>{{ group.title }}</h3>
                      <div v-if="group.key === 'cpu'" class="container-resource-cpu-metric-grid">
                        <div
                          v-for="metric in cpuDetailMetrics"
                          :key="metric.key"
                          class="container-resource-cpu-metric"
                          :class="{
                            'container-resource-cpu-metric--muted': metric.muted,
                            'container-resource-cpu-metric--warning': metric.emphasized,
                          }"
                        >
                          <span class="container-resource-cpu-metric__label">{{ metric.label }}</span>
                          <strong class="container-resource-cpu-metric__value">{{ metric.value }}</strong>
                          <span v-if="metric.hint" class="container-resource-cpu-metric__hint">
                            {{ metric.hint }}
                          </span>
                        </div>
                      </div>
                      <div
                        v-else
                        class="container-resource-detail-card__body"
                        :class="{ 'container-resource-detail-card__body--memory': group.key === 'memory' }"
                      >
                        <div
                          v-for="row in group.rows"
                          :key="row.key"
                          class="container-resource-detail-row"
                          :class="{ 'container-resource-detail-row--placeholder': row.type === 'placeholder' }"
                        >
                          <span class="container-resource-detail-row__label">{{ row.label }}</span>
                          <span class="container-resource-detail-row__value">
                            <t-tag v-if="row.type === 'tag'" :theme="row.theme" variant="light-outline">
                              {{ row.value }}
                            </t-tag>
                            <span v-else>{{ row.value }}</span>
                          </span>
                        </div>
                      </div>
                    </article>
                  </div>
                </section>
              </section>
            </t-tab-panel>

            <t-tab-panel value="logs" :label="t('container.detail.tabs.logs')" :destroy-on-hide="false">
              <section class="container-detail-section">
                <log-viewer
                  v-model:line-limit="logLineLimit"
                  :lines="logs?.lines ?? []"
                  :loading="logsLoading"
                  :error="logsError"
                  :truncated="logs?.truncated"
                  :refresh-label="t('container.detail.logs.refresh')"
                  :copy-label="t('container.detail.copy')"
                  :download-label="t('container.detail.logs.download')"
                  :retry-label="t('container.list.retry')"
                  :search-placeholder="t('container.detail.logs.searchPlaceholder')"
                  :wrap-label="t('container.detail.logs.wrap')"
                  :refresh-scroll-label="t('container.detail.logs.refreshScroll')"
                  :refresh-scroll-tooltip-label="t('container.detail.logs.refreshScrollTooltip')"
                  :level-filter-label="t('container.detail.logs.levelFilter')"
                  :all-levels-label="t('container.detail.logs.allLevels')"
                  :match-count-label="t('container.detail.logs.matchCount')"
                  :empty-label="t('container.detail.logs.empty')"
                  :truncated-label="t('container.detail.logs.truncated')"
                  :detail-title-label="t('container.detail.logs.detailTitle')"
                  :important-fields-label="t('container.detail.logs.importantFields')"
                  :basic-info-label="t('container.detail.logs.basicInfo')"
                  :time-label="t('container.detail.logs.time')"
                  :level-label="t('container.detail.logs.level')"
                  :source-label="t('container.detail.logs.source')"
                  :view-detail-label="t('container.detail.logs.viewDetail')"
                  :collapse-detail-label="t('container.detail.logs.collapseDetail')"
                  :metadata-label="t('container.detail.logs.metadata')"
                  :message-label="t('container.detail.logs.message')"
                  :raw-label="t('container.detail.logs.raw')"
                  :copy-message-label="t('container.detail.logs.copyMessage')"
                  :copy-line-label="t('container.detail.logs.copyLine')"
                  :copy-json-label="t('container.detail.logs.copyJson')"
                  :copy-success-label="t('container.detail.copySuccess')"
                  :copy-error-label="t('container.detail.copyError')"
                  @refresh="loadLogs"
                />
              </section>
            </t-tab-panel>

            <t-tab-panel value="health" :label="t('container.detail.tabs.health')" :destroy-on-hide="false">
              <section class="container-detail-section">
                <t-descriptions :column="2" item-layout="vertical" bordered table-layout="fixed">
                  <t-descriptions-item :label="t('container.detail.health.status')">
                    {{ healthLabel(detail.health) }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.detail.health.restartCount')">
                    {{ detail.restart_count ?? '-' }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.list.fields.restartPolicy')">
                    {{ detail.restart_policy || '-' }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.list.detail.inspectUpdatedAt')">
                    {{ formatTime(detail.inspect_updated_at) }}
                  </t-descriptions-item>
                </t-descriptions>
              </section>
            </t-tab-panel>

            <t-tab-panel value="config" :label="t('container.detail.tabs.config')" :destroy-on-hide="false">
              <section class="container-detail-section">
                <t-descriptions :column="2" item-layout="vertical" bordered table-layout="fixed">
                  <t-descriptions-item :label="t('container.list.detail.command')">
                    {{ joinList(detail.command) }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.list.detail.entrypoint')">
                    {{ joinList(detail.entrypoint) }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.list.detail.workingDir')">
                    {{ detail.working_dir || '-' }}
                  </t-descriptions-item>
                </t-descriptions>
                <div class="container-detail-subsection">
                  <h3>{{ t('container.detail.config.environment') }}</h3>
                  <t-table
                    v-if="environmentRows.length"
                    row-key="name"
                    size="small"
                    :columns="environmentColumns"
                    :data="environmentRows"
                    :pagination="undefined"
                    table-layout="fixed"
                    cell-empty-content="-"
                  >
                    <template #value="{ row }">
                      <span>{{ row.value || '-' }}</span>
                    </template>
                    <template #policy="{ row }">
                      <t-tag :theme="policyTheme(row.policy)" variant="light-outline">
                        {{ policyLabel(row.policy) }}
                      </t-tag>
                    </template>
                    <template #operation="{ row }">
                      <t-button
                        v-if="row.copyable"
                        data-testid="env-copy"
                        size="small"
                        theme="default"
                        variant="text"
                        @click="copyEnvironmentValue(row)"
                      >
                        {{ t('container.detail.copy') }}
                      </t-button>
                    </template>
                  </t-table>
                  <t-empty v-else size="small" :description="t('container.detail.config.environmentUnavailable')" />
                </div>
              </section>
            </t-tab-panel>

            <t-tab-panel value="network" :label="t('container.detail.tabs.network')" :destroy-on-hide="false">
              <section class="container-detail-section">
                <t-table
                  v-if="detail.networks.length"
                  row-key="name"
                  size="small"
                  :columns="networkColumns"
                  :data="detail.networks"
                  :pagination="undefined"
                  table-layout="fixed"
                  cell-empty-content="-"
                />
                <t-empty v-else size="small" :description="t('container.list.detail.networkEmpty')" />
              </section>
            </t-tab-panel>

            <t-tab-panel value="storage" :label="t('container.detail.tabs.storage')" :destroy-on-hide="false">
              <section class="container-detail-section">
                <t-table
                  v-if="detail.mounts.length"
                  row-key="destination"
                  size="small"
                  :columns="mountColumns"
                  :data="detail.mounts"
                  :pagination="undefined"
                  table-layout="fixed"
                  cell-empty-content="-"
                >
                  <template #read_only="{ row }">
                    {{ row.read_only ? 'ro' : 'rw' }}
                  </template>
                </t-table>
                <t-empty v-else size="small" :description="t('container.list.detail.mountEmpty')" />
              </section>
            </t-tab-panel>

            <t-tab-panel value="raw" :label="t('container.detail.tabs.raw')" :destroy-on-hide="false">
              <section class="container-detail-section">
                <json-viewer
                  :value="detail"
                  :title="t('container.detail.raw.title')"
                  :description="t('container.detail.raw.description')"
                  :root-label="t('container.detail.raw.root')"
                  :source-label="t('container.detail.raw.source')"
                  :tree-label="t('container.detail.raw.tree')"
                  :copy-label="t('container.detail.copy')"
                  :copy-success-label="t('container.detail.copySuccess')"
                  :copy-error-label="t('container.detail.copyError')"
                  :empty-label="t('container.detail.raw.empty')"
                  :error-label="t('container.detail.raw.error')"
                />
              </section>
            </t-tab-panel>
          </t-tabs>
        </t-card>
      </template>

      <t-empty v-else-if="!error" size="small" :description="t('container.detail.empty')" />
    </t-loading>
  </div>
</template>
<script setup lang="ts">
import type { TableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { ManagementPageHeader } from '@/shared/components/management';
import { MetricCard } from '@/shared/components/metrics';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import {
  copyText as copyTextToClipboard,
  formatBytes,
  formatLocaleDateTime,
  formatNanosecondsAsDuration,
  formatPercent,
  JsonViewer,
  LogViewer,
  toProgressPercent,
} from '@/shared/observability';
import { createLogger } from '@/utils/logger';

import { getContainer, getContainerLogs } from '../../api/container';
import { CONTAINER_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
import type { ContainerDetail, ContainerHealth, ContainerLogResponse, ContainerState } from '../../types/container';
import ContainerOverviewPanel from './components/ContainerOverviewPanel.vue';
import CopyableDetailValue from './components/CopyableDetailValue.vue';
import type { ContainerOverviewInfoSection } from './components/overview';
import { buildCpuDetailMetrics, formatCpuCountText } from './components/resource-cpu-presenter';

defineOptions({
  name: 'ContainerDetailIndex',
});

type DetailTab = 'overview' | 'resources' | 'logs' | 'health' | 'config' | 'network' | 'storage' | 'raw';
type EnvironmentPolicy = 'plain' | 'masked' | 'hidden' | 'unknown';
type EnvironmentRow = {
  copyable: boolean;
  name: string;
  policy: EnvironmentPolicy;
  rawValue: string;
  value: string;
};
type ResourceStatusTheme = 'success' | 'warning' | 'default';
type ResourceMetricFormat = 'bytes' | 'number' | 'percent' | 'text';
type ContainerResourceSummary = NonNullable<ContainerDetail['resource']>;
type ResourceMetricKey = Extract<
  keyof ContainerResourceSummary,
  | 'cpu_percent'
  | 'cpu_usage_in_kernelmode'
  | 'cpu_usage_in_usermode'
  | 'memory_active_file'
  | 'memory_cache'
  | 'memory_inactive_file'
  | 'memory_limit_bytes'
  | 'memory_percent'
  | 'memory_pgfault'
  | 'memory_pgmajfault'
  | 'memory_rss'
  | 'memory_usage_bytes'
  | 'online_cpus'
  | 'pids_current'
  | 'pids_limit'
  | 'rx_bytes'
  | 'rx_dropped'
  | 'rx_errors'
  | 'rx_packets'
  | 'system_cpu_usage'
  | 'throttling_periods'
  | 'throttling_throttled_periods'
  | 'throttling_throttled_time'
  | 'total_cpu_usage'
  | 'tx_bytes'
  | 'tx_dropped'
  | 'tx_errors'
  | 'tx_packets'
>;
type ResourceDetailRow =
  | {
      key: string;
      label: string;
      type: 'text';
      value: string;
    }
  | {
      key: string;
      label: string;
      type: 'placeholder';
      value: string;
    }
  | {
      key: string;
      label: string;
      theme: ResourceStatusTheme;
      type: 'tag';
      value: string;
    };
type ResourceDetailGroup = {
  key: string;
  rows: ResourceDetailRow[];
  title: string;
};
type ResourceMetricDefinition = [ResourceMetricKey, string, ResourceMetricFormat];

const DETAIL_TABS: DetailTab[] = ['overview', 'resources', 'logs', 'health', 'config', 'network', 'storage', 'raw'];
const DEFAULT_LOG_QUERY = {
  tail: 200,
  since: undefined,
  timestamps: false,
  stdout: true,
  stderr: true,
};

const { locale, t } = useI18n();
const route = useRoute();
const router = useRouter();
const logger = createLogger('container.detail');

const detail = ref<ContainerDetail | null>(null);
const loading = ref(false);
const error = ref('');
const logs = ref<ContainerLogResponse | null>(null);
const logsLoading = ref(false);
const logsError = ref('');
const logLineLimit = ref(DEFAULT_LOG_QUERY.tail);
const activeTab = ref<DetailTab>(normalizeTab(route.query.tab));

const containerId = computed(() => String(route.params.id ?? '').trim());
const detailBreadcrumb = computed(() => [
  { labelKey: 'container.list.eyebrow', fallback: t('container.list.eyebrow') },
  { labelKey: 'container.detail.title', fallback: t('container.detail.title') },
]);
const pageTitle = computed(() => {
  if (detail.value) {
    return displayName(detail.value);
  }
  return containerId.value || t('container.detail.title');
});
const environmentRows = computed(() => normalizeEnvironmentRows(detail.value));
const resourceMetrics = computed(() => {
  const current = detail.value;
  const resource = current?.resource;
  const cpuValue = formatPercent(resource?.cpu_percent);
  const memoryPercent = formatPercent(resource?.memory_percent);
  const status = current ? resourceStatus(current) : emptyResourceStatus();

  return {
    cpu: {
      description:
        cpuValue === '-' ? t('container.detail.resources.noData') : t('container.detail.resources.currentSnapshot'),
      progress: toProgressPercent(resource?.cpu_percent),
      progressLabel: cpuValue,
      value: cpuValue,
    },
    memory: {
      description: memoryPercent,
      progress: toProgressPercent(resource?.memory_percent),
      progressLabel: memoryPercent,
      value: current ? memorySummary(current) : '-',
    },
    status,
  };
});
const cpuDetailMetrics = computed(() =>
  buildCpuDetailMetrics(
    detail.value?.resource,
    {
      cpuLimit: t('container.detail.resources.cpuLimitWithOnline'),
      cpuPercent: t('container.detail.resources.cpuPercent'),
      kernelTime: t('container.detail.resources.cpuKernelTime'),
      systemCpuTime: t('container.detail.resources.systemCpuTime'),
      throttlingCount: t('container.detail.resources.throttlingCount'),
      throttlingInactiveHint: t('container.detail.resources.throttlingInactiveHint'),
      throttlingSignalHint: t('container.detail.resources.throttlingSignalHint'),
      throttlingTime: t('container.detail.resources.throttlingTime'),
      totalCpuTime: t('container.detail.resources.totalCpuTime'),
      userTime: t('container.detail.resources.cpuUserTime'),
    },
    locale.value,
  ),
);
const resourceDetailGroups = computed<ResourceDetailGroup[]>(() => {
  const current = detail.value;
  if (!current) {
    return [];
  }
  const status = resourceStatus(current);

  return [
    {
      key: 'memory',
      title: t('container.detail.resources.memoryDetails'),
      rows: [
        ...metricRows([
          ['memory_usage_bytes', t('container.detail.resources.memoryUsage'), 'bytes'],
          ['memory_cache', t('container.detail.resources.memoryCache'), 'bytes'],
          ['memory_limit_bytes', t('container.detail.resources.memoryLimit'), 'bytes'],
          ['memory_rss', t('container.detail.resources.memoryRss'), 'bytes'],
          ['memory_percent', t('container.detail.resources.memoryPercent'), 'percent'],
          ['memory_active_file', t('container.detail.resources.memoryActiveFile'), 'bytes'],
          ['memory_inactive_file', t('container.detail.resources.memoryInactiveFile'), 'bytes'],
          ['memory_pgfault', t('container.detail.resources.memoryPgfault'), 'number'],
          ['memory_pgmajfault', t('container.detail.resources.memoryPgmajfault'), 'number'],
        ]),
        {
          key: 'memory-placeholder',
          label: '',
          type: 'placeholder',
          value: '—',
        },
      ],
    },
    {
      key: 'cpu',
      title: t('container.detail.resources.cpuDetails'),
      rows: [],
    },
    {
      key: 'network',
      title: t('container.detail.resources.networkIo'),
      rows: metricRows([
        ['rx_bytes', t('container.detail.resources.rxBytes'), 'bytes'],
        ['tx_bytes', t('container.detail.resources.txBytes'), 'bytes'],
        ['rx_packets', t('container.detail.resources.rxPackets'), 'number'],
        ['tx_packets', t('container.detail.resources.txPackets'), 'number'],
        ['rx_errors', t('container.detail.resources.rxErrors'), 'number'],
        ['tx_errors', t('container.detail.resources.txErrors'), 'number'],
        ['rx_dropped', t('container.detail.resources.rxDropped'), 'number'],
        ['tx_dropped', t('container.detail.resources.txDropped'), 'number'],
      ]),
    },
    {
      key: 'process',
      title: t('container.detail.resources.processInfo'),
      rows: [
        ...metricRows([
          ['pids_current', t('container.detail.resources.pidsCurrent'), 'number'],
          ['pids_limit', t('container.detail.resources.pidsLimit'), 'number'],
        ]),
        {
          key: 'status',
          label: t('container.detail.resources.status'),
          theme: status.theme,
          type: 'tag',
          value: status.value,
        },
        {
          key: 'collected-at',
          label: t('container.detail.resources.collectedAt'),
          type: 'text',
          value: status.collectedAt === '-' ? notCollectedLabel() : status.collectedAt,
        },
      ],
    },
  ];
});
const overviewSections = computed<ContainerOverviewInfoSection[]>(() => {
  const current = detail.value;
  if (!current) {
    return [];
  }

  const imageId = readableImageId(current.image_id);

  return [
    {
      key: 'basic',
      title: t('container.detail.overview.basicInfo'),
      rows: [
        {
          displayValue: displayName(current),
          key: 'name',
          label: t('container.detail.overview.fields.name'),
          type: 'text',
        },
        {
          code: true,
          copyValue: current.id,
          displayValue: shortContainerId(current),
          key: 'container-id',
          label: t('container.detail.overview.fields.containerId'),
          testId: 'container-id-copy',
          type: 'copy',
        },
        {
          copyValue: current.image,
          displayValue: current.image || '-',
          key: 'image',
          label: t('container.detail.overview.fields.image'),
          type: 'copy',
        },
        {
          code: true,
          copyValue: imageId,
          displayValue: shortIdentifier(imageId),
          key: 'image-id',
          label: t('container.detail.overview.fields.imageId'),
          testId: 'image-id-copy',
          type: 'copy',
        },
        {
          displayValue: runtimeLabel(current),
          key: 'runtime',
          label: t('container.detail.overview.fields.runtime'),
          type: 'text',
        },
      ],
    },
    {
      key: 'runtime',
      title: t('container.detail.overview.runtimeInfo'),
      rows: [
        {
          key: 'status',
          label: t('container.detail.overview.fields.status'),
          tagLabel: stateLabel(current.state),
          tagTheme: stateTheme(current.state),
          type: 'tag',
        },
        {
          displayValue: current.state || '-',
          key: 'state',
          label: t('container.detail.overview.fields.state'),
          type: 'text',
        },
        {
          key: 'health',
          label: t('container.detail.overview.fields.health'),
          tagLabel: healthLabel(current.health),
          tagTheme: healthTheme(current.health),
          type: 'tag',
        },
        {
          displayValue: formatTime(current.created_at),
          key: 'created-at',
          label: t('container.detail.overview.fields.createdAt'),
          type: 'text',
        },
        {
          displayValue: formatTime(current.started_at),
          key: 'started-at',
          label: t('container.detail.overview.fields.startedAt'),
          type: 'text',
        },
        {
          displayValue: formatTime(current.inspect_updated_at),
          key: 'updated-at',
          label: t('container.detail.overview.fields.updatedAt'),
          type: 'text',
        },
      ],
    },
    {
      key: 'resource-network',
      title: t('container.detail.overview.resourceNetwork'),
      rows: [
        {
          displayValue: formatPercent(current.resource?.cpu_percent),
          key: 'cpu',
          label: t('container.detail.resources.cpu'),
          type: 'text',
        },
        {
          displayValue: memorySummary(current),
          key: 'memory',
          label: t('container.detail.resources.memory'),
          type: 'text',
        },
        {
          displayValue: current.primary_ip || '-',
          key: 'primary-ip',
          label: t('container.detail.network.primaryIp'),
          type: 'text',
        },
        {
          displayValue: networkSummary(current),
          key: 'network-mode',
          label: t('container.detail.overview.fields.networkMode'),
          type: 'text',
        },
        {
          displayValue: primaryNetworkName(current),
          key: 'network-name',
          label: t('container.detail.overview.fields.networkName'),
          type: 'text',
        },
        {
          emptyLabel: t('container.detail.network.noPublicPorts'),
          key: 'ports',
          label: t('container.detail.network.ports'),
          ports: current.ports.map((port) => portLabel(port)),
          type: 'ports',
        },
      ],
    },
  ];
});
const environmentColumns = computed<TableProps['columns']>(() => [
  { colKey: 'name', title: t('container.detail.config.envName'), minWidth: 220, ellipsis: true },
  { colKey: 'value', title: t('container.detail.config.envValue'), minWidth: 260, ellipsis: true },
  { colKey: 'policy', title: t('container.detail.config.envPolicy'), width: 160, align: 'center' },
  { colKey: 'operation', title: t('container.detail.operation'), width: 112, align: 'center' },
]);
const networkColumns = computed<TableProps['columns']>(() => [
  { colKey: 'name', title: t('container.detail.network.name'), minWidth: 180, ellipsis: true },
  { colKey: 'ip_address', title: t('container.detail.network.ipAddress'), minWidth: 160, ellipsis: true },
  { colKey: 'gateway', title: t('container.detail.network.gateway'), minWidth: 160, ellipsis: true },
  { colKey: 'mac_address', title: t('container.detail.network.macAddress'), minWidth: 180, ellipsis: true },
]);
const mountColumns = computed<TableProps['columns']>(() => [
  { colKey: 'destination', title: t('container.detail.storage.destination'), minWidth: 240, ellipsis: true },
  { colKey: 'source', title: t('container.detail.storage.source'), minWidth: 260, ellipsis: true },
  { colKey: 'type', title: t('container.detail.storage.type'), width: 120, align: 'center' },
  { colKey: 'mode', title: t('container.detail.storage.mode'), width: 120, align: 'center' },
  { colKey: 'read_only', title: t('container.detail.storage.access'), width: 120, align: 'center' },
]);

onMounted(() => {
  void loadDetail();
  if (activeTab.value === 'logs') {
    void loadLogs();
  }
});

watch(
  () => route.params.id,
  () => {
    resetDetailState();
    void loadDetail();
    if (activeTab.value === 'logs') {
      void loadLogs();
    }
  },
);

watch(
  () => route.query.tab,
  (tab) => {
    const normalized = normalizeTab(tab);
    activeTab.value = normalized;
    if (normalized === 'logs' && !logs.value) {
      void loadLogs();
    }
  },
);

watch(logLineLimit, () => {
  if (activeTab.value === 'logs') {
    void loadLogs();
  }
});

async function loadDetail() {
  if (!containerId.value) {
    detail.value = null;
    logs.value = null;
    error.value = t('container.detail.missingId');
    return;
  }

  loading.value = true;
  error.value = '';
  try {
    detail.value = await getContainer(containerId.value);
  } catch (loadError) {
    detail.value = null;
    error.value = resolveLocalizedErrorMessage(t, loadError, t('container.list.detail.loadFailed'));
    logger.warn('failed to fetch container detail', loadError);
  } finally {
    loading.value = false;
  }
}

async function loadLogs() {
  if (!containerId.value) {
    logs.value = null;
    return;
  }
  logsLoading.value = true;
  logsError.value = '';
  try {
    logs.value = await getContainerLogs(containerId.value, {
      ...DEFAULT_LOG_QUERY,
      tail: logLineLimit.value,
    });
  } catch (loadError) {
    logsError.value = resolveLocalizedErrorMessage(t, loadError, t('container.list.logs.loadFailed'));
    logger.warn('failed to fetch container logs', loadError);
  } finally {
    logsLoading.value = false;
  }
}

function resetDetailState() {
  detail.value = null;
  error.value = '';
  logs.value = null;
  logsError.value = '';
}

function handleTabChange(value: string | number) {
  const tab = normalizeTab(value);
  activeTab.value = tab;
  void router.replace({
    params: route.params,
    query: {
      ...route.query,
      tab,
    },
  });
  if (tab === 'logs' && !logs.value) {
    void loadLogs();
  }
}

function goBack() {
  if (window.history.length > 1) {
    router.back();
    return;
  }
  void router.push({ name: CONTAINER_BOOTSTRAP_ROUTE.LIST.routeName });
}

async function copyEnvironmentValue(row: EnvironmentRow) {
  await copyDetailText(row.rawValue);
}

async function copyDetailText(text: string) {
  if (!text) return;
  const copied = await copyTextToClipboard(text);
  if (copied) {
    MessagePlugin.success(t('container.detail.copySuccess'));
    return;
  }
  MessagePlugin.error(t('container.detail.copyError'));
}

function normalizeTab(value: unknown): DetailTab {
  const raw = Array.isArray(value) ? value[0] : value;
  return typeof raw === 'string' && DETAIL_TABS.includes(raw as DetailTab) ? (raw as DetailTab) : 'overview';
}

function normalizeEnvironmentRows(nextDetail: ContainerDetail | null): EnvironmentRow[] {
  const detailRecord = readUnknownRecord(nextDetail);
  const source = detailRecord?.environment;
  if (!Array.isArray(source)) {
    return [];
  }

  return source.flatMap((item) => {
    const record = readUnknownRecord(item);
    const name = readString(record?.name ?? record?.key);
    if (!name) {
      return [];
    }

    const rawPolicy = readString(record?.policy ?? record?.visibility ?? record?.state);
    const masked = record?.masked === true;
    const rawValue = readRawString(record?.value);
    const policy = normalizeEnvironmentPolicy(
      rawPolicy,
      readString(detailRecord?.environment_policy),
      masked,
      rawValue,
    );
    const value = policy === 'hidden' ? '' : rawValue;

    return [
      {
        copyable: Boolean(rawValue) && policy !== 'hidden',
        name,
        policy,
        rawValue,
        value: value || environmentValueFallback(policy),
      },
    ];
  });
}

function readUnknownRecord(value: unknown): Record<string, unknown> | null {
  return value && typeof value === 'object' && !Array.isArray(value) ? (value as Record<string, unknown>) : null;
}

function readString(value: unknown) {
  return typeof value === 'string' ? value.trim() : '';
}

function readRawString(value: unknown) {
  return typeof value === 'string' ? value : '';
}

function normalizeEnvironmentPolicy(
  value: string,
  detailPolicy = '',
  masked = false,
  rawValue = '',
): EnvironmentPolicy {
  if (value === 'plain' || value === 'masked' || value === 'hidden') {
    return value;
  }
  if (rawValue && !masked) {
    return 'plain';
  }
  if (masked) {
    return 'masked';
  }
  if (detailPolicy === 'plain' || detailPolicy === 'masked' || detailPolicy === 'hidden') {
    return detailPolicy;
  }
  return 'unknown';
}

function environmentValueFallback(policy: EnvironmentPolicy) {
  if (policy === 'masked') return t('container.detail.config.maskedValue');
  if (policy === 'hidden') return t('container.detail.config.hiddenValue');
  return '-';
}

function policyLabel(policy: EnvironmentPolicy) {
  return t(`container.detail.config.policy.${policy}`);
}

function policyTheme(policy: EnvironmentPolicy) {
  if (policy === 'plain') return 'success';
  if (policy === 'masked') return 'warning';
  if (policy === 'hidden') return 'danger';
  return 'default';
}

function displayName(row: ContainerDetail) {
  return row.name || row.names[0] || shortContainerId(row);
}

function stateLabel(state: ContainerState) {
  return t(`container.list.states.${state}`);
}

function healthLabel(health?: ContainerHealth | null) {
  return t(`container.list.health.${health || 'none'}`);
}

function healthTheme(health?: ContainerHealth | null) {
  if (health === 'healthy') return 'success';
  if (health === 'unhealthy') return 'danger';
  if (health === 'starting') return 'warning';
  return 'default';
}

function stateTheme(state: ContainerState) {
  if (state === 'running') return 'success';
  if (state === 'created' || state === 'paused' || state === 'restarting') return 'warning';
  if (state === 'dead') return 'danger';
  return 'default';
}

function formatTime(value?: string | null) {
  return formatLocaleDateTime(value, locale);
}

function joinList(values?: string[]) {
  return values?.length ? values.join(' ') : '-';
}

function runtimeLabel(nextDetail: ContainerDetail) {
  return nextDetail.runtime || nextDetail.runtime_info?.runtime || '-';
}

function shortContainerId(nextDetail: ContainerDetail) {
  return shortIdentifier(nextDetail.id, nextDetail.short_id, 12);
}

function networkSummary(nextDetail: ContainerDetail) {
  return nextDetail.network_summary || primaryNetworkName(nextDetail);
}

function primaryNetworkName(nextDetail: ContainerDetail) {
  return nextDetail.networks[0]?.name || '-';
}

function resourceStatus(nextDetail: ContainerDetail) {
  const resource = nextDetail.resource;
  if (resource?.stats_available || resource?.available) {
    return {
      collectedAt: formatTime(nextDetail.inspect_updated_at),
      description: formatTime(nextDetail.inspect_updated_at),
      theme: 'success' as const,
      value: t('container.detail.resources.available'),
    };
  }
  if (resource?.stats_error_message || resource?.stats_error_key || resource?.unavailable_reason) {
    return {
      collectedAt: '-',
      description: resource.stats_error_message || resource.stats_error_key || resource.unavailable_reason || '-',
      theme: 'warning' as const,
      value: t('container.detail.resources.unavailable'),
    };
  }
  return emptyResourceStatus();
}

function emptyResourceStatus() {
  return {
    collectedAt: '-',
    description: '-',
    theme: 'default' as const,
    value: t('container.detail.resources.noData'),
  };
}

function metricRows(definitions: ResourceMetricDefinition[]): ResourceDetailRow[] {
  return definitions.map(([key, label, format]) => ({
    key,
    label,
    type: 'text',
    value: readMetricText(key, format),
  }));
}

function readMetricText(key: ResourceMetricKey, format: ResourceMetricFormat) {
  const value = readResourceMetric(key);
  if (format === 'bytes') {
    return formatResourceValue(formatBytes(value));
  }
  if (format === 'percent') {
    return formatResourceValue(formatPercent(value));
  }
  if (format === 'number') {
    return formatResourceValue(formatNumberMetric(value));
  }
  return formatResourceValue(readRawString(value));
}

function readCpuSystemTimeText() {
  return formatResourceValue(formatNanosecondsAsDuration(readResourceMetric('system_cpu_usage'), '-', locale.value));
}

function readCpuCountText() {
  return formatResourceValue(formatCpuCountText(readResourceMetric('online_cpus'), locale.value));
}

function readResourceMetric(key: ResourceMetricKey) {
  return detail.value?.resource?.[key];
}

function formatNumberMetric(value: unknown) {
  if (typeof value === 'number' && Number.isFinite(value)) {
    const currentLocale = typeof locale === 'string' ? locale : locale.value;
    return new Intl.NumberFormat(currentLocale).format(value);
  }
  if (typeof value === 'string' && value.trim()) {
    return value.trim();
  }
  return '-';
}

function formatResourceValue(value: string) {
  return value && value !== '-' ? value : notCollectedLabel();
}

function notCollectedLabel() {
  return t('container.detail.resources.notCollected');
}

function memorySummary(nextDetail: ContainerDetail) {
  const resource = nextDetail.resource;
  const usage = formatBytes(resource?.memory_usage_bytes);
  const limit = formatBytes(resource?.memory_limit_bytes);
  if (usage === '-' && limit === '-') {
    return '-';
  }
  return `${usage} / ${limit}`;
}

function shortIdentifier(value?: string | null, preferred?: string | null, maxLength = 28) {
  const normalized = value?.trim();
  if (!normalized) return '-';
  if (preferred?.trim()) return preferred.trim();
  if (normalized.length <= maxLength) return normalized;
  if (maxLength <= 12) return normalized.slice(0, maxLength);
  return `${normalized.slice(0, 18)}...${normalized.slice(-10)}`;
}

function readableImageId(value?: string | null) {
  const normalized = value?.trim();
  if (!normalized) return '-';
  return normalized.startsWith('sha256:') ? normalized.slice('sha256:'.length) : normalized;
}

function portLabel(port: ContainerDetail['ports'][number]) {
  const privatePort = port.private_port ? `${port.private_port}` : '-';
  const publicPort = port.public_port ? `${port.public_port}` : '';
  return publicPort ? `${publicPort}:${privatePort}/${port.type}` : `${privatePort}/${port.type}`;
}
</script>
<style scoped lang="less">
.container-detail-page {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.container-detail-summary {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.container-detail-summary-card {
  background: color-mix(in srgb, var(--td-bg-color-container) 92%, transparent);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 64%, transparent);
  height: 100%;
  min-width: 0;
}

.container-detail-summary-card :deep(.t-card__body) {
  height: calc(100% - var(--td-comp-size-xxxl));
}

.container-detail-metric,
.container-detail-section,
.container-detail-subsection {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.container-detail-summary-list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
  height: 100%;
  min-width: 0;
}

.container-detail-kv {
  display: grid;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-detail-kv--inline {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
}

.container-detail-kv > span {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
}

.container-detail-kv strong,
.container-detail-kv code {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-detail-kv code,
.container-detail-header-id {
  font-family: var(
    --td-font-family-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    monospace
  );
}

.container-detail-header-id {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  font: var(--td-font-body-small);
  min-height: var(--td-comp-size-xs);
}

.container-detail-header-meta {
  max-width: min(100%, 680px);
  min-width: 0;
}

.container-detail-header-meta :deep(.t-space-item) {
  min-width: 0;
}

.container-detail-header-meta :deep(.t-tag) {
  max-width: 100%;
}

.container-detail-summary__resource,
.container-detail-resource-grid {
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  min-width: 0;
}

.container-detail-resource-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.container-detail-summary__resource {
  grid-template-columns: 1fr;
}

.container-detail-metric strong,
.container-detail-subsection h3,
.container-detail-overview-group h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0;
}

.container-detail-metric span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  overflow-wrap: anywhere;
}

.container-detail-tag-row,
.container-detail-port-list {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.container-detail-resource-meter {
  border-radius: var(--td-radius-medium);
  display: flex;
  gap: var(--graft-density-gap-10);
  min-width: 0;
}

.container-detail-resource-meter--cpu {
  align-items: center;
  justify-content: space-between;
}

.container-detail-resource-meter--memory {
  flex-direction: column;
  justify-content: center;
}

.container-detail-resource-meter__content {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-detail-resource-meter__content span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.container-detail-resource-meter__content strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  min-width: 0;
  overflow-wrap: anywhere;
}

.container-detail-resource-meter__content em {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  font-style: normal;
}

.container-detail-resource-meter--memory :deep(.t-progress) {
  width: 100%;
}

.container-detail-port-chip {
  font-family: var(
    --td-font-family-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    monospace
  );
  max-width: 100%;
}

.container-detail-tabs-card {
  min-width: 0;
}

.container-detail-tabs-card :deep(.t-card__body) {
  padding: 0;
}

.container-detail-tabs-card :deep(.t-tabs__content) {
  padding-top: var(--graft-density-gap-12);
}

/*
 * Short detail tabs use the page scrollbar. Long-form tabs such as logs and raw JSON own
 * their internal scrolling so the page does not fight a second nested scrollbar.
 */
.container-detail-section {
  padding: 0;
}

.container-detail-section--overview {
  min-height: 0;
  padding: 0 var(--graft-density-gap-16) var(--graft-density-gap-16);
}

.container-detail-resource-status-card {
  align-items: center;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
  padding: var(--graft-density-gap-14);
}

.container-detail-resource-status-card__content {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.container-detail-resource-status-card__content strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  min-width: 0;
  overflow-wrap: anywhere;
}

.container-detail-resource-status-card__content span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  min-width: 0;
  overflow-wrap: anywhere;
}

.container-detail-resource-status-card__title {
  color: var(--td-text-color-secondary);
}

.container-resource-dashboard-section,
.container-resource-detail-section {
  background: var(--td-bg-color-container);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  margin-top: var(--graft-density-gap-12);
  min-width: 0;
  overflow: hidden;
  width: 100%;
}

.container-resource-dashboard-section {
  gap: var(--graft-density-gap-14);
  padding: var(--graft-density-gap-14) var(--graft-density-gap-16) var(--graft-density-gap-16);
}

.container-resource-dashboard-section__title,
.container-resource-detail-section__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-weight: 600;
  line-height: 22px;
}

.container-resource-detail-section__title {
  background: color-mix(in srgb, var(--td-bg-color-container) 86%, var(--td-bg-color-page));
  border-bottom: 1px solid color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  padding: var(--graft-density-gap-12) var(--graft-density-gap-16);
}

.container-resource-dashboard-grid,
.container-resource-detail-grid {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  min-width: 0;
}

.container-resource-dashboard-panel,
.container-resource-detail-card {
  background: color-mix(in srgb, var(--td-bg-color-container) 96%, var(--td-bg-color-page));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 56%, transparent);
  border-radius: var(--td-radius-medium);
  min-width: 0;
}

.container-resource-dashboard-panel {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  padding: var(--graft-density-gap-14);
}

.container-resource-dashboard-panel__heading {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
}

.container-resource-dashboard-panel__heading span {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
  min-width: 0;
}

.container-resource-dashboard-panel__heading strong,
.container-resource-dashboard-panel__usage {
  color: var(--td-brand-color);
  font: var(--td-font-title-small);
  font-weight: 600;
  min-width: 0;
  overflow-wrap: anywhere;
}

.container-resource-dashboard-panel__usage {
  color: var(--td-text-color-primary);
}

.container-resource-dashboard-panel :deep(.t-progress) {
  width: 100%;
}

.container-resource-dashboard-panel__meta {
  display: grid;
  gap: var(--graft-density-gap-8) var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  min-width: 0;
}

.container-resource-dashboard-panel__meta span {
  color: var(--td-text-color-secondary);
  display: flex;
  flex-direction: column;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-2);
  min-width: 0;
}

.container-resource-dashboard-panel__meta strong {
  color: var(--td-text-color-primary);
  font-weight: 500;
  min-width: 0;
  overflow-wrap: anywhere;
}

.container-resource-detail-grid {
  padding: var(--graft-density-gap-12) var(--graft-density-gap-16) var(--graft-density-gap-16);
}

.container-resource-detail-card {
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.container-resource-detail-card--cpu,
.container-resource-detail-card--memory {
  grid-column: 1 / -1;
}

.container-resource-detail-card h3 {
  border-bottom: 1px solid color-mix(in srgb, var(--td-component-stroke) 44%, transparent);
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 600;
  line-height: 22px;
  margin: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

.container-resource-detail-card__body {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: var(--graft-density-gap-6) var(--graft-density-gap-12);
}

.container-resource-detail-card__body--memory {
  display: grid;
  gap: 0 var(--graft-density-gap-16);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  padding: var(--graft-density-gap-6) var(--graft-density-gap-12);
}

.container-resource-cpu-metric-grid {
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: repeat(4, minmax(0, 1fr));
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.container-resource-cpu-metric {
  background: color-mix(in srgb, var(--td-bg-color-container) 94%, var(--td-bg-color-page));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 36%, transparent);
  border-radius: var(--td-radius-medium);
  display: grid;
  gap: var(--graft-density-gap-4);
  min-height: 78px;
  min-width: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

.container-resource-cpu-metric--warning {
  background: color-mix(in srgb, var(--td-warning-color-1) 42%, var(--td-bg-color-container));
  border-color: color-mix(in srgb, var(--td-warning-color) 52%, var(--td-component-stroke));
}

.container-resource-cpu-metric--muted .container-resource-cpu-metric__value,
.container-resource-cpu-metric--muted .container-resource-cpu-metric__hint {
  color: var(--td-text-color-placeholder);
}

.container-resource-cpu-metric__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  line-height: 20px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-resource-cpu-metric__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-weight: 600;
  justify-self: end;
  line-height: 24px;
  max-width: 100%;
  min-width: 0;
  overflow-wrap: anywhere;
  text-align: right;
}

.container-resource-cpu-metric--warning .container-resource-cpu-metric__value {
  color: var(--td-warning-color);
}

.container-resource-cpu-metric__hint {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  line-height: 18px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-resource-detail-row {
  align-items: center;
  column-gap: var(--graft-density-gap-12);
  display: grid;
  grid-template-columns: minmax(112px, 34%) minmax(0, 1fr);
  min-height: 34px;
  min-width: 0;
  width: 100%;
}

.container-resource-detail-row + .container-resource-detail-row {
  border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 30%, transparent);
}

.container-resource-detail-card__body--memory .container-resource-detail-row + .container-resource-detail-row {
  border-top: 0;
}

.container-resource-detail-card__body--memory .container-resource-detail-row {
  border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 30%, transparent);
  grid-template-columns: minmax(112px, 42%) minmax(0, 1fr);
}

.container-resource-detail-card__body--memory .container-resource-detail-row:nth-child(-n + 2) {
  border-top: 0;
}

.container-resource-detail-row--placeholder .container-resource-detail-row__label,
.container-resource-detail-row--placeholder .container-resource-detail-row__value {
  color: var(--td-text-color-placeholder);
}

.container-resource-detail-row__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  line-height: 20px;
  min-width: 0;
}

.container-resource-detail-row__value {
  align-items: center;
  color: var(--td-text-color-primary);
  display: inline-flex;
  font: var(--td-font-body-small);
  font-weight: 500;
  gap: var(--graft-density-gap-6);
  justify-content: flex-end;
  line-height: 22px;
  min-width: 0;
  overflow: hidden;
  text-align: right;
}

.container-resource-detail-row__value > span:not(.t-tag) {
  display: inline-block;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (width <= 1360px) {
  .container-detail-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 960px) {
  .container-detail-resource-grid,
  .container-resource-dashboard-grid,
  .container-resource-detail-grid,
  .container-resource-detail-card__body--memory,
  .container-resource-cpu-metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 720px) {
  .container-detail-summary,
  .container-detail-summary__resource {
    grid-template-columns: 1fr;
  }

  .container-detail-resource-grid,
  .container-resource-dashboard-grid,
  .container-resource-detail-grid,
  .container-resource-detail-card__body--memory,
  .container-resource-cpu-metric-grid,
  .container-resource-dashboard-panel__meta {
    grid-template-columns: 1fr;
  }

  .container-resource-detail-card__body--memory .container-resource-detail-row:nth-child(-n + 2) {
    border-top: 1px solid color-mix(in srgb, var(--td-component-stroke) 30%, transparent);
  }

  .container-resource-detail-card__body--memory .container-resource-detail-row:first-child {
    border-top: 0;
  }

  .container-resource-detail-row {
    align-items: flex-start;
    gap: var(--graft-density-gap-4);
    grid-template-columns: 1fr;
    padding: var(--graft-density-gap-8) 0;
  }

  .container-resource-detail-row__value {
    justify-content: flex-start;
    text-align: left;
    width: 100%;
  }

  .container-detail-header-meta {
    max-width: 100%;
  }

  .container-detail-header-meta :deep(.t-space) {
    width: 100%;
  }

  .container-detail-header-meta :deep(.t-space-item) {
    max-width: 100%;
  }
}
</style>
