<template>
  <footer class="assignment-footer">
    <div class="assignment-footer__summary">
      <span v-if="summary" class="assignment-footer__summary-line assignment-footer__summary-line--primary">
        {{ summary }}
      </span>
      <span v-for="item in details" :key="item" class="assignment-footer__summary-line">
        {{ item }}
      </span>
    </div>
    <div class="assignment-footer__actions">
      <t-button :data-testid="cancelTestId" variant="outline" @click="emit('cancel')">{{ cancelLabel }}</t-button>
      <t-button
        v-if="showConfirm"
        :data-testid="confirmTestId"
        theme="primary"
        :disabled="confirmDisabled"
        :loading="confirmLoading"
        @click="emit('confirm')"
      >
        {{ confirmLabel }}
      </t-button>
    </div>
  </footer>
</template>
<script setup lang="ts">
withDefaults(
  defineProps<{
    cancelLabel: string;
    cancelTestId?: string;
    confirmDisabled?: boolean;
    confirmLabel: string;
    confirmLoading?: boolean;
    confirmTestId?: string;
    details?: string[];
    showConfirm?: boolean;
    summary?: string;
  }>(),
  {
    cancelTestId: undefined,
    confirmTestId: undefined,
    details: () => [],
    showConfirm: true,
    summary: undefined,
  },
);

const emit = defineEmits<{
  cancel: [];
  confirm: [];
}>();
</script>
<style scoped lang="less">
.assignment-footer,
.assignment-footer__actions,
.assignment-footer__summary {
  display: flex;
}

.assignment-footer {
  align-items: center;
  background: color-mix(in srgb, var(--td-bg-color-container) 92%, var(--td-bg-color-page));
  gap: var(--td-comp-margin-l);
  justify-content: space-between;
  padding: var(--td-comp-paddingTB-l) 0 var(--graft-density-gap-2);
}

.assignment-footer__summary {
  color: var(--td-text-color-secondary);
  flex: 1;
  flex-direction: column;
  gap: var(--td-comp-margin-xs);
  min-width: 0;
}

.assignment-footer__summary-line {
  font: var(--td-font-body-medium);
}

.assignment-footer__summary-line--primary {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-large);
}

.assignment-footer__actions {
  align-items: center;
  flex-shrink: 0;
  gap: var(--td-comp-margin-s);
  justify-content: flex-end;
}

@media (width <= 768px) {
  .assignment-footer,
  .assignment-footer__actions {
    align-items: stretch;
    flex-direction: column;
  }

  .assignment-footer__actions {
    width: 100%;
  }
}
</style>
