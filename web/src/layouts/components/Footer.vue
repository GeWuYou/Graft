<template>
  <footer :class="`${prefix}-footer`">{{ content }}</footer>
</template>
<script setup lang="ts">
import { prefix } from '@/config/global';

defineProps<{
  content: string;
}>();
</script>
<style lang="less" scoped>
.@{starter-prefix}-footer {
  align-items: center;
  box-sizing: border-box;
  color: var(--td-text-color-placeholder);
  display: flex;
  font: var(--td-font-body-small);
  justify-content: center;
  line-height: 20px;
  min-height: var(--graft-page-footer-height);
  padding: 8px 0 var(--graft-page-bottom-safe-space);
  text-align: center;
}
</style>
