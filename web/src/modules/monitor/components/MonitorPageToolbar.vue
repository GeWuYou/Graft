<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <refresh-control-bar
    :status="refreshControlStatus"
    :countdown-seconds="remainingRefreshSeconds"
    :interval="refreshIntervalValue"
    :interval-options="refreshIntervalOptions"
    :refreshing="loading"
    :show-countdown="true"
    :show-trend-window="false"
    :status-tone="status"
    :status-label="statusLabel"
    variant="page"
    @refresh="$emit('refresh')"
    @pause="$emit('toggle-auto-refresh')"
    @resume="$emit('toggle-auto-refresh')"
    @update:interval="$emit('update:refresh-interval-value', $event)"
  />
</template>
<script setup lang="ts">
import { RefreshControlBar, type RefreshControlStatus } from '@/shared/components/refresh';

import type { RefreshIntervalOption } from '../composables/use-monitor-refresh-preferences';
import type { ServerStatusTone } from './server-status-ui';

defineProps<{
  refreshControlStatus: RefreshControlStatus;
  remainingRefreshSeconds: number | null;
  loading: boolean;
  refreshIntervalOptions: RefreshIntervalOption[];
  refreshIntervalValue: number | string;
  status: ServerStatusTone;
  statusLabel: string;
}>();

defineEmits<{
  refresh: [];
  'toggle-auto-refresh': [];
  'update:refresh-interval-value': [value: number | string];
}>();
</script>
