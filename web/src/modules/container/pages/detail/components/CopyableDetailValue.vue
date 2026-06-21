<template>
  <span class="container-detail-copyable-value">
    <t-tooltip :content="copyValue || resolvedDisplayValue" placement="top-left">
      <code v-if="code" class="container-detail-copyable-value__text">{{ resolvedDisplayValue }}</code>
      <span v-else class="container-detail-copyable-value__text">{{ resolvedDisplayValue }}</span>
    </t-tooltip>
    <t-tooltip v-if="copyValue" :content="copyLabel" placement="top">
      <t-button
        :aria-label="copyLabel"
        class="container-detail-copyable-value__button"
        :data-testid="testId"
        shape="square"
        size="small"
        theme="default"
        variant="text"
        @click="emit('copy', copyValue)"
      >
        <template #icon><copy-icon /></template>
      </t-button>
    </t-tooltip>
  </span>
</template>
<script setup lang="ts">
import { CopyIcon } from 'tdesign-icons-vue-next';
import { computed, useAttrs } from 'vue';

defineOptions({
  inheritAttrs: false,
});

const props = withDefaults(
  defineProps<{
    code?: boolean;
    copyLabel: string;
    displayValue?: string;
    value?: string;
  }>(),
  {
    code: false,
    displayValue: '',
    value: '',
  },
);

const emit = defineEmits<{
  copy: [value: string];
}>();

const attrs = useAttrs();
const resolvedDisplayValue = computed(() => props.displayValue || props.value || '-');
const copyValue = computed(() => {
  const value = props.value.trim();
  return value === '-' ? '' : value;
});
const testId = computed(() => {
  const value = attrs['data-testid'];
  return typeof value === 'string' ? value : undefined;
});
</script>
<style scoped lang="less">
.container-detail-copyable-value {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-6);
  max-width: 100%;
  min-width: 0;
  vertical-align: bottom;
}

.container-detail-copyable-value__text,
.container-detail-copyable-value code {
  color: var(--td-text-color-primary);
  display: inline-block;
  font-family: var(
    --td-font-family-mono,
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    'Liberation Mono',
    monospace
  );
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.container-detail-copyable-value > :first-child {
  min-width: 0;
}

.container-detail-copyable-value__button {
  flex: 0 0 auto;
}
</style>
