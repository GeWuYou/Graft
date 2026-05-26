<template>
  <t-drawer
    :visible="visible"
    :header="title"
    :size="size"
    placement="right"
    destroy-on-close
    @update:visible="emit('update:visible', $event)"
    @close="emit('close')"
  >
    <div class="assignment-drawer">
      <div class="assignment-drawer__body">
        <slot />
      </div>
    </div>
  </t-drawer>
</template>
<script setup lang="ts">
withDefaults(
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
</script>
<style scoped lang="less">
.assignment-drawer,
.assignment-drawer__body {
  display: flex;
  flex-direction: column;
}

.assignment-drawer {
  min-height: 100%;
}

.assignment-drawer__body {
  flex: 1;
  gap: var(--td-comp-margin-l);
}
</style>
