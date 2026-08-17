<template>
  <t-list-item
    :class="itemClasses"
    :data-status="item.status"
    :data-evidence="item.evidenceState"
    :[itemDataAttribute]="item.id"
  >
    <div class="workbench-row" :class="{ 'workbench-row--compact': variant !== 'attention' }">
      <workbench-status-indicator
        :status="item.status"
        :label="t(`dashboard.workbench.status.${item.status}`)"
        :show-label="variant === 'attention'"
      />
      <div class="workbench-row__copy">
        <div v-if="variant === 'activity'" class="workbench-row__title-line">
          <strong>{{ itemTitle }}</strong>
          <time v-if="item.occurredAt" :datetime="item.occurredAt">{{ formatTime(item.occurredAt) }}</time>
        </div>
        <strong v-else>{{ itemTitle }}</strong>
        <p>{{ itemDescription }}</p>
      </div>
    </div>
    <template #action>
      <t-button
        v-if="item.action"
        :variant="variant === 'attention' ? 'outline' : 'text'"
        :theme="variant === 'attention' ? 'default' : 'primary'"
        size="small"
        :loading="retryingId === item.id"
        @click="emit('activate', item)"
      >
        {{ actionLabel }}
      </t-button>
    </template>
  </t-list-item>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { currentLocale, t } from '@/locales';
import { formatLocaleDateTime, MEDIUM_DATE_TIME_FORMAT_OPTIONS } from '@/shared/observability';

import type { PresentationItem } from '../../presentation/workbench';
import { hasDashboardTranslation, resolveDashboardText } from '../widgets/widget-i18n';
import WorkbenchStatusIndicator from './WorkbenchStatusIndicator.vue';

// 单行组件统一首屏与折叠区的事实、时间和操作渲染，避免两套标记产生语义漂移。
const props = withDefaults(
  defineProps<{
    item: PresentationItem;
    retryingId?: string;
    variant: 'attention' | 'health' | 'activity';
  }>(),
  { retryingId: '' },
);

const emit = defineEmits<{
  activate: [item: PresentationItem];
}>();

const itemClasses = computed(() => [
  `${props.variant}-row`,
  props.variant === 'attention' ? `attention-row--${props.item.status}` : undefined,
]);
const itemDataAttribute = computed(() => `data-${props.variant}-id`);
const itemTitle = computed(() => {
  if (hasDashboardTranslation(props.item.titleKey)) {
    return t(props.item.titleKey, props.item.titleParams ?? {});
  }
  return resolveDashboardText(props.item.titleKey, props.item.titleFallback || props.item.id);
});
const itemDescription = computed(() => {
  if (hasDashboardTranslation(props.item.descriptionKey)) {
    return t(props.item.descriptionKey, props.item.descriptionParams ?? {});
  }
  return resolveDashboardText(props.item.descriptionKey, props.item.descriptionFallback || '');
});
const actionLabel = computed(() => {
  if (!props.item.action) {
    return '';
  }
  return resolveDashboardText(props.item.action.labelKey, props.item.action.labelFallback || '');
});

function formatTime(value: string) {
  return formatLocaleDateTime(value, currentLocale, MEDIUM_DATE_TIME_FORMAT_OPTIONS);
}
</script>
<style scoped lang="less">
.attention-row {
  border-left: 3px solid transparent;
  padding-left: var(--graft-density-gap-16) !important;
}

.attention-row--warning {
  background: color-mix(in srgb, var(--td-warning-color-light) 12%, transparent) !important;
  border-left-color: var(--td-warning-color);
}

.attention-row--error {
  background: color-mix(in srgb, var(--td-error-color-light) 10%, transparent) !important;
  border-left-color: var(--td-error-color);
}

.attention-row--unknown {
  background: color-mix(in srgb, var(--td-bg-color-container-hover) 22%, transparent) !important;
  border-left-color: var(--td-text-color-placeholder);
}

.workbench-row {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.workbench-row--compact {
  gap: var(--graft-density-gap-8);
}

.workbench-row__copy {
  flex: 1 1 auto;
  min-width: 0;
}

.workbench-row__copy strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.attention-row .workbench-row__copy > strong {
  font: var(--td-font-title-small);
}

.workbench-row__copy p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-large);
  margin: var(--graft-density-gap-6) 0 0;
}

.health-row .workbench-row__copy p {
  font: var(--td-font-body-medium);
  margin-top: var(--graft-density-gap-2);
}

.workbench-row__title-line {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
}

.workbench-row__title-line time {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-large);
}
</style>
