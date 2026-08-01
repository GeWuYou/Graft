<template>
  <t-drawer
    v-if="presentation === 'overlay' && variant.density !== 'compact'"
    :visible="visible"
    attach="body"
    :size="drawerSize"
    placement="right"
    :footer="false"
    :header="false"
    :close-on-overlay-click="true"
    destroy-on-close
    @update:visible="emitVisible"
  >
    <resource-detail-content :title="title" :back-label="backLabel" @back="close">
      <template v-if="$slots.actions" #actions><slot name="actions" /></template>
      <slot />
      <template v-if="$slots.footer" #footer><slot name="footer" /></template>
    </resource-detail-content>
  </t-drawer>

  <t-dialog
    v-else-if="presentation === 'overlay'"
    :visible="visible"
    :footer="false"
    :header="false"
    :close-btn="false"
    :close-on-overlay-click="true"
    :width="'100vw'"
    class="resource-detail-layout__fullscreen-dialog"
    destroy-on-close
    @overlay-click="close"
    @update:visible="emitVisible"
  >
    <resource-detail-content :title="title" :back-label="backLabel" @back="close">
      <template v-if="$slots.actions" #actions><slot name="actions" /></template>
      <slot />
      <template v-if="$slots.footer" #footer><slot name="footer" /></template>
    </resource-detail-content>
  </t-dialog>

  <resource-detail-content
    v-else
    class="resource-detail-layout--page"
    :title="title"
    :back-label="backLabel"
    @back="close"
  >
    <template v-if="$slots.actions" #actions><slot name="actions" /></template>
    <slot />
    <template v-if="$slots.footer" #footer><slot name="footer" /></template>
  </resource-detail-content>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { useViewportResponsiveVariant } from '@/shared/composables';

import ResourceDetailContent from './ResourceDetailContent.vue';

/** 资源详情表面在共享层统一选择 Drawer、全屏覆盖层或路由页骨架，模块不感知设备宽度。 */
type ResourceDetailPresentation = 'overlay' | 'page';
type ResourceDetailSize = 'compact' | 'medium' | 'large';

const props = withDefaults(
  defineProps<{
    backLabel: string;
    presentation?: ResourceDetailPresentation;
    size?: ResourceDetailSize;
    title: string;
    visible?: boolean;
  }>(),
  { presentation: 'overlay', size: 'medium', visible: true },
);

const emit = defineEmits<{ 'update:visible': [visible: boolean] }>();
const variant = useViewportResponsiveVariant();
const drawerSize = computed(() => {
  if (props.size === 'large') {
    return 'var(--graft-resource-detail-large-fluid-width)';
  }
  if (variant.value.density === 'comfortable') return '70%';
  return `var(--graft-resource-detail-${props.size}-width)`;
});

function emitVisible(value: boolean) {
  emit('update:visible', value);
}

function close() {
  emitVisible(false);
}
</script>
<style scoped lang="less">
.resource-detail-layout--page {
  min-block-size: 0;
}

:deep(.resource-detail-layout__fullscreen-dialog .t-dialog) {
  block-size: 100dvh;
  border-radius: 0;
  margin: 0;
  max-block-size: none;
  max-inline-size: none;
  padding: 0;
}

:deep(.resource-detail-layout__fullscreen-dialog .t-dialog__body) {
  block-size: 100%;
  padding: 0;
}
</style>
