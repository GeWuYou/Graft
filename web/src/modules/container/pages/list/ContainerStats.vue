<template>
  <section class="container-stats" :aria-label="label">
    <t-tag v-for="item in items" :key="item.label" :theme="item.theme" variant="light-outline" shape="round">
      {{ `${item.label} ${item.value}` }}
    </t-tag>
  </section>
</template>
<script setup lang="ts">
/** 容器状态统计只负责紧凑呈现，统计口径由列表页提供。 */
type ContainerStat = { label: string; value: number; theme: 'default' | 'success' | 'warning' | 'danger' };

defineProps<{ items: ContainerStat[]; label: string }>();
</script>
<style scoped lang="less">
.container-stats {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  min-width: 0;
}
</style>
