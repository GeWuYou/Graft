<template>
  <t-tooltip :content="tooltip" placement="top">
    <div class="realtime-resource-metric" :class="`realtime-resource-metric--${change}`" :data-available="available">
      <t-progress
        v-if="available"
        theme="circle"
        :label="false"
        :percentage="percentage"
        :size="36"
        :status="progressStatus"
        :stroke-width="4"
      />
      <span v-else class="realtime-resource-metric__empty" />
      <span>{{ value }}</span>
    </div>
  </t-tooltip>
</template>
<script setup lang="ts">
import { computed } from 'vue';

const props = withDefaults(
  defineProps<{
    available: boolean;
    change?: 'up' | 'down' | 'none';
    percentage: number;
    tooltip: string;
    value: string;
  }>(),
  { change: 'none' },
);

const progressStatus = computed(() =>
  props.change === 'up' ? 'warning' : props.change === 'down' ? 'success' : undefined,
);
</script>
<style scoped lang="less">
.realtime-resource-metric {
  align-items: center;
  border-radius: 999px;
  display: inline-flex;
  gap: var(--graft-density-gap-8);
  justify-content: center;
  min-width: 0;
  overflow: hidden;
  padding: var(--graft-density-gap-2) var(--graft-density-gap-8) var(--graft-density-gap-2) var(--graft-density-gap-2);
  transition:
    background-color 180ms ease,
    opacity 180ms ease,
    transform 180ms ease;
  white-space: nowrap;
}

.realtime-resource-metric[data-available='true'] {
  background: var(--td-bg-color-container-hover);
}

.realtime-resource-metric > span:last-child {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-small);
}

.realtime-resource-metric[data-available='false'] > span:last-child {
  color: var(--td-text-color-secondary);
}

.realtime-resource-metric__empty {
  background: var(--td-bg-color-component-disabled);
  border-radius: 50%;
  height: 36px;
  width: 36px;
}

.realtime-resource-metric--up {
  animation: realtime-resource-up 480ms ease;
  background: color-mix(in srgb, var(--td-warning-color-1) 58%, transparent);
}

.realtime-resource-metric--down {
  animation: realtime-resource-down 480ms ease;
  background: color-mix(in srgb, var(--td-success-color-1) 60%, transparent);
}

@keyframes realtime-resource-up {
  50% {
    transform: translateY(-1px);
  }
}

@keyframes realtime-resource-down {
  50% {
    opacity: 0.82;
  }
}

@media (prefers-reduced-motion: reduce) {
  .realtime-resource-metric {
    animation: none;
    transition-duration: 0ms;
  }
}
</style>
