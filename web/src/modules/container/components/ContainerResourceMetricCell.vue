<template>
  <t-tooltip :content="metric.tooltip" placement="top">
    <div
      class="container-resource-meter"
      :class="metric.changeClass"
      :data-available="metric.available"
      :data-testid="testId"
    >
      <t-progress
        v-if="metric.available"
        theme="circle"
        :label="false"
        :percentage="metric.percentage"
        :size="36"
        :status="metric.progressStatus"
        :stroke-width="4"
      />
      <span v-else class="container-resource-meter__empty"></span>
      <span>{{ metric.value }}</span>
    </div>
  </t-tooltip>
</template>
<script setup lang="ts">
defineOptions({
  name: 'ContainerResourceMetricCell',
});
defineProps<{
  metric: {
    available: boolean;
    changeClass: Record<string, boolean>;
    percentage: number;
    progressStatus: 'success' | 'warning' | undefined;
    tooltip: string;
    value: string;
  };
  testId: string;
}>();
</script>
