<template>
  <t-list class="workbench-list" :class="{ 'workbench-list--quiet': variant === 'health' }" split>
    <t-list-item
      v-for="item in visibleItems"
      :key="item.id"
      :class="itemClasses(item)"
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
            <strong>{{ itemTitle(item) }}</strong>
            <time v-if="item.occurredAt" :datetime="item.occurredAt">{{ formatTime(item.occurredAt) }}</time>
          </div>
          <strong v-else>{{ itemTitle(item) }}</strong>
          <p>{{ itemDescription(item) }}</p>
        </div>
      </div>
      <template #action>
        <t-button
          v-if="item.action"
          :variant="variant === 'attention' ? 'outline' : 'text'"
          :theme="variant === 'attention' ? 'default' : 'primary'"
          size="small"
          :loading="props.retryingId === item.id"
          @click="handleAction(item)"
        >
          {{ actionLabel(item) }}
        </t-button>
      </template>
    </t-list-item>
  </t-list>

  <p v-if="!items.length && emptyKey" class="workbench-empty">
    {{ t(emptyKey) }}
  </p>

  <t-collapse
    v-if="remainingItems.length"
    v-model="expandedPanels"
    borderless
    :expand-icon="false"
    :expand-on-row-click="false"
    class="workbench-collapse"
  >
    <t-collapse-panel :value="panelValue">
      <template #header>
        <t-button
          block
          theme="primary"
          type="button"
          variant="text"
          :aria-controls="panelContentId"
          :aria-expanded="overflowExpanded"
          :data-collapse-trigger="panelValue"
          @click.stop="toggleOverflow"
          @keydown.enter.prevent.stop="toggleOverflow"
          @keydown.space.prevent.stop="toggleOverflow"
        >
          {{ t(expandKey, { count: remainingItems.length }) }}
          <template #suffix>
            <chevron-down-icon :style="{ transform: overflowExpanded ? 'rotate(180deg)' : undefined }" />
          </template>
        </t-button>
      </template>
      <div :id="panelContentId" class="workbench-collapse__content" :data-collapse-content="panelValue">
        <t-list class="workbench-list" :class="{ 'workbench-list--quiet': variant === 'health' }" split>
          <t-list-item
            v-for="item in remainingItems"
            :key="item.id"
            :class="itemClasses(item)"
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
                <strong>{{ itemTitle(item) }}</strong>
                <p>{{ itemDescription(item) }}</p>
              </div>
            </div>
            <template #action>
              <t-button
                v-if="item.action"
                :variant="variant === 'attention' ? 'outline' : 'text'"
                :theme="variant === 'attention' ? 'default' : 'primary'"
                size="small"
                :loading="props.retryingId === item.id"
                @click="handleAction(item)"
              >
                {{ actionLabel(item) }}
              </t-button>
            </template>
          </t-list-item>
        </t-list>
      </div>
    </t-collapse-panel>
  </t-collapse>
</template>
<script setup lang="ts">
import { ChevronDownIcon } from 'tdesign-icons-vue-next';
import { computed, ref } from 'vue';

import { currentLocale, t } from '@/locales';
import { formatLocaleDateTime, MEDIUM_DATE_TIME_FORMAT_OPTIONS } from '@/shared/observability';

import type { PresentationItem } from '../../presentation/workbench';
import { hasDashboardTranslation, resolveDashboardText } from '../widgets/widget-i18n';
import WorkbenchStatusIndicator from './WorkbenchStatusIndicator.vue';

// 展示列表统一工作台事实行、折叠预算和下钻语义，父组件只决定区域信息层级。
const props = withDefaults(
  defineProps<{
    emptyKey?: string;
    expandKey?: string;
    items: PresentationItem[];
    retryingId?: string;
    variant: 'attention' | 'health' | 'activity';
    visibleLimit?: number;
  }>(),
  {
    emptyKey: '',
    expandKey: 'dashboard.workbench.expand.attention',
    retryingId: '',
    visibleLimit: Number.MAX_SAFE_INTEGER,
  },
);

const emit = defineEmits<{
  navigate: [route: string];
  'retry-item': [item: PresentationItem];
}>();

const expandedPanels = ref<string[]>([]);
const visibleItems = computed(() => props.items.slice(0, props.visibleLimit));
const remainingItems = computed(() => props.items.slice(props.visibleLimit));
const panelValue = computed(() => `${props.variant}-more`);
const panelContentId = computed(() => `workbench-${panelValue.value}-content`);
const overflowExpanded = computed(() => expandedPanels.value.includes(panelValue.value));
const itemDataAttribute = computed(() => `data-${props.variant}-id`);

function toggleOverflow() {
  expandedPanels.value = overflowExpanded.value ? [] : [panelValue.value];
}

function itemClasses(item: PresentationItem) {
  return [`${props.variant}-row`, props.variant === 'attention' ? `attention-row--${item.status}` : undefined];
}

function formatTime(value: string) {
  return formatLocaleDateTime(value, currentLocale, MEDIUM_DATE_TIME_FORMAT_OPTIONS);
}

function handleAction(item: PresentationItem) {
  if (item.action?.kind === 'retry') {
    emit('retry-item', item);
    return;
  }
  if (item.action?.kind === 'navigate') {
    emit('navigate', item.action.route);
  }
}

function itemTitle(item: PresentationItem) {
  if (hasDashboardTranslation(item.titleKey)) {
    return t(item.titleKey, item.titleParams ?? {});
  }
  return resolveDashboardText(item.titleKey, item.titleFallback || item.id);
}

function itemDescription(item: PresentationItem) {
  if (hasDashboardTranslation(item.descriptionKey)) {
    return t(item.descriptionKey, item.descriptionParams ?? {});
  }
  return resolveDashboardText(item.descriptionKey, item.descriptionFallback || '');
}

function actionLabel(item: PresentationItem) {
  if (!item.action) {
    return '';
  }
  return resolveDashboardText(item.action.labelKey, item.action.labelFallback || '');
}
</script>
<style scoped lang="less">
.workbench-list,
.workbench-list :deep(.t-list-item) {
  background: transparent;
}

.workbench-list :deep(.t-list-item) {
  padding: var(--graft-density-gap-16) 0;
}

.workbench-list--quiet :deep(.t-list-item) {
  padding: var(--graft-density-gap-8) 0;
}

.workbench-empty {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-large);
  margin: 0;
  padding: var(--graft-density-gap-16) 0;
}

.workbench-collapse {
  background: transparent;
  margin-top: var(--graft-density-gap-8);
}

.workbench-collapse :deep(.t-collapse-panel__body) {
  background: transparent;
}

.workbench-collapse :deep(.t-collapse-panel__content) {
  color: inherit;
  padding: 0;
}

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
