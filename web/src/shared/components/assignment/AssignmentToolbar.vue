<template>
  <section class="assignment-toolbar">
    <t-input
      :model-value="searchValue"
      clearable
      class="assignment-toolbar__search"
      :placeholder="searchPlaceholder"
      @update:model-value="emit('update:searchValue', $event)"
    />
    <div class="assignment-toolbar__mode">
      <span class="assignment-toolbar__label">{{ modeLabel }}</span>
      <t-select
        :model-value="modeValue"
        class="assignment-toolbar__select"
        :disabled="disabled"
        :options="modeOptions"
        @update:model-value="emit('update:modeValue', $event as string)"
      />
    </div>
  </section>
</template>
<script setup lang="ts">
type Option = {
  label: string;
  value: string;
};

defineProps<{
  disabled?: boolean;
  modeLabel: string;
  modeOptions: Option[];
  modeValue: string;
  searchPlaceholder: string;
  searchValue: string;
}>();

const emit = defineEmits<{
  'update:modeValue': [value: string];
  'update:searchValue': [value: string];
}>();
</script>
<style scoped lang="less">
.assignment-toolbar,
.assignment-toolbar__mode {
  display: flex;
}

.assignment-toolbar {
  align-items: center;
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  gap: var(--td-comp-margin-l);
  justify-content: space-between;
  padding: var(--td-comp-paddingTB-m) var(--td-comp-paddingLR-l);
}

.assignment-toolbar__search {
  flex: 1 1 320px;
}

.assignment-toolbar__mode {
  align-items: center;
  flex: 0 0 auto;
  gap: var(--td-comp-margin-s);
}

.assignment-toolbar__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  white-space: nowrap;
}

.assignment-toolbar__select {
  min-width: 220px;
}

@media (width <= 768px) {
  .assignment-toolbar,
  .assignment-toolbar__mode {
    align-items: stretch;
    flex-direction: column;
  }

  .assignment-toolbar__select {
    min-width: 100%;
  }
}
</style>
