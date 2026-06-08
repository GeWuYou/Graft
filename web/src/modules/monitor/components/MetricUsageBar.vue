<template>
  <div
    class="metric-usage-bar"
    :class="[`metric-usage-bar--${effectiveStatus}`, { 'metric-usage-bar--loading': loading }]"
    :data-usage-status="effectiveStatus"
    :data-usage-percent="dataPercent"
    :title="titleText"
    role="meter"
    :aria-label="label || titleText"
    aria-valuemin="0"
    aria-valuemax="100"
    :aria-valuenow="ariaValueNow"
  >
    <div class="metric-usage-bar__track">
      <span v-if="loading" class="metric-usage-bar__placeholder" />
      <span v-else class="metric-usage-bar__fill" :style="{ width: fillWidth }" />
    </div>
    <span v-if="!loading && !hasValue" class="metric-usage-bar__empty">{{ emptyText }}</span>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';

export type MetricUsageStatus = 'healthy' | 'warning' | 'danger' | 'unknown';

const props = withDefaults(
  defineProps<{
    value: number | null;
    max?: number;
    label?: string;
    status?: MetricUsageStatus;
    tooltip?: string;
    loading?: boolean;
    emptyText?: string;
  }>(),
  {
    max: 100,
    label: '',
    status: 'unknown',
    tooltip: '',
    loading: false,
    emptyText: '',
  },
);

const hasValue = computed(() => props.value !== null && Number.isFinite(props.value) && props.max > 0);
const rawPercent = computed(() => (hasValue.value ? (Number(props.value) / props.max) * 100 : null));
const clampedPercent = computed(() => {
  if (rawPercent.value === null) {
    return 0;
  }

  return Math.min(Math.max(rawPercent.value, 0), 100);
});

const effectiveStatus = computed<MetricUsageStatus>(() => {
  if (!hasValue.value || props.loading) {
    return 'unknown';
  }
  if (props.status !== 'unknown') {
    return props.status;
  }
  if (rawPercent.value === null) {
    return 'unknown';
  }
  if (rawPercent.value >= 85) {
    return 'danger';
  }
  if (rawPercent.value >= 70) {
    return 'warning';
  }

  return 'healthy';
});

const fillWidth = computed(() => `${clampedPercent.value}%`);
const dataPercent = computed(() => (hasValue.value ? clampedPercent.value.toFixed(2) : 'none'));
const ariaValueNow = computed(() => (hasValue.value ? String(clampedPercent.value) : undefined));
const titleText = computed(() => {
  if (props.loading) {
    return props.label || '';
  }
  if (!hasValue.value) {
    return props.tooltip || props.emptyText;
  }

  return props.tooltip || `${props.label ? `${props.label} ` : ''}${rawPercent.value?.toFixed(2)}%`;
});
</script>
<style scoped lang="less">
.metric-usage-bar {
  --metric-usage-color: var(--td-text-color-placeholder);
  --metric-usage-track: color-mix(in srgb, var(--td-component-stroke) 48%, transparent);

  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
  width: 100%;
}

.metric-usage-bar--healthy {
  --metric-usage-color: var(--td-success-color-5);
}

.metric-usage-bar--warning {
  --metric-usage-color: var(--td-warning-color-5);
}

.metric-usage-bar--danger {
  --metric-usage-color: var(--td-error-color-5);
}

.metric-usage-bar__track {
  background: var(--metric-usage-track);
  border-radius: 999px;
  height: 6px;
  overflow: hidden;
  width: 100%;
}

.metric-usage-bar__fill {
  background: color-mix(in srgb, var(--metric-usage-color) 82%, var(--td-bg-color-container));
  border-radius: inherit;
  display: block;
  height: 100%;
  min-width: 0;
  transition:
    background-color 180ms ease,
    width 220ms ease;
}

.metric-usage-bar__placeholder {
  animation: metric-usage-loading 1.2s ease-in-out infinite;
  background: linear-gradient(
    90deg,
    transparent,
    color-mix(in srgb, var(--td-text-color-placeholder) 24%, transparent),
    transparent
  );
  display: block;
  height: 100%;
  width: 42%;
}

.metric-usage-bar__empty {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  line-height: 1.2;
}

@keyframes metric-usage-loading {
  0% {
    transform: translateX(-100%);
  }

  100% {
    transform: translateX(240%);
  }
}
</style>
