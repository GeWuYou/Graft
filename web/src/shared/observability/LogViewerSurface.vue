<template>
  <content-viewer-frame
    v-if="framed"
    v-bind="$attrs"
    class="log-viewer log-viewer--framed"
    :storage-key="storageKey"
    :fullscreen-label="fullscreenLabel"
    :exit-fullscreen-label="exitFullscreenLabel"
    :resize-handle-label="resizeHandleLabel"
    :show-fullscreen-button="false"
    surface-padding="none"
    fullscreen-surface-padding="none"
  >
    <template #header-actions="slotProps">
      <slot name="header-actions" v-bind="slotProps" />
    </template>
    <template #toolbar>
      <slot name="toolbar" />
    </template>
    <div class="log-viewer__body">
      <slot />
    </div>
  </content-viewer-frame>

  <section v-else v-bind="$attrs" class="log-viewer">
    <slot name="toolbar" />
    <slot />
  </section>
</template>
<script setup lang="ts">
/** 负责在普通容器与可全屏、可调整高度的阅读器框架之间转发日志视图插槽。 */
defineOptions({ inheritAttrs: false });

import ContentViewerFrame from '@/shared/components/viewer/ContentViewerFrame.vue';

defineProps<{
  exitFullscreenLabel: string;
  framed: boolean;
  fullscreenLabel: string;
  resizeHandleLabel: string;
  storageKey: string;
}>();
</script>
<style scoped lang="less">
.log-viewer__body {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-10);
  min-height: 0;
  min-width: 0;
  padding: var(--graft-density-gap-12);
}
</style>
