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
import { useQuery } from '@tanstack/vue-query';
import { computed, onActivated, onDeactivated, onMounted, onUnmounted, ref, shallowReactive, watch } from 'vue';
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
import { SYSTEM_CONFIG_PERMISSION_CODE } from '@/modules/system-config/contract/permissions';
import { useSystemConfigsQuery } from '@/modules/system-config/contract/queries';
import { queryClient } from '@/shared/query';
import { usePermissionStore } from '@/store/modules/permission';
import type { ApiRequestError } from '@/types/axios';
import { createLogger } from '@/utils/logger';

import { getDashboardSummary, getDashboardWidget } from '../api/dashboard';
import DashboardWorkbench from '../components/workbench/DashboardWorkbench.vue';
import { useDashboardQuickActions } from '../composables/use-dashboard-quick-actions';
import { DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG, resolveDashboardQuickActionConfig } from '../contract/quick-actions';
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
import type { DashboardWidgetResponse } from '../types/dashboard';

defineOptions({ name: 'DashboardHomePage' });

const logger = createLogger('dashboard.home');
const router = useRouter();
const permissionStore = usePermissionStore();
const summaryQueryKey = ['dashboard', 'summary'] as const;
const containerSummaryQueryKey = ['dashboard', 'container-summary'] as const;
const initialTimestamp = new Date().toISOString();
const dashboardPageActive = ref(false);
const retryingPresentationItemId = ref('');
const retriedWidgets = shallowReactive(new Map<string, DashboardWidgetResponse>());
let dashboardContainerRealtimeSubscribed = false;

const canViewContainerOverview = computed(() => permissionStore.hasPermission(CONTAINER_PERMISSION_CODE.VIEW));
const canReadSystemConfig = computed(() => permissionStore.hasPermission(SYSTEM_CONFIG_PERMISSION_CODE.READ));
const summaryQuery = useQuery(
  {
    queryKey: summaryQueryKey,
    queryFn: async () => ({ summary: await getDashboardSummary() }),
    enabled: dashboardPageActive,
    retry: false,
    // 每次成功返回都是新的摘要快照，即使响应字段与上一次完全相同。
    structuralSharing: false,
  },
  queryClient,
);
const configQuery = useSystemConfigsQuery(computed(() => dashboardPageActive.value && canReadSystemConfig.value));
const containerQuery = useQuery(
  {
    queryKey: containerSummaryQueryKey,
    queryFn: containerModuleFacades.getContainerDashboardSummary,
    enabled: computed(() => dashboardPageActive.value && canViewContainerOverview.value),
    retry: false,
  },
  queryClient,
);
const summary = computed(() => summaryQuery.data.value?.summary);
const loading = computed(
  () => summaryQuery.isFetching.value || configQuery.isFetching.value || containerQuery.isFetching.value,
);
const errorMessage = computed(() =>
  summaryQuery.error.value ? requestErrorMessage(summaryQuery.error.value, t('dashboard.error.fallback')) : '',
);
const lastUpdatedAt = computed(() =>
  summaryQuery.dataUpdatedAt.value ? new Date(summaryQuery.dataUpdatedAt.value).toISOString() : '',
);
const quickActionConfig = computed(() => {
  if (!canReadSystemConfig.value || configQuery.isError.value || !configQuery.data.value) {
    return DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG;
  }
  return resolveDashboardQuickActionConfig(configQuery.data.value.items ?? [], {
    onInvalidConfigValue: ({ key, error }) => {
      logger.warn('dashboard quick-action config value parse failed', { key, error });
    },
  });
});
const containerResourceState = computed<DashboardResourceLoadState>(() => {
  if (!canViewContainerOverview.value) return 'hidden';
  if (containerQuery.isFetching.value) return 'loading';
  if (containerQuery.isError.value) return 'failed';
  return containerQuery.data.value ? 'loaded' : 'loading';
});

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
const containerDashboardSummary = computed<ContainerDashboardSummary>(
  () => selectContainerDashboardSummaryView() ?? emptyContainerDashboardSummary(),
);
const presentation = computed(() => {
  const snapshot = summary.value;
  if (!snapshot) {
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
    summary: retriedWidgets.size
      ? { ...snapshot, widgets: snapshot.widgets.map((widget) => retriedWidgets.get(widget.id) ?? widget) }
      : snapshot,
  });
});

onMounted(activateDashboard);
onActivated(activateDashboard);
onDeactivated(deactivateDashboard);
onUnmounted(deactivateDashboard);

// 单组件重试只修正展示，整页摘要及其请求状态仍由摘要 Query 持有。
watch(summaryQuery.data, () => retriedWidgets.clear(), { flush: 'sync' });
watch(summaryQuery.error, (error) => {
  if (error) logger.error('dashboard summary request failed', error);
});
watch(configQuery.error, (error) => {
  if (error && canReadSystemConfig.value) logger.error('dashboard quick-action config request failed', error);
});
// 只有页面活跃且有权限时才把 HTTP 缓存接入容器实时投影。
watch(
  [dashboardPageActive, canViewContainerOverview, containerQuery.data, containerQuery.error],
  ([active, permitted, snapshot, error]) => {
    if (!permitted) {
      releaseDashboardContainerRealtimeSubscription();
      clearContainerDashboardSummary();
    } else if (active && error) {
      logger.warn('dashboard container resource seed request failed', error);
      if (shouldResetContainerRealtimeState(error)) releaseDashboardContainerRealtimeSubscription();
      clearContainerDashboardSummary();
    } else if (active && snapshot) {
      seedContainerDashboardSummary(snapshot);
      acquireDashboardContainerRealtimeSubscription();
    }
  },
  { immediate: true },
);

function activateDashboard() {
  dashboardPageActive.value = true;
}

function deactivateDashboard() {
  dashboardPageActive.value = false;
  releaseDashboardContainerRealtimeSubscription();
}

function loadSummary() {
  if (!dashboardPageActive.value || loading.value) return;
  void summaryQuery.refetch();
  if (canReadSystemConfig.value) void configQuery.refetch();
  if (canViewContainerOverview.value) void containerQuery.refetch();
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
  const widgetId = item.sourceWidgetId;
  const snapshot = summaryQuery.data.value;
  if (!widgetId || !snapshot || retryingPresentationItemId.value) {
    return;
  }

  const previous = retriedWidgets.get(widgetId) ?? snapshot.summary.widgets.find((widget) => widget.id === widgetId);
  if (!previous) return;

  retryingPresentationItemId.value = item.id;
  try {
    const widget = await queryClient.fetchQuery({
      queryKey: ['dashboard', 'widget', widgetId],
      queryFn: () => getDashboardWidget(widgetId),
      staleTime: 0,
      retry: false,
    });
    if (summaryQuery.data.value === snapshot) retriedWidgets.set(widget.id, widget);
  } catch (error) {
    logger.error('dashboard widget refresh failed', error);
    if (summaryQuery.data.value === snapshot) {
      retriedWidgets.set(widgetId, {
        ...previous,
        status: 'error',
        error: {
          code: requestErrorCode(error),
          message_key: requestErrorMessageKey(error),
          message: requestErrorMessage(error, t('dashboard.widget.errorFallback')),
        },
      });
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
