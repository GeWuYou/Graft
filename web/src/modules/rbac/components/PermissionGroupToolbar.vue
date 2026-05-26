<template>
  <header class="permission-group-toolbar">
    <div class="permission-group-toolbar__summary">
      <h3 class="permission-group-toolbar__title">{{ title }}</h3>
      <p class="permission-group-toolbar__meta">{{ meta }}</p>
    </div>

    <div class="permission-group-toolbar__controls">
      <div class="permission-group-toolbar__strategy">
        <span class="permission-group-toolbar__label">{{ strategyLabel }}</span>
        <t-select
          :model-value="modelValue"
          class="permission-group-toolbar__select"
          size="small"
          :options="options"
          :disabled="disabled"
          @update:model-value="emit('update:modelValue', $event)"
        />
      </div>

      <div class="permission-group-toolbar__actions">
        <t-button
          size="small"
          variant="text"
          theme="default"
          :disabled="quickActionDisabled"
          @click="emit('select-group')"
        >
          {{ selectGroupLabel }}
        </t-button>
        <t-button
          size="small"
          variant="text"
          theme="default"
          :disabled="quickActionDisabled"
          @click="emit('clear-group')"
        >
          {{ clearGroupLabel }}
        </t-button>
      </div>
    </div>
  </header>
</template>
<script setup lang="ts">
type MutationMode = 'replace' | 'add' | 'remove';

defineProps<{
  clearGroupLabel: string;
  disabled: boolean;
  meta: string;
  modelValue: MutationMode;
  options: Array<{ label: string; value: MutationMode }>;
  quickActionDisabled: boolean;
  selectGroupLabel: string;
  strategyLabel: string;
  title: string;
}>();

const emit = defineEmits<{
  'clear-group': [];
  'select-group': [];
  'update:modelValue': [value: MutationMode];
}>();
</script>
<style scoped lang="less">
.permission-group-toolbar {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--rbac-permission-toolbar-gap);
  justify-content: space-between;
}

.permission-group-toolbar__summary,
.permission-group-toolbar__controls,
.permission-group-toolbar__strategy,
.permission-group-toolbar__actions {
  align-items: center;
  display: flex;
  gap: var(--rbac-permission-toolbar-gap);
}

.permission-group-toolbar__summary,
.permission-group-toolbar__controls {
  flex-wrap: wrap;
}

.permission-group-toolbar__summary {
  flex: 1 1 auto;
  min-width: 0;
}

.permission-group-toolbar__controls {
  flex: 0 1 auto;
  justify-content: flex-end;
}

.permission-group-toolbar__strategy {
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-default);
  padding: var(--rbac-permission-toolbar-inset-block) var(--rbac-permission-toolbar-inset-inline);
}

.permission-group-toolbar__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0;
}

.permission-group-toolbar__meta,
.permission-group-toolbar__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.permission-group-toolbar__meta {
  margin: 0;
}

.permission-group-toolbar__select {
  min-width: var(--rbac-permission-mode-min-width);
}

@media (width <= 768px) {
  .permission-group-toolbar,
  .permission-group-toolbar__summary,
  .permission-group-toolbar__controls {
    align-items: stretch;
    flex-direction: column;
  }

  .permission-group-toolbar__controls,
  .permission-group-toolbar__strategy,
  .permission-group-toolbar__actions {
    width: 100%;
  }

  .permission-group-toolbar__controls {
    justify-content: flex-start;
  }

  .permission-group-toolbar__actions {
    justify-content: flex-start;
  }
}
</style>
