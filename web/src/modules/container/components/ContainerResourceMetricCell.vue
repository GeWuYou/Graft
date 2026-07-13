<template>
  <realtime-resource-metric-cell
    :available="metric.available"
    :class="metric.changeClass"
    :change="change"
    :percentage="metric.percentage"
    :tooltip="metric.tooltip"
    :value="metric.value"
    :data-testid="testId"
  />
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { RealtimeResourceMetricCell } from '@/shared/components/metrics';

import type { ContainerResourceMetric } from '../shared/resource-table';

defineOptions({
  name: 'ContainerResourceMetricCell',
});
const props = defineProps<{
  metric: ContainerResourceMetric;
  testId: string;
}>();

const change = computed(() => {
  if (props.metric.changeClass['container-metric-change--up']) return 'up';
  if (props.metric.changeClass['container-metric-change--down']) return 'down';
  return 'none';
});
</script>
