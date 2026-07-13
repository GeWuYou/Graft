<template>
  <realtime-resource-metric-cell
    :available="metric.available"
    :class="metric.changeClass"
    :change="metric.change"
    :percentage="metric.percentage"
    :tooltip="metric.tooltip"
    :value="metric.value"
    :data-testid="testId"
  />
</template>
<script setup lang="ts">
import { RealtimeResourceMetricCell } from '@/shared/components/metrics';

import type { ContainerResourceMetric } from '../shared/resource-table';

defineOptions({
  name: 'ContainerResourceMetricCell',
});
defineProps<{
  metric: ContainerResourceMetric;
  testId: string;
}>();
</script>
