<template>
  <t-timeline v-if="payload && payload.items.length" mode="same" theme="dot">
    <t-timeline-item
      v-for="item in payload.items"
      :key="item.id"
      :dot-color="timelineDotColor(item.status)"
      :label="formatDashboardDateTime(item.occurred_at, currentLocale)"
    >
      <div class="dashboard-timeline__item">
        <strong>{{ resolveDashboardText(item.title_key, item.title) }}</strong>
        <p v-if="item.description_key || item.description">
          {{ resolveDashboardText(item.description_key, item.description) }}
        </p>
        <t-button
          v-if="item.route_location"
          variant="text"
          theme="primary"
          size="small"
          @click="go(item.route_location)"
        >
          {{ t('dashboard.actions.open') }}
        </t-button>
      </div>
    </t-timeline-item>
  </t-timeline>
  <t-empty
    v-else-if="payload"
    size="small"
    :description="resolveDashboardText(payload.empty_key, payload.empty || t('dashboard.widget.empty'))"
  />
  <t-empty v-else size="small" :description="t('dashboard.widget.invalidPayload')" />
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';

import { currentLocale, t } from '@/locales';

import type { DashboardTimelinePayload, DashboardWidget } from '../../types/dashboard';
import { asTimelinePayload } from './payload';
import { formatDashboardDateTime, openDashboardRoute } from './widget-actions';
import { resolveDashboardText } from './widget-i18n';

const props = defineProps<{
  widget: DashboardWidget;
}>();

type TimelineStatus = DashboardTimelinePayload['items'][number]['status'];

const router = useRouter();
const payload = computed(() => asTimelinePayload(props.widget.payload));

function timelineDotColor(status: TimelineStatus) {
  if (status === 'error') return 'error';
  if (status === 'warning') return 'warning';
  return 'primary';
}

function go(location: string) {
  openDashboardRoute(router, location);
}
</script>
<style lang="less" scoped>
.dashboard-timeline__item {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xxs);
  min-width: 0;
}

.dashboard-timeline__item strong,
.dashboard-timeline__item p {
  overflow-wrap: anywhere;
}

.dashboard-timeline__item p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}
</style>
