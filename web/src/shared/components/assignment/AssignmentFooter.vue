<template>
  <footer class="assignment-footer">
    <span class="assignment-footer__summary">{{ summary }}</span>
    <div class="assignment-footer__actions">
      <t-button :data-testid="cancelTestId" variant="outline" @click="emit('cancel')">{{ cancelLabel }}</t-button>
      <t-button
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
defineProps<{
  cancelLabel: string;
  cancelTestId?: string;
  confirmDisabled?: boolean;
  confirmLabel: string;
  confirmLoading?: boolean;
  confirmTestId?: string;
  summary: string;
}>();

const emit = defineEmits<{
  cancel: [];
  confirm: [];
}>();
</script>
<style scoped lang="less">
.assignment-footer,
.assignment-footer__actions {
  display: flex;
}

.assignment-footer {
  align-items: center;
  background: color-mix(in srgb, var(--td-bg-color-container) 92%, var(--td-bg-color-page));
  border-top: 1px solid var(--td-component-stroke);
  bottom: 0;
  gap: var(--td-comp-margin-l);
  justify-content: space-between;
  margin-top: auto;
  padding: var(--td-comp-paddingTB-l) 0 0;
  position: sticky;
}

.assignment-footer__summary {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
}

.assignment-footer__actions {
  gap: var(--td-comp-margin-s);
  justify-content: flex-end;
}

@media (width <= 768px) {
  .assignment-footer,
  .assignment-footer__actions {
    align-items: stretch;
    flex-direction: column;
  }
}
</style>
