<template>
  <span
    class="workbench-status"
    :class="`workbench-status--${status}`"
    :data-status="status"
    :aria-label="label"
    role="status"
  >
    <component :is="statusIcon" size="16px" />
    <span v-if="showLabel">{{ label }}</span>
  </span>
</template>
<script setup lang="ts">
import {
  CheckCircleIcon,
  ErrorCircleIcon,
  ErrorTriangleIcon,
  HelpCircleIcon,
  InfoCircleIcon,
} from 'tdesign-icons-vue-next';
import { computed } from 'vue';

import type { PresentationStatus } from '../../presentation/workbench';

const props = withDefaults(defineProps<{ status: PresentationStatus; label: string; showLabel?: boolean }>(), {
  showLabel: true,
});

const statusIcon = computed(
  () =>
    ({
      error: ErrorCircleIcon,
      warning: ErrorTriangleIcon,
      unknown: HelpCircleIcon,
      info: InfoCircleIcon,
      healthy: CheckCircleIcon,
    })[props.status],
);
</script>
<style scoped lang="less">
.workbench-status {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  flex: 0 0 auto;
  font: var(--td-font-body-small);
  gap: var(--graft-density-gap-4);
  white-space: nowrap;
}

.workbench-status--error {
  color: var(--td-error-color);
}

.workbench-status--warning {
  color: var(--td-warning-color);
}

.workbench-status--unknown {
  color: var(--td-text-color-placeholder);
}

.workbench-status--healthy {
  color: var(--td-text-color-secondary);
}
</style>
