import type { Ref } from 'vue';
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import type { RefreshControlStatus } from '@/shared/components/refresh';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';
import { useRealtimeSchedulerStore } from '@/store';

import { getServerStatus } from '../api/server-status';
import { useMonitorRefreshPreferences } from '../composables/use-monitor-refresh-preferences';
import { MONITOR_TREND_RANGE } from '../contract/trend';
import type { ServerStatusConnectionPool } from '../types/server-status';
import type { ServerStatusResponse } from '../types/server-status';

export type DependencyDisplayStatus = 'healthy' | 'abnormal' | 'notConfigured' | 'unknown';

/**
 * 创建服务器状态快照的响应式状态与自动刷新调度。
 *
 * @returns 包含服务器状态、加载状态、自动刷新配置、倒计时信息以及手动刷新方法的响应式对象。
 */
export function useServerStatusSnapshot() {
  const { t } = useI18n();
  const realtimeSchedulerStore = useRealtimeSchedulerStore();
  const {
    autoRefreshEnabled,
    refreshIntervalOptions,
    selectedRefreshInterval,
    selectedRefreshIntervalLabel,
    toggleAutoRefresh: toggleSharedAutoRefresh,
  } = useMonitorRefreshPreferences();

  const loading = ref(false);
  const initialized = ref(false);
  const errorMessage = ref('');
  const isPageVisible = ref(typeof document === 'undefined' ? true : document.visibilityState === 'visible');
  const remainingRefreshSeconds = ref<number | null>(null);
  const serverStatus = ref<ServerStatusResponse | null>(null);
  const consecutiveFailures = ref(0);

  let nextRefreshAt: number | null = null;
  let refreshTickTimer: number | null = null;

  async function refreshSnapshot() {
    // 先停止倒计时，避免手动刷新或重试与上一轮定时器并发发起请求。
    stopRefreshTick();

    if (loading.value) {
      return;
    }

    loading.value = true;
    errorMessage.value = '';

    try {
      serverStatus.value = await getServerStatus(MONITOR_TREND_RANGE.TEN_MINUTES);
      consecutiveFailures.value = 0;
    } catch (error) {
      consecutiveFailures.value += 1;
      errorMessage.value = resolveLocalizedErrorMessage(t, error, t('monitor.shared.loadFailed'));
    } finally {
      loading.value = false;
      initialized.value = true;
      scheduleNextRefresh();
    }
  }

  const refreshCountdownText = computed(() => {
    if (selectedRefreshInterval.value <= 0) {
      return t('app.refreshControl.status.off');
    }

    if (!autoRefreshEnabled.value) {
      return t('monitor.serverStatus.nextRefreshPausedByUser');
    }

    if (!isPageVisible.value) {
      return t('monitor.serverStatus.nextRefreshPaused');
    }

    if (remainingRefreshSeconds.value === null) {
      return t('monitor.serverStatus.nextRefreshPending');
    }

    if (consecutiveFailures.value > 0) {
      return t('monitor.serverStatus.nextRefreshRetryIn', {
        seconds: String(remainingRefreshSeconds.value),
        interval: selectedRefreshIntervalLabel.value,
      });
    }

    return t('monitor.serverStatus.nextRefreshIn', {
      seconds: String(remainingRefreshSeconds.value),
    });
  });

  const refreshControlStatus = computed<RefreshControlStatus>(() => {
    if (selectedRefreshInterval.value <= 0) {
      return 'off';
    }

    if (!autoRefreshEnabled.value) {
      return 'paused';
    }

    if (!isPageVisible.value) {
      return 'paused';
    }

    return 'running';
  });

  function canRunAutoRefreshCycle() {
    return (
      autoRefreshEnabled.value &&
      isPageVisible.value &&
      selectedRefreshInterval.value > 0 &&
      realtimeSchedulerStore.allowPolling
    );
  }

  function clearRefreshSchedule() {
    stopRefreshTick();
    remainingRefreshSeconds.value = null;
  }

  function handleVisibilityChange() {
    isPageVisible.value = document.visibilityState === 'visible';

    if (canRunAutoRefreshCycle()) {
      void refreshSnapshot();
      return;
    }

    clearRefreshSchedule();
  }

  onMounted(() => {
    if (realtimeSchedulerStore.allowPolling) {
      void refreshSnapshot();
    } else {
      clearRefreshSchedule();
    }
    document.addEventListener('visibilitychange', handleVisibilityChange, false);
  });

  onUnmounted(() => {
    stopRefreshTick();
    document.removeEventListener('visibilitychange', handleVisibilityChange);
  });

  watch(selectedRefreshInterval, () => {
    scheduleNextRefresh();
  });

  watch(
    () => realtimeSchedulerStore.allowPolling,
    (allowPolling) => {
      if (!allowPolling) {
        clearRefreshSchedule();
        return;
      }

      if (!initialized.value && !serverStatus.value) {
        void refreshSnapshot();
        return;
      }
      scheduleNextRefresh();
    },
  );

  function scheduleNextRefresh() {
    stopRefreshTick();

    if (!canRunAutoRefreshCycle()) {
      remainingRefreshSeconds.value = null;
      return;
    }

    // 服务端连续失败时扩大下一次间隔，并限制上限，避免故障期间形成请求风暴。
    const backoffMultiplier = consecutiveFailures.value > 0 ? 2 ** consecutiveFailures.value : 1;
    const delaySeconds = Math.min(selectedRefreshInterval.value * backoffMultiplier, 5 * 60);
    nextRefreshAt = Date.now() + delaySeconds * 1000;
    updateRemainingRefreshSeconds();

    refreshTickTimer = window.setInterval(() => {
      updateRemainingRefreshSeconds();

      if (remainingRefreshSeconds.value === 0) {
        void refreshSnapshot();
      }
    }, 1000);
  }

  function stopRefreshTick() {
    if (refreshTickTimer !== null) {
      window.clearInterval(refreshTickTimer);
      refreshTickTimer = null;
    }

    nextRefreshAt = null;
  }

  function toggleAutoRefresh() {
    toggleSharedAutoRefresh();

    if (canRunAutoRefreshCycle()) {
      void refreshSnapshot();
      return;
    }

    clearRefreshSchedule();
  }

  function updateRemainingRefreshSeconds() {
    if (nextRefreshAt === null) {
      remainingRefreshSeconds.value = null;
      return;
    }

    remainingRefreshSeconds.value = Math.max(0, Math.ceil((nextRefreshAt - Date.now()) / 1000));
  }

  return {
    autoRefreshEnabled,
    loading,
    initialized,
    errorMessage,
    refreshCountdownText,
    remainingRefreshSeconds,
    refreshControlStatus,
    refreshIntervalOptions,
    selectedRefreshInterval,
    serverStatus,
    refreshSnapshot,
    observedAt: computed(() => serverStatus.value?.observed_at ?? ''),
    toggleAutoRefresh,
  };
}

export function normalizeDependencyStatus(status?: string): DependencyDisplayStatus {
  switch ((status ?? '').trim().toLowerCase()) {
    case 'healthy':
      return 'healthy';
    case 'degraded':
      return 'abnormal';
    case 'disabled':
      return 'notConfigured';
    default:
      return 'unknown';
  }
}

export function formatBytes(bytes?: number | null) {
  if (!Number.isFinite(bytes) || bytes === null || bytes === undefined || bytes < 0) {
    return '--';
  }

  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let value = bytes;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  const digits = value >= 100 || unitIndex === 0 ? 0 : value >= 10 ? 1 : 2;
  return `${value.toFixed(digits)} ${units[unitIndex]}`;
}

export function formatTimestamp(value?: string | null, locale?: string | Ref<string | undefined> | null) {
  const formatted = formatLocaleDateTime(value, locale);
  return formatted === '-' ? '--' : formatted;
}

export function formatUptime(totalSeconds?: number | null) {
  if (!Number.isFinite(totalSeconds) || totalSeconds === null || totalSeconds === undefined || totalSeconds < 0) {
    return '--';
  }

  const remainingSeconds = Math.floor(totalSeconds);
  const days = Math.floor(remainingSeconds / 86400);
  const hours = Math.floor((remainingSeconds % 86400) / 3600);
  const minutes = Math.floor((remainingSeconds % 3600) / 60);
  const seconds = remainingSeconds % 60;
  const parts = [
    days > 0 ? `${days}d` : '',
    hours > 0 ? `${hours}h` : '',
    minutes > 0 ? `${minutes}m` : '',
    seconds > 0 || (days === 0 && hours === 0 && minutes === 0) ? `${seconds}s` : '',
  ].filter(Boolean);

  return parts.join(' ');
}

export function formatLatency(latencyMs?: number | null) {
  if (!Number.isFinite(latencyMs) || latencyMs === null || latencyMs === undefined) {
    return '--';
  }

  return `${latencyMs.toFixed(2)} ms`;
}

export function formatPoolWait(pool?: ServerStatusConnectionPool | null) {
  if (!pool) {
    return '--';
  }

  return `${pool.wait_count} · ${pool.wait_duration_ms.toFixed(2)} ms`;
}

export function displayText(value?: string | null) {
  if (!value) {
    return '--';
  }

  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : '--';
}
