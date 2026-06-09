<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div v-if="payload && payload.items.length" class="dashboard-stat-group">
    <div v-for="item in payload.items" :key="item.key" class="dashboard-stat-group__item">
      <span class="dashboard-stat-group__label">{{ resolveDashboardText(item.label_key, item.label) }}</span>
      <strong class="dashboard-stat-group__value">
        {{ item.value
        }}<span v-if="item.unit_key || item.unit">{{ resolveDashboardText(item.unit_key, item.unit) }}</span>
      </strong>
      <p v-if="item.description_key || item.description" class="dashboard-stat-group__description">
        {{ itemDescription(item) }}
      </p>
      <t-button v-if="item.route_location" variant="text" theme="primary" size="small" @click="go(item.route_location)">
        {{ t('dashboard.actions.open') }}
      </t-button>
    </div>
  </div>
  <t-empty v-else-if="payload" size="small" :description="t('dashboard.widget.empty')" />
  <t-empty v-else size="small" :description="t('dashboard.widget.invalidPayload')" />
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';

import { t } from '@/locales';

import type { DashboardWidget } from '../../types/dashboard';
import { asStatGroupPayload } from './payload';
import { openDashboardRoute } from './widget-actions';
import { resolveDashboardRelatedText, resolveDashboardText } from './widget-i18n';

const props = defineProps<{
  widget: DashboardWidget;
}>();

const router = useRouter();
const payload = computed(() => asStatGroupPayload(props.widget.payload));

type StatGroupItem = NonNullable<ReturnType<typeof asStatGroupPayload>>['items'][number];

function itemDescription(item: StatGroupItem) {
  return item.description_key
    ? resolveDashboardText(item.description_key, item.description)
    : resolveDashboardRelatedText(item.label_key, 'description', item.description);
}

function go(location: string) {
  openDashboardRoute(router, location);
}
</script>
<style lang="less" scoped>
.dashboard-stat-group {
  display: grid;
  gap: var(--td-comp-margin-m);
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
}

.dashboard-stat-group__item {
  background: var(--graft-card-bg-hover);
  border: 1px solid var(--graft-card-border-color);
  border-radius: var(--td-radius-medium);
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xs);
  min-width: 0;
  padding: var(--td-comp-paddingTB-m) var(--td-comp-paddingLR-l);
}

.dashboard-stat-group__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.dashboard-stat-group__value {
  color: var(--graft-metric-value-color);
  display: flex;
  font: var(--td-font-headline-small);
  gap: var(--td-comp-margin-xxs);
  line-height: 1.2;
  overflow-wrap: anywhere;
}

.dashboard-stat-group__value span {
  align-self: flex-end;
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.dashboard-stat-group__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}
</style>
