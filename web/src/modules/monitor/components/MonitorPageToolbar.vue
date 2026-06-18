<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <refresh-control-bar
    :auto-refresh-enabled="refreshControlAutoRefreshEnabled"
    :interval="refreshIntervalValue"
    :interval-label="refreshIntervalLabel"
    :interval-options="refreshIntervalOptions"
    :paused="refreshControlPaused"
    :pause-label="pauseAutoRefreshLabel"
    :refresh-label="refreshNowLabel"
    :refreshing="loading"
    :resume-label="resumeAutoRefreshLabel"
    :show-trend-window="false"
    :status="status"
    :status-label="statusLabel"
    @refresh="$emit('refresh')"
    @pause="$emit('toggle-auto-refresh')"
    @resume="$emit('toggle-auto-refresh')"
    @update:interval="$emit('update:refresh-interval-value', $event)"
  />
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { RefreshControlBar } from '@/shared/components/refresh';

import type { RefreshIntervalOption } from '../composables/use-monitor-refresh-preferences';
import type { ServerStatusTone } from './server-status-ui';

const props = defineProps<{
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

const refreshControlAutoRefreshEnabled = computed(() => Number(props.refreshIntervalValue) > 0);
const refreshControlPaused = computed(() => refreshControlAutoRefreshEnabled.value && !props.autoRefreshEnabled);
</script>
