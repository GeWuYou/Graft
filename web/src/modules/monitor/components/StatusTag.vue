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
  font-size: 12px;
  font-weight: 600;
  line-height: 20px;
  padding-inline: 10px;
}

.monitor-status-tag--disabled {
  color: var(--td-text-color-secondary);
}

.monitor-status-tag--unknown {
  color: var(--td-text-color-placeholder);
}
</style>
