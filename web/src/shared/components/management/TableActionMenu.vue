<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="table-action-menu">
    <t-button
      v-if="primaryAction"
      :data-testid="primaryAction.testId"
      :disabled="primaryAction.disabled"
      size="small"
      theme="default"
      variant="outline"
      @click="handlePrimaryClick"
    >
      {{ primaryAction.label }}
    </t-button>
    <t-dropdown v-if="menuOptions.length > 0" :options="menuOptions" trigger="click" @click="handleMenuClick">
      <t-button size="small" theme="default" variant="outline">
        {{ resolvedMoreLabel }}
      </t-button>
    </t-dropdown>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue';

type ActionOption = {
  disabled?: boolean;
  fallbackLabel?: string;
  label: string;
  testId?: string;
  value: string;
};

const props = withDefaults(
  defineProps<{
    actions: ActionOption[];
    moreLabel?: string;
    moreLabelFallback?: string;
  }>(),
  {
    moreLabel: 'More',
    moreLabelFallback: 'More',
  },
);

const emit = defineEmits<{
  action: [value: string];
}>();

const I18N_KEY_PATTERN = /^[a-z][\w-]*(\.[A-Za-z0-9_-]+)+$/;

function resolveLabel(label: string, fallbackLabel?: string) {
  if (!label || I18N_KEY_PATTERN.test(label)) {
    return fallbackLabel ?? label;
  }

  return label;
}

const resolvedMoreLabel = computed(() => resolveLabel(props.moreLabel, props.moreLabelFallback));
const primaryAction = computed(() => {
  const action = props.actions[0];
  if (!action) {
    return null;
  }

  return {
    ...action,
    label: resolveLabel(action.label, action.fallbackLabel),
  };
});
const menuOptions = computed(() =>
  props.actions.slice(1).map((action) => ({
    content: resolveLabel(action.label, action.fallbackLabel),
    disabled: action.disabled,
    testId: action.testId,
    value: action.value,
  })),
);

function handlePrimaryClick() {
  const action = primaryAction.value;

  if (action && !action.disabled) {
    emit('action', action.value);
  }
}

function handleMenuClick(payload: { value?: string | number | Record<string, unknown> } | string | number) {
  const action = typeof payload === 'object' && payload ? payload.value : payload;
  if (typeof action === 'string') {
    emit('action', action);
  }
}
</script>
<style scoped lang="less">
.table-action-menu {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-8);
  justify-content: center;
  width: 100%;
}

.table-action-menu :deep(.t-button) {
  min-width: 64px;
  white-space: nowrap;
}

.table-action-menu :deep(.t-dropdown) {
  flex: none;
}
</style>
