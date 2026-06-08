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
        <t-button theme="primary" :loading="loading" @click="loadSummary">
          {{ t('dashboard.actions.refresh') }}
        </t-button>
      </template>
    </page-header>

    <t-loading :loading="loading" size="large" :text="t('dashboard.loading')">
      <t-alert v-if="errorMessage" theme="error" :title="t('dashboard.error.title')" :message="errorMessage">
        <template #operation>
          <t-button variant="text" theme="primary" size="small" @click="loadSummary">
            {{ t('dashboard.actions.retry') }}
          </t-button>
        </template>
      </t-alert>

      <template v-if="summary">
        <section class="dashboard-page__summary" :aria-label="t('dashboard.systemSummary.title')">
          <div v-for="item in systemSummaryItems" :key="item.key" class="dashboard-page__summary-item">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
            <p>{{ item.description }}</p>
          </div>
        </section>

        <dashboard-quick-actions :links="quickLinks" />

        <dashboard-renderer
          :widgets="widgets"
          :refreshing-widget-id="refreshingWidgetId"
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
import DashboardQuickActions from '../components/DashboardQuickActions.vue';
import DashboardRenderer from '../components/DashboardRenderer.vue';
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

const systemSummaryItems = computed(() => {
  const systemSummary = summary.value?.system_summary;
  if (!systemSummary) {
    return [];
  }

  return [
    {
      key: 'current-user',
      label: t('dashboard.systemSummary.currentUser.label'),
      value: systemSummary.current_user.display_name || systemSummary.current_user.username,
      description: systemSummary.current_user.username,
    },
    {
      key: 'environment',
      label: t('dashboard.systemSummary.environment.label'),
      value: systemSummary.app_env,
      description: t('dashboard.systemSummary.environment.description'),
    },
    {
      key: 'locale',
      label: t('dashboard.systemSummary.locale.label'),
      value: systemSummary.locale.default_locale,
      description: t('dashboard.systemSummary.locale.description', {
        fallback: systemSummary.locale.fallback_locale,
      }),
    },
    {
      key: 'modules',
      label: t('dashboard.systemSummary.modules.label'),
      value: String(systemSummary.modules.enabled_modules),
      description: t('dashboard.systemSummary.modules.description', {
        total: systemSummary.modules.total_modules,
        degraded: systemSummary.modules.degraded_modules,
      }),
    },
    {
      key: 'widgets',
      label: t('dashboard.systemSummary.widgets.label'),
      value: String(systemSummary.visible_widgets),
      description: t('dashboard.systemSummary.widgets.description'),
    },
  ];
});

onMounted(() => {
  void loadSummary();
});

async function loadSummary() {
  loading.value = true;
  errorMessage.value = '';

  try {
    const response = await getDashboardSummary();
    summary.value = response;
    quickLinks.value = response.quick_links;
    widgets.value = response.widgets;
  } catch (error) {
    logger.error('dashboard summary request failed', error);
    errorMessage.value = requestErrorMessage(error, t('dashboard.error.fallback'));
  } finally {
    loading.value = false;
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
</script>
<style lang="less" scoped>
.dashboard-page {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xl);
  min-width: 0;
}

.dashboard-page__summary {
  display: grid;
  gap: var(--td-comp-margin-m);
  grid-template-columns: repeat(5, minmax(0, 1fr));
}

.dashboard-page__summary-item {
  background: linear-gradient(180deg, var(--td-bg-color-container) 0%, var(--td-bg-color-container-hover) 100%);
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
  .dashboard-page__summary {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (width <= 768px) {
  .dashboard-page__summary {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
