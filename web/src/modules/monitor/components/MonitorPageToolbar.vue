<template>
  <monitor-toolbar
    :auto-refresh-enabled="autoRefreshEnabled"
    :loading="loading"
    :pause-auto-refresh-label="pauseAutoRefreshLabel"
    :refresh-interval-label="refreshIntervalLabel"
    :refresh-interval-options="refreshIntervalOptions"
    :refresh-interval-value="refreshIntervalValue"
    :refresh-now-label="refreshNowLabel"
    :resume-auto-refresh-label="resumeAutoRefreshLabel"
    :show-trend-range="false"
    :status="status"
    :status-label="statusLabel"
    :trend-range-label-placeholder="trendRangeLabelPlaceholder"
    @refresh="$emit('refresh')"
    @toggle-auto-refresh="$emit('toggle-auto-refresh')"
    @update:refresh-interval-value="$emit('update:refresh-interval-value', $event)"
  />
</template>
<script setup lang="ts">
import type { RefreshIntervalOption } from '../composables/use-monitor-refresh-preferences';
import MonitorToolbar from './MonitorToolbar.vue';
import type { ServerStatusTone } from './server-status-ui';

defineProps<{
  autoRefreshEnabled: boolean;
  loading: boolean;
  pauseAutoRefreshLabel: string;
  refreshIntervalLabel: string;
  refreshIntervalOptions: RefreshIntervalOption[];
  refreshIntervalValue: number | string;
  refreshNowLabel: string;
  resumeAutoRefreshLabel: string;
  status: ServerStatusTone;
  statusLabel: string;
  trendRangeLabelPlaceholder: string;
}>();

defineEmits<{
  refresh: [];
  'toggle-auto-refresh': [];
  'update:refresh-interval-value': [value: number | string];
}>();
</script>
