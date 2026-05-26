<template>
  <section class="assignment-grid">
    <slot v-if="!empty" />
    <t-empty v-else :description="emptyDescription" />
  </section>
</template>
<script setup lang="ts">
defineProps<{
  empty: boolean;
  emptyDescription: string;
}>();
</script>
<style scoped lang="less">
.assignment-grid {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-l);
}
</style>
