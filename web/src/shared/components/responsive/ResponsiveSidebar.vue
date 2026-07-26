<template>
  <aside v-if="mode !== 'drawer'" :class="['responsive-sidebar', `responsive-sidebar--${mode}`]">
    <slot :compact="mode === 'compact'" />
  </aside>
  <t-drawer
    v-else
    :visible="visible"
    attach="body"
    :close-btn="true"
    drawer-class-name="graft-mobile-navigation-drawer"
    :footer="false"
    header
    placement="left"
    :prevent-scroll-through="true"
    :size="'var(--graft-shell-mobile-drawer-width)'"
    @update:visible="emit('update:visible', $event)"
  >
    <slot name="drawer" />
  </t-drawer>
</template>
<script setup lang="ts">
defineProps<{
  mode: 'desktop' | 'compact' | 'drawer';
  visible?: boolean;
}>();

const emit = defineEmits<{
  'update:visible': [visible: boolean];
}>();
</script>
<style scoped lang="less">
.responsive-sidebar {
  min-width: 0;
}
</style>
