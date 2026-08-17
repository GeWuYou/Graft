<template>
  <button
    class="workbench-quick-action-item"
    :class="`workbench-quick-action-item--${variant}`"
    type="button"
    @click="emit('activate', action)"
  >
    <span class="workbench-quick-action-item__icon">
      <graft-menu-icon :icon-key="action.iconKey" />
    </span>
    <span class="workbench-quick-action-item__copy">
      <strong>{{ resolveDashboardText(action.titleKey, action.titleFallback || action.id) }}</strong>
      <span>{{ resolveDashboardText(action.descriptionKey, action.descriptionFallback || '') }}</span>
    </span>
    <chevron-right-icon
      v-if="variant === 'drawer-featured' && action.kind === 'action'"
      class="workbench-quick-action-item__arrow"
    />
  </button>
</template>
<script setup lang="ts">
import { ChevronRightIcon } from 'tdesign-icons-vue-next';

import GraftMenuIcon from '@/shared/icons/MenuIcon.vue';

import type { QuickAction } from '../../presentation/workbench';
import { resolveDashboardText } from '../widgets/widget-i18n';

// 工作台快捷入口统一首页与抽屉的动作语义，数据与可见范围仍由 presentation model 决定。
withDefaults(
  defineProps<{
    action: QuickAction;
    variant?: 'default' | 'drawer-featured';
  }>(),
  { variant: 'default' },
);

const emit = defineEmits<{
  activate: [action: QuickAction];
}>();
</script>
<style scoped lang="less">
.workbench-quick-action-item {
  align-items: center;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  color: inherit;
  cursor: pointer;
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: auto minmax(0, 1fr);
  min-height: 64px;
  min-width: 0;
  padding: var(--graft-density-gap-10) var(--graft-density-gap-12);
  text-align: left;
  transition:
    background-color 120ms ease,
    border-color 120ms ease;
  width: 100%;
}

.workbench-quick-action-item:hover {
  background: var(--td-bg-color-container-hover);
  border-color: var(--td-border-level-2-color);
}

.workbench-quick-action-item:focus-visible {
  border-color: var(--td-brand-color);
  box-shadow: inset 0 0 0 1px var(--td-brand-color);
  outline: none;
}

.workbench-quick-action-item__icon {
  align-items: center;
  background: var(--td-bg-color-container-hover);
  border-radius: var(--td-radius-small);
  color: var(--td-text-color-secondary);
  display: inline-flex;
  height: 28px;
  justify-content: center;
  width: 28px;
}

.workbench-quick-action-item__icon :deep(svg) {
  height: 16px;
  width: 16px;
}

.workbench-quick-action-item__copy {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
  min-width: 0;
}

.workbench-quick-action-item__copy strong,
.workbench-quick-action-item__copy > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.workbench-quick-action-item__copy strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
  font-weight: 500;
}

.workbench-quick-action-item__copy > span {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.workbench-quick-action-item--drawer-featured {
  border-color: transparent;
  grid-template-columns: auto minmax(0, 1fr) auto;
  min-height: 56px;
  padding-inline: var(--graft-density-gap-8);
}

.workbench-quick-action-item--drawer-featured:hover {
  border-color: transparent;
}

.workbench-quick-action-item__arrow {
  color: var(--td-text-color-placeholder);
}
</style>
