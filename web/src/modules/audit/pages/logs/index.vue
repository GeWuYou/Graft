<template>
  <div class="audit-page" data-page-type="log-audit">
    <management-page-content>
      <management-page-header :title="t('audit.logList.listTitle')" :description="t('audit.logList.hint')">
        <template #eyebrow>{{ t('menu.audit.logs.title') }}</template>
        <template #actions>
          <t-space size="small" wrap>
            <t-button
              v-for="preset in presetViews"
              :key="preset.key"
              v-permission="AUDIT_PERMISSION_CODE.READ"
              size="small"
              :theme="activePreset === preset.key ? 'primary' : 'default'"
              :variant="activePreset === preset.key ? 'base' : 'outline'"
              @click="applyPreset(preset.key)"
            >
              {{ preset.title }}
            </t-button>
            <t-button
              v-permission="AUDIT_PERMISSION_CODE.READ"
              theme="default"
              variant="outline"
              :loading="loading"
              @click="fetchAuditLogs"
            >
              {{ t('audit.logList.refresh') }}
            </t-button>
          </t-space>
        </template>
      </management-page-header>

      <div class="audit-correlation-toolbar">
        <div class="audit-correlation-toolbar__copy">
          <strong>{{ t('audit.logList.correlationTitle') }}</strong>
          <p>{{ t('audit.logList.correlationSubtitle') }}</p>
        </div>
        <div class="audit-correlation-toolbar__filters">
          <t-input
            v-model="filters.request_id"
            clearable
            class="toolbar__search"
            :placeholder="t('audit.logList.filters.requestIdPlaceholder')"
          />
          <t-input
            v-model="filters.actor"
            clearable
            class="toolbar__search"
            :placeholder="t('audit.logList.filters.actorPlaceholder')"
          />
          <t-input
            v-model="filters.resource"
            clearable
            class="toolbar__search"
            :placeholder="t('audit.logList.filters.resourcePlaceholder')"
          />
          <t-input
            v-model="filters.session"
            clearable
            class="toolbar__search"
            :placeholder="t('audit.logList.filters.sessionPlaceholder')"
          />
          <t-date-range-picker
            v-model="createdRange"
            allow-input
            clearable
            class="toolbar__date"
            enable-time-picker
            format="YYYY-MM-DD HH:mm:ss"
            :placeholder="[
              t('audit.logList.filters.createdRangePlaceholder'),
              t('audit.logList.filters.createdRangePlaceholder'),
            ]"
          />
        </div>
      </div>

      <management-toolbar>
        <template #filters>
          <t-input
            v-model="filters.action"
            clearable
            class="toolbar__search"
            :placeholder="t('audit.logList.filters.actionPlaceholder')"
          />
          <t-input
            v-model="filters.resource_type"
            clearable
            class="toolbar__select"
            :placeholder="t('audit.logList.filters.resourceTypePlaceholder')"
          />
          <t-input
            v-model="filters.resource_name"
            clearable
            class="toolbar__select"
            :placeholder="t('audit.logList.filters.resourceNamePlaceholder')"
          />
          <t-select
            v-model="filters.successValue"
            clearable
            class="toolbar__select"
            :options="successOptions"
            :placeholder="t('audit.logList.filters.successPlaceholder')"
          />
        </template>
        <template #actions>
          <t-space size="small" wrap>
            <t-button v-permission="AUDIT_PERMISSION_CODE.READ" theme="default" variant="text" @click="toggleDensity">
              {{ densityButtonLabel }}
            </t-button>
            <t-button v-permission="AUDIT_PERMISSION_CODE.READ" theme="default" variant="text" @click="resetFilters">
              {{ t('audit.logList.clearFilters') }}
            </t-button>
          </t-space>
        </template>
      </management-toolbar>

      <section class="audit-workbench-grid">
        <div class="inline-note">
          <p>{{ t('audit.logList.readonlyNotice') }}</p>
          <p>{{ t('audit.logList.factSourceHint') }}</p>
        </div>

        <div class="audit-investigation-cards">
          <article v-for="signal in investigationSignals" :key="signal.key" class="audit-investigation-card">
            <div class="audit-investigation-card__head">
              <strong>{{ signal.title }}</strong>
              <t-tag :theme="signal.tone" variant="light-outline" size="small">{{ signal.value }}</t-tag>
            </div>
            <p>{{ signal.description }}</p>
            <button type="button" class="audit-investigation-card__action" @click="applyInvestigationSignal(signal)">
              {{ signal.action }}
            </button>
          </article>
        </div>
      </section>

      <management-table-card>
        <template #head>
          <div class="table-head">
            <div>
              <p class="table-head__summary">{{ t('audit.logList.summary', { count: rows.length }) }}</p>
              <p class="table-head__description">{{ t('audit.logList.tableHint') }}</p>
            </div>
            <div class="table-head__meta">
              <t-tag theme="default" variant="light-outline" size="small">{{ densitySummary }}</t-tag>
              <t-tag v-if="activePreset !== 'all'" theme="warning" variant="light-outline" size="small">
                {{ activePresetLabel }}
              </t-tag>
            </div>
          </div>
        </template>

        <management-empty-state
          v-if="listError && !loading"
          tone="error"
          :title="t('audit.logList.errorTitle')"
          :description="listError"
        >
          <template #actions>
            <t-button theme="primary" variant="outline" @click="fetchAuditLogs">
              {{ t('audit.logList.retry') }}
            </t-button>
          </template>
        </management-empty-state>

        <t-table
          v-else
          row-key="id"
          :class="[`audit-table--${densityMode}`]"
          :data="rows"
          :columns="columns"
          :loading="loading"
          table-layout="fixed"
          :table-content-width="tableContentWidth"
          cell-empty-content="-"
          hover
          max-height="720"
          header-affixed-top
        >
          <template #action="{ row }">
            <div class="action-cell">
              <strong class="action-cell__primary">{{ auditActionTitle(row) }}</strong>
              <span class="action-cell__secondary">{{ row.resource_type || '-' }}</span>
            </div>
          </template>

          <template #actor="{ row }">
            <div class="stack-cell">
              <strong>{{ actorLabel(row) }}</strong>
              <span class="stack-cell__secondary">{{ actorSecondaryLabel(row) }}</span>
            </div>
          </template>

          <template #resource="{ row }">
            <div class="stack-cell">
              <strong>{{ resourceLabel(row) }}</strong>
              <span class="stack-cell__secondary">{{ resourceSecondaryLabel(row) }}</span>
            </div>
          </template>

          <template #result="{ row }">
            <div class="result-cell">
              <t-tag :theme="riskTone(row)" variant="light-outline" size="small" shape="round">
                {{ row.success ? t('audit.logList.result.success') : t('audit.logList.result.failed') }}
              </t-tag>
              <span class="result-cell__risk">{{ riskLabel(row) }}</span>
            </div>
          </template>

          <template #created_at="{ row }">
            <span>{{ formatTimestamp(row.created_at) }}</span>
          </template>

          <template #operation="{ row }">
            <table-action-menu
              :actions="[
                {
                  label: t('audit.logList.detail'),
                  testId: 'audit-detail',
                  value: 'detail',
                },
                {
                  label: t('audit.logList.quickActions.sameRequest'),
                  testId: 'audit-same-request',
                  value: 'same-request',
                },
                {
                  label: t('audit.logList.quickActions.sameActor'),
                  testId: 'audit-same-actor',
                  value: 'same-actor',
                },
              ]"
              :more-label="t('audit.logList.more')"
              @action="(action) => handleRowAction(action, row)"
            />
          </template>

          <template #expandedRow="{ row }">
            <div class="expanded-panel">
              <div class="expanded-panel__summary">
                <div>
                  <strong>{{ t('audit.logList.expanded.correlationTitle') }}</strong>
                  <p>{{ correlationSummary(row) }}</p>
                </div>
                <t-space size="small" wrap>
                  <t-tag
                    v-for="tag in expandedTags(row)"
                    :key="tag"
                    theme="default"
                    variant="light-outline"
                    size="small"
                  >
                    {{ tag }}
                  </t-tag>
                </t-space>
              </div>
              <div class="expanded-panel__timeline">
                <article v-for="item in timelineForRow(row)" :key="item.key" class="expanded-panel__timeline-item">
                  <strong>{{ item.title }}</strong>
                  <span>{{ item.description }}</span>
                </article>
              </div>
            </div>
          </template>

          <template #empty>
            <div class="table-empty-state">
              <t-empty :title="t('audit.logList.emptyTitle')" :description="t('audit.logList.emptyDescription')">
                <template #action>
                  <div class="table-empty-state__actions">
                    <t-button
                      v-if="hasActiveFilters"
                      v-permission="AUDIT_PERMISSION_CODE.READ"
                      theme="default"
                      variant="outline"
                      @click="resetFilters"
                    >
                      {{ t('audit.logList.clearFilters') }}
                    </t-button>
                  </div>
                </template>
              </t-empty>
            </div>
          </template>
        </t-table>

        <template #footer>
          <management-table-pagination :summary="t('audit.logList.footerTotal', { count: total })">
            <t-pagination
              v-model:current="pagination.current"
              v-model:page-size="pagination.pageSize"
              :total="total"
              :page-size-options="[10, 20, 50]"
              @change="handlePageChange"
            />
          </management-table-pagination>
        </template>
      </management-table-card>
    </management-page-content>

    <t-drawer
      v-model:visible="detailDrawerVisible"
      :header="t('audit.logList.detailTitle')"
      size="640px"
      placement="right"
      destroy-on-close
    >
      <div v-if="detailRecord" class="drawer-panel audit-detail-panel">
        <div class="detail-hero">
          <div>
            <strong>{{ auditActionTitle(detailRecord) }}</strong>
            <p>{{ detailHeroText(detailRecord) }}</p>
          </div>
          <t-tag :theme="riskTone(detailRecord)" variant="light-outline" size="small">
            {{ riskLabel(detailRecord) }}
          </t-tag>
        </div>

        <div class="detail-section">
          <h4 class="detail-section__title">{{ t('audit.logList.detailSections.basic') }}</h4>
          <div class="detail-grid">
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.columns.action') }}</span>
              <span class="detail-item__value">{{ auditActionTitle(detailRecord) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.columns.result') }}</span>
              <span class="detail-item__value">
                {{ detailRecord.success ? t('audit.logList.result.success') : t('audit.logList.result.failed') }}
              </span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.columns.createdAt') }}</span>
              <span class="detail-item__value">{{ formatTimestamp(detailRecord.created_at) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.columns.actor') }}</span>
              <span class="detail-item__value">{{ actorLabel(detailRecord) }}</span>
            </div>
            <div class="detail-item detail-item--full">
              <span class="detail-item__label">{{ t('audit.logList.columns.resource') }}</span>
              <span class="detail-item__value">{{ resourceDetailLabel(detailRecord) }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <h4 class="detail-section__title">{{ t('audit.logList.detailSections.request') }}</h4>
          <div class="detail-grid">
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.detailFields.requestId') }}</span>
              <span class="detail-item__value detail-item__value--mono">{{ detailRecord.request_id || '-' }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.detailFields.traceId') }}</span>
              <span class="detail-item__value detail-item__value--mono">{{ traceIdForRecord(detailRecord) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.detailFields.sessionId') }}</span>
              <span class="detail-item__value detail-item__value--mono">{{ sessionIdForRecord(detailRecord) }}</span>
            </div>
            <div class="detail-item">
              <span class="detail-item__label">{{ t('audit.logList.detailFields.ip') }}</span>
              <span class="detail-item__value">{{ detailRecord.ip || '-' }}</span>
            </div>
            <div class="detail-item detail-item--full">
              <span class="detail-item__label">{{ t('audit.logList.detailFields.userAgent') }}</span>
              <span class="detail-item__value">{{ detailRecord.user_agent || '-' }}</span>
            </div>
            <div class="detail-item detail-item--full">
              <span class="detail-item__label">{{ t('audit.logList.detailFields.message') }}</span>
              <span class="detail-item__value">{{ detailRecord.message || '-' }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <h4 class="detail-section__title">{{ t('audit.logList.detailSections.context') }}</h4>
          <div class="detail-grid">
            <div v-for="field in contextualFields(detailRecord)" :key="field.label" class="detail-item">
              <span class="detail-item__label">{{ field.label }}</span>
              <span class="detail-item__value">{{ field.value }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section">
          <h4 class="detail-section__title">{{ t('audit.logList.detailSections.correlation') }}</h4>
          <div class="detail-correlation-list">
            <article v-for="item in correlationItems(detailRecord)" :key="item.title" class="detail-correlation-item">
              <strong>{{ item.title }}</strong>
              <p>{{ item.description }}</p>
            </article>
          </div>
        </div>

        <div class="detail-section">
          <h4 class="detail-section__title">{{ t('audit.logList.detailSections.timeline') }}</h4>
          <div class="detail-timeline">
            <article v-for="item in timelineForRow(detailRecord)" :key="item.key" class="detail-timeline__item">
              <strong>{{ item.title }}</strong>
              <span>{{ item.description }}</span>
            </article>
          </div>
        </div>

        <div class="detail-section">
          <h4 class="detail-section__title">{{ t('audit.logList.detailSections.risk') }}</h4>
          <div class="detail-risk-list">
            <t-tag
              v-for="item in riskSignals(detailRecord)"
              :key="item"
              theme="warning"
              variant="light-outline"
              size="small"
            >
              {{ item }}
            </t-tag>
          </div>
        </div>

        <div class="detail-section">
          <div class="detail-section__header">
            <h4 class="detail-section__title">{{ t('audit.logList.detailSections.metadata') }}</h4>
            <t-button theme="default" variant="text" size="small" @click="copyMetadata(detailRecord)">
              {{ t('audit.logList.copyMetadata') }}
            </t-button>
          </div>
          <pre class="detail-code">{{ metadataDetail(detailRecord.metadata) }}</pre>
        </div>
      </div>
    </t-drawer>
  </div>
</template>
<script setup lang="ts">
import type { TdBaseTableProps } from 'tdesign-vue-next';
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import {
  calculateTableContentWidth,
  createActionColumn,
  createStatusColumn,
  createTextColumn,
  createTimeColumn,
  formatCompactDateTime,
  ManagementEmptyState,
  ManagementPageContent,
  ManagementPageHeader,
  ManagementTableCard,
  ManagementTablePagination,
  ManagementToolbar,
  TableActionMenu,
} from '@/shared/components/management';
import { createLogger } from '@/utils/logger';

import { getAuditLogs } from '../../api/audit';
import { AUDIT_PERMISSION_CODE } from '../../contract/permissions';
import type { AuditLogListItem, AuditLogQuery } from '../../types/audit';

defineOptions({
  name: 'AuditLogListIndex',
});

type AuditFilterState = {
  action: string;
  resource_type: string;
  resource_name: string;
  request_id: string;
  actor: string;
  resource: string;
  session: string;
  successValue: '' | 'true' | 'false';
};

type PresetKey = 'all' | 'failed-auth' | 'rbac-changes' | 'permission-denied' | 'plugin-ops' | 'sensitive-ops';
type DensityMode = 'comfortable' | 'compact';

const logger = createLogger('audit.logs');
const { t, locale } = useI18n();

const loading = ref(false);
const listError = ref('');
const latestRequestSeq = ref(0);
const rows = ref<AuditLogListItem[]>([]);
const total = ref(0);
const createdRange = ref<string[]>([]);
const detailDrawerVisible = ref(false);
const detailRecord = ref<AuditLogListItem | null>(null);
const activePreset = ref<PresetKey>('all');
const densityMode = ref<DensityMode>('comfortable');
const filters = ref<AuditFilterState>({
  action: '',
  resource_type: '',
  resource_name: '',
  request_id: '',
  actor: '',
  resource: '',
  session: '',
  successValue: '',
});
const pagination = ref({
  current: 1,
  pageSize: 10,
});

const presetViews = computed(() => [
  { key: 'all' as const, title: t('audit.logList.presets.all') },
  { key: 'failed-auth' as const, title: t('audit.logList.presets.failedAuth') },
  { key: 'rbac-changes' as const, title: t('audit.logList.presets.rbacChanges') },
  { key: 'permission-denied' as const, title: t('audit.logList.presets.permissionDenied') },
  { key: 'plugin-ops' as const, title: t('audit.logList.presets.pluginOps') },
  { key: 'sensitive-ops' as const, title: t('audit.logList.presets.sensitiveOps') },
]);

const successOptions = computed(() => [
  { label: t('audit.logList.filters.successAll'), value: '' },
  { label: t('audit.logList.filters.successTrue'), value: 'true' },
  { label: t('audit.logList.filters.successFalse'), value: 'false' },
]);

const densitySummary = computed(() =>
  densityMode.value === 'compact' ? t('audit.logList.density.compact') : t('audit.logList.density.comfortable'),
);
const densityButtonLabel = computed(() =>
  densityMode.value === 'compact'
    ? t('audit.logList.density.switchComfortable')
    : t('audit.logList.density.switchCompact'),
);
const activePresetLabel = computed(
  () => presetViews.value.find((item) => item.key === activePreset.value)?.title ?? '',
);

const hasActiveFilters = computed(() => {
  return Boolean(
    filters.value.action.trim() ||
    filters.value.resource_type.trim() ||
    filters.value.resource_name.trim() ||
    filters.value.request_id.trim() ||
    filters.value.actor.trim() ||
    filters.value.resource.trim() ||
    filters.value.session.trim() ||
    filters.value.successValue ||
    createdRange.value.length ||
    activePreset.value !== 'all',
  );
});

const investigationSignals = computed(() => [
  {
    key: 'request-chain',
    title: t('audit.logList.investigationSignals.requestChain.title'),
    description: t('audit.logList.investigationSignals.requestChain.description'),
    value: requestCoverageValue(),
    action: t('audit.logList.investigationSignals.requestChain.action'),
    tone: 'primary' as const,
    handler: () => {
      if (rows.value[0]?.request_id) {
        filters.value.request_id = rows.value[0].request_id ?? '';
      }
    },
  },
  {
    key: 'rbac-risk',
    title: t('audit.logList.investigationSignals.rbacRisk.title'),
    description: t('audit.logList.investigationSignals.rbacRisk.description'),
    value: highRiskCount().toString(),
    action: t('audit.logList.investigationSignals.rbacRisk.action'),
    tone: 'warning' as const,
    handler: () => applyPreset('rbac-changes'),
  },
  {
    key: 'failed-flow',
    title: t('audit.logList.investigationSignals.failedFlow.title'),
    description: t('audit.logList.investigationSignals.failedFlow.description'),
    value: failedCount().toString(),
    action: t('audit.logList.investigationSignals.failedFlow.action'),
    tone: 'danger' as const,
    handler: () => applyPreset('failed-auth'),
  },
]);

const columns = computed<TdBaseTableProps['columns']>(() => {
  void locale.value;

  return [
    createTextColumn(t('audit.logList.columns.action'), 'action', {
      fixed: 'left',
      minWidth: densityMode.value === 'compact' ? 300 : 360,
    }),
    createTextColumn(t('audit.logList.columns.actor'), 'actor', {
      width: densityMode.value === 'compact' ? 180 : 210,
    }),
    createTextColumn(t('audit.logList.columns.resource'), 'resource', {
      width: densityMode.value === 'compact' ? 220 : 260,
    }),
    createStatusColumn(t('audit.logList.columns.result'), 'result', 132),
    createTimeColumn(t('audit.logList.columns.createdAt'), 'created_at', densityMode.value === 'compact' ? 160 : 180),
    createActionColumn(t('components.commonTable.operation'), 108),
  ];
});

const tableContentWidth = computed(() => calculateTableContentWidth(columns.value));

function toQuery(): AuditLogQuery {
  const query: AuditLogQuery = {
    page: pagination.value.current,
    page_size: pagination.value.pageSize,
  };

  if (filters.value.action.trim()) {
    query.action = filters.value.action.trim();
  }
  if (filters.value.resource_type.trim()) {
    query.resource_type = filters.value.resource_type.trim();
  }
  if (filters.value.resource_name.trim()) {
    query.resource_name = filters.value.resource_name.trim();
  }
  if (filters.value.request_id.trim()) {
    query.request_id = filters.value.request_id.trim();
  }
  if (filters.value.successValue === 'true') {
    query.success = true;
  } else if (filters.value.successValue === 'false') {
    query.success = false;
  }
  if (createdRange.value[0]) {
    query.created_from = toISOStringOrRaw(createdRange.value[0]);
  }
  if (createdRange.value[1]) {
    query.created_to = toISOStringOrRaw(createdRange.value[1]);
  }

  if (filters.value.actor.trim()) {
    query.resource_name = query.resource_name || filters.value.actor.trim();
  }
  if (filters.value.resource.trim()) {
    query.resource_id = filters.value.resource.trim();
  }

  return query;
}

async function fetchAuditLogs() {
  const requestSeq = ++latestRequestSeq.value;
  loading.value = true;
  listError.value = '';

  try {
    const response = await getAuditLogs(toQuery());
    if (requestSeq !== latestRequestSeq.value) {
      return;
    }
    rows.value = response.items;
    total.value = response.total;
  } catch (error) {
    if (requestSeq !== latestRequestSeq.value) {
      return;
    }
    rows.value = [];
    total.value = 0;
    logger.error('failed to fetch audit logs', error);
    listError.value = resolveLocalizedErrorMessage(t, error, t('audit.logList.loadFailed'));
    MessagePlugin.error(listError.value);
  } finally {
    if (requestSeq === latestRequestSeq.value) {
      loading.value = false;
    }
  }
}

function resetFilters() {
  filters.value = {
    action: '',
    resource_type: '',
    resource_name: '',
    request_id: '',
    actor: '',
    resource: '',
    session: '',
    successValue: '',
  };
  activePreset.value = 'all';
  createdRange.value = [];
  pagination.value.current = 1;
}

function handlePageChange() {
  fetchAuditLogs();
}

function actorLabel(row: AuditLogListItem) {
  return row.actor_display_name || row.actor_username || t('audit.logList.actor.anonymous');
}

function actorSecondaryLabel(row: AuditLogListItem) {
  return row.actor_username || row.actor_user_id?.toString() || t('audit.logList.actor.anonymous');
}

function auditActionTitle(row: AuditLogListItem) {
  return row.action || row.request_id || '-';
}

function resourceLabel(row: AuditLogListItem) {
  return row.resource_name || t('audit.logList.resource.unknown');
}

function resourceSecondaryLabel(row: AuditLogListItem) {
  return row.resource_id ? `${row.resource_type || '-'} / ${row.resource_id}` : row.resource_type || '-';
}

function resourceDetailLabel(row: AuditLogListItem) {
  const detailParts = [resourceLabel(row)];

  if (row.resource_type) {
    detailParts.push(row.resource_type);
  }
  if (row.resource_id) {
    detailParts.push(row.resource_id);
  }

  return detailParts.join(' / ');
}

function metadataDetail(metadata: AuditLogListItem['metadata']) {
  if (!metadata || typeof metadata !== 'object' || Object.keys(metadata).length === 0) {
    return '-';
  }

  return JSON.stringify(metadata, null, 2);
}

async function copyMetadata(row: AuditLogListItem) {
  try {
    await navigator.clipboard.writeText(metadataDetail(row.metadata));
    MessagePlugin.success(t('audit.logList.copyMetadataSuccess'));
  } catch (error) {
    logger.error('failed to copy audit metadata', error);
    MessagePlugin.error(t('audit.logList.copyMetadataFailed'));
  }
}

function openDetailDrawer(row: AuditLogListItem) {
  detailRecord.value = row;
  detailDrawerVisible.value = true;
}

function formatTimestamp(value?: string | null) {
  return formatCompactDateTime(value);
}

function toISOStringOrRaw(value: string) {
  const date = new Date(value.replace(' ', 'T'));
  return Number.isNaN(date.getTime()) ? value : date.toISOString();
}

function applyPreset(preset: PresetKey) {
  activePreset.value = preset;
  pagination.value.current = 1;

  if (preset === 'all') {
    filters.value.action = '';
    filters.value.resource_type = '';
    filters.value.successValue = '';
    return;
  }

  if (preset === 'failed-auth') {
    filters.value.action = 'auth';
    filters.value.successValue = 'false';
    filters.value.resource_type = 'session';
  } else if (preset === 'rbac-changes') {
    filters.value.action = 'role';
    filters.value.resource_type = 'role';
    filters.value.successValue = '';
  } else if (preset === 'permission-denied') {
    filters.value.action = 'permission';
    filters.value.successValue = 'false';
    filters.value.resource_type = '';
  } else if (preset === 'plugin-ops') {
    filters.value.action = 'plugin';
    filters.value.resource_type = 'plugin';
    filters.value.successValue = '';
  } else if (preset === 'sensitive-ops') {
    filters.value.action = 'delete';
    filters.value.resource_type = '';
    filters.value.successValue = '';
  }
}

function toggleDensity() {
  densityMode.value = densityMode.value === 'comfortable' ? 'compact' : 'comfortable';
}

function requestCoverageValue() {
  const requestCount = rows.value.filter((row) => row.request_id).length;
  return `${requestCount}/${rows.value.length || 0}`;
}

function failedCount() {
  return rows.value.filter((row) => !row.success).length;
}

function highRiskCount() {
  return rows.value.filter((row) => riskTone(row) !== 'success').length;
}

function riskTone(row: AuditLogListItem) {
  if (!row.success) {
    return 'danger' as const;
  }
  if (isSensitiveAction(row)) {
    return 'warning' as const;
  }
  return 'success' as const;
}

function riskLabel(row: AuditLogListItem) {
  if (!row.success) {
    return t('audit.logList.risk.failed');
  }
  if (isSensitiveAction(row)) {
    return t('audit.logList.risk.sensitive');
  }
  return t('audit.logList.risk.normal');
}

function isSensitiveAction(row: AuditLogListItem) {
  const action = row.action?.toLowerCase() ?? '';
  return ['delete', 'role', 'permission', 'plugin', 'password'].some((keyword) => action.includes(keyword));
}

function timelineForRow(row: AuditLogListItem) {
  return [
    {
      key: 'actor',
      title: t('audit.logList.timeline.actorTitle'),
      description: `${actorLabel(row)} · ${formatTimestamp(row.created_at)}`,
    },
    {
      key: 'request',
      title: t('audit.logList.timeline.requestTitle'),
      description: `${row.request_id || '-'} · ${traceIdForRecord(row)}`,
    },
    {
      key: 'resource',
      title: t('audit.logList.timeline.resourceTitle'),
      description: resourceDetailLabel(row),
    },
  ];
}

function correlationSummary(row: AuditLogListItem) {
  return t('audit.logList.expanded.correlationSummary', {
    actor: actorLabel(row),
    resource: resourceLabel(row),
    requestId: row.request_id || '-',
  });
}

function expandedTags(row: AuditLogListItem) {
  return [
    row.request_id
      ? `${t('audit.logList.expanded.tags.request')}: ${row.request_id}`
      : t('audit.logList.expanded.tags.noRequest'),
    `${t('audit.logList.expanded.tags.actor')}: ${actorLabel(row)}`,
    `${t('audit.logList.expanded.tags.resource')}: ${row.resource_type || '-'}`,
  ];
}

function traceIdForRecord(row: AuditLogListItem) {
  const metadata = row.metadata;
  if (metadata && typeof metadata === 'object' && 'trace_id' in metadata && typeof metadata.trace_id === 'string') {
    return metadata.trace_id;
  }
  return row.request_id ? `trace/${row.request_id}` : '-';
}

function sessionIdForRecord(row: AuditLogListItem) {
  const metadata = row.metadata;
  if (metadata && typeof metadata === 'object' && 'session_id' in metadata && typeof metadata.session_id === 'string') {
    return metadata.session_id;
  }
  return filters.value.session || '-';
}

function contextualFields(row: AuditLogListItem) {
  return [
    { label: t('audit.logList.detailFields.plugin'), value: metadataLookup(row, 'plugin', '-') },
    { label: t('audit.logList.detailFields.endpoint'), value: metadataLookup(row, 'endpoint', '-') },
    { label: t('audit.logList.detailFields.relatedRole'), value: metadataLookup(row, 'role', '-') },
    { label: t('audit.logList.detailFields.relatedPermission'), value: metadataLookup(row, 'permission', '-') },
    { label: t('audit.logList.detailFields.beforeSnapshot'), value: metadataLookup(row, 'before', '-') },
    { label: t('audit.logList.detailFields.afterSnapshot'), value: metadataLookup(row, 'after', '-') },
  ];
}

function metadataLookup(row: AuditLogListItem, key: string, fallback: string) {
  const metadata = row.metadata;
  if (!metadata || typeof metadata !== 'object' || !(key in metadata)) {
    return fallback;
  }

  const value = metadata[key];
  return typeof value === 'string' ? value : JSON.stringify(value);
}

function correlationItems(row: AuditLogListItem) {
  return [
    {
      title: t('audit.logList.correlationItems.sameRequest.title'),
      description: row.request_id
        ? t('audit.logList.correlationItems.sameRequest.description', { requestId: row.request_id })
        : t('audit.logList.correlationItems.sameRequest.empty'),
    },
    {
      title: t('audit.logList.correlationItems.sameActor.title'),
      description: t('audit.logList.correlationItems.sameActor.description', { actor: actorLabel(row) }),
    },
    {
      title: t('audit.logList.correlationItems.sameResource.title'),
      description: t('audit.logList.correlationItems.sameResource.description', { resource: resourceLabel(row) }),
    },
  ];
}

function riskSignals(row: AuditLogListItem) {
  const signals = [riskLabel(row)];

  if (!row.success) {
    signals.push(t('audit.logList.riskSignals.authAnomaly'));
  }
  if (isSensitiveAction(row)) {
    signals.push(t('audit.logList.riskSignals.privilegeSensitive'));
  }
  if (row.request_id) {
    signals.push(t('audit.logList.riskSignals.requestTraceable'));
  }

  return signals;
}

function detailHeroText(row: AuditLogListItem) {
  return t('audit.logList.detailHero', {
    actor: actorLabel(row),
    resource: resourceLabel(row),
    requestId: row.request_id || '-',
  });
}

function handleRowAction(action: string, row: AuditLogListItem) {
  if (action === 'same-request') {
    filters.value.request_id = row.request_id || '';
    return;
  }
  if (action === 'same-actor') {
    filters.value.actor = actorLabel(row);
    return;
  }
  openDetailDrawer(row);
}

function applyInvestigationSignal(signal: { handler: () => void }) {
  signal.handler();
}

onMounted(() => {
  fetchAuditLogs();
});

watch(
  () =>
    [
      filters.value.action,
      filters.value.resource_type,
      filters.value.resource_name,
      filters.value.request_id,
      filters.value.actor,
      filters.value.resource,
      filters.value.session,
      filters.value.successValue,
      createdRange.value[0],
      createdRange.value[1],
    ] as const,
  () => {
    pagination.value.current = 1;
    fetchAuditLogs();
  },
);
</script>
<style scoped lang="less">
@import '../../../rbac/shared/list-page.less';

.audit-page {
  display: flex;
  flex-direction: column;
  gap: 16px;

  .management-list-toolbar();
  .management-list-header();
  .management-list-table-empty();
  .management-list-table-shell();
  .management-list-mobile();
}

.audit-correlation-toolbar,
.inline-note,
.audit-investigation-card,
.expanded-panel,
.detail-hero,
.detail-correlation-item,
.detail-timeline__item {
  border-radius: var(--td-radius-large);
}

.audit-correlation-toolbar {
  align-items: flex-start;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--td-warning-color-1) 88%, white),
    var(--td-bg-color-container)
  );
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 90%, var(--td-warning-color-4));
  display: grid;
  gap: 14px;
  grid-template-columns: minmax(220px, 0.85fr) minmax(0, 2fr);
  padding: 16px;
}

.audit-correlation-toolbar__copy strong,
.audit-investigation-card__head strong,
.detail-hero strong {
  color: var(--td-text-color-primary);
}

.audit-correlation-toolbar__copy p,
.audit-investigation-card p,
.detail-hero p,
.detail-correlation-item p,
.detail-timeline__item span {
  color: var(--td-text-color-secondary);
  margin: 4px 0 0;
}

.audit-correlation-toolbar__filters {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.toolbar__date {
  min-width: min(100%, 320px);
}

.audit-workbench-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: minmax(280px, 0.8fr) minmax(0, 1.6fr);
}

.inline-note {
  --audit-note-bg: color-mix(in srgb, var(--td-warning-color-1) 75%, var(--td-bg-color-container));

  background: var(--audit-note-bg);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 92%, var(--td-warning-color-5));
  border-inline-start: 4px solid var(--td-warning-color-5);
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--td-warning-color-4) 12%, transparent);
  color: var(--td-text-color-placeholder);
  display: grid;
  gap: 6px;
  padding: 14px 16px 14px 18px;
}

.inline-note p,
.table-head__summary,
.table-head__description {
  margin: 0;
}

.audit-investigation-cards {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.audit-investigation-card {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  display: grid;
  gap: 12px;
  padding: 16px;
}

.audit-investigation-card__head,
.table-head,
.table-head__meta,
.result-cell,
.detail-section__header {
  align-items: center;
  display: flex;
  gap: 12px;
  justify-content: space-between;
}

.audit-investigation-card__action {
  background: transparent;
  border: 0;
  color: var(--td-warning-color-7);
  cursor: pointer;
  justify-self: flex-start;
  padding: 0;
}

.action-cell,
.stack-cell,
.detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.action-cell__primary,
.stack-cell strong,
.detail-item__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.action-cell__secondary,
.stack-cell__secondary,
.detail-item__label,
.result-cell__risk {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.expanded-panel {
  background: color-mix(in srgb, var(--td-warning-color-1) 60%, var(--td-bg-color-container));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 90%, var(--td-warning-color-4));
  display: grid;
  gap: 12px;
  padding: 14px 16px;
}

.expanded-panel__summary,
.expanded-panel__timeline,
.audit-detail-panel,
.detail-section,
.detail-correlation-list,
.detail-timeline,
.detail-risk-list {
  display: grid;
  gap: 12px;
}

.expanded-panel__summary {
  align-items: flex-start;
  grid-template-columns: minmax(0, 1fr) auto;
}

.expanded-panel__summary p {
  color: var(--td-text-color-secondary);
  margin: 4px 0 0;
}

.expanded-panel__timeline {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.expanded-panel__timeline-item,
.detail-correlation-item,
.detail-timeline__item {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: grid;
  gap: 6px;
  padding: 12px;
}

.detail-hero {
  align-items: flex-start;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--td-warning-color-1) 78%, white),
    var(--td-bg-color-container)
  );
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 88%, var(--td-warning-color-4));
  display: flex;
  justify-content: space-between;
  padding: 16px;
}

.detail-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.detail-item--full {
  grid-column: 1 / -1;
}

.detail-item__value--mono,
.detail-code {
  font-family: var(--td-font-family-medium);
}

.detail-code {
  background: var(--td-bg-color-page);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  margin: 0;
  max-height: 240px;
  overflow: auto;
  padding: 12px;
  white-space: pre-wrap;
}

.detail-risk-list {
  grid-template-columns: repeat(auto-fit, minmax(140px, max-content));
}

.audit-table--compact :deep(.t-table__body td) {
  padding-block: 8px;
}

.audit-table--comfortable :deep(.t-table__body td) {
  padding-block: 12px;
}

@media (width <= 1200px) {
  .audit-correlation-toolbar,
  .audit-workbench-grid {
    grid-template-columns: 1fr;
  }

  .audit-correlation-toolbar__filters,
  .audit-investigation-cards,
  .expanded-panel__timeline {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 768px) {
  .audit-correlation-toolbar__filters,
  .audit-investigation-cards,
  .expanded-panel__timeline,
  .detail-grid {
    grid-template-columns: 1fr;
  }

  .detail-hero,
  .expanded-panel__summary,
  .table-head,
  .audit-investigation-card__head {
    align-items: flex-start;
    flex-direction: column;
  }

  .toolbar__date {
    min-width: 100%;
  }
}
</style>
