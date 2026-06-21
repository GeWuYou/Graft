<template>
  <section class="dashboard-renderer">
    <template v-if="loading">
      <div class="dashboard-renderer__skeleton-grid" aria-hidden="true">
        <t-card
          v-for="item in skeletonItems"
          :key="item.key"
          class="dashboard-renderer__widget"
          :class="`dashboard-renderer__widget--${item.size}`"
          :bordered="true"
          size="small"
        >
          <t-skeleton animation="gradient" :row-col="skeletonRowCol" />
        </t-card>
      </div>
    </template>
    <template v-else-if="groupedWidgets.length">
      <section v-for="group in groupedWidgets" :key="group.category" class="dashboard-renderer__category">
        <header class="dashboard-renderer__category-header">
          <h2>{{ categoryLabel(group.category) }}</h2>
          <span>{{ t('dashboard.category.count', { count: group.widgets.length }) }}</span>
        </header>
        <div class="dashboard-renderer__grid">
          <t-card
            v-for="widget in group.widgets"
            :key="widget.id"
            class="dashboard-renderer__widget"
            :class="[
              `dashboard-renderer__widget--${widget.size}`,
              `dashboard-renderer__widget-state--${widget.state}`,
              { 'dashboard-renderer__widget--disabled': isDisabled(widget) },
            ]"
            :bordered="true"
            size="small"
          >
            <template #title>
              <div class="dashboard-renderer__heading">
                <span class="dashboard-renderer__title">{{ widgetTitle(widget) }}</span>
                <span class="dashboard-renderer__badges">
                  <t-tag v-if="widget.module_key" variant="light-outline" size="small">
                    {{ moduleLabel(widget.module_key) }}
                  </t-tag>
                  <t-tag :theme="priorityTheme(widget.priority)" variant="light-outline" size="small">
                    {{ priorityLabel(widget.priority) }}
                  </t-tag>
                </span>
              </div>
            </template>
            <template v-if="widgetActions(widget).length" #actions>
              <div class="dashboard-renderer__actions">
                <t-button
                  v-for="action in widgetActions(widget)"
                  :key="action.key"
                  variant="text"
                  theme="primary"
                  size="small"
                  :loading="action.key === 'retry' && refreshingWidgetId === widget.id"
                  @click="action.run"
                >
                  {{ action.label }}
                </t-button>
              </div>
            </template>

            <p v-if="widget.description_key || widget.description" class="dashboard-renderer__description">
              {{ resolveDashboardText(widget.description_key, widget.description) }}
            </p>

            <t-alert
              v-if="widget.status === 'error'"
              theme="error"
              :title="t('dashboard.widget.errorTitle')"
              :message="widgetErrorMessage(widget)"
            />
            <t-alert v-else-if="isDisabled(widget)" theme="info" :message="t('dashboard.widget.disabledDescription')" />
            <component :is="resolveWidgetComponent(widget.type)" v-else :widget="widget" />
          </t-card>
        </div>
      </section>
    </template>
    <t-empty v-else size="large" :description="t('dashboard.widget.empty')" />
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';

import { t } from '@/locales';

import type { DashboardWidget, DashboardWidgetCategory, DashboardWidgetType } from '../types/dashboard';
import AlertListWidget from './widgets/AlertListWidget.vue';
import HealthWidget from './widgets/HealthWidget.vue';
import LinkListWidget from './widgets/LinkListWidget.vue';
import StatGroupWidget from './widgets/StatGroupWidget.vue';
import TimelineWidget from './widgets/TimelineWidget.vue';
import { openDashboardRoute } from './widgets/widget-actions';
import { resolveDashboardText } from './widgets/widget-i18n';

const props = defineProps<{
  widgets: DashboardWidget[];
  refreshingWidgetId?: string;
  loading?: boolean;
}>();

const emit = defineEmits<{
  'refresh-widget': [widgetId: string];
}>();

const router = useRouter();

const skeletonItems = [
  { key: 'system', size: 'medium' },
  { key: 'security', size: 'medium' },
  { key: 'operation', size: 'medium' },
] as const;
const skeletonRowCol = [
  { width: '48%', height: '20px' },
  { width: '92%', height: '14px' },
  { width: '76%', height: '14px' },
  [
    { width: '28%', height: '28px' },
    { width: '28%', height: '28px', marginLeft: '12px' },
  ],
];

type WidgetGroup = {
  category: DashboardWidgetCategory;
  widgets: DashboardWidget[];
};

const groupedWidgets = computed<WidgetGroup[]>(() => {
  const groups = new Map<DashboardWidgetCategory, DashboardWidget[]>();
  for (const widget of sortedVisibleWidgets.value) {
    const items = groups.get(widget.category) ?? [];
    items.push(widget);
    groups.set(widget.category, items);
  }

  return [...groups.entries()]
    .map(([category, widgets]) => ({ category, widgets }))
    .sort((left, right) => {
      const priorityDiff = groupPriorityWeight(left.widgets) - groupPriorityWeight(right.widgets);
      if (priorityDiff !== 0) return priorityDiff;
      return categoryWeight(left.category) - categoryWeight(right.category);
    });
});

const sortedVisibleWidgets = computed(() =>
  props.widgets
    .filter((widget) => widget.visible && widget.state !== 'hidden')
    .sort((left, right) => {
      const priorityDiff = priorityWeight(left.priority) - priorityWeight(right.priority);
      if (priorityDiff !== 0) return priorityDiff;
      if (left.order !== right.order) return left.order - right.order;
      return left.id.localeCompare(right.id);
    }),
);

function resolveWidgetComponent(type: DashboardWidgetType) {
  const components = {
    'stat-group': StatGroupWidget,
    'alert-list': AlertListWidget,
    'link-list': LinkListWidget,
    timeline: TimelineWidget,
    health: HealthWidget,
  } satisfies Record<DashboardWidgetType, unknown>;

  return components[type];
}

function widgetTitle(widget: DashboardWidget) {
  return resolveDashboardText(widget.title_key, widget.title || widget.id);
}

function widgetErrorMessage(widget: DashboardWidget) {
  return resolveDashboardText(widget.error?.message_key, widget.error?.message || t('dashboard.widget.errorFallback'));
}

function isDisabled(widget: DashboardWidget) {
  return widget.status === 'disabled';
}

function canRefresh(widget: DashboardWidget) {
  return widget.status === 'error';
}

function widgetActions(widget: DashboardWidget) {
  const actions: Array<{ key: string; label: string; run: () => void }> = [];
  if (widget.action) {
    actions.push({
      key: 'details',
      label: resolveDashboardText(widget.action.label_key, widget.action.label, t('dashboard.actions.details')),
      run: () => openDashboardRoute(router, widget.action?.route ?? ''),
    });
  }
  if (canRefresh(widget)) {
    actions.push({
      key: 'retry',
      label: t('dashboard.actions.retry'),
      run: () => emit('refresh-widget', widget.id),
    });
  }
  return actions;
}

function priorityTheme(priority: DashboardWidget['priority']) {
  if (priority === 'critical') return 'danger';
  if (priority === 'warning') return 'warning';
  if (priority === 'normal') return 'primary';
  return 'default';
}

function priorityLabel(priority: DashboardWidget['priority']) {
  return t(`dashboard.widget.priority.${priority}`);
}

function categoryLabel(category: DashboardWidgetCategory) {
  return t(`dashboard.category.${category}`);
}

function moduleLabel(moduleKey: string) {
  const key = `dashboard.module.${moduleKey.replaceAll('.', '_')}`;
  return resolveDashboardText(key, moduleKey, moduleKey);
}

function priorityWeight(priority: DashboardWidget['priority']) {
  if (priority === 'critical') return 0;
  if (priority === 'warning') return 1;
  if (priority === 'normal') return 2;
  return 3;
}

function groupPriorityWeight(widgets: DashboardWidget[]) {
  return widgets.reduce((current, widget) => Math.min(current, priorityWeight(widget.priority)), 3);
}

function categoryWeight(category: DashboardWidgetCategory) {
  if (category === 'system') return 0;
  if (category === 'security') return 1;
  if (category === 'operation') return 2;
  return 3;
}
</script>
<style lang="less" scoped>
.dashboard-renderer {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xl);
  min-width: 0;
}

.dashboard-renderer__category {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-m);
  min-width: 0;
}

.dashboard-renderer__category-header {
  align-items: flex-end;
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: space-between;
  min-width: 0;
}

.dashboard-renderer__category-header h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
}

.dashboard-renderer__category-header span {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
}

.dashboard-renderer__grid,
.dashboard-renderer__skeleton-grid {
  display: grid;
  gap: var(--td-comp-margin-l);
  grid-template-columns: repeat(12, minmax(0, 1fr));
}

.dashboard-renderer__widget {
  grid-column: span 6;
  min-width: 0;
}

.dashboard-renderer__widget--small {
  grid-column: span 4;
}

.dashboard-renderer__widget--medium {
  grid-column: span 6;
}

.dashboard-renderer__widget--large {
  grid-column: 1 / -1;
}

.dashboard-renderer__widget-state--critical {
  border-color: color-mix(in srgb, var(--td-error-color-5) 36%, transparent);
}

.dashboard-renderer__widget-state--warning {
  border-color: color-mix(in srgb, var(--td-warning-color-5) 42%, transparent);
}

.dashboard-renderer__widget--disabled {
  opacity: 0.72;
}

.dashboard-renderer__heading,
.dashboard-renderer__actions {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-s);
  min-width: 0;
}

.dashboard-renderer__heading {
  justify-content: space-between;
}

.dashboard-renderer__title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-renderer__badges {
  align-items: center;
  display: flex;
  flex-shrink: 0;
  gap: var(--td-comp-margin-xs);
}

.dashboard-renderer__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0 0 var(--td-comp-margin-s);
}

@media (width <= 1200px) {
  .dashboard-renderer__widget,
  .dashboard-renderer__widget--small,
  .dashboard-renderer__widget--medium {
    grid-column: span 6;
  }
}

@media (width <= 768px) {
  .dashboard-renderer__grid,
  .dashboard-renderer__skeleton-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .dashboard-renderer__widget,
  .dashboard-renderer__widget--small,
  .dashboard-renderer__widget--medium,
  .dashboard-renderer__widget--large {
    grid-column: 1 / -1;
  }
}
</style>
