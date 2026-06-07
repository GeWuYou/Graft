<template>
  <section class="dashboard-renderer">
    <div v-if="widgets.length" class="dashboard-renderer__grid">
      <t-card
        v-for="widget in sortedWidgets"
        :key="widget.id"
        class="dashboard-renderer__widget"
        :class="[
          `dashboard-renderer__widget--${widget.size}`,
          { 'dashboard-renderer__widget--disabled': isDisabled(widget) },
        ]"
        :bordered="true"
        size="small"
      >
        <template #title>
          <div class="dashboard-renderer__heading">
            <span>{{ widgetTitle(widget) }}</span>
            <t-tag
              v-if="widget.status && widget.status !== 'normal'"
              :theme="statusTheme(widget.status)"
              variant="light"
            >
              {{ statusLabel(widget.status) }}
            </t-tag>
          </div>
        </template>
        <template v-if="canRefresh(widget)" #actions>
          <t-button
            variant="text"
            theme="primary"
            size="small"
            :loading="refreshingWidgetId === widget.id"
            @click="emit('refresh-widget', widget.id)"
          >
            {{ t('dashboard.actions.retry') }}
          </t-button>
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
    <t-empty v-else size="large" :description="t('dashboard.widget.empty')" />
  </section>
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { t } from '@/locales';

import type { DashboardWidget, DashboardWidgetStatus, DashboardWidgetType } from '../types/dashboard';
import AlertListWidget from './widgets/AlertListWidget.vue';
import HealthWidget from './widgets/HealthWidget.vue';
import LinkListWidget from './widgets/LinkListWidget.vue';
import StatGroupWidget from './widgets/StatGroupWidget.vue';
import TimelineWidget from './widgets/TimelineWidget.vue';
import { resolveDashboardText } from './widgets/widget-i18n';

const props = defineProps<{
  widgets: DashboardWidget[];
  refreshingWidgetId?: string;
}>();

const emit = defineEmits<{
  'refresh-widget': [widgetId: string];
}>();

const sortedWidgets = computed(() => [...props.widgets].sort((left, right) => left.order - right.order));

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

function statusTheme(status: DashboardWidgetStatus) {
  if (status === 'error') return 'danger';
  if (status === 'warning') return 'warning';
  return 'default';
}

function statusLabel(status: DashboardWidgetStatus) {
  return t(`dashboard.widget.status.${status}`);
}
</script>
<style lang="less" scoped>
.dashboard-renderer {
  min-width: 0;
}

.dashboard-renderer__grid {
  display: grid;
  gap: var(--td-comp-margin-l);
  grid-template-columns: repeat(12, minmax(0, 1fr));
}

.dashboard-renderer__widget {
  grid-column: span 6;
  min-width: 0;
}

.dashboard-renderer__widget--small {
  grid-column: span 3;
}

.dashboard-renderer__widget--medium {
  grid-column: span 6;
}

.dashboard-renderer__widget--large {
  grid-column: span 9;
}

.dashboard-renderer__widget--full {
  grid-column: 1 / -1;
}

.dashboard-renderer__widget--disabled {
  opacity: 0.72;
}

.dashboard-renderer__heading {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-s);
  justify-content: space-between;
  min-width: 0;
}

.dashboard-renderer__heading span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-renderer__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0 0 var(--td-comp-margin-s);
}

@media (width <= 1200px) {
  .dashboard-renderer__widget,
  .dashboard-renderer__widget--small,
  .dashboard-renderer__widget--medium,
  .dashboard-renderer__widget--large {
    grid-column: span 6;
  }
}

@media (width <= 768px) {
  .dashboard-renderer__grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .dashboard-renderer__widget,
  .dashboard-renderer__widget--small,
  .dashboard-renderer__widget--medium,
  .dashboard-renderer__widget--large,
  .dashboard-renderer__widget--full {
    grid-column: 1 / -1;
  }
}
</style>
