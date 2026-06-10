<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-list v-if="payload && payload.items.length" size="small" split>
    <t-list-item v-for="item in payload.items" :key="item.id">
      <div class="dashboard-alert-list__item">
        <t-tag :theme="levelTheme(item.level)" variant="light">{{ levelLabel(item.level) }}</t-tag>
        <div class="dashboard-alert-list__content">
          <div class="dashboard-alert-list__title-row">
            <strong>{{ alertTitle(item) }}</strong>
            <t-tag v-if="item.count && item.count > 1" size="small" variant="light-outline">
              {{ t('dashboard.alert.count', { count: item.count }) }}
            </t-tag>
          </div>
          <p v-if="itemDescription(item)">{{ itemDescription(item) }}</p>
          <time v-if="item.occurred_at">
            {{ t('dashboard.alert.latestAt', { time: formatDashboardDateTime(item.occurred_at) }) }}
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
const KNOWN_ALERT_TITLE_KEYS = {
  token_expired: 'dashboard.alert.known.tokenExpired',
} as const;

const router = useRouter();
const payload = computed(() => asAlertListPayload(props.widget.payload));

function levelTheme(level: AlertLevel) {
  if (level === 'error') return 'danger';
  if (level === 'warning') return 'warning';
  return 'primary';
}

function levelLabel(level: AlertLevel) {
  return t(`dashboard.alert.level.${level}`);
}

function alertTitle(item: DashboardAlertListPayload['items'][number]) {
  return normalizedTitle(resolveDashboardText(item.title_key, item.title));
}

function itemDescription(item: DashboardAlertListPayload['items'][number]) {
  return item.description_key
    ? resolveDashboardText(item.description_key, item.description)
    : resolveDashboardRelatedText(item.title_key, 'description', item.description);
}

function normalizedTitle(value: string) {
  const knownAlertTitleKey = KNOWN_ALERT_TITLE_KEYS[value as keyof typeof KNOWN_ALERT_TITLE_KEYS];
  if (knownAlertTitleKey) {
    return resolveDashboardText(knownAlertTitleKey, value);
  }

  return value.replaceAll('_', ' ').replace(/\b\w/g, (match) => match.toUpperCase());
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
