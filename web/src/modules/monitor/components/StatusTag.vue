<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-tag class="monitor-status-tag" :class="`monitor-status-tag--${status}`" :theme="theme" variant="light">
    {{ label }}
  </t-tag>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { type ServerStatusTagTheme, serverStatusTagTheme, type ServerStatusTone } from './server-status-ui';

const props = defineProps<{
  label: string;
  status: ServerStatusTone;
  themeOverride?: ServerStatusTagTheme;
}>();

const theme = computed(() => props.themeOverride ?? serverStatusTagTheme(props.status));
</script>
<style scoped lang="less">
.monitor-status-tag {
  border-radius: 999px;
  font: var(--td-font-body-small);
  font-weight: 600;
  line-height: 20px;
  padding-inline: var(--graft-density-gap-10);
}

.monitor-status-tag--disabled {
  color: var(--td-text-color-secondary);
}

.monitor-status-tag--unknown {
  color: var(--td-text-color-placeholder);
}
</style>
