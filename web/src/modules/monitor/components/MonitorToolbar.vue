<template>
  <div class="monitor-toolbar" data-monitor-refresh-toolbar="true">
    <status-tag v-if="statusLabel" class="monitor-toolbar__status" :label="statusLabel" :status="status" />

    <div class="monitor-toolbar__field">
      <span class="monitor-toolbar__label">{{ refreshIntervalLabel }}</span>
      <t-select
        class="monitor-toolbar__select"
        :model-value="refreshIntervalValue"
        :options="refreshIntervalOptions"
        size="small"
        data-monitor-refresh-interval-select="true"
        @update:model-value="handleRefreshIntervalChange"
      />
    </div>

    <div v-if="showTrendRange" class="monitor-toolbar__field">
      <span class="monitor-toolbar__label">{{ trendRangeLabel }}</span>
      <t-select
        class="monitor-toolbar__select"
        :model-value="trendRangeValue"
        :options="trendRangeOptions"
        :placeholder="trendRangeLabelPlaceholder"
        size="small"
        data-monitor-refresh-extra-select="true"
        @update:model-value="handleTrendRangeChange"
      />
    </div>

    <t-button class="monitor-toolbar__button" theme="primary" size="small" :loading="loading" @click="emit('refresh')">
      <template #icon>
        <refresh-icon />
      </template>
      {{ refreshNowLabel }}
    </t-button>

    <t-button class="monitor-toolbar__button" variant="outline" size="small" @click="emit('toggle-auto-refresh')">
      {{ autoRefreshEnabled ? pauseAutoRefreshLabel : resumeAutoRefreshLabel }}
    </t-button>
  </div>
</template>
<script setup lang="ts">
import { RefreshIcon } from 'tdesign-icons-vue-next';

import type { ServerStatusTone } from './server-status-ui';
import StatusTag from './StatusTag.vue';

type ToolbarOptionValue = number | string;

type ToolbarOption = {
  label: string;
  value: ToolbarOptionValue;
};

const props = withDefaults(
  defineProps<{
    autoRefreshEnabled: boolean;
    loading?: boolean;
    pauseAutoRefreshLabel: string;
    refreshIntervalLabel: string;
    refreshIntervalOptions: ToolbarOption[];
    refreshIntervalValue: ToolbarOptionValue;
    refreshNowLabel: string;
    resumeAutoRefreshLabel: string;
    showTrendRange?: boolean;
    status?: ServerStatusTone;
    statusLabel?: string;
    trendRangeLabel?: string;
    trendRangeLabelPlaceholder?: string;
    trendRangeOptions?: ToolbarOption[];
    trendRangeValue?: ToolbarOptionValue;
  }>(),
  {
    loading: false,
    showTrendRange: false,
    status: 'unknown',
    statusLabel: '',
    trendRangeLabel: '',
    trendRangeLabelPlaceholder: '',
    trendRangeOptions: () => [],
    trendRangeValue: undefined,
  },
);

const emit = defineEmits<{
  refresh: [];
  'toggle-auto-refresh': [];
  'update:refresh-interval-value': [value: ToolbarOptionValue];
  'update:trend-range-value': [value: ToolbarOptionValue];
}>();

function handleRefreshIntervalChange(value: ToolbarOptionValue) {
  const nextValue = resolveOptionValue(value, props.refreshIntervalOptions);
  if (nextValue !== undefined) {
    emit('update:refresh-interval-value', nextValue);
  }
}

function handleTrendRangeChange(value: ToolbarOptionValue) {
  const nextValue = resolveOptionValue(value, props.trendRangeOptions);
  if (nextValue !== undefined) {
    emit('update:trend-range-value', nextValue);
  }
}

function resolveOptionValue(value: ToolbarOptionValue, options: ToolbarOption[]) {
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
.monitor-toolbar {
  align-items: center;
  display: flex;
  flex-wrap: nowrap;
  gap: 10px 12px;
  justify-content: flex-end;
  max-width: 100%;
}

.monitor-toolbar__status {
  flex: 0 0 auto;
}

.monitor-toolbar__field {
  align-items: center;
  display: inline-flex;
  gap: 8px;
  min-width: 0;
}

.monitor-toolbar__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  white-space: nowrap;
}

.monitor-toolbar__select {
  width: 124px;
}

.monitor-toolbar__button {
  flex: 0 0 auto;
  white-space: nowrap;
}

@media (width <= 1279px) {
  .monitor-toolbar {
    flex-wrap: wrap;
  }
}

@media (width <= 991px) {
  .monitor-toolbar {
    justify-content: flex-start;
  }
}

@media (width <= 767px) {
  .monitor-toolbar {
    align-items: stretch;
  }

  .monitor-toolbar__field {
    flex-wrap: wrap;
  }

  .monitor-toolbar__select {
    width: min(100%, 168px);
  }
}
</style>
