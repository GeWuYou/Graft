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
          <t-card
            class="container-detail-summary-card container-detail-summary-card--identity"
            size="small"
            :bordered="false"
            :title="t('container.detail.summary.identity')"
          >
            <div class="container-detail-summary__main">
              <strong>{{ displayName(detail) }}</strong>
              <span>{{ detail.image }}</span>
              <code>{{ detail.short_id || detail.id }}</code>
              <div class="container-detail-tag-row">
                <t-tag :theme="stateTheme(detail.state)" variant="light-outline">
                  {{ stateLabel(detail.state) }}
                </t-tag>
                <t-tag v-if="detail.health" :theme="healthTheme(detail.health)" variant="light-outline">
                  {{ healthLabel(detail.health) }}
                </t-tag>
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
                  <strong>{{ t('container.detail.resources.currentSnapshot') }}</strong>
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
              <strong v-else>-</strong>
            </div>
          </t-card>
        </section>

        <t-card class="container-detail-tabs-card" :bordered="true">
          <t-tabs v-model:value="activeTab" theme="card" @change="handleTabChange">
            <t-tab-panel value="overview" :label="t('container.detail.tabs.overview')" :destroy-on-hide="false">
              <section class="container-detail-section">
                <div class="container-detail-overview-groups">
                  <section class="container-detail-overview-group">
                    <h3>{{ t('container.detail.overview.basicInfo') }}</h3>
                    <t-descriptions :column="2" item-layout="vertical" bordered table-layout="fixed">
                      <t-descriptions-item :label="t('container.list.fields.name')">
                        {{ displayName(detail) }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('container.list.fields.id')">
                        <span class="container-detail-copyable-value">
                          <t-tooltip :content="detail.id" placement="top-left">
                            <code>{{ shortIdentifier(detail.id, detail.short_id) }}</code>
                          </t-tooltip>
                          <t-button
                            v-if="detail.id"
                            data-testid="container-id-copy"
                            size="small"
                            theme="primary"
                            variant="text"
                            @click="copyDetailText(detail.id)"
                          >
                            {{ t('container.detail.copy') }}
                          </t-button>
                        </span>
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('container.list.fields.image')">
                        {{ detail.image }}
                      </t-descriptions-item>
                      <t-descriptions-item :label="t('container.list.fields.imageId')">
                        <span class="container-detail-copyable-value">
                          <t-tooltip :content="readableImageId(detail.image_id)" placement="top-left">
                            <code>{{ shortIdentifier(readableImageId(detail.image_id)) }}</code>
                          </t-tooltip>
                          <t-button
                            v-if="detail.image_id"
                            data-testid="image-id-copy"
                            size="small"
                            theme="primary"
                            variant="text"
                            @click="copyDetailText(readableImageId(detail.image_id))"
                          >
                            {{ t('container.detail.copy') }}
                          </t-button>
                        </span>
                      </t-descriptions-item>
                    </t-descriptions>
                  </section>

                  <section class="container-detail-overview-group">
                    <h3>{{ t('container.detail.overview.runtimeInfo') }}</h3>
                    <t-descriptions :column="2" item-layout="vertical" bordered table-layout="fixed">
                      <t-descriptions-item :label="t('container.list.fields.state')">
                        <t-tag :theme="stateTheme(detail.state)" variant="light-outline">
                          {{ stateLabel(detail.state) }}
                        </t-tag>
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
                </div>
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
  copyText as copyTextToClipboard,
  formatBytes,
  formatLocaleDateTime,
  formatPercent,
  JsonViewer,
  LogViewer,
  toProgressPercent,
} from '@/shared/observability';
import { createLogger } from '@/utils/logger';

import { getContainer, getContainerLogs } from '../../api/container';
import { CONTAINER_BOOTSTRAP_ROUTE } from '../../contract/bootstrap';
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

function shortIdentifier(value?: string | null, preferred?: string | null) {
  const normalized = value?.trim();
  if (!normalized) return '-';
  if (preferred?.trim()) return preferred.trim();
  if (normalized.length <= 32) return normalized;
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
  grid-template-columns: repeat(3, minmax(0, 1fr));
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

.container-detail-summary__resource {
  grid-template-columns: minmax(112px, 0.8fr) minmax(0, 1.2fr);
}

.container-detail-resource-descriptions {
  margin-top: var(--graft-density-gap-12);
}

.container-detail-summary__main strong,
.container-detail-metric strong,
.container-detail-subsection h3,
.container-detail-overview-group h3 {
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

.container-detail-overview-groups {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.container-detail-overview-group {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.container-detail-copyable-value {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-8);
  max-width: 100%;
  min-width: 0;
}

.container-detail-copyable-value code {
  color: var(--td-text-color-primary);
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
