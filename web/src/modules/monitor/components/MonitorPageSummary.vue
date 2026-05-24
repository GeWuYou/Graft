<template>
  <template v-for="metric in items" :key="metric.key">
    <summary-metric-card :title="metric.label" :value="metric.value" :description="metric.description" />
  </template>
</template>
<script setup lang="ts">
import SummaryMetricCard from './SummaryMetricCard.vue';

defineProps<{
  items: Array<{
    key: string;
    label: string;
    value: string;
    description: string;
  }>;
}>();
</script>
