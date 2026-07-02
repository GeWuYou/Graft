<template>
  <div class="project-import-section-heading">
    <div>
      <h3>{{ title }}</h3>
      <p>{{ description }}</p>
    </div>
  </div>
</template>
<script setup lang="ts">
defineOptions({
  name: 'ProjectImportSectionHeading',
});

defineProps<{
  description: string;
  title: string;
}>();
</script>
<style scoped lang="less">
.project-import-section-heading h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
}

.project-import-section-heading p {
  color: var(--td-text-color-secondary);
  margin: var(--graft-density-gap-4) 0 0;
}
</style>
