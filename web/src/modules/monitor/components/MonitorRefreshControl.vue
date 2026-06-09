<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="monitor-refresh-control" data-monitor-refresh-toolbar="true">
    <div v-if="statusTag" class="monitor-refresh-control__status">
      <t-tag :theme="statusTag.theme" variant="light">
        {{ statusTag.label }}
      </t-tag>
    </div>

    <div class="monitor-refresh-control__field">
      <span class="monitor-refresh-control__label">{{ refreshIntervalLabel }}</span>
      <t-select
        class="monitor-refresh-control__select"
        :model-value="refreshIntervalValue"
        :options="refreshIntervalOptions"
        size="small"
        data-monitor-refresh-interval-select="true"
        @update:model-value="handleRefreshIntervalChange"
      />
    </div>

    <div v-if="extraSelectLabel && extraSelectOptions.length > 0" class="monitor-refresh-control__field">
      <span class="monitor-refresh-control__label">{{ extraSelectLabel }}</span>
      <t-select
        class="monitor-refresh-control__select monitor-refresh-control__select--extra"
        :model-value="extraSelectValue"
        :options="extraSelectOptions"
        size="small"
        data-monitor-refresh-extra-select="true"
        @update:model-value="handleExtraSelectChange"
      />
    </div>

    <t-button
      class="monitor-refresh-control__button"
      theme="primary"
      size="small"
      :loading="loading"
      @click="emit('refresh')"
    >
      <template #icon>
        <refresh-icon />
      </template>
      {{ refreshNowLabel }}
    </t-button>
    <t-button
      class="monitor-refresh-control__button"
      variant="outline"
      size="small"
      @click="emit('toggle-auto-refresh')"
    >
      {{ autoRefreshEnabled ? pauseAutoRefreshLabel : resumeAutoRefreshLabel }}
    </t-button>
  </div>
</template>
<script setup lang="ts">
import { RefreshIcon } from 'tdesign-icons-vue-next';
import type { PropType } from 'vue';

type ControlOptionValue = number | string;

type ControlOption = {
  label: string;
  value: ControlOptionValue;
};

type StatusTag = {
  label: string;
  theme: 'success' | 'warning' | 'danger' | 'default';
};

const props = defineProps({
  autoRefreshEnabled: {
    type: Boolean,
    required: true,
  },
  extraSelectLabel: {
    type: String,
    default: '',
  },
  extraSelectOptions: {
    type: Array as PropType<ControlOption[]>,
    default: () => [],
  },
  extraSelectValue: {
    type: [Number, String] as PropType<ControlOptionValue | undefined>,
    default: undefined,
  },
  loading: {
    type: Boolean,
    default: false,
  },
  pauseAutoRefreshLabel: {
    type: String,
    required: true,
  },
  refreshIntervalLabel: {
    type: String,
    required: true,
  },
  refreshIntervalOptions: {
    type: Array as PropType<ControlOption[]>,
    required: true,
  },
  refreshIntervalValue: {
    type: [Number, String] as PropType<ControlOptionValue>,
    required: true,
  },
  refreshNowLabel: {
    type: String,
    required: true,
  },
  resumeAutoRefreshLabel: {
    type: String,
    required: true,
  },
  statusTag: {
    type: Object as PropType<StatusTag | null>,
    default: null,
  },
});

const emit = defineEmits<{
  refresh: [];
  'toggle-auto-refresh': [];
  'update:extra-select-value': [value: ControlOptionValue];
  'update:refresh-interval-value': [value: ControlOptionValue];
}>();

function handleExtraSelectChange(value: ControlOptionValue) {
  const nextValue = resolveOptionValue(value, props.extraSelectOptions);
  if (nextValue !== undefined) {
    emit('update:extra-select-value', nextValue);
  }
}

function handleRefreshIntervalChange(value: ControlOptionValue) {
  const nextValue = resolveOptionValue(value, props.refreshIntervalOptions);
  if (nextValue !== undefined) {
    emit('update:refresh-interval-value', nextValue);
  }
}

function resolveOptionValue(value: ControlOptionValue, options: ControlOption[]) {
  return options.find((option) => String(option.value) === String(value))?.value;
}
</script>
<style scoped lang="less">
.monitor-refresh-control {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-10) var(--graft-density-gap-12);
  justify-content: flex-end;
  min-width: 0;
}

.monitor-refresh-control__status {
  align-items: center;
  display: flex;
  flex: 0 0 auto;
}

.monitor-refresh-control__field {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}

.monitor-refresh-control__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  white-space: nowrap;
}

.monitor-refresh-control__select {
  width: 136px;
}

.monitor-refresh-control__select--extra {
  width: 128px;
}

.monitor-refresh-control__button {
  flex: 0 0 auto;
  white-space: nowrap;
}

@media (width <= 767px) {
  .monitor-refresh-control {
    justify-content: flex-start;
  }

  .monitor-refresh-control__field {
    flex-wrap: wrap;
  }

  .monitor-refresh-control__select,
  .monitor-refresh-control__select--extra {
    width: min(100%, 168px);
  }
}
</style>
