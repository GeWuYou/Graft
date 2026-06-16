<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <server-status-page-shell
    :eyebrow="t('monitor.sectionTitle')"
    title-key="monitor.moduleRuntime.title"
    description-key="monitor.moduleRuntime.subtitle"
    compact-header
  >
    <template #toolbar>
      <div class="module-runtime-toolbar">
        <status-tag :label="headerStatusLabel" :status="headerStatus" />
      </div>
    </template>

    <template #summary>
      <div v-for="metric in summaryMetrics" :key="metric.key" class="module-runtime-summary-card">
        <t-statistic :title="metric.label" :value="metric.value" :loading="loading" />
        <p>{{ metric.description }}</p>
      </div>
    </template>

    <template #feedback>
      <t-alert
        v-if="errorMessage"
        theme="error"
        :title="t('monitor.moduleRuntime.errorTitle')"
        :message="errorMessage"
      />
    </template>

    <management-table-card
      :title="t('monitor.moduleRuntime.table.title')"
      :description="t('monitor.moduleRuntime.table.description')"
    >
      <template #toolbar>
        <table-view-toolbar
          :column-settings-label="t('monitor.moduleRuntime.table.columnSettings')"
          :density-label="tableDensityLabel"
          :refresh-label="t('monitor.moduleRuntime.actions.refresh')"
          :refresh-loading="loading"
          @column-settings="columnDrawerVisible = true"
          @density="toggleTableDensity"
          @refresh="refreshSnapshot"
        />
      </template>

      <p class="module-runtime-table__note">{{ t('monitor.moduleRuntime.table.note') }}</p>

      <div ref="tableHostRef" class="module-runtime-table-host" :data-table-mode="tableWidthPolicy.mode">
        <t-table
          row-key="module_key"
          hover
          :data="items"
          :columns="visibleColumns"
          :loading="loading"
          :empty="emptyTableContent"
          :size="tableDensity"
          :table-content-width="tableWidthPolicy.tableContentWidth"
          table-layout="fixed"
          cell-empty-content="-"
        >
          <template #module_key="{ row }">
            <div class="module-runtime-table__identity">
              <strong class="module-runtime-table__key">{{ row.module_key }}</strong>
              <span>{{ runtimeStatusLabel(row.runtime_status) }}</span>
            </div>
          </template>

          <template #enabled="{ row }">
            <status-tag :label="booleanLabel(row.enabled)" :status="row.enabled ? 'healthy' : 'disabled'" />
          </template>

          <template #registered="{ row }">
            <status-tag :label="booleanLabel(row.registered)" :status="row.registered ? 'healthy' : 'unknown'" />
          </template>

          <template #health="{ row }">
            <status-tag :label="healthLabel(row.health)" :status="healthTone(row.health)" />
          </template>

          <template #dependencies="{ row }">
            <t-popup placement="top-left" :disabled="!row.dependencies.length" show-arrow>
              <span class="module-runtime-table__dependency-summary">
                {{ dependencySummary(row.dependencies) }}
              </span>
              <template #content>
                <div class="module-runtime-table__dependency-popover">
                  <div
                    v-for="dependency in row.dependencies"
                    :key="dependency.module_key"
                    class="module-runtime-table__dependency-line"
                  >
                    <strong>{{ dependency.module_key }}</strong>
                    <status-tag
                      :label="dependencyStatusLabel(dependency.status)"
                      :status="dependencyTone(dependency.status)"
                    />
                  </div>
                </div>
              </template>
            </t-popup>
          </template>

          <template #resource_status="{ row }">
            <div class="module-runtime-table__resource-status">
              <span class="module-runtime-table__resource-status-item">
                <span>{{ t('monitor.moduleRuntime.columns.migration') }}</span>
                <status-tag
                  :label="migrationStatusLabel(row.migration_status.status)"
                  :status="declaredTone(row.migration_status.status)"
                />
              </span>
              <span class="module-runtime-table__resource-status-item">
                <span>{{ t('monitor.moduleRuntime.columns.schema') }}</span>
                <status-tag
                  :label="schemaStatusLabel(row.schema_status.status)"
                  :status="declaredTone(row.schema_status.status)"
                />
              </span>
              <span class="module-runtime-table__resource-status-item">
                <span>{{ t('monitor.moduleRuntime.columns.config') }}</span>
                <status-tag
                  :label="configStatusLabel(row.config_status.status)"
                  :status="configTone(row.config_status.status)"
                />
              </span>
            </div>
          </template>

          <template #operation="{ row }">
            <t-button variant="text" theme="primary" size="small" @click="openDetail(row)">
              {{ t('monitor.moduleRuntime.actions.detail') }}
            </t-button>
          </template>
        </t-table>
      </div>
    </management-table-card>

    <advanced-query-column-drawer
      v-model:visible="columnDrawerVisible"
      v-model:selected-keys="visibleColumnKeys"
      :columns="columnSettingOptions"
      :default-selected-keys="DEFAULT_VISIBLE_COLUMNS"
      :disabled-keys="ALWAYS_VISIBLE_COLUMNS"
      :reset-label="t('monitor.moduleRuntime.table.resetColumns')"
      :title="t('monitor.moduleRuntime.table.columnSettings')"
    />

    <t-drawer
      v-model:visible="detailVisible"
      :header="detailHeader"
      :footer="false"
      size="520px"
      attach="body"
      destroy-on-close
    >
      <div v-if="selectedModule" class="module-runtime-detail">
        <section class="module-runtime-detail__section">
          <h3>{{ t('monitor.moduleRuntime.detail.basicInfo') }}</h3>
          <div class="module-runtime-detail__grid">
            <div class="module-runtime-detail__field module-runtime-detail__field--wide">
              <span>{{ t('monitor.moduleRuntime.detail.moduleKey') }}</span>
              <strong>{{ selectedModule.module_key }}</strong>
            </div>
            <div class="module-runtime-detail__field">
              <span>{{ t('monitor.moduleRuntime.detail.enabled') }}</span>
              <status-tag
                :label="booleanLabel(selectedModule.enabled)"
                :status="selectedModule.enabled ? 'healthy' : 'disabled'"
              />
            </div>
            <div class="module-runtime-detail__field">
              <span>{{ t('monitor.moduleRuntime.detail.registered') }}</span>
              <status-tag
                :label="booleanLabel(selectedModule.registered)"
                :status="selectedModule.registered ? 'healthy' : 'unknown'"
              />
            </div>
            <div class="module-runtime-detail__field">
              <span>{{ t('monitor.moduleRuntime.detail.health') }}</span>
              <status-tag :label="healthLabel(selectedModule.health)" :status="healthTone(selectedModule.health)" />
            </div>
            <div class="module-runtime-detail__field">
              <span>{{ t('monitor.moduleRuntime.detail.runtimeStatus') }}</span>
              <status-tag
                :label="runtimeStatusLabel(selectedModule.runtime_status)"
                :status="runtimeTone(selectedModule.runtime_status)"
              />
            </div>
            <div class="module-runtime-detail__field module-runtime-detail__field--wide">
              <span>{{ t('monitor.moduleRuntime.detail.enablementSource') }}</span>
              <strong>{{ enablementSourceLabel(selectedModule.enablement_source) }}</strong>
            </div>
          </div>
        </section>

        <section class="module-runtime-detail__section">
          <h3>{{ t('monitor.moduleRuntime.detail.dependencies') }}</h3>
          <div class="module-runtime-detail__field module-runtime-detail__field--wide">
            <span>{{ t('monitor.moduleRuntime.detail.dependencySatisfaction') }}</span>
            <strong>{{ dependencySummary(selectedModule.dependencies) }}</strong>
          </div>
          <div class="module-runtime-detail__subhead">{{ t('monitor.moduleRuntime.detail.declaredDependencies') }}</div>
          <div v-if="selectedModule.dependencies.length" class="module-runtime-detail__list">
            <div
              v-for="dependency in selectedModule.dependencies"
              :key="dependency.module_key"
              class="module-runtime-detail__line"
            >
              <strong>{{ dependency.module_key }}</strong>
              <status-tag
                :label="dependencyStatusLabel(dependency.status)"
                :status="dependencyTone(dependency.status)"
              />
            </div>
          </div>
          <div v-else class="module-runtime-detail__empty">
            {{ t('monitor.moduleRuntime.values.emptyDependencies') }}
          </div>
        </section>

        <section class="module-runtime-detail__section">
          <h3>{{ t('monitor.moduleRuntime.detail.migration') }}</h3>
          <div class="module-runtime-detail__grid">
            <div class="module-runtime-detail__field">
              <span>{{ t('monitor.moduleRuntime.detail.migrationStatus') }}</span>
              <status-tag
                :label="migrationStatusLabel(selectedModule.migration_status.status)"
                :status="declaredTone(selectedModule.migration_status.status)"
              />
            </div>
            <div class="module-runtime-detail__field">
              <span>{{ t('monitor.moduleRuntime.detail.migrationDir') }}</span>
              <strong>{{
                t('monitor.moduleRuntime.values.migrationDirCount', {
                  count: selectedModule.migration_status.declared_dirs.length,
                })
              }}</strong>
            </div>
          </div>
          <div v-if="selectedModule.migration_status.declared_dirs.length" class="module-runtime-detail__paths">
            <code v-for="directory in selectedModule.migration_status.declared_dirs" :key="directory">
              {{ directory }}
            </code>
          </div>
          <div v-else class="module-runtime-detail__empty">
            {{ t('monitor.moduleRuntime.values.emptyMigrationDir') }}
          </div>
        </section>

        <section class="module-runtime-detail__section">
          <h3>{{ t('monitor.moduleRuntime.detail.schema') }}</h3>
          <div class="module-runtime-detail__grid">
            <div class="module-runtime-detail__field">
              <span>{{ t('monitor.moduleRuntime.detail.schemaOwner') }}</span>
              <strong>{{ schemaOwnerLabel(selectedModule.schema_status.status) }}</strong>
            </div>
            <div class="module-runtime-detail__field">
              <span>{{ t('monitor.moduleRuntime.detail.schemaStatus') }}</span>
              <status-tag
                :label="schemaStatusLabel(selectedModule.schema_status.status)"
                :status="declaredTone(selectedModule.schema_status.status)"
              />
            </div>
          </div>
        </section>

        <section class="module-runtime-detail__section">
          <h3>{{ t('monitor.moduleRuntime.detail.config') }}</h3>
          <div class="module-runtime-detail__grid">
            <div class="module-runtime-detail__field">
              <span>{{ t('monitor.moduleRuntime.detail.configStatus') }}</span>
              <status-tag
                :label="configStatusLabel(selectedModule.config_status.status)"
                :status="configTone(selectedModule.config_status.status)"
              />
            </div>
            <div class="module-runtime-detail__field module-runtime-detail__field--wide">
              <span>{{ t('monitor.moduleRuntime.detail.configDescription') }}</span>
              <strong>{{ configDescriptionLabel(selectedModule.config_status.status) }}</strong>
            </div>
          </div>
        </section>

        <section class="module-runtime-detail__section">
          <h3>{{ t('monitor.moduleRuntime.detail.diagnostics') }}</h3>
          <div v-if="diagnosticEntries.length" class="module-runtime-detail__diagnostics">
            <div v-for="[key, value] in diagnosticEntries" :key="key" class="module-runtime-detail__diagnostic">
              <span>{{ key }}</span>
              <strong>{{ value }}</strong>
            </div>
          </div>
          <div v-else class="module-runtime-detail__empty">{{ t('monitor.moduleRuntime.values.noDiagnostics') }}</div>
        </section>

        <section class="module-runtime-detail__section">
          <t-collapse borderless>
            <t-collapse-panel value="runtime-json" :header="t('monitor.moduleRuntime.detail.rawJson')">
              <pre class="module-runtime-detail__json">{{ selectedModuleRuntimeJson }}</pre>
            </t-collapse-panel>
          </t-collapse>
        </section>
      </div>
    </t-drawer>
  </server-status-page-shell>
</template>
<script setup lang="ts">
import type { TdBaseTableProps } from 'tdesign-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  buildVisibleColumns,
  ManagementTableCard,
  resolveTableWidthPolicy,
  TableViewToolbar,
  useTableHostWidth,
} from '@/shared/components/management';
import { AdvancedQueryColumnDrawer } from '@/shared/components/query-list';
import { createLogger } from '@/utils/logger';

import { getModuleRuntimeDetail, getModuleRuntimeSnapshot } from '../../api/module-runtime';
import type { ServerStatusTone } from '../../components/server-status-ui';
import ServerStatusPageShell from '../../components/ServerStatusPageShell.vue';
import StatusTag from '../../components/StatusTag.vue';
import type {
  ModuleRuntimeConfigStatus,
  ModuleRuntimeDependency,
  ModuleRuntimeItem,
  ModuleRuntimeMigrationStatus,
  ModuleRuntimeSchemaStatus,
  ModuleRuntimeSnapshot,
} from '../../types/module-runtime';

const { t } = useI18n();
const moduleRuntimeLogger = createLogger('monitor.module-runtime.page');
const DEFAULT_VISIBLE_COLUMNS = [
  'module_key',
  'enabled',
  'registered',
  'health',
  'dependencies',
  'resource_status',
  'operation',
];
const ALWAYS_VISIBLE_COLUMNS = ['module_key', 'health', 'operation'];

const snapshot = ref<ModuleRuntimeSnapshot | null>(null);
const loading = ref(false);
const initialized = ref(false);
const errorMessage = ref('');
const detailVisible = ref(false);
const columnDrawerVisible = ref(false);
const visibleColumnKeys = ref<string[]>([...DEFAULT_VISIBLE_COLUMNS]);
const tableDensity = ref<'medium' | 'small'>('medium');
const selectedModule = ref<ModuleRuntimeItem | null>(null);

const items = computed(() => snapshot.value?.items ?? []);
const summary = computed(() => snapshot.value?.summary);

const summaryMetrics = computed(() => [
  {
    key: 'total',
    label: t('monitor.moduleRuntime.summary.total'),
    value: summary.value?.total_modules ?? 0,
    description: t('monitor.moduleRuntime.summary.totalDescription'),
  },
  {
    key: 'enabled',
    label: t('monitor.moduleRuntime.summary.enabled'),
    value: summary.value?.enabled_modules ?? 0,
    description: t('monitor.moduleRuntime.summary.enabledDescription'),
  },
  {
    key: 'healthy',
    label: t('monitor.moduleRuntime.summary.healthy'),
    value: summary.value?.healthy_modules ?? 0,
    description: t('monitor.moduleRuntime.summary.healthyDescription'),
  },
  {
    key: 'degradedUnknown',
    label: t('monitor.moduleRuntime.summary.degradedUnknown'),
    value: (summary.value?.degraded_modules ?? 0) + (summary.value?.unknown_modules ?? 0),
    description: t('monitor.moduleRuntime.summary.degradedUnknownDescription'),
  },
]);

const headerStatus = computed<ServerStatusTone>(() => {
  const currentSummary = summary.value;
  if (!currentSummary) {
    return 'unknown';
  }

  if (currentSummary.degraded_modules > 0) {
    return 'warning';
  }

  if (currentSummary.unknown_modules > 0) {
    return 'unknown';
  }

  return 'healthy';
});

const headerStatusLabel = computed(() => {
  switch (headerStatus.value) {
    case 'healthy':
      return t('monitor.moduleRuntime.status.ready');
    case 'warning':
      return t('monitor.moduleRuntime.status.attention');
    default:
      return t('monitor.moduleRuntime.status.unknown');
  }
});

const emptyTableContent = computed(() =>
  initialized.value && !loading.value && !items.value.length && !errorMessage.value
    ? t('monitor.moduleRuntime.empty')
    : '',
);

const detailHeader = computed(() =>
  selectedModule.value
    ? t('monitor.moduleRuntime.detail.titleWithKey', { key: selectedModule.value.module_key })
    : t('monitor.moduleRuntime.detail.title'),
);

const diagnosticEntries = computed(() => Object.entries(selectedModule.value?.diagnostics ?? {}));
const selectedModuleRuntimeJson = computed(() => JSON.stringify(selectedModule.value, null, 2));

const tableDensityLabel = computed(() =>
  tableDensity.value === 'medium'
    ? t('monitor.moduleRuntime.table.compactDensity')
    : t('monitor.moduleRuntime.table.defaultDensity'),
);

const columnSettingOptions = computed(() => [
  { label: t('monitor.moduleRuntime.columns.moduleKey'), value: 'module_key' },
  { label: t('monitor.moduleRuntime.columns.enabled'), value: 'enabled' },
  { label: t('monitor.moduleRuntime.columns.registered'), value: 'registered' },
  { label: t('monitor.moduleRuntime.columns.health'), value: 'health' },
  { label: t('monitor.moduleRuntime.columns.dependencies'), value: 'dependencies' },
  { label: t('monitor.moduleRuntime.columns.resourceStatus'), value: 'resource_status' },
  { label: t('monitor.moduleRuntime.columns.action'), value: 'operation' },
]);

const allColumns = computed<TdBaseTableProps['columns']>(() => [
  {
    colKey: 'module_key',
    title: t('monitor.moduleRuntime.columns.moduleKey'),
    minWidth: 180,
    ellipsis: { theme: 'default', placement: 'top-left' },
  },
  {
    colKey: 'enabled',
    title: t('monitor.moduleRuntime.columns.enabled'),
    width: 104,
    align: 'center',
    ellipsis: false,
  },
  {
    colKey: 'registered',
    title: t('monitor.moduleRuntime.columns.registered'),
    width: 104,
    align: 'center',
    ellipsis: false,
  },
  {
    colKey: 'health',
    title: t('monitor.moduleRuntime.columns.health'),
    width: 112,
    align: 'center',
    ellipsis: false,
  },
  {
    colKey: 'dependencies',
    title: t('monitor.moduleRuntime.columns.dependencies'),
    minWidth: 170,
    ellipsis: { theme: 'default', placement: 'top-left' },
  },
  {
    colKey: 'resource_status',
    title: t('monitor.moduleRuntime.columns.resourceStatus'),
    minWidth: 260,
    ellipsis: false,
  },
  {
    colKey: 'operation',
    title: t('monitor.moduleRuntime.columns.action'),
    width: 104,
    fixed: 'right',
    align: 'center',
    ellipsis: false,
  },
]);

const visibleColumns = computed<TdBaseTableProps['columns']>(() =>
  buildVisibleColumns(allColumns.value, visibleColumnKeys.value, ALWAYS_VISIBLE_COLUMNS),
);
const { tableHostRef, tableHostWidth } = useTableHostWidth(() => visibleColumns.value);
const tableWidthPolicy = computed(() => resolveTableWidthPolicy(visibleColumns.value, tableHostWidth.value));

onMounted(() => {
  void refreshSnapshot();
});

async function refreshSnapshot() {
  loading.value = true;
  errorMessage.value = '';

  try {
    snapshot.value = await getModuleRuntimeSnapshot();
  } catch (error) {
    moduleRuntimeLogger.error(error instanceof Error ? error : 'load module runtime snapshot failed', {
      operation: 'module_runtime_snapshot',
    });
    errorMessage.value = t('monitor.moduleRuntime.errorFallback');
  } finally {
    loading.value = false;
    initialized.value = true;
  }
}

async function openDetail(row: ModuleRuntimeItem) {
  errorMessage.value = '';

  try {
    selectedModule.value = await getModuleRuntimeDetail(row.module_key);
    detailVisible.value = true;
  } catch (error) {
    moduleRuntimeLogger.error(error instanceof Error ? error : 'load module runtime detail failed', {
      moduleKey: row.module_key,
      operation: 'module_runtime_detail',
    });
    errorMessage.value = t('monitor.moduleRuntime.errorFallback');
  }
}

function booleanLabel(value: boolean) {
  return value ? t('monitor.moduleRuntime.values.yes') : t('monitor.moduleRuntime.values.no');
}

function healthTone(status: ModuleRuntimeItem['health']): ServerStatusTone {
  switch (status) {
    case 'healthy':
      return 'healthy';
    case 'degraded':
      return 'warning';
    case 'disabled':
      return 'disabled';
    default:
      return 'unknown';
  }
}

function healthLabel(status: ModuleRuntimeItem['health']) {
  return t(`monitor.moduleRuntime.health.${status}`);
}

function runtimeTone(status: ModuleRuntimeItem['runtime_status']): ServerStatusTone {
  switch (status) {
    case 'registered':
      return 'healthy';
    case 'degraded':
      return 'warning';
    case 'disabled':
      return 'disabled';
    default:
      return 'unknown';
  }
}

function runtimeStatusLabel(status: ModuleRuntimeItem['runtime_status']) {
  return t(`monitor.moduleRuntime.runtimeStatus.${status}`);
}

function declaredTone(
  status: ModuleRuntimeMigrationStatus['status'] | ModuleRuntimeSchemaStatus['status'],
): ServerStatusTone {
  return status === 'declared' ? 'healthy' : 'unknown';
}

function migrationStatusLabel(status: ModuleRuntimeMigrationStatus['status']) {
  return t(`monitor.moduleRuntime.migrationStatus.${status}`);
}

function schemaStatusLabel(status: ModuleRuntimeSchemaStatus['status']) {
  return t(`monitor.moduleRuntime.schemaStatus.${status}`);
}

function configTone(status: ModuleRuntimeConfigStatus['status']): ServerStatusTone {
  return status === 'not_required' ? 'disabled' : 'unknown';
}

function configStatusLabel(status: ModuleRuntimeConfigStatus['status']) {
  return t(`monitor.moduleRuntime.configStatus.${status}`);
}

function dependencyTone(status: ModuleRuntimeDependency['status']): ServerStatusTone {
  switch (status) {
    case 'satisfied':
      return 'healthy';
    case 'disabled':
      return 'disabled';
    default:
      return 'warning';
  }
}

function dependencyStatusLabel(status: ModuleRuntimeDependency['status']) {
  return t(`monitor.moduleRuntime.dependencyStatus.${status}`);
}

function dependencySummary(dependencies: ModuleRuntimeDependency[]) {
  if (!dependencies.length) {
    return t('monitor.moduleRuntime.values.none');
  }

  const satisfied = dependencies.filter((dependency) => dependency.status === 'satisfied').length;
  return t('monitor.moduleRuntime.values.dependencySummary', {
    satisfied,
    total: dependencies.length,
  });
}

function enablementSourceLabel(source: ModuleRuntimeItem['enablement_source']) {
  return t(`monitor.moduleRuntime.enablementSource.${source}`);
}

function schemaOwnerLabel(status: ModuleRuntimeSchemaStatus['status']) {
  return status === 'declared'
    ? t('monitor.moduleRuntime.values.moduleOwnedSchema')
    : t('monitor.moduleRuntime.values.emptySchema');
}

function configDescriptionLabel(status: ModuleRuntimeConfigStatus['status']) {
  return status === 'not_required'
    ? t('monitor.moduleRuntime.values.notRequiredConfig')
    : t('monitor.moduleRuntime.values.unknownConfig');
}

function toggleTableDensity() {
  tableDensity.value = tableDensity.value === 'medium' ? 'small' : 'medium';
}
</script>
<style scoped lang="less">
.module-runtime-toolbar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-10);
  justify-content: flex-end;
}

.module-runtime-summary-card {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  min-height: 112px;
  padding: var(--graft-density-gap-16);
}

.module-runtime-summary-card p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-8) 0 0;
}

.module-runtime-table__note {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0 0 var(--graft-density-gap-12);
}

.module-runtime-table-host {
  max-width: 100%;
  min-width: 0;
  overflow-x: hidden;
  width: 100%;
}

.module-runtime-table-host[data-table-mode='scroll'] {
  overflow-x: auto;
}

.module-runtime-table-host :deep(.t-table__content) {
  min-width: 0;
}

.module-runtime-table-host :deep(.t-table__content table) {
  min-width: 100%;
}

.module-runtime-table__key {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  overflow-wrap: anywhere;
}

.module-runtime-table__identity {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.module-runtime-table__identity span,
.module-runtime-table__dependency-summary {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.module-runtime-table__dependency-summary {
  cursor: default;
}

.module-runtime-table__dependency-popover {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-8);
  max-width: 280px;
  min-width: 180px;
}

.module-runtime-table__dependency-line {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
}

.module-runtime-table__dependency-line strong {
  color: var(--td-text-color-primary);
  font-weight: 500;
  min-width: 0;
  overflow-wrap: anywhere;
}

.module-runtime-table__resource-status {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
}

.module-runtime-table__resource-status-item {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.module-runtime-table__resource-status-item > span:first-child {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  white-space: nowrap;
}

.module-runtime-empty {
  margin-top: var(--graft-density-gap-20);
}

.module-runtime-detail {
  padding-bottom: var(--graft-density-gap-12);
}

.module-runtime-detail__section {
  border-bottom: 1px solid var(--td-border-level-1-color);
  padding: var(--graft-density-gap-16) 0;
}

.module-runtime-detail__section:first-child {
  padding-top: 0;
}

.module-runtime-detail__section:last-child {
  border-bottom: 0;
  padding-bottom: 0;
}

.module-runtime-detail__section h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  font-weight: 600;
  margin: 0 0 var(--graft-density-gap-12);
}

.module-runtime-detail__grid {
  display: grid;
  gap: var(--graft-density-gap-10);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.module-runtime-detail__field {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-default);
  min-width: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

.module-runtime-detail__field--wide {
  grid-column: 1 / -1;
}

.module-runtime-detail__field span,
.module-runtime-detail__subhead {
  color: var(--td-text-color-secondary);
  display: block;
  font: var(--td-font-body-small);
}

.module-runtime-detail__field strong {
  color: var(--td-text-color-primary);
  display: block;
  font-weight: 500;
  margin-top: var(--graft-density-gap-6);
  min-width: 0;
  overflow-wrap: anywhere;
}

.module-runtime-detail__subhead {
  margin: var(--graft-density-gap-12) 0 var(--graft-density-gap-8);
}

.module-runtime-detail__list,
.module-runtime-detail__diagnostics,
.module-runtime-detail__paths {
  display: grid;
  gap: var(--graft-density-gap-10);
}

.module-runtime-detail__line,
.module-runtime-detail__diagnostic {
  align-items: center;
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-default);
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

.module-runtime-detail__line span,
.module-runtime-detail__line strong,
.module-runtime-detail__diagnostic span,
.module-runtime-detail__diagnostic strong {
  min-width: 0;
  overflow-wrap: anywhere;
}

.module-runtime-detail__line strong {
  color: var(--td-text-color-primary);
  font-weight: 500;
}

.module-runtime-detail__diagnostic span {
  color: var(--td-text-color-secondary);
}

.module-runtime-detail__diagnostic strong {
  color: var(--td-text-color-primary);
  font-weight: 500;
  text-align: right;
}

.module-runtime-detail__paths code,
.module-runtime-detail__json {
  background: var(--td-bg-color-container-hover);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  font-family: var(--td-font-family-monospace, ui-monospace, SFMono-Regular, Menlo, Consolas, monospace);
}

.module-runtime-detail__paths code {
  display: block;
  overflow-wrap: anywhere;
  padding: var(--graft-density-gap-8) var(--graft-density-gap-10);
  white-space: normal;
}

.module-runtime-detail__json {
  margin: 0;
  max-height: 320px;
  overflow: auto;
  padding: var(--graft-density-gap-12);
  white-space: pre-wrap;
}

.module-runtime-detail__empty {
  background: var(--td-bg-color-container-hover);
  border: 1px dashed var(--td-border-level-2-color);
  border-radius: var(--td-radius-default);
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
}

@media (width <= 767px) {
  .module-runtime-toolbar {
    justify-content: flex-start;
  }

  .module-runtime-table__note {
    margin-bottom: var(--graft-density-gap-10);
  }

  .module-runtime-summary-card {
    min-height: 96px;
  }

  .module-runtime-detail__grid {
    grid-template-columns: 1fr;
  }
}
</style>
