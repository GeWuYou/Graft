<template>
  <div class="table-action-menu" @click.stop>
    <t-button
      v-if="primaryAction"
      :data-testid="primaryAction.testId"
      :disabled="primaryAction.disabled"
      size="small"
      theme="primary"
      variant="outline"
      @click="handlePrimaryClick"
    >
      {{ primaryAction.label }}
    </t-button>
    <t-dropdown v-if="menuOptions.length > 0" :options="menuOptions" trigger="click" @click="handleMenuClick">
      <t-button :aria-label="resolvedMoreLabel" size="small" theme="default" variant="outline" @click.stop>
        {{ resolvedMoreLabel }}
      </t-button>
    </t-dropdown>
  </div>
</template>
<script setup lang="ts">
import type { DropdownProps } from 'tdesign-vue-next';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

type ActionOption = {
  danger?: boolean;
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
    moreLabel: '',
    moreLabelFallback: 'components.commonTable.more',
  },
);

const emit = defineEmits<{
  action: [value: string];
}>();

const I18N_KEY_PATTERN = /^[a-z][\w-]*(\.[A-Za-z0-9_-]+)+$/;
const { t } = useI18n();

function resolveLabel(label: string, fallbackLabel?: string) {
  if (!label || I18N_KEY_PATTERN.test(label)) {
    const fallback = fallbackLabel ?? label;
    if (I18N_KEY_PATTERN.test(fallback)) {
      return t(fallback);
    }

    return fallback;
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
    theme: action.danger ? ('error' as const) : undefined,
    value: action.value,
  })),
);

function handlePrimaryClick(event?: MouseEvent) {
  event?.stopPropagation();

  const action = primaryAction.value;

  if (action && !action.disabled) {
    emit('action', action.value);
  }
}

const handleMenuClick: NonNullable<DropdownProps['onClick']> = (payload, context) => {
  context?.e?.stopPropagation();

  const action = typeof payload === 'object' && payload ? payload.value : payload;
  if (typeof action === 'string') {
    emit('action', action);
  }
};
</script>
<style scoped lang="less">
.table-action-menu {
  align-items: center;
  display: inline-flex;
  gap: var(--graft-density-gap-6);
  justify-content: center;
  width: 100%;
}

.table-action-menu :deep(.t-button) {
  min-width: 56px;
  white-space: nowrap;
}

.table-action-menu :deep(.t-button.t-button--variant-outline.t-button--theme-default) {
  background: color-mix(in srgb, var(--graft-card-elevated-bg) 88%, var(--td-bg-color-page));
  border-color: color-mix(in srgb, var(--graft-card-border-color) 92%, transparent);
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--graft-card-bg) 88%, transparent);
  color: var(--td-text-color-primary);
}

.table-action-menu :deep(.t-button.t-button--variant-outline.t-button--theme-default:hover),
.table-action-menu :deep(.t-button.t-button--variant-outline.t-button--theme-default:focus-visible) {
  background: color-mix(in srgb, var(--td-brand-color) 5%, var(--graft-card-elevated-bg));
  border-color: color-mix(in srgb, var(--td-brand-color) 20%, var(--graft-card-border-color));
}

.table-action-menu :deep(.t-button.t-button--variant-outline.t-button--theme-default:disabled) {
  background: color-mix(in srgb, var(--graft-card-bg) 82%, var(--td-bg-color-page));
  border-color: color-mix(in srgb, var(--graft-card-border-color) 76%, transparent);
  box-shadow: none;
  color: var(--td-text-color-disabled);
  opacity: 1;
}

.table-action-menu :deep(.t-dropdown) {
  flex: none;
}
</style>
