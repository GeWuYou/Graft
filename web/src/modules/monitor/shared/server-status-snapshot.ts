import type { Ref } from 'vue';
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import type { RefreshControlStatus } from '@/shared/components/refresh';
import { useCurrentTabRefresh } from '@/shared/composables/useTabRefresh';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';

import { getServerStatus } from '../api/server-status';
import { useMonitorRefreshPreferences } from '../composables/use-monitor-refresh-preferences';
import { MONITOR_TREND_RANGE } from '../contract/trend';
import type { ServerStatusConnectionPool } from '../types/server-status';
import type { ServerStatusResponse } from '../types/server-status';

export type DependencyDisplayStatus = 'healthy' | 'abnormal' | 'notConfigured' | 'unknown';

/**
 * Creates reactive state and automatic scheduling for fetching and refreshing server status data.
 *
 * @returns An object with reactive state for server status and loading indicators, configurable auto-refresh controls, and methods to manually trigger refreshes.
 */
export function useServerStatusSnapshot() {
  const { t } = useI18n();
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

  function handleVisibilityChange() {
    isPageVisible.value = document.visibilityState === 'visible';

    if (isPageVisible.value && autoRefreshEnabled.value && selectedRefreshInterval.value > 0) {
      void refreshSnapshot();
      return;
    }

    stopRefreshTick();
    remainingRefreshSeconds.value = null;
  }

  onMounted(() => {
    void refreshSnapshot();
    document.addEventListener('visibilitychange', handleVisibilityChange, false);
  });

  useCurrentTabRefresh(async () => {
    await refreshSnapshot();
  });

  onUnmounted(() => {
    stopRefreshTick();
    document.removeEventListener('visibilitychange', handleVisibilityChange);
  });

  watch(selectedRefreshInterval, () => {
    scheduleNextRefresh();
  });

  function scheduleNextRefresh() {
    stopRefreshTick();

    if (!autoRefreshEnabled.value || !isPageVisible.value || selectedRefreshInterval.value <= 0) {
      remainingRefreshSeconds.value = null;
      return;
    }

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

    if (autoRefreshEnabled.value && isPageVisible.value && selectedRefreshInterval.value > 0) {
      void refreshSnapshot();
      return;
    }

    stopRefreshTick();
    remainingRefreshSeconds.value = null;
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
