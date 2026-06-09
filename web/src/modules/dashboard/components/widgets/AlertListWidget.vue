<template>
  <t-list v-if="payload && groupedItems.length" size="small" split>
    <t-list-item v-for="item in groupedItems" :key="item.id">
      <div class="dashboard-alert-list__item">
        <t-tag :theme="levelTheme(item.level)" variant="light">{{ levelLabel(item.level) }}</t-tag>
        <div class="dashboard-alert-list__content">
          <div class="dashboard-alert-list__title-row">
            <strong>{{ item.title }}</strong>
            <t-tag v-if="item.count > 1" size="small" variant="light-outline">
              {{ t('dashboard.alert.count', { count: item.count }) }}
            </t-tag>
          </div>
          <p v-if="item.description">{{ item.description }}</p>
          <time v-if="item.latestAt">
            {{ t('dashboard.alert.latestAt', { time: formatDashboardDateTime(item.latestAt) }) }}
          </time>
        </div>
      </div>
      <template v-if="item.route_location" #action>
        <t-button variant="text" theme="primary" size="small" @click="go(item.route_location)">
          {{ t('dashboard.actions.open') }}
        </t-button>
      </template>
    </t-list-item>
  </t-list>
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

import { t } from '@/locales';

import type { DashboardAlertListPayload, DashboardWidget } from '../../types/dashboard';
import { asAlertListPayload } from './payload';
import { formatDashboardDateTime, openDashboardRoute } from './widget-actions';
import { resolveDashboardRelatedText, resolveDashboardText } from './widget-i18n';

const props = defineProps<{
  widget: DashboardWidget;
}>();

type AlertLevel = DashboardAlertListPayload['items'][number]['level'];

const router = useRouter();
const payload = computed(() => asAlertListPayload(props.widget.payload));
const groupedItems = computed(() => {
  const currentPayload = payload.value;
  if (!currentPayload) {
    return [];
  }

  const groups = new Map<string, AlertGroup>();
  for (const item of currentPayload.items) {
    const title = normalizedTitle(resolveDashboardText(item.title_key, item.title));
    const key = alertGroupKey(item, title);
    const existing = groups.get(key);
    const occurredAt = item.occurred_at || '';
    if (!existing) {
      groups.set(key, {
        id: key,
        count: 1,
        description: itemDescription(item),
        latestAt: occurredAt,
        level: item.level,
        route_location: item.route_location,
        title,
      });
      continue;
    }

    existing.count += 1;
    if (isAfter(occurredAt, existing.latestAt)) {
      existing.latestAt = occurredAt;
      existing.description = itemDescription(item);
      existing.route_location = item.route_location || existing.route_location;
    }
  }

  return [...groups.values()].sort((left, right) => {
    const levelDelta = levelWeight(left.level) - levelWeight(right.level);
    if (levelDelta !== 0) {
      return levelDelta;
    }
    return timestamp(right.latestAt) - timestamp(left.latestAt);
  });
});

type AlertGroup = {
  id: string;
  level: AlertLevel;
  title: string;
  description: string;
  latestAt: string;
  route_location?: string;
  count: number;
};

function levelTheme(level: AlertLevel) {
  if (level === 'error') return 'danger';
  if (level === 'warning') return 'warning';
  return 'primary';
}

function levelLabel(level: AlertLevel) {
  return t(`dashboard.alert.level.${level}`);
}

function itemDescription(item: DashboardAlertListPayload['items'][number]) {
  return item.description_key
    ? resolveDashboardText(item.description_key, item.description)
    : resolveDashboardRelatedText(item.title_key, 'description', item.description);
}

function alertGroupKey(item: DashboardAlertListPayload['items'][number], title: string) {
  const statusCode = item.description?.match(/\b([1-5]\d{2})\b/)?.[1];
  return [item.level, statusCode || title].join(':');
}

function normalizedTitle(value: string) {
  if (value === 'token_expired') {
    return t('dashboard.alert.known.tokenExpired');
  }

  return value.replaceAll('_', ' ').replace(/\b\w/g, (match) => match.toUpperCase());
}

function timestamp(value: string) {
  const date = new Date(value).getTime();
  return Number.isFinite(date) ? date : 0;
}

function isAfter(left: string, right: string) {
  return timestamp(left) > timestamp(right);
}

function levelWeight(level: AlertLevel) {
  if (level === 'error') return 0;
  if (level === 'warning') return 1;
  return 2;
}

function go(location: string) {
  openDashboardRoute(router, location);
}
</script>
<style lang="less" scoped>
.dashboard-alert-list__item {
  align-items: flex-start;
  display: flex;
  gap: var(--td-comp-margin-s);
  min-width: 0;
}

.dashboard-alert-list__content {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: var(--td-comp-margin-xxs);
  min-width: 0;
}

.dashboard-alert-list__title-row {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-xs);
  min-width: 0;
}

.dashboard-alert-list__content strong,
.dashboard-alert-list__content p {
  overflow-wrap: anywhere;
}

.dashboard-alert-list__content p,
.dashboard-alert-list__content time {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}
</style>
