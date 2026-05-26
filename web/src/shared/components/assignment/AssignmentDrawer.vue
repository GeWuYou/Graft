<template>
  <t-drawer
    :visible="visible"
    :header="title"
    :footer="false"
    :size="size"
    placement="right"
    destroy-on-close
    @update:visible="handleVisibleChange"
    @close="emit('close')"
  >
    <div class="assignment-drawer permission-drawer">
      <div v-if="$slots.header" class="assignment-drawer__header permission-drawer__header">
        <slot name="header" />
      </div>
      <div ref="bodyRef" class="assignment-drawer__body permission-drawer__body">
        <slot />
      </div>
      <div v-if="$slots.footer" class="assignment-drawer__footer permission-drawer__footer">
        <slot name="footer" />
      </div>
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
import { nextTick, ref, watch } from 'vue';

const props = withDefaults(
  defineProps<{
    size?: string;
    title: string;
    visible: boolean;
  }>(),
  {
    size: '760px',
  },
);

const emit = defineEmits<{
  close: [];
  'update:visible': [value: boolean];
}>();

const bodyRef = ref<HTMLDivElement | null>(null);

function handleVisibleChange(value: boolean) {
  emit('update:visible', value);
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
  scrollbar-color: var(--td-scrollbar-color, rgb(255 255 255 / 24%)) transparent;
  scrollbar-width: thin;
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

.assignment-drawer__body::-webkit-scrollbar {
  width: 8px;
}

.assignment-drawer__body::-webkit-scrollbar-track {
  background: transparent;
}

.assignment-drawer__body::-webkit-scrollbar-thumb {
  background: rgb(255 255 255 / 24%);
  border-radius: 999px;
}

:deep(.t-drawer__content) {
  display: flex;
  flex-direction: column;
  height: 100%;
}

:deep(.t-drawer__body) {
  box-sizing: border-box;
  flex: 1 1 auto;
  height: auto;
  min-height: 0;
  overflow: hidden;
  padding: 0;
}
</style>
