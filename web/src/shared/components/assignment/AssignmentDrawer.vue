<template>
  <responsive-dialog
    :visible="visible"
    :close-label="closeLabel"
    :title="title"
    purpose="workspace"
    size="large"
    :close-on-esc-keydown="false"
    :close-on-overlay-click="false"
    @update:visible="handleVisibleChange"
  >
    <div class="assignment-drawer permission-drawer">
      <div v-if="$slots.header" class="assignment-drawer__header permission-drawer__header">
        <slot name="header" />
      </div>
      <div ref="bodyRef" class="assignment-drawer__body permission-drawer__body graft-scrollbar">
        <slot />
      </div>
      <div v-if="$slots.footer" class="assignment-drawer__footer permission-drawer__footer">
        <slot name="footer" />
      </div>
    </div>
  </responsive-dialog>
</template>
<script setup lang="ts">
import { nextTick, ref, watch } from 'vue';

import ResponsiveDialog from '@/shared/components/responsive/ResponsiveDialog.vue';

const props = defineProps<{
  title: string;
  visible: boolean;
  closeLabel: string;
}>();

const emit = defineEmits<{
  close: [];
  'update:visible': [value: boolean];
}>();

const bodyRef = ref<HTMLDivElement | null>(null);
const closeRequestPending = ref(false);

function requestClose() {
  if (closeRequestPending.value) {
    return;
  }

  closeRequestPending.value = true;
  emit('close');
  void nextTick(() => {
    closeRequestPending.value = false;
  });
}

function handleVisibleChange(value: boolean) {
  if (value) {
    emit('update:visible', value);
    return;
  }

  requestClose();
}

watch(
  () => props.visible,
  async (nextVisible) => {
    if (!nextVisible) {
      return;
    }

    await nextTick();
    const body = bodyRef.value;
    if (!body) {
      return;
    }

    if (typeof body.scrollTo === 'function') {
      body.scrollTo({ top: 0, left: 0 });
      return;
    }

    body.scrollTop = 0;
    body.scrollLeft = 0;
  },
);
</script>
<style scoped lang="less">
.assignment-drawer {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.assignment-drawer__header,
.assignment-drawer__footer {
  flex: 0 0 auto;
}

.assignment-drawer__body {
  box-sizing: border-box;
  flex: 1 1 auto;
  min-height: 0;
  overflow: hidden auto;
  padding: 0 var(--td-comp-paddingLR-l) var(--td-comp-paddingTB-l);
  width: 100%;
}

.assignment-drawer__header {
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l) 0;
}

.assignment-drawer__footer {
  background: var(--td-bg-color-container);
  border-top: 1px solid var(--td-component-border);
  padding: 0 var(--td-comp-paddingLR-l) var(--td-comp-paddingTB-l);
}
</style>
