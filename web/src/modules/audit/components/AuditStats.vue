<template>
  <section class="audit-stats">
    <t-card v-for="item in items" :key="item.key" class="audit-stats__card" size="small">
      <div class="audit-stats__head">
        <span class="audit-stats__title">{{ item.title }}</span>
        <t-tag :theme="item.theme" variant="light-outline" size="small">{{ item.meta }}</t-tag>
      </div>
      <div class="audit-stats__value-row">
        <strong class="audit-stats__value">{{ item.value }}</strong>
        <span class="audit-stats__unit">{{ item.unit }}</span>
      </div>
    </t-card>
  </section>
</template>
<script setup lang="ts">
defineProps<{
  items: Array<{
    key: string;
    meta: string;
    theme: 'default' | 'primary' | 'warning' | 'danger' | 'success';
    title: string;
    unit: string;
    value: string;
  }>;
}>();
</script>
<style scoped lang="less">
.audit-stats {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.audit-stats__card {
  border-radius: var(--td-radius-large);
}

.audit-stats__head,
.audit-stats__value-row {
  align-items: center;
  display: flex;
  justify-content: space-between;
}

.audit-stats__head {
  gap: 12px;
}

.audit-stats__title,
.audit-stats__unit {
  color: var(--td-text-color-secondary);
}

.audit-stats__value-row {
  margin-top: 16px;
}

.audit-stats__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
}

@media (width <= 1280px) {
  .audit-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 768px) {
  .audit-stats {
    grid-template-columns: 1fr;
  }
}
</style>
