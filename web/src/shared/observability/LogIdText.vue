<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <span class="log-id-text">
    <t-tooltip :content="tooltipContent" placement="top-left">
      <span class="log-id-text__value">
        {{ displayValue || emptyText }}
      </span>
    </t-tooltip>
    <t-tooltip v-if="copyable && copySource" :content="copyLabel" placement="top">
      <t-button
        class="log-id-text__copy"
        shape="square"
        size="small"
        theme="default"
        variant="text"
        @click.stop="copyValue"
      >
        <template #icon><copy-icon /></template>
      </t-button>
    </t-tooltip>
  </span>
</template>
<script setup lang="ts">
import { CopyIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin } from 'tdesign-vue-next/es/message';
import { computed } from 'vue';

import { copyText } from './copy';

const props = defineProps<{
  copyFailLabel?: string;
  copyLabel?: string;
  copySuccessLabel?: string;
  copyable?: boolean;
  displayValue?: string | null;
  emptyText?: string;
  tooltip?: string | null;
}>();

const emptyText = computed(() => props.emptyText ?? '-');
const copyLabel = computed(() => props.copyLabel ?? '');
const tooltipContent = computed(() => props.tooltip || props.displayValue || emptyText.value);
const copySource = computed(() => {
  const value = props.tooltip || props.displayValue || '';
  return value === emptyText.value ? '' : value;
});

async function copyValue() {
  if (!copySource.value) {
    return;
  }

  const copied = await copyText(copySource.value);
  if (copied) {
    if (props.copySuccessLabel) {
      MessagePlugin.success(props.copySuccessLabel);
    }
    return;
  }

  if (props.copyFailLabel) {
    MessagePlugin.error(props.copyFailLabel);
  }
}
</script>
<style scoped lang="less">
.log-id-text {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-4);
  max-width: 100%;
  min-width: 0;
  vertical-align: bottom;
}

.log-id-text__value {
  display: inline-block;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-weight: 600;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.log-id-text__copy {
  flex: 0 0 auto;
}
</style>
