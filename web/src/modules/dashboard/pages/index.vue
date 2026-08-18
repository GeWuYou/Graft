<template>
  <dashboard-workbench
    :error-message="errorMessage"
    :generated-at="lastUpdatedAt || initialTimestamp"
    :loading="loading"
    :navigation-links="quickLinks"
    :presentation="presentation"
    :quick-actions-enabled="quickActionConfig.enabled"
    :ready="Boolean(summary)"
    :refreshing="loading && Boolean(summary)"
    :retrying-id="retryingPresentationItemId"
    @navigate="navigate"
    @refresh="loadSummary"
    @retry-item="retryItem"
  />
</template>
<script setup lang="ts">
// 正式首页把注册表事实投影为工作台语义；loader 状态、业务状态与页面展示分别处理。
import { computed, onActivated, onDeactivated, onMounted, onUnmounted, ref } from 'vue';
import { useRouter } from 'vue-router';

import { API_CODE } from '@/contracts/api/codes';
import { CONTAINER_PERMISSION_CODE } from '@/contracts/generated/modules/container';
import type { SupportedLocale } from '@/contracts/i18n/locales';
import { currentLocale, t } from '@/locales';
import { containerModuleFacades } from '@/modules/container';
import type { ContainerDashboardSummary } from '@/modules/container/contract/dashboard-summary';
import { CONTAINER_ROUTE_PATH } from '@/modules/container/contract/paths';
import {
  acquireContainerDashboardSummarySubscription,
  clearContainerDashboardSummary,
  releaseContainerDashboardSummarySubscription,
  seedContainerDashboardSummary,
  selectContainerDashboardSummaryView,
} from '@/modules/container/shared/stats-manager';
import { usePermissionStore } from '@/store/modules/permission';
import type { ApiRequestError } from '@/types/axios';
import { createLogger } from '@/utils/logger';

import { getDashboardSummary, getDashboardWidget } from '../api/dashboard';
import { getDashboardSystemConfigs } from '../api/quick-actions-config';
import DashboardWorkbench from '../components/workbench/DashboardWorkbench.vue';
import { useDashboardQuickActions } from '../composables/use-dashboard-quick-actions';
import {
  type DashboardQuickActionConfig,
  DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG,
  resolveDashboardQuickActionConfig,
} from '../contract/quick-actions';
import { buildDashboardQuickActionLinks } from '../contract/sidebar-quick-actions';
import {
  type DashboardResourceLoadState,
  projectDashboardSummaryToWorkbench,
} from '../presentation/production-workbench';
import {
  type PresentationItem,
  projectWorkbenchScenario,
  type WorkbenchNavigationSource,
} from '../presentation/workbench';
import type { DashboardSummaryResponse } from '../types/dashboard';

defineOptions({ name: 'DashboardHomePage' });

const logger = createLogger('dashboard.home');
const router = useRouter();
const permissionStore = usePermissionStore();
const initialTimestamp = new Date().toISOString();
const loading = ref(false);
const errorMessage = ref('');
const summary = ref<DashboardSummaryResponse | null>(null);
const lastUpdatedAt = ref('');
const retryingPresentationItemId = ref('');
const quickActionConfig = ref<DashboardQuickActionConfig>({ ...DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG });
const containerResourceState = ref<DashboardResourceLoadState>('hidden');
const dashboardPageActive = ref(false);
let dashboardContainerRealtimeSubscribed = false;

const emptyPresentation = projectWorkbenchScenario({
  generatedAt: initialTimestamp,
  operational: { enabledModules: 0, failedTasks: 0, highRiskEvents: 0 },
  items: [],
  quickActions: [],
});
const quickLinks = computed(() =>
  buildDashboardQuickActionLinks(permissionStore.routers, currentLocale.value as SupportedLocale),
);
const { rankedLinks, recordAccess } = useDashboardQuickActions(
  () => quickLinks.value,
  () => quickActionConfig.value,
);
const canViewContainerOverview = computed(() => permissionStore.hasPermission(CONTAINER_PERMISSION_CODE.VIEW));
const containerDashboardSummary = computed<ContainerDashboardSummary>(
  () => selectContainerDashboardSummaryView() ?? emptyContainerDashboardSummary(),
);
const presentation = computed(() => {
  if (!summary.value) {
    return emptyPresentation;
  }

  return projectDashboardSummaryToWorkbench({
    generatedAt: lastUpdatedAt.value || initialTimestamp,
    quickActionConfig: quickActionConfig.value,
    rankedQuickLinks: rankedLinks.value,
    resources: {
      route: CONTAINER_ROUTE_PATH.LIST,
      state: canViewContainerOverview.value ? containerResourceState.value : 'hidden',
      summary: containerDashboardSummary.value,
    },
    summary: summary.value,
  });
});

onMounted(() => {
  dashboardPageActive.value = true;
  void loadSummary();
});

onUnmounted(() => {
  dashboardPageActive.value = false;
  releaseDashboardContainerRealtimeSubscription();
});

onActivated(() => {
  dashboardPageActive.value = true;
  acquireDashboardContainerRealtimeSubscription();
});

onDeactivated(() => {
  dashboardPageActive.value = false;
  releaseDashboardContainerRealtimeSubscription();
});

async function loadSummary() {
  if (loading.value) {
    return;
  }
  loading.value = true;
  errorMessage.value = '';

  try {
    const [response] = await Promise.all([
      getDashboardSummary(),
      loadQuickActionConfig(),
      loadDashboardContainerResources(),
    ]);
    summary.value = response;
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
    quickActionConfig.value = resolveDashboardQuickActionConfig(response.items ?? [], {
      onInvalidConfigValue: ({ key, error }) => {
        logger.warn('dashboard quick-action config value parse failed', { key, error });
      },
    });
  } catch (error) {
    logger.error('dashboard quick-action config request failed', error);
    quickActionConfig.value = { ...DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG };
  }
}

async function loadDashboardContainerResources() {
  if (!canViewContainerOverview.value) {
    containerResourceState.value = 'hidden';
    releaseDashboardContainerRealtimeSubscription();
    clearContainerDashboardSummary();
    return;
  }

  containerResourceState.value = 'loading';
  try {
    const nextSummary = await containerModuleFacades.getContainerDashboardSummary();
    seedContainerDashboardSummary(nextSummary);
    containerResourceState.value = 'loaded';
    acquireDashboardContainerRealtimeSubscription();
  } catch (error) {
    logger.warn('dashboard container resource seed request failed', error);
    containerResourceState.value = 'failed';
    if (shouldResetContainerRealtimeState(error)) {
      releaseDashboardContainerRealtimeSubscription();
    }
    clearContainerDashboardSummary();
  }
}

function emptyContainerDashboardSummary(): ContainerDashboardSummary {
  return {
    overview: {
      abnormalContainers: 0,
      collectedAt: null,
      cpuTotalPercent: 0,
      memoryTotalLimitBytes: null,
      memoryTotalPercent: null,
      memoryTotalUsageBytes: null,
      runningContainers: 0,
    },
    hotspots: { cpu: [], memory: [] },
    anomalies: [],
  };
}

function acquireDashboardContainerRealtimeSubscription() {
  if (
    !dashboardPageActive.value ||
    !canViewContainerOverview.value ||
    dashboardContainerRealtimeSubscribed ||
    !selectContainerDashboardSummaryView()
  ) {
    return;
  }
  dashboardContainerRealtimeSubscribed = true;
  acquireContainerDashboardSummarySubscription();
}

function releaseDashboardContainerRealtimeSubscription() {
  if (!dashboardContainerRealtimeSubscribed) {
    return;
  }
  dashboardContainerRealtimeSubscribed = false;
  releaseContainerDashboardSummarySubscription();
}

function navigate(route: string, source: WorkbenchNavigationSource) {
  if (source === 'quick-entry') {
    recordAccess(route);
  }
  void router.push(route);
}

async function retryItem(item: PresentationItem) {
  if (!item.sourceWidgetId || retryingPresentationItemId.value) {
    return;
  }

  retryingPresentationItemId.value = item.id;
  try {
    const widget = await getDashboardWidget(item.sourceWidgetId);
    if (summary.value) {
      summary.value = {
        ...summary.value,
        widgets: summary.value.widgets.map((current) => (current.id === widget.id ? widget : current)),
      };
    }
  } catch (error) {
    logger.error('dashboard widget refresh failed', error);
    if (summary.value) {
      summary.value = {
        ...summary.value,
        widgets: summary.value.widgets.map((current) =>
          current.id === item.sourceWidgetId
            ? {
                ...current,
                status: 'error',
                error: {
                  code: requestErrorCode(error),
                  message_key: requestErrorMessageKey(error),
                  message: requestErrorMessage(error, t('dashboard.widget.errorFallback')),
                },
              }
            : current,
        ),
      };
    }
  } finally {
    retryingPresentationItemId.value = '';
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

function shouldResetContainerRealtimeState(error: unknown) {
  if (!isApiRequestError(error)) {
    return false;
  }
  return (
    error.status === 401 ||
    error.status === 403 ||
    error.code === API_CODE.AUTH_FORBIDDEN ||
    error.code === API_CODE.AUTH_MISSING_PERMISSION
  );
}
</script>
