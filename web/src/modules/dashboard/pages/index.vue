<template>
  <section class="dashboard-page" data-page-type="overview-dashboard">
    <page-header
      :breadcrumb="[{ labelKey: 'dashboard.page.title', fallback: t('dashboard.page.title') }]"
      title-key="dashboard.page.title"
      :title-fallback="t('dashboard.page.title')"
      description-key="dashboard.page.description"
      :description-fallback="t('dashboard.page.description')"
      :source="{
        labelKey: 'dashboard.page.eyebrow',
        fallback: t('dashboard.page.eyebrow'),
        color: 'var(--td-brand-color-6)',
      }"
    >
      <template #actions>
        <div class="dashboard-page__header-actions">
          <span v-if="lastUpdatedAt" class="dashboard-page__updated-at">
            {{ t('dashboard.page.lastUpdated', { time: lastUpdatedLabel }) }}
          </span>
          <t-button theme="primary" :loading="loading" @click="loadSummary">
            {{ t('dashboard.actions.refresh') }}
          </t-button>
        </div>
      </template>
    </page-header>

    <t-loading :loading="loading && Boolean(summary)" size="large" :text="t('dashboard.loading')">
      <t-alert v-if="errorMessage" theme="error" :title="t('dashboard.error.title')" :message="errorMessage">
        <template #operation>
          <t-button variant="text" theme="primary" size="small" @click="loadSummary">
            {{ t('dashboard.actions.retry') }}
          </t-button>
        </template>
      </t-alert>

      <template v-if="summary || loading">
        <section class="dashboard-page__summary" :aria-label="t('dashboard.systemSummary.title')">
          <header class="dashboard-page__summary-header">
            <span>{{ t('dashboard.systemSummary.eyebrow') }}</span>
            <h2>{{ t('dashboard.systemSummary.title') }}</h2>
          </header>
          <div class="dashboard-page__summary-grid">
            <div v-for="item in systemSummaryItems" :key="item.key" class="dashboard-page__summary-item">
              <template v-if="loading && !summary">
                <t-skeleton animation="gradient" :row-col="summarySkeletonRowCol" />
              </template>
              <template v-else>
                <span>{{ item.label }}</span>
                <strong>{{ item.value }}</strong>
                <p>{{ item.description }}</p>
              </template>
            </div>
          </div>
        </section>

        <dashboard-quick-actions v-if="summary" :links="quickLinks" :config="quickActionConfig" />

        <dashboard-renderer
          :widgets="widgets"
          :refreshing-widget-id="refreshingWidgetId"
          :loading="loading && !summary"
          @refresh-widget="refreshWidget"
        />
      </template>

      <t-empty v-else-if="!loading" size="large" :description="t('dashboard.empty')" />
    </t-loading>
  </section>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';

import { API_CODE } from '@/contracts/api/codes';
import { t } from '@/locales';
import { PageHeader } from '@/shared/components/page';
import type { ApiRequestError } from '@/types/axios';
import { createLogger } from '@/utils/logger';

import { getDashboardSummary, getDashboardWidget } from '../api/dashboard';
import { getDashboardSystemConfigs } from '../api/quick-actions-config';
import DashboardQuickActions from '../components/DashboardQuickActions.vue';
import DashboardRenderer from '../components/DashboardRenderer.vue';
import {
  type DashboardQuickActionConfig,
  DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG,
  resolveDashboardQuickActionConfig,
} from '../contract/quick-actions';
import type { DashboardQuickLink, DashboardSummaryResponse, DashboardWidget } from '../types/dashboard';

defineOptions({
  name: 'DashboardHomePage',
});

const logger = createLogger('dashboard.home');
const loading = ref(false);
const refreshingWidgetId = ref('');
const errorMessage = ref('');
const summary = ref<DashboardSummaryResponse | null>(null);
const quickLinks = ref<DashboardQuickLink[]>([]);
const widgets = ref<DashboardWidget[]>([]);
const lastUpdatedAt = ref('');
const quickActionConfig = ref<DashboardQuickActionConfig>({ ...DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG });
const summarySkeletonRowCol = [
  { width: '52%', height: '14px' },
  { width: '36%', height: '28px' },
  { width: '80%', height: '14px' },
];

const systemSummaryItems = computed(() => {
  const systemSummary = summary.value?.system_summary;
  if (!systemSummary) {
    return [
      { key: 'modules', label: '', value: '', description: '' },
      { key: 'abnormal-services', label: '', value: '', description: '' },
      { key: 'failed-tasks', label: '', value: '', description: '' },
      { key: 'high-risk-events', label: '', value: '', description: '' },
    ];
  }

  return [
    {
      key: 'modules',
      label: t('dashboard.systemSummary.modules.label'),
      value: t('dashboard.systemSummary.modules.value', {
        count: systemSummary.modules.enabled_modules,
      }),
      description: t('dashboard.systemSummary.modules.description', {
        total: systemSummary.modules.total_modules,
        degraded: systemSummary.modules.degraded_modules,
      }),
    },
    {
      key: 'abnormal-services',
      label: t('dashboard.systemSummary.abnormalServices.label'),
      value: t('dashboard.systemSummary.abnormalServices.value', {
        count: systemSummary.abnormal_services,
      }),
      description: t('dashboard.systemSummary.abnormalServices.description'),
    },
    {
      key: 'failed-tasks',
      label: t('dashboard.systemSummary.failedTasks.label'),
      value: t('dashboard.systemSummary.failedTasks.value', {
        count: systemSummary.failed_tasks,
      }),
      description: t('dashboard.systemSummary.failedTasks.description'),
    },
    {
      key: 'high-risk-events',
      label: t('dashboard.systemSummary.highRiskEvents.label'),
      value: t('dashboard.systemSummary.highRiskEvents.value', {
        count: systemSummary.high_risk_events,
      }),
      description: t('dashboard.systemSummary.highRiskEvents.description'),
    },
  ];
});

const lastUpdatedLabel = computed(() => formatDashboardUpdatedAt(lastUpdatedAt.value));

onMounted(() => {
  void loadSummary();
});

async function loadSummary() {
  loading.value = true;
  errorMessage.value = '';

  try {
    const [response] = await Promise.all([getDashboardSummary(), loadQuickActionConfig()]);
    summary.value = response;
    quickLinks.value = response.quick_links;
    widgets.value = response.widgets;
    lastUpdatedAt.value = new Date().toISOString();
  } catch (error) {
    logger.error('dashboard summary request failed', error);
    errorMessage.value = requestErrorMessage(error, t('dashboard.error.fallback'));
  } finally {
    loading.value = false;
  }
}

async function loadQuickActionConfig() {
  try {
    const response = await getDashboardSystemConfigs();
    quickActionConfig.value = resolveDashboardQuickActionConfig(response.items ?? []);
  } catch (error) {
    logger.error('dashboard quick-action config request failed', error);
    quickActionConfig.value = { ...DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG };
  }
}

async function refreshWidget(widgetId: string) {
  if (refreshingWidgetId.value) {
    return;
  }

  refreshingWidgetId.value = widgetId;
  try {
    const widget = await getDashboardWidget(widgetId);
    widgets.value = widgets.value.map((item) => (item.id === widgetId ? widget : item));
  } catch (error) {
    logger.error('dashboard widget refresh failed', error);
    widgets.value = widgets.value.map((item) =>
      item.id === widgetId
        ? {
            ...item,
            status: 'error',
            error: {
              code: requestErrorCode(error),
              message_key: requestErrorMessageKey(error),
              message: requestErrorMessage(error, t('dashboard.widget.errorFallback')),
            },
          }
        : item,
    );
  } finally {
    refreshingWidgetId.value = '';
  }
}

function isApiRequestError(error: unknown): error is ApiRequestError {
  return Boolean(error && typeof error === 'object' && (error as Partial<ApiRequestError>).isApiRequestError);
}

function requestErrorMessage(error: unknown, fallback: string) {
  if (isApiRequestError(error)) {
    if (error.messageKey) {
      const translated = t(error.messageKey);
      if (translated !== error.messageKey) {
        return translated;
      }
    }

    return error.message || fallback;
  }

  return error instanceof Error ? error.message : fallback;
}

function requestErrorMessageKey(error: unknown) {
  return isApiRequestError(error) ? error.messageKey : undefined;
}

function requestErrorCode(error: unknown) {
  return isApiRequestError(error) ? error.code : API_CODE.COMMON_INTERNAL_ERROR;
}

function formatDashboardUpdatedAt(value: string) {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) {
    return value;
  }

  return new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'medium',
  }).format(date);
}
</script>
<style lang="less" scoped>
.dashboard-page {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xl);
  min-width: 0;
}

.dashboard-page__header-actions {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-s);
}

.dashboard-page__updated-at {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  white-space: nowrap;
}

.dashboard-page__summary {
  background-color: var(--td-bg-color-container);
  border-radius: var(--td-radius-medium);
  box-shadow: inset 0 0 0 1px var(--td-border-level-1-color);
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-m);
  padding: var(--td-comp-paddingTB-xl) var(--td-comp-paddingLR-xl);
}

.dashboard-page__summary-header {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xxs);
}

.dashboard-page__summary-header span {
  color: var(--td-brand-color);
  font: var(--td-font-body-small);
}

.dashboard-page__summary-header h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
  margin: 0;
}

.dashboard-page__summary-grid {
  display: grid;
  gap: var(--td-comp-margin-m);
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.dashboard-page__summary-item {
  background: var(--td-bg-color-container-hover);
  border-color: var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  border-style: solid;
  border-width: 1px;
  display: grid;
  gap: var(--td-comp-margin-xs);
  min-width: 0;
  overflow: hidden;
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
}

.dashboard-page__summary-item span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.dashboard-page__summary-item strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-headline-small);
  overflow-wrap: anywhere;
}

.dashboard-page__summary-item p {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  margin: 0;
  overflow-wrap: anywhere;
}

@media (width <= 1200px) {
  .dashboard-page__summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 768px) {
  .dashboard-page__header-actions {
    align-items: flex-end;
    flex-direction: column;
    gap: var(--td-comp-margin-xs);
  }

  .dashboard-page__summary-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
