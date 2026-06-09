<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

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
  min-height: var(--graft-footer-height);
  padding: 0;
  text-align: center;
}
</style>
