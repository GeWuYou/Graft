import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { MONITOR_REFRESH_INTERVAL, type MonitorRefreshInterval } from '../contract/refresh';

type RefreshIntervalOption = {
  label: string;
  value: MonitorRefreshInterval;
};

const selectedRefreshInterval = ref<MonitorRefreshInterval>(MONITOR_REFRESH_INTERVAL.FIVE_SECONDS);
const autoRefreshEnabled = ref(true);

export function useMonitorRefreshPreferences() {
  const { t } = useI18n();

  const refreshIntervalOptions = computed<RefreshIntervalOption[]>(() => [
    {
      label: t('monitor.serverStatus.refreshInterval5Seconds'),
      value: MONITOR_REFRESH_INTERVAL.FIVE_SECONDS,
    },
    {
      label: t('monitor.serverStatus.refreshInterval10Seconds'),
      value: MONITOR_REFRESH_INTERVAL.TEN_SECONDS,
    },
    {
      label: t('monitor.serverStatus.refreshInterval30Seconds'),
      value: MONITOR_REFRESH_INTERVAL.THIRTY_SECONDS,
    },
    {
      label: t('monitor.serverStatus.refreshInterval1Minute'),
      value: MONITOR_REFRESH_INTERVAL.ONE_MINUTE,
    },
  ]);

  const selectedRefreshIntervalLabel = computed(() => {
    return refreshIntervalOptions.value.find((option) => option.value === selectedRefreshInterval.value)?.label ?? '--';
  });

  function setRefreshInterval(value: MonitorRefreshInterval) {
    selectedRefreshInterval.value = value;
  }

  function toggleAutoRefresh() {
    autoRefreshEnabled.value = !autoRefreshEnabled.value;
  }

  return {
    autoRefreshEnabled,
    refreshIntervalOptions,
    selectedRefreshInterval,
    selectedRefreshIntervalLabel,
    setRefreshInterval,
    toggleAutoRefresh,
  };
}
