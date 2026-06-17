<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="refresh-control-bar" data-refresh-control-bar="true">
    <t-tag
      v-if="statusLabel"
      class="refresh-control-bar__status"
      :theme="statusTheme"
      variant="light"
      size="small"
      data-refresh-status-label="true"
    >
      {{ statusLabel }}
    </t-tag>

    <div class="refresh-control-bar__field">
      <span class="refresh-control-bar__label">{{ intervalLabel }}</span>
      <t-select
        class="refresh-control-bar__select"
        :model-value="interval"
        :options="intervalOptions"
        size="small"
        :disabled="disabled"
        data-refresh-interval-select="true"
        @update:model-value="handleIntervalChange"
      />
    </div>

    <div v-if="showTrendWindow" class="refresh-control-bar__field">
      <span class="refresh-control-bar__label">{{ trendWindowLabel }}</span>
      <t-select
        class="refresh-control-bar__select"
        :model-value="trendWindow"
        :options="trendWindowOptions"
        size="small"
        :disabled="disabled"
        data-refresh-trend-window-select="true"
        @update:model-value="handleTrendWindowChange"
      />
    </div>

    <span v-if="lastUpdatedAt" class="refresh-control-bar__updated">
      {{ lastUpdatedAt }}
    </span>

    <t-tag
      v-if="showCountdownStatus"
      class="refresh-control-bar__countdown"
      :theme="countdownTheme"
      variant="light"
      size="small"
      data-refresh-countdown="true"
    >
      <span v-if="countdownStatusLabel" class="refresh-control-bar__countdown-label">
        {{ countdownStatusLabel }}
      </span>
      <span class="refresh-control-bar__countdown-value">{{ countdownStatusValue }}</span>
    </t-tag>

    <t-button
      class="refresh-control-bar__button"
      theme="primary"
      size="small"
      :loading="refreshing"
      :disabled="disabled"
      data-refresh-now="true"
      @click="emit('refresh')"
    >
      <template #icon>
        <refresh-icon />
      </template>
      {{ refreshLabel }}
    </t-button>

    <t-button
      v-if="autoRefreshEnabled"
      class="refresh-control-bar__button"
      theme="default"
      variant="outline"
      size="small"
      :disabled="disabled"
      data-refresh-toggle-auto="true"
      @click="handleAutoRefreshClick"
    >
      {{ paused ? resumeLabel : pauseLabel }}
    </t-button>
  </div>
</template>
<script setup lang="ts">
import { RefreshIcon } from 'tdesign-icons-vue-next';
import { computed } from 'vue';

import { formatRefreshCountdown } from './countdown';
import type { RefreshControlOption, RefreshControlValue } from './types';

type StatusTone = 'healthy' | 'success' | 'warning' | 'danger' | 'error' | 'disabled' | 'unknown' | 'default';

const props = withDefaults(
  defineProps<{
    autoRefreshEnabled: boolean;
    interval: RefreshControlValue;
    intervalOptions: RefreshControlOption[];
    refreshing?: boolean;
    disabled?: boolean;
    showTrendWindow?: boolean;
    trendWindow?: RefreshControlValue;
    trendWindowOptions?: RefreshControlOption[];
    statusLabel?: string;
    status?: StatusTone;
    lastUpdatedAt?: string;
    intervalLabel?: string;
    trendWindowLabel?: string;
    countdownSeconds?: number | null;
    showCountdown?: boolean;
    paused?: boolean;
    countdownLabel?: string;
    manualLabel?: string;
    pausedLabel?: string;
    refreshLabel?: string;
    pauseLabel?: string;
    resumeLabel?: string;
  }>(),
  {
    countdownLabel: '',
    countdownSeconds: null,
    disabled: false,
    intervalLabel: '',
    lastUpdatedAt: '',
    manualLabel: '',
    pauseLabel: '',
    paused: false,
    pausedLabel: '',
    refreshLabel: '',
    refreshing: false,
    resumeLabel: '',
    showCountdown: false,
    showTrendWindow: false,
    status: 'default',
    statusLabel: '',
    trendWindow: undefined,
    trendWindowLabel: '',
    trendWindowOptions: () => [],
  },
);

const emit = defineEmits<{
  'update:interval': [value: RefreshControlValue];
  'update:trendWindow': [value: RefreshControlValue];
  refresh: [];
  pause: [];
  resume: [];
}>();

const statusTheme = computed(() => {
  if (props.status === 'healthy' || props.status === 'success') return 'success';
  if (props.status === 'warning') return 'warning';
  if (props.status === 'danger' || props.status === 'error') return 'danger';
  return 'default';
});

const showCountdownStatus = computed(() => props.showCountdown);

const countdownTheme = computed(() => {
  if (!props.autoRefreshEnabled || props.paused) {
    return 'default';
  }
  return 'primary';
});

const countdownStatusLabel = computed(() => {
  if (!props.autoRefreshEnabled || props.paused) {
    return '';
  }
  return props.countdownLabel;
});

const countdownStatusValue = computed(() => {
  if (!props.autoRefreshEnabled) {
    return props.manualLabel;
  }
  if (props.paused) {
    return props.pausedLabel;
  }
  return formatRefreshCountdown(props.countdownSeconds);
});

function handleIntervalChange(value: RefreshControlValue) {
  const nextValue = resolveOptionValue(value, props.intervalOptions);
  if (nextValue !== undefined) {
    emit('update:interval', nextValue);
  }
}

function handleTrendWindowChange(value: RefreshControlValue) {
  const nextValue = resolveOptionValue(value, props.trendWindowOptions);
  if (nextValue !== undefined) {
    emit('update:trendWindow', nextValue);
  }
}

function handleAutoRefreshClick() {
  if (props.paused) {
    emit('resume');
    return;
  }
  emit('pause');
}

function resolveOptionValue(value: RefreshControlValue, options: RefreshControlOption[]) {
  const directMatch = options.find((option) => option.value === value);
  if (directMatch) {
    return directMatch.value;
  }

  if (typeof value !== 'string') {
    return undefined;
  }

  return options.find((option) => typeof option.value === 'number' && String(option.value) === value)?.value;
}
</script>
<style scoped lang="less">
.refresh-control-bar {
  align-items: center;
  display: flex;
  flex-wrap: nowrap;
  gap: var(--graft-density-gap-10) var(--graft-density-gap-12);
  justify-content: flex-end;
  max-width: 100%;
  min-width: 0;
}

.refresh-control-bar__status,
.refresh-control-bar__countdown,
.refresh-control-bar__button {
  flex: 0 0 auto;
  white-space: nowrap;
}

.refresh-control-bar__field {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.refresh-control-bar__label,
.refresh-control-bar__updated {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  white-space: nowrap;
}

.refresh-control-bar__updated {
  color: var(--td-text-color-placeholder);
}

.refresh-control-bar__countdown {
  justify-content: center;
  min-width: 116px;
}

.refresh-control-bar__countdown-label {
  color: var(--td-text-color-secondary);
  margin-inline-end: var(--graft-density-gap-4);
}

.refresh-control-bar__countdown-value {
  color: var(--td-text-color-primary);
  font-variant-numeric: tabular-nums;
}

.refresh-control-bar__select {
  width: 124px;
}

@media (width <= 1279px) {
  .refresh-control-bar {
    flex-wrap: wrap;
  }
}

@media (width <= 991px) {
  .refresh-control-bar {
    justify-content: flex-start;
  }
}

@media (width <= 767px) {
  .refresh-control-bar {
    align-items: stretch;
  }

  .refresh-control-bar__field {
    flex-wrap: wrap;
  }

  .refresh-control-bar__select {
    width: min(100%, 168px);
  }

  .refresh-control-bar__countdown {
    min-width: 72px;
  }

  .refresh-control-bar__countdown-label {
    display: none;
  }
}
</style>
