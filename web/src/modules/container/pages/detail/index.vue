<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="container-detail-page" data-page-type="operations-detail">
    <management-page-header
      title-key="container.detail.title"
      :title="pageTitle"
      description-key="container.detail.description"
      :description="t('container.detail.description')"
      :source="{ labelKey: 'container.list.eyebrow', fallback: t('container.list.eyebrow') }"
    >
      <template #meta>
        <t-space break-line size="small">
          <t-tag v-if="detail" :theme="stateTheme(detail.state)" variant="light-outline">
            {{ stateLabel(detail.state) }}
          </t-tag>
          <t-tag v-if="detail?.health" :theme="healthTheme(detail.health)" variant="light-outline">
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
          <t-card size="small" :bordered="true" :title="t('container.detail.summary.identity')">
            <div class="container-detail-summary__main">
              <strong>{{ displayName(detail) }}</strong>
              <span>{{ detail.image }}</span>
              <code>{{ detail.short_id || detail.id }}</code>
              <t-tag :theme="stateTheme(detail.state)" variant="light-outline">
                {{ stateLabel(detail.state) }}
              </t-tag>
            </div>
          </t-card>
          <t-card size="small" :bordered="true" :title="t('container.detail.summary.resources')">
            <div class="container-detail-summary__resource">
              <metric-card
                :title="t('container.detail.resources.cpu')"
                :value="formatPercent(detail.resource?.cpu_percent)"
                :progress="toProgressPercent(detail.resource?.cpu_percent)"
                :progress-label="formatPercent(detail.resource?.cpu_percent)"
              />
              <metric-card
                :title="t('container.detail.resources.memory')"
                :value="formatPercent(detail.resource?.memory_percent)"
                :description="memorySummary(detail)"
                :progress="toProgressPercent(detail.resource?.memory_percent)"
                :progress-label="formatPercent(detail.resource?.memory_percent)"
              />
            </div>
          </t-card>
          <t-card size="small" :bordered="true" :title="t('container.detail.summary.network')">
            <div class="container-detail-metric">
              <span>{{ t('container.detail.network.primaryIp') }}</span>
              <strong>{{ detail.primary_ip || '-' }}</strong>
            </div>
            <div class="container-detail-metric">
              <span>{{ t('container.detail.network.summary') }}</span>
              <strong>{{ detail.network_summary || '-' }}</strong>
            </div>
            <div class="container-detail-metric">
              <span>{{ t('container.detail.network.ports') }}</span>
              <strong>{{ portSummary(detail) }}</strong>
            </div>
          </t-card>
        </section>

        <t-card class="container-detail-tabs-card" :bordered="true">
          <t-tabs v-model:value="activeTab" theme="card" @change="handleTabChange">
            <t-tab-panel value="overview" :label="t('container.detail.tabs.overview')" :destroy-on-hide="false">
              <section class="container-detail-section">
                <t-descriptions :column="2" item-layout="vertical" bordered table-layout="fixed">
                  <t-descriptions-item :label="t('container.list.fields.name')">
                    {{ displayName(detail) }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.list.fields.id')">
                    {{ detail.id }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.list.fields.image')">
                    {{ detail.image }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.list.fields.imageId')">
                    {{ detail.image_id || '-' }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.list.fields.state')">
                    {{ stateLabel(detail.state) }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.list.fields.status')">
                    {{ detail.status || '-' }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.list.fields.createdAt')">
                    {{ formatTime(detail.created_at) }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.list.fields.startedAt')">
                    {{ formatTime(detail.started_at) }}
                  </t-descriptions-item>
                </t-descriptions>
              </section>
            </t-tab-panel>

            <t-tab-panel value="resources" :label="t('container.detail.tabs.resources')" :destroy-on-hide="false">
              <section class="container-detail-section">
                <div class="container-detail-resource-grid">
                  <metric-card
                    :title="t('container.detail.resources.cpu')"
                    :value="formatPercent(detail.resource?.cpu_percent)"
                    :description="t('container.detail.resources.currentSnapshot')"
                    :progress="toProgressPercent(detail.resource?.cpu_percent)"
                    :progress-label="formatPercent(detail.resource?.cpu_percent)"
                  />
                  <metric-card
                    :title="t('container.detail.resources.memory')"
                    :value="memorySummary(detail)"
                    :description="formatPercent(detail.resource?.memory_percent)"
                    :progress="toProgressPercent(detail.resource?.memory_percent)"
                    :progress-label="formatPercent(detail.resource?.memory_percent)"
                  />
                  <metric-card
                    :title="t('container.detail.resources.status')"
                    :value="resourceAvailability(detail)"
                    :description="formatTime(detail.inspect_updated_at)"
                  />
                </div>
                <t-descriptions
                  class="container-detail-resource-descriptions"
                  :column="2"
                  item-layout="vertical"
                  bordered
                  table-layout="fixed"
                >
                  <t-descriptions-item :label="t('container.detail.resources.cpu')">
                    {{ formatPercent(detail.resource?.cpu_percent) }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.detail.resources.memoryUsage')">
                    {{ formatBytes(detail.resource?.memory_usage_bytes) }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.detail.resources.memoryLimit')">
                    {{ formatBytes(detail.resource?.memory_limit_bytes) }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.detail.resources.memoryPercent')">
                    {{ formatPercent(detail.resource?.memory_percent) }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.detail.resources.status')">
                    {{ resourceAvailability(detail) }}
                  </t-descriptions-item>
                  <t-descriptions-item :label="t('container.detail.resources.collectedAt')">
                    {{ formatTime(detail.inspect_updated_at) }}
                  </t-descriptions-item>
                </t-descriptions>
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
                  :search-placeholder="t('container.detail.logs.searchPlaceholder')"
                  :wrap-label="t('container.detail.logs.wrap')"
                  :follow-tail-label="t('container.detail.logs.followTail')"
                  :empty-label="t('container.detail.logs.empty')"
                  :truncated-label="t('container.detail.logs.truncated')"
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
  formatBytes,
  formatLocaleDateTime,
  formatPercent,
  JsonViewer,
  LogViewer,
  toProgressPercent,
} from '@/shared/observability';
import { createLogger } from '@/utils/logger';

import { getContainer, getContainerLogs } from '../../api/container';
import type { ContainerDetail, ContainerHealth, ContainerLogResponse, ContainerState } from '../../types/container';

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
const pageTitle = computed(() => {
  const name = detail.value ? displayName(detail.value) : containerId.value;
  return name ? `${t('container.detail.title')} - ${name}` : t('container.detail.title');
});
const environmentRows = computed(() => normalizeEnvironmentRows(detail.value));
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
    void loadDetail();
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
    error.value = t('container.detail.missingId');
    return;
  }

  loading.value = true;
  error.value = '';
  try {
    detail.value = await getContainer(containerId.value);
  } catch (loadError) {
    error.value = resolveLocalizedErrorMessage(t, loadError, t('container.list.detail.loadFailed'));
    logger.warn('failed to fetch container detail', loadError);
  } finally {
    loading.value = false;
  }
}

async function loadLogs() {
  if (!containerId.value) return;
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
  void router.push({ name: 'ContainerList' });
}

async function copyEnvironmentValue(row: EnvironmentRow) {
  await copyText(row.rawValue);
}

async function copyText(text: string) {
  if (!text) return;
  try {
    await navigator.clipboard.writeText(text);
    MessagePlugin.success(t('container.detail.copySuccess'));
  } catch (copyError) {
    logger.warn('failed to copy container detail text', copyError);
    MessagePlugin.error(t('container.detail.copyError'));
  }
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
    const rawValue = readString(record?.value);
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
  return row.name || row.names[0] || row.id;
}

function stateLabel(state: ContainerState) {
  return t(`container.list.states.${state}`);
}

function healthLabel(health?: ContainerHealth | null) {
  return t(`container.list.health.${health || 'unavailable'}`);
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

function resourceAvailability(nextDetail: ContainerDetail) {
  const resource = nextDetail.resource;
  if (resource?.stats_available || resource?.available) {
    return t('container.detail.resources.available');
  }
  return resource?.stats_error_message || resource?.stats_error_key || resource?.unavailable_reason || '-';
}

function memorySummary(nextDetail: ContainerDetail) {
  const resource = nextDetail.resource;
  return `${formatBytes(resource?.memory_usage_bytes)} / ${formatBytes(resource?.memory_limit_bytes)}`;
}

function portSummary(nextDetail: ContainerDetail) {
  if (!nextDetail.ports.length) {
    return '-';
  }

  const firstPorts = nextDetail.ports.slice(0, 2).map((port) => {
    const privatePort = port.private_port ? `${port.private_port}` : '-';
    const publicPort = port.public_port ? `${port.public_port}` : '';
    return publicPort ? `${publicPort}:${privatePort}/${port.type}` : `${privatePort}/${port.type}`;
  });
  const restCount = nextDetail.ports.length - firstPorts.length;
  return restCount > 0 ? `${firstPorts.join(', ')} +${restCount}` : firstPorts.join(', ');
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
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.container-detail-summary__main,
.container-detail-metric,
.container-detail-section,
.container-detail-subsection {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-width: 0;
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

.container-detail-resource-descriptions {
  margin-top: var(--graft-density-gap-12);
}

.container-detail-summary__main strong,
.container-detail-metric strong,
.container-detail-subsection h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0;
}

.container-detail-summary__main span,
.container-detail-summary__main code,
.container-detail-metric span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  overflow-wrap: anywhere;
}

.container-detail-tabs-card {
  min-width: 0;
}

.container-detail-tabs-card :deep(.t-card__body) {
  padding: 0;
}

.container-detail-section {
  padding: var(--graft-density-gap-16) 0 0;
}

@media (width <= 960px) {
  .container-detail-summary,
  .container-detail-summary__resource,
  .container-detail-resource-grid {
    grid-template-columns: 1fr;
  }
}
</style>
