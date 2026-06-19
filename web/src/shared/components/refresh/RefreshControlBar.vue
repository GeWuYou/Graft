<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    :class="['refresh-control-bar', `refresh-control-bar--${variant}`, `refresh-control-bar--${appearance}`]"
    data-refresh-control-bar="true"
    :data-refresh-variant="variant"
    :data-refresh-appearance="appearance"
  >
    <div class="refresh-control-bar__summary">
      <div v-if="showHealthStatus" class="refresh-control-bar__status" data-refresh-status-label="true">
        <span class="refresh-control-bar__status-dot" :data-tone="statusTheme" aria-hidden="true" />
        <span class="refresh-control-bar__status-text">{{ statusLabel }}</span>
      </div>

      <div class="refresh-control-bar__items">
        <div class="refresh-control-bar__item refresh-control-bar__item--interval">
          <template v-if="showIntervalSelect">
            <span class="refresh-control-bar__label">{{ t('app.refreshControl.labels.interval') }}</span>
            <span class="refresh-control-bar__chip">
              <t-select
                class="refresh-control-bar__select refresh-control-bar__select--compact"
                :model-value="interval"
                :options="intervalOptions"
                auto-width
                borderless
                size="small"
                :disabled="disabled"
                data-refresh-interval-select="true"
                @update:model-value="handleIntervalChange"
              />
            </span>
          </template>
          <span v-else class="refresh-control-bar__value" data-refresh-auto-state="true">{{
            autoRefreshSummaryText
          }}</span>
        </div>

        <div v-if="showTrendWindow" class="refresh-control-bar__item">
          <span class="refresh-control-bar__label">{{ resolvedTrendWindowLabel }}</span>
          <span class="refresh-control-bar__chip">
            <t-select
              class="refresh-control-bar__select refresh-control-bar__select--compact"
              :model-value="trendWindow"
              :options="trendWindowOptions"
              auto-width
              borderless
              size="small"
              :disabled="disabled"
              data-refresh-trend-window-select="true"
              @update:model-value="handleTrendWindowChange"
            />
          </span>
        </div>

        <div
          v-if="showCountdownStatus"
          class="refresh-control-bar__item refresh-control-bar__item--countdown"
          data-refresh-countdown="true"
        >
          <span class="refresh-control-bar__value refresh-control-bar__value--countdown">{{
            countdownSummaryText
          }}</span>
        </div>

        <div v-if="lastUpdatedAt" class="refresh-control-bar__item refresh-control-bar__item--muted">
          <span class="refresh-control-bar__updated">{{ lastUpdatedAt }}</span>
        </div>
      </div>
    </div>

    <div class="refresh-control-bar__actions">
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
        {{ refreshActionLabel }}
      </t-button>

      <t-button
        v-if="showAutoRefreshToggle"
        class="refresh-control-bar__button"
        theme="default"
        variant="outline"
        size="small"
        :disabled="disabled"
        data-refresh-toggle-auto="true"
        @click="handleAutoRefreshClick"
      >
        {{ toggleActionLabel }}
      </t-button>
    </div>
  </div>
</template>
<script setup lang="ts">
import { RefreshIcon } from 'tdesign-icons-vue-next';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import { formatRefreshCountdown } from './countdown';
import type { RefreshControlOption, RefreshControlValue } from './types';

type StatusTone = 'healthy' | 'success' | 'warning' | 'danger' | 'error' | 'disabled' | 'unknown' | 'default';
type RefreshStatus = 'running' | 'paused' | 'off';

const props = withDefaults(
  defineProps<{
    status: RefreshStatus;
    interval: RefreshControlValue;
    intervalOptions: RefreshControlOption[];
    refreshing?: boolean;
    disabled?: boolean;
    showTrendWindow?: boolean;
    trendWindow?: RefreshControlValue;
    trendWindowOptions?: RefreshControlOption[];
    statusLabel?: string;
    statusTone?: StatusTone;
    variant?: 'page' | 'compact';
    appearance?: 'outlined' | 'plain';
    lastUpdatedAt?: string;
    trendWindowLabel?: string;
    countdownSeconds?: number | null;
    showCountdown?: boolean;
  }>(),
  {
    countdownSeconds: null,
    disabled: false,
    lastUpdatedAt: '',
    refreshing: false,
    showCountdown: false,
    showTrendWindow: false,
    statusTone: 'default',
    statusLabel: '',
    trendWindow: undefined,
    trendWindowLabel: '',
    trendWindowOptions: () => [],
    appearance: 'outlined',
    variant: 'page',
  },
);

const emit = defineEmits<{
  'update:interval': [value: RefreshControlValue];
  'update:trendWindow': [value: RefreshControlValue];
  refresh: [];
  pause: [];
  resume: [];
}>();

const { t } = useI18n();

const statusTheme = computed(() => {
  if (props.statusTone === 'healthy' || props.statusTone === 'success') return 'success';
  if (props.statusTone === 'warning') return 'warning';
  if (props.statusTone === 'danger' || props.statusTone === 'error') return 'danger';
  return 'default';
});

const showHealthStatus = computed(() => props.variant === 'page' && Boolean(props.statusLabel));

const showIntervalSelect = computed(() => props.status === 'running');

const resolvedIntervalLabel = computed(() => resolveOptionLabel(props.interval, props.intervalOptions));
const resolvedTrendWindowLabel = computed(() => {
  if (!props.trendWindowLabel) {
    return t('app.refreshControl.labels.trendWindow');
  }
  return props.trendWindowLabel;
});

const autoRefreshSummaryText = computed(() => {
  if (props.status === 'off') {
    return t('app.refreshControl.status.off');
  }
  if (props.status === 'paused') {
    return t('app.refreshControl.status.paused');
  }
  return t('app.refreshControl.status.running', { interval: resolvedIntervalLabel.value });
});

const showCountdownStatus = computed(() => props.showCountdown && props.status === 'running');

const countdownSummaryText = computed(() => {
  if (props.countdownSeconds === null || props.countdownSeconds === undefined) {
    return t('app.refreshControl.pending');
  }

  return t('app.refreshControl.countdown', {
    countdown: formatRefreshCountdown(props.countdownSeconds),
  });
});

const toggleActionLabel = computed(() => {
  if (props.status === 'off') {
    return props.variant === 'compact'
      ? t('app.refreshControl.actions.enableCompact')
      : t('app.refreshControl.actions.enable');
  }
  if (props.status === 'paused') {
    return props.variant === 'compact'
      ? t('app.refreshControl.actions.resumeCompact')
      : t('app.refreshControl.actions.resume');
  }
  return props.variant === 'compact'
    ? t('app.refreshControl.actions.pauseCompact')
    : t('app.refreshControl.actions.pause');
});

const showAutoRefreshToggle = computed(() => Boolean(toggleActionLabel.value));
const refreshActionLabel = computed(() => t('app.refreshControl.actions.refresh'));

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
  if (props.status === 'off') {
    emit('resume');
    return;
  }
  if (props.status === 'paused') {
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

  return undefined;
}

function resolveOptionLabel(value: RefreshControlValue, options: RefreshControlOption[]) {
  return options.find((option) => option.value === value)?.label ?? String(value ?? '');
}
</script>
<style scoped lang="less">
.refresh-control-bar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-10) var(--graft-density-gap-12);
  justify-content: space-between;
  max-width: 100%;
  min-width: 0;
}

.refresh-control-bar__summary,
.refresh-control-bar__items {
  align-items: center;
  display: flex;
  flex: 1 1 auto;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-10) var(--graft-density-gap-12);
  min-width: 0;
}

.refresh-control-bar--outlined.refresh-control-bar--page {
  background: color-mix(in srgb, var(--td-bg-color-container) 92%, var(--td-bg-color-page));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  border-radius: var(--td-radius-large);
  inline-size: min(100%, 900px);
  padding: var(--graft-density-gap-10) var(--graft-density-gap-16);
}

.refresh-control-bar--compact {
  align-items: center;
  padding: 0;
}

.refresh-control-bar--compact.refresh-control-bar--plain {
  flex-wrap: nowrap;
}

.refresh-control-bar--plain {
  background: transparent;
  border: 0;
  border-radius: 0;
  box-shadow: none;
}

.refresh-control-bar__status {
  align-items: center;
  display: inline-flex;
  flex: 0 0 auto;
  gap: var(--graft-density-gap-6);
  min-height: 28px;
  white-space: nowrap;
}

.refresh-control-bar__status-dot {
  block-size: 8px;
  border-radius: var(--td-radius-circle);
  display: block;
  inline-size: 8px;
}

.refresh-control-bar__status-dot[data-tone='success'] {
  background: var(--td-success-color);
}

.refresh-control-bar__status-dot[data-tone='warning'] {
  background: var(--td-warning-color);
}

.refresh-control-bar__status-dot[data-tone='danger'] {
  background: var(--td-error-color);
}

.refresh-control-bar__status-dot[data-tone='default'] {
  background: var(--td-text-color-placeholder);
}

.refresh-control-bar__status-text {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  font-weight: 600;
}

.refresh-control-bar__item {
  align-items: center;
  display: inline-flex;
  flex: 0 1 auto;
  gap: var(--graft-density-gap-6);
  min-height: 28px;
  min-width: 0;
  white-space: nowrap;
}

.refresh-control-bar__item--muted {
  color: var(--td-text-color-placeholder);
}

.refresh-control-bar__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  white-space: nowrap;
}

.refresh-control-bar__chip {
  align-items: center;
  background: var(--td-bg-color-container);
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  border-radius: var(--td-radius-circle);
  display: inline-flex;
  min-height: 28px;
  padding: 0 var(--graft-density-gap-8);
}

.refresh-control-bar__value,
.refresh-control-bar__updated {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
  white-space: nowrap;
}

.refresh-control-bar__value--countdown {
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

.refresh-control-bar__select {
  min-width: 0;
}

.refresh-control-bar__select--compact {
  border-radius: var(--td-radius-circle);
  inline-size: auto;
  min-width: 88px;
}

.refresh-control-bar--compact .refresh-control-bar__summary {
  flex: 1 1 280px;
}

.refresh-control-bar--compact.refresh-control-bar--plain .refresh-control-bar__summary,
.refresh-control-bar--compact.refresh-control-bar--plain .refresh-control-bar__items {
  flex-wrap: nowrap;
}

.refresh-control-bar--compact.refresh-control-bar--plain .refresh-control-bar__summary {
  flex: 0 1 auto;
}

.refresh-control-bar--compact .refresh-control-bar__items {
  gap: var(--graft-density-gap-8);
}

.refresh-control-bar--compact .refresh-control-bar__chip {
  background: transparent;
  border: 0;
  min-height: 24px;
  padding: 0;
}

.refresh-control-bar--plain .refresh-control-bar__chip {
  background: transparent;
  border: 0;
  padding-inline: 0;
}

.refresh-control-bar--compact .refresh-control-bar__item--countdown::before {
  color: var(--td-text-color-placeholder);
  content: '·';
  margin-inline-end: var(--graft-density-gap-4);
}

.refresh-control-bar__actions {
  align-items: center;
  display: flex;
  flex: 0 0 auto;
  flex-wrap: nowrap;
  gap: var(--graft-density-gap-10);
  justify-content: flex-end;
  min-width: fit-content;
}

.refresh-control-bar__button {
  flex: 0 0 auto;
  white-space: nowrap;
}

@media (width <= 1279px) {
  .refresh-control-bar--page {
    inline-size: min(100%, 820px);
  }

  .refresh-control-bar--page .refresh-control-bar__summary {
    flex-basis: 100%;
  }

  .refresh-control-bar--page .refresh-control-bar__actions {
    flex-basis: 100%;
    justify-content: flex-end;
  }
}

@media (width <= 767px) {
  .refresh-control-bar--page {
    inline-size: 100%;
    padding-inline: var(--graft-density-gap-12);
  }

  .refresh-control-bar--compact.refresh-control-bar--plain {
    flex-wrap: wrap;
  }

  .refresh-control-bar--compact.refresh-control-bar--plain .refresh-control-bar__summary,
  .refresh-control-bar--compact.refresh-control-bar--plain .refresh-control-bar__items {
    flex-wrap: wrap;
  }

  .refresh-control-bar--compact.refresh-control-bar--plain .refresh-control-bar__summary {
    flex: 1 1 100%;
  }

  .refresh-control-bar__actions {
    justify-content: flex-start;
  }
}
</style>
