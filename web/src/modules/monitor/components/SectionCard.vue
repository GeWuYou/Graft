<template>
  <section class="server-status-section-card" :style="cardStyle">
    <header v-if="title || description || $slots.actions" class="server-status-section-card__header">
      <div class="server-status-section-card__copy">
        <h2 v-if="title" class="server-status-section-card__title">{{ title }}</h2>
        <p v-if="description" class="server-status-section-card__description">{{ description }}</p>
      </div>
      <div v-if="$slots.actions" class="server-status-section-card__actions">
        <slot name="actions" />
      </div>
    </header>

    <div class="server-status-section-card__body">
      <slot />
    </div>
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{
  title?: string;
  description?: string;
  minHeight?: number | string;
}>();

const cardStyle = computed(() => {
  if (props.minHeight === undefined) {
    return undefined;
  }

  return {
    minHeight: typeof props.minHeight === 'number' ? `${props.minHeight}px` : props.minHeight,
  };
});
</script>
<style scoped lang="less">
@import './card-surface.less';

.server-status-section-card {
  .server-status-card-surface();

  display: flex;
  flex-direction: column;
  padding: 18px;
}

.server-status-section-card__header {
  align-items: flex-start;
  display: flex;
  gap: 16px;
  justify-content: space-between;
  margin-bottom: 14px;
}

.server-status-section-card__copy {
  min-width: 0;
}

.server-status-section-card__title {
  color: var(--td-text-color-primary);
  font-size: 18px;
  font-weight: 600;
  line-height: 28px;
  margin: 0;
}

.server-status-section-card__description {
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
  margin: 4px 0 0;
}

.server-status-section-card__actions {
  flex: 0 0 auto;
}

.server-status-section-card__body {
  flex: 1;
  min-height: 0;
}
</style>
