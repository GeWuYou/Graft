<template>
  <section class="container-danger-zone">
    <h3>{{ t('container.resourceContext.dangerZone') }}</h3>
    <p>{{ description }}</p>
    <t-button theme="danger" variant="outline" :loading="loading" @click="emit('action')">
      {{ actionLabel }}
    </t-button>
  </section>
</template>
<script setup lang="ts">
// 容器模块危险区统一高风险资源操作的视觉层级，具体确认与提交仍由所属资源页面负责。
import { useI18n } from 'vue-i18n';

withDefaults(
  defineProps<{
    actionLabel: string;
    description: string;
    loading?: boolean;
  }>(),
  { loading: false },
);
const emit = defineEmits<{ action: [] }>();
const { t } = useI18n();
</script>
<style scoped lang="less">
.container-danger-zone {
  background: color-mix(in srgb, var(--td-error-color) 3%, var(--td-bg-color-container));
  border: 1px solid var(--td-error-color);
  border-radius: var(--td-radius-small);
  padding: var(--graft-density-gap-12);
}

.container-danger-zone h3 {
  color: var(--td-text-color-primary);
  font-size: var(--td-font-size-body-large);
  margin: 0 0 var(--graft-density-gap-8);
}

.container-danger-zone p {
  color: var(--td-text-color-secondary);
  margin: 0 0 var(--graft-density-gap-12);
}
</style>
