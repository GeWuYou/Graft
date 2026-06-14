<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="container-page" data-page-type="list-form-detail">
    <management-page-header
      title-key="container.list.title"
      :title="t('container.list.title')"
      description-key="container.list.description"
      :description="t('container.list.description')"
      :source="{ labelKey: 'container.list.eyebrow', fallback: t('container.list.eyebrow') }"
    >
      <template #meta>
        <t-space break-line size="small">
          <t-tag :theme="runtimeStatusTheme" variant="light-outline">
            {{ t('container.list.runtimeLabel') }}: {{ runtimeSummary }}
          </t-tag>
          <t-tag theme="default" variant="light-outline">
            {{ t('container.list.runtimeContainers', runtimeCountParams) }}
          </t-tag>
        </t-space>
      </template>
      <template #actions>
        <t-button theme="primary" :loading="loading" @click="refreshContainers">
          <template #icon><refresh-icon /></template>
          {{ t('container.list.refresh') }}
        </t-button>
      </template>
    </management-page-header>

    <management-toolbar>
      <template #filters>
        <t-input
          v-model="filters.keyword"
          class="management-list-search"
          clearable
          :placeholder="t('container.list.filters.searchPlaceholder')"
          @enter="applyFilters"
        >
          <template #prefix-icon><search-icon /></template>
        </t-input>
        <t-select
          v-model="filters.status"
          class="management-toolbar__select"
          :placeholder="t('container.list.filters.status')"
        >
          <t-option value="all" :label="t('container.list.filters.allStatuses')" />
          <t-option v-for="status in statusOptions" :key="status" :value="status" :label="stateLabel(status)" />
        </t-select>
        <t-button theme="primary" @click="applyFilters">
          {{ t('container.list.filters.query') }}
        </t-button>
        <t-button theme="default" variant="text" @click="resetFilters">
          {{ t('container.list.filters.reset') }}
        </t-button>
      </template>
    </management-toolbar>

    <management-table-card>
      <template #head>
        <div class="container-table-head">
          <p class="container-table-head__summary">
            {{ t('container.list.tableSummary', { count: filteredRows.length }) }}
          </p>
          <p>{{ t('container.list.tableHint') }}</p>
        </div>
      </template>
      <template #toolbar>
        <table-view-toolbar
          :refresh-label="t('container.list.refresh')"
          :refresh-loading="loading"
          @refresh="refreshContainers"
        />
      </template>

      <t-alert v-if="listError" class="container-alert" theme="error" :title="listError">
        <template #operation>
          <t-button theme="danger" variant="text" @click="refreshContainers">
            {{ t('container.list.retry') }}
          </t-button>
        </template>
      </t-alert>

      <t-table
        row-key="id"
        :columns="columns"
        :data="filteredRows"
        :loading="loading"
        :table-content-width="tableContentWidth"
        cell-empty-content="-"
        table-layout="fixed"
        hover
      >
        <template #state="{ row }">
          <t-tag :theme="stateTheme(row.state)" variant="light-outline">
            {{ stateLabel(row.state) }}
          </t-tag>
        </template>

        <template #name="{ row }">
          <div class="container-identity">
            <span class="container-identity__name">{{ displayName(row) }}</span>
            <span class="container-identity__id">{{ row.id }}</span>
          </div>
        </template>

        <template #image="{ row }">
          <div class="container-image">
            <span>{{ row.image }}</span>
            <span v-if="row.runtime" class="container-muted">{{ row.runtime }}</span>
          </div>
        </template>

        <template #ports="{ row }">
          <div v-if="formatPorts(row.ports).length" class="container-port-list">
            <t-tag v-for="port in formatPorts(row.ports)" :key="port" size="small" theme="default" variant="light">
              {{ port }}
            </t-tag>
          </div>
          <span v-else>-</span>
        </template>

        <template #created_at="{ row }">
          {{ formatTime(row.created_at) }}
        </template>

        <template #started_at="{ row }">
          {{ formatTime(row.started_at) }}
        </template>

        <template #restart_policy="{ row }">
          {{ row.restart_policy || '-' }}
        </template>

        <template #operation="{ row }">
          <t-space class="container-actions" size="small" align="center">
            <t-button
              v-permission="permissionCodes.DETAIL"
              data-testid="container-action-detail"
              theme="primary"
              variant="text"
              size="small"
              @click="openDetail(row)"
            >
              {{ t('container.list.actions.detail') }}
            </t-button>
            <t-button
              v-permission="permissionCodes.LOGS"
              data-testid="container-action-logs"
              theme="primary"
              variant="text"
              size="small"
              @click="openLogs(row)"
            >
              {{ t('container.list.actions.logs') }}
            </t-button>
            <t-popconfirm
              v-permission="permissionCodes.START"
              :content="t('container.list.actions.confirmStart')"
              :confirm-btn="t('container.list.actions.confirm')"
              :cancel-btn="t('container.list.actions.cancel')"
              theme="warning"
              @confirm="runAction('start', row)"
            >
              <t-button
                theme="success"
                variant="text"
                size="small"
                :loading="actionLoadingKey === actionKey('start', row)"
              >
                {{ t('container.list.actions.start') }}
              </t-button>
            </t-popconfirm>
            <t-popconfirm
              v-permission="permissionCodes.STOP"
              :content="t('container.list.actions.confirmStop')"
              :confirm-btn="t('container.list.actions.confirm')"
              :cancel-btn="t('container.list.actions.cancel')"
              theme="danger"
              @confirm="runAction('stop', row)"
            >
              <t-button
                theme="danger"
                variant="text"
                size="small"
                :loading="actionLoadingKey === actionKey('stop', row)"
              >
                {{ t('container.list.actions.stop') }}
              </t-button>
            </t-popconfirm>
            <t-popconfirm
              v-permission="permissionCodes.RESTART"
              :content="t('container.list.actions.confirmRestart')"
              :confirm-btn="t('container.list.actions.confirm')"
              :cancel-btn="t('container.list.actions.cancel')"
              theme="warning"
              @confirm="runAction('restart', row)"
            >
              <t-button
                theme="warning"
                variant="text"
                size="small"
                :loading="actionLoadingKey === actionKey('restart', row)"
              >
                {{ t('container.list.actions.restart') }}
              </t-button>
            </t-popconfirm>
          </t-space>
        </template>

        <template #empty>
          <t-empty
            :title="t('container.list.emptyTitle')"
            :description="
              hasActiveFilters ? t('container.list.emptyFilteredDescription') : t('container.list.emptyDescription')
            "
          >
            <template v-if="hasActiveFilters" #action>
              <t-button theme="primary" variant="outline" @click="resetFilters">
                {{ t('container.list.clearFilters') }}
              </t-button>
            </template>
          </t-empty>
        </template>
      </t-table>
    </management-table-card>

    <t-drawer
      v-model:visible="detailDrawerVisible"
      :header="t('container.list.detail.title')"
      :footer="false"
      size="560px"
    >
      <div class="container-drawer-panel">
        <t-alert v-if="detailError" theme="error" :title="detailError">
          <template #operation>
            <t-button v-if="selectedContainer" theme="danger" variant="text" @click="loadDetail(selectedContainer.id)">
              {{ t('container.list.retry') }}
            </t-button>
          </template>
        </t-alert>
        <t-loading :loading="detailLoading">
          <section v-if="activeDetail" class="container-detail-stack">
            <t-descriptions :title="t('container.list.detail.identity')" :column="1" bordered>
              <t-descriptions-item :label="t('container.list.fields.name')">{{
                displayName(activeDetail)
              }}</t-descriptions-item>
              <t-descriptions-item :label="t('container.list.fields.id')">{{ activeDetail.id }}</t-descriptions-item>
              <t-descriptions-item :label="t('container.list.fields.image')">{{
                activeDetail.image
              }}</t-descriptions-item>
              <t-descriptions-item :label="t('container.list.fields.state')">
                {{ stateLabel(activeDetail.state) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('container.list.fields.status')">{{
                activeDetail.status
              }}</t-descriptions-item>
              <t-descriptions-item :label="t('container.list.fields.restartPolicy')">
                {{ activeDetail.restart_policy || '-' }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('container.list.fields.createdAt')">
                {{ formatTime(activeDetail.created_at) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('container.list.fields.startedAt')">
                {{ formatTime(activeDetail.started_at) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('container.list.detail.command')">
                {{ activeDetail.command || '-' }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('container.list.detail.entrypoint')">
                {{ joinList(activeDetail.entrypoint) }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('container.list.detail.workingDir')">
                {{ activeDetail.working_dir || '-' }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('container.list.detail.inspectUpdatedAt')">
                {{ formatTime(activeDetail.inspect_updated_at) }}
              </t-descriptions-item>
            </t-descriptions>

            <t-descriptions :title="t('container.list.detail.runtime')" :column="1" bordered>
              <t-descriptions-item :label="t('container.list.fields.runtime')">
                {{ activeDetail.runtime_info.runtime }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('container.list.fields.endpoint')">
                {{ activeDetail.runtime_info.endpoint || '-' }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('container.list.fields.apiVersion')">
                {{ activeDetail.runtime_info.api_version || '-' }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('container.list.fields.serverVersion')">
                {{ activeDetail.runtime_info.server_version || '-' }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('container.list.fields.operatingSystem')">
                {{ activeDetail.runtime_info.operating_system || '-' }}
              </t-descriptions-item>
              <t-descriptions-item :label="t('container.list.fields.architecture')">
                {{ activeDetail.runtime_info.architecture || '-' }}
              </t-descriptions-item>
            </t-descriptions>

            <section class="container-detail-section">
              <h3>{{ t('container.list.detail.mounts') }}</h3>
              <div v-if="activeDetail.mounts.length" class="container-detail-list">
                <div
                  v-for="mount in activeDetail.mounts"
                  :key="`${mount.type}:${mount.destination}`"
                  class="container-detail-item"
                >
                  <strong>{{ mount.destination }}</strong>
                  <span>{{ mount.type }} / {{ mount.mode || '-' }} / {{ mount.read_only ? 'ro' : 'rw' }}</span>
                  <span>{{ mount.source || mount.name || '-' }}</span>
                </div>
              </div>
              <t-empty v-else size="small" :description="t('container.list.detail.mountEmpty')" />
            </section>

            <section class="container-detail-section">
              <h3>{{ t('container.list.detail.networks') }}</h3>
              <div v-if="activeDetail.networks.length" class="container-detail-list">
                <div v-for="network in activeDetail.networks" :key="network.name" class="container-detail-item">
                  <strong>{{ network.name }}</strong>
                  <span>{{ network.ip_address || '-' }}</span>
                  <span>{{ network.gateway || network.mac_address || '-' }}</span>
                </div>
              </div>
              <t-empty v-else size="small" :description="t('container.list.detail.networkEmpty')" />
            </section>
          </section>
        </t-loading>
      </div>
    </t-drawer>

    <t-drawer v-model:visible="logsDrawerVisible" :header="logsDrawerTitle" :footer="false" size="720px">
      <div class="container-drawer-panel container-logs-panel">
        <t-form class="container-log-controls" layout="inline" label-align="top" :data="logQuery">
          <t-form-item :label="t('container.list.logs.tail')" name="tail">
            <t-input-number v-model="logQuery.tail" theme="normal" :min="1" :max="2000" :step="100" />
          </t-form-item>
          <t-form-item :label="t('container.list.logs.since')" name="since">
            <t-input v-model="logQuery.since" clearable :placeholder="t('container.list.logs.sincePlaceholder')" />
          </t-form-item>
          <t-form-item name="streams">
            <t-space break-line size="small">
              <t-checkbox v-model="logQuery.timestamps">{{ t('container.list.logs.timestamps') }}</t-checkbox>
              <t-checkbox v-model="logQuery.stdout">{{ t('container.list.logs.stdout') }}</t-checkbox>
              <t-checkbox v-model="logQuery.stderr">{{ t('container.list.logs.stderr') }}</t-checkbox>
            </t-space>
          </t-form-item>
          <t-form-item>
            <t-space size="small">
              <t-button theme="primary" :loading="logsLoading" @click="refreshLogs">
                {{ t('container.list.logs.refresh') }}
              </t-button>
              <t-button theme="default" variant="outline" :disabled="!activeLogs?.lines.length" @click="copyLogs">
                {{ t('container.list.logs.copy') }}
              </t-button>
            </t-space>
          </t-form-item>
        </t-form>

        <t-alert v-if="logsError" class="container-alert" theme="error" :title="logsError" />
        <t-alert
          v-if="activeLogs?.truncated"
          class="container-alert"
          theme="warning"
          :title="t('container.list.logs.truncated')"
        />

        <t-loading :loading="logsLoading">
          <pre v-if="activeLogs?.lines.length" class="container-log-output">{{ activeLogs.lines.join('\n') }}</pre>
          <t-empty v-else size="small" :description="t('container.list.logs.empty')" />
        </t-loading>
      </div>
    </t-drawer>
  </div>
</template>
<script setup lang="ts">
import { RefreshIcon, SearchIcon } from 'tdesign-icons-vue-next';
import type { TdBaseTableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  ManagementPageHeader,
  ManagementTableCard,
  ManagementToolbar,
  TableViewToolbar,
} from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';
import { createLogger } from '@/utils/logger';

import { getContainer, getContainerLogs, getContainers, runContainerAction } from '../../api/container';
import { CONTAINER_PERMISSION_CODE } from '../../contract/permissions';
import type {
  ContainerAction,
  ContainerDetail,
  ContainerFilters,
  ContainerLogQuery,
  ContainerLogResponse,
  ContainerPort,
  ContainerRuntimeInfo,
  ContainerState,
  ContainerSummary,
} from '../../types/container';

defineOptions({
  name: 'ContainerListIndex',
});

const { locale, t } = useI18n();
const logger = createLogger('container.list');
const permissionCodes = CONTAINER_PERMISSION_CODE;

const statusOptions: ContainerState[] = [
  'created',
  'running',
  'paused',
  'restarting',
  'removing',
  'exited',
  'dead',
  'unknown',
];
const DEFAULT_LOG_QUERY: Required<ContainerLogQuery> = {
  tail: 200,
  since: '',
  timestamps: false,
  stdout: true,
  stderr: true,
};

const loading = ref(false);
const listError = ref('');
const rows = ref<ContainerSummary[]>([]);
const runtime = ref<ContainerRuntimeInfo | null>(null);
const detailDrawerVisible = ref(false);
const detailLoading = ref(false);
const detailError = ref('');
const selectedContainer = ref<ContainerSummary | null>(null);
const activeDetail = ref<ContainerDetail | null>(null);
const logsDrawerVisible = ref(false);
const logsLoading = ref(false);
const logsError = ref('');
const activeLogs = ref<ContainerLogResponse | null>(null);
const actionLoadingKey = ref('');
const filters = reactive<ContainerFilters>({
  keyword: '',
  status: 'all',
});
const logQuery = reactive<Required<ContainerLogQuery>>({ ...DEFAULT_LOG_QUERY });

const columns = computed<TdBaseTableProps['columns']>(() => [
  { title: t('container.list.columns.status'), colKey: 'state', width: 108, align: 'center', ellipsis: false },
  {
    title: t('container.list.columns.name'),
    colKey: 'name',
    minWidth: 260,
    ellipsis: { theme: 'default', placement: 'top-left' },
  },
  {
    title: t('container.list.columns.image'),
    colKey: 'image',
    minWidth: 240,
    ellipsis: { theme: 'default', placement: 'top-left' },
  },
  { title: t('container.list.columns.ports'), colKey: 'ports', width: 220, ellipsis: false },
  { title: t('container.list.columns.createdAt'), colKey: 'created_at', width: 168, align: 'center' },
  { title: t('container.list.columns.startedAt'), colKey: 'started_at', width: 168, align: 'center' },
  { title: t('container.list.columns.restartPolicy'), colKey: 'restart_policy', width: 140, align: 'center' },
  {
    title: t('container.list.columns.operation'),
    colKey: 'operation',
    width: 360,
    fixed: 'right',
    align: 'center',
    ellipsis: false,
  },
]);
const tableContentWidth = computed(() => {
  const totalWidth = (columns.value ?? []).reduce((sum, column) => {
    if (typeof column?.width === 'number') return sum + column.width;
    if (typeof column?.minWidth === 'number') return sum + column.minWidth;
    return sum + 160;
  }, 0);
  return `max(100%, ${totalWidth}px)`;
});
const filteredRows = computed(() => {
  const keyword = filters.keyword.trim().toLowerCase();
  return rows.value.filter((row) => {
    const matchesStatus = filters.status === 'all' || row.state === filters.status;
    if (!matchesStatus) return false;
    if (!keyword) return true;
    return [row.id, row.image, row.status, row.restart_policy, ...row.names, ...formatPorts(row.ports)].some(
      (value) => value?.toLowerCase().includes(keyword) ?? false,
    );
  });
});
const hasActiveFilters = computed(() => Boolean(filters.keyword.trim()) || filters.status !== 'all');
const runtimeStatusTheme = computed(() => {
  if (runtime.value?.status === 'enabled') return 'success';
  if (runtime.value?.status === 'disabled') return 'warning';
  return 'danger';
});
const runtimeSummary = computed(() => {
  if (!runtime.value) return t('container.list.runtimeUnavailable');
  const version = runtime.value.server_version || runtime.value.api_version || '';
  return version ? `${runtime.value.runtime} / ${version}` : runtime.value.runtime;
});
const runtimeCountParams = computed(() => ({
  running: runtime.value?.containers_running ?? 0,
  total: runtime.value?.containers_total ?? rows.value.length,
}));
const logsDrawerTitle = computed(() => {
  const containerName = selectedContainer.value ? displayName(selectedContainer.value) : '';
  return containerName ? `${t('container.list.logs.title')} - ${containerName}` : t('container.list.logs.title');
});

onMounted(() => {
  void refreshContainers();
});

async function refreshContainers() {
  loading.value = true;
  listError.value = '';
  try {
    const payload = await getContainers();
    rows.value = payload.items;
    runtime.value = payload.runtime;
  } catch (error) {
    rows.value = [];
    runtime.value = null;
    listError.value = resolveLocalizedErrorMessage(t, error, t('container.list.loadFailed'));
    logger.error('failed to fetch containers', error);
  } finally {
    loading.value = false;
  }
}

function applyFilters() {
  filters.keyword = filters.keyword.trim();
}

function resetFilters() {
  filters.keyword = '';
  filters.status = 'all';
}

async function openDetail(row: ContainerSummary) {
  selectedContainer.value = row;
  activeDetail.value = null;
  detailDrawerVisible.value = true;
  await loadDetail(row.id);
}

async function loadDetail(containerId: string) {
  detailLoading.value = true;
  detailError.value = '';
  try {
    activeDetail.value = await getContainer(containerId);
  } catch (error) {
    detailError.value = resolveLocalizedErrorMessage(t, error, t('container.list.detail.loadFailed'));
    logger.warn('failed to fetch container detail', error);
  } finally {
    detailLoading.value = false;
  }
}

async function openLogs(row: ContainerSummary) {
  selectedContainer.value = row;
  activeLogs.value = null;
  logsError.value = '';
  Object.assign(logQuery, DEFAULT_LOG_QUERY);
  logsDrawerVisible.value = true;
  await refreshLogs();
}

async function refreshLogs() {
  if (!selectedContainer.value) return;
  logsLoading.value = true;
  logsError.value = '';
  try {
    activeLogs.value = await getContainerLogs(selectedContainer.value.id, normalizeLogQuery());
  } catch (error) {
    logsError.value = resolveLocalizedErrorMessage(t, error, t('container.list.logs.loadFailed'));
    logger.warn('failed to fetch container logs', error);
  } finally {
    logsLoading.value = false;
  }
}

function normalizeLogQuery(): ContainerLogQuery {
  return {
    tail: logQuery.tail,
    since: logQuery.since.trim() || undefined,
    timestamps: logQuery.timestamps,
    stdout: logQuery.stdout,
    stderr: logQuery.stderr,
  };
}

async function copyLogs() {
  const text = activeLogs.value?.lines.join('\n') ?? '';
  if (!text) return;
  try {
    await navigator.clipboard.writeText(text);
    MessagePlugin.success(t('container.list.copySuccess'));
  } catch (error) {
    logger.warn('failed to copy container logs', error);
    MessagePlugin.error(t('container.list.copyError'));
  }
}

async function runAction(action: ContainerAction, row: ContainerSummary) {
  const key = actionKey(action, row);
  actionLoadingKey.value = key;
  try {
    const result = await runContainerAction(action, row.id);
    MessagePlugin.success(localizedActionMessage(result.message_key, result.message));
    await refreshContainers();
  } catch (error) {
    logger.warn('failed to run container action', error);
    MessagePlugin.error(resolveLocalizedErrorMessage(t, error, t('container.list.actionFailed')));
  } finally {
    if (actionLoadingKey.value === key) {
      actionLoadingKey.value = '';
    }
  }
}

function localizedActionMessage(messageKey?: string, fallback?: string) {
  if (messageKey) {
    const translated = t(messageKey);
    if (translated !== messageKey) {
      return translated;
    }
  }
  return fallback?.trim() || t('container.list.actionSuccess');
}

function actionKey(action: ContainerAction, row: ContainerSummary) {
  return `${action}:${row.id}`;
}

function displayName(row: ContainerSummary | ContainerDetail) {
  return row.names[0] || row.id;
}

function formatPorts(ports: ContainerPort[]) {
  return ports.map((port) => {
    const target = `${port.private_port}/${port.type}`;
    if (port.public_port === undefined) {
      return target;
    }
    return `${port.ip ? `${port.ip}:` : ''}${port.public_port}->${target}`;
  });
}

function formatTime(value?: string | null) {
  return formatLocaleDateTime(value, locale);
}

function joinList(values?: string[]) {
  return values?.length ? values.join(' ') : '-';
}

function stateLabel(state: ContainerState) {
  return t(`container.list.states.${state}`);
}

function stateTheme(state: ContainerState) {
  if (state === 'running') return 'success';
  if (state === 'created' || state === 'paused' || state === 'restarting') return 'warning';
  if (state === 'dead') return 'danger';
  return 'default';
}
</script>
<style scoped lang="less">
.container-page {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.container-table-head,
.container-image,
.container-identity,
.container-detail-item {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.container-table-head p,
.container-detail-section h3 {
  margin: 0;
}

.container-table-head__summary,
.container-identity__name,
.container-detail-item strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.container-table-head p:not(.container-table-head__summary),
.container-identity__id,
.container-muted,
.container-detail-item span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.container-identity__id {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-port-list,
.container-actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6);
}

.container-alert {
  margin-bottom: var(--graft-density-gap-12);
}

.container-drawer-panel,
.container-detail-stack,
.container-logs-panel,
.container-detail-list {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
}

.container-detail-section {
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-12);
  padding: var(--graft-density-gap-14);
}

.container-detail-list {
  gap: var(--graft-density-gap-10);
}

.container-detail-item {
  border-bottom: 1px solid var(--td-component-stroke);
  padding-bottom: var(--graft-density-gap-10);
}

.container-detail-item:last-child {
  border-bottom: 0;
  padding-bottom: 0;
}

.container-log-controls {
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  padding: var(--graft-density-gap-14);
}

.container-log-output {
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  color: var(--td-text-color-primary);
  font-family: var(--td-font-family-monospace);
  line-height: var(--td-line-height-body-medium);
  margin: 0;
  max-height: min(60vh, 640px);
  overflow: auto;
  overflow-wrap: anywhere;
  padding: var(--graft-density-gap-14);
  white-space: pre-wrap;
}

@media (width <= 768px) {
  .container-actions {
    justify-content: flex-start;
  }
}
</style>
