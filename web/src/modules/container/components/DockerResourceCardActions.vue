<template>
  <div class="docker-resource-card-actions" @click.stop>
    <t-button theme="primary" variant="outline" @click="$emit('detail')">
      {{ detailLabel }}
    </t-button>
    <t-dropdown v-if="dropdownOptions.length" :options="dropdownOptions" trigger="click" @click="handleAction">
      <t-button theme="default" variant="outline">
        {{ moreLabel }}
      </t-button>
    </t-dropdown>
  </div>
</template>
<script setup lang="ts">
import type { DropdownProps } from 'tdesign-vue-next';
import { computed } from 'vue';

type DockerResourceCardAction = {
  danger?: boolean;
  disabled?: boolean;
  label: string;
  value: string;
};
/** DockerResourceCardActions 只统一资源卡片的行级动作外观，动作权限和业务处理仍由页面拥有。 */
const props = defineProps<{
  detailLabel: string;
  moreActions: DockerResourceCardAction[];
  moreLabel: string;
}>();

const emit = defineEmits<{
  action: [value: string];
  detail: [];
}>();

const dropdownOptions = computed<NonNullable<DropdownProps['options']>>(() =>
  props.moreActions.map((action) => ({
    content: action.label,
    disabled: action.disabled,
    theme: action.danger ? ('error' as const) : undefined,
    value: action.value,
  })),
);

function handleAction(payload: { value?: unknown } | string | number) {
  const action = typeof payload === 'object' && payload ? payload.value : payload;
  if (typeof action === 'string') emit('action', action);
}
</script>
<style scoped lang="less">
.docker-resource-card-actions {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-6);
  justify-content: flex-end;
  margin-top: auto;
  min-width: 0;
}
</style>
