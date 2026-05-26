<template>
  <button
    class="assignment-card"
    :class="{
      'assignment-card--assigned': assigned,
      'assignment-card--disabled': disabled,
      'assignment-card--selected': selected,
    }"
    :aria-checked="selected"
    :disabled="disabled"
    role="checkbox"
    type="button"
    @click="emit('toggle')"
  >
    <div class="assignment-card__surface">
      <div class="assignment-card__top">
        <div class="assignment-card__state">
          <div class="assignment-card__check">
            <span v-if="selected">✓</span>
          </div>
          <t-tag v-if="assigned" theme="primary" variant="light" size="small">{{ assignedLabel }}</t-tag>
        </div>
        <div v-if="tags.length > 0" class="assignment-card__tags">
          <t-tag
            v-for="tag in tags"
            :key="`${tag.label}-${tag.theme ?? 'default'}`"
            :theme="tag.theme ?? 'default'"
            :variant="tag.variant ?? 'light'"
            size="small"
          >
            {{ tag.label }}
          </t-tag>
        </div>
      </div>

      <div class="assignment-card__body">
        <span class="assignment-card__title">{{ title }}</span>
        <span class="assignment-card__code">{{ code }}</span>
        <span class="assignment-card__description">{{ description }}</span>
      </div>
    </div>
  </button>
</template>
<script setup lang="ts">
type CardTag = {
  label: string;
  theme?: 'danger' | 'default' | 'primary' | 'success' | 'warning';
  variant?: 'dark' | 'light' | 'light-outline' | 'outline';
};

withDefaults(
  defineProps<{
    assigned?: boolean;
    assignedLabel?: string;
    code: string;
    description: string;
    disabled?: boolean;
    selected?: boolean;
    tags?: CardTag[];
    title: string;
  }>(),
  {
    assigned: false,
    assignedLabel: '',
    disabled: false,
    selected: false,
    tags: () => [],
  },
);

const emit = defineEmits<{
  toggle: [];
}>();
</script>
<style scoped lang="less">
.assignment-card,
.assignment-card__surface,
.assignment-card__top,
.assignment-card__state,
.assignment-card__tags,
.assignment-card__body,
.assignment-card__check {
  display: flex;
}

.assignment-card {
  appearance: none;
  background: transparent;
  border: 0;
  cursor: pointer;
  padding: 0;
  text-align: left;
  width: 100%;
}

.assignment-card__surface {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  flex: 1;
  flex-direction: column;
  gap: var(--td-comp-margin-m);
  min-height: 164px;
  padding: var(--td-comp-paddingTB-l) var(--td-comp-paddingLR-l);
  transition:
    border-color 180ms ease,
    box-shadow 180ms ease,
    transform 180ms ease;
  width: 100%;
}

.assignment-card:hover .assignment-card__surface {
  border-color: color-mix(in srgb, var(--td-brand-color) 26%, var(--td-component-stroke));
  box-shadow: var(--td-shadow-1);
}

.assignment-card--selected .assignment-card__surface {
  border-color: var(--td-brand-color);
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--td-brand-color) 28%, transparent);
}

.assignment-card--disabled {
  cursor: not-allowed;
}

.assignment-card:focus {
  outline: none;
}

.assignment-card:focus-visible .assignment-card__surface {
  box-shadow:
    0 0 0 2px color-mix(in srgb, var(--td-brand-color) 18%, transparent),
    var(--td-shadow-1);
}

.assignment-card--disabled .assignment-card__surface {
  opacity: 0.56;
}

.assignment-card__top {
  align-items: flex-start;
  gap: var(--td-comp-margin-s);
  justify-content: space-between;
}

.assignment-card__state,
.assignment-card__tags {
  align-items: center;
  flex-wrap: wrap;
  gap: var(--td-comp-margin-s);
}

.assignment-card__check {
  align-items: center;
  border: 1px solid var(--td-component-stroke);
  border-radius: 999px;
  color: transparent;
  flex: 0 0 24px;
  font: var(--td-font-title-small);
  height: 24px;
  justify-content: center;
  width: 24px;
}

.assignment-card--selected .assignment-card__check {
  background: var(--td-brand-color);
  border-color: var(--td-brand-color);
  color: var(--td-text-color-anti);
}

.assignment-card__body {
  flex: 1;
  flex-direction: column;
  gap: 6px;
}

.assignment-card__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
}

.assignment-card__code,
.assignment-card__description {
  color: var(--td-text-color-secondary);
}

.assignment-card__description {
  line-height: 1.6;
}
</style>
