<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div v-if="payload" class="dashboard-health">
    <div class="dashboard-health__summary">
      <t-tag :theme="healthTheme(payload.summary.status)" variant="light">
        {{
          resolveDashboardText(payload.summary.label_key, payload.summary.label || healthLabel(payload.summary.status))
        }}
      </t-tag>
    </div>
    <t-list v-if="payload.items.length" size="small" split>
      <t-list-item v-for="item in payload.items" :key="item.key">
        <div class="dashboard-health__item">
          <strong>{{ resolveDashboardText(item.label_key, item.label) }}</strong>
          <p v-if="item.description_key || item.description">
            {{ resolveDashboardText(item.description_key, item.description) }}
          </p>
          <div v-if="usagePercent(item) !== null" class="dashboard-health__usage">
            <t-progress
              theme="line"
              size="small"
              :percentage="usagePercent(item) ?? 0"
              :status="usageStatus(item.status)"
            />
          </div>
        </div>
        <template #action>
          <t-tag :theme="healthTheme(item.status)" variant="light">{{ healthLabel(item.status) }}</t-tag>
        </template>
      </t-list-item>
    </t-list>
    <p v-else class="dashboard-health__summary-description">
      {{ summaryDescription }}
    </p>
  </div>
  <t-empty v-else size="small" :description="t('dashboard.widget.invalidPayload')" />
</template>
<script setup lang="ts">
import { computed } from 'vue';

import { t } from '@/locales';

import type { DashboardHealthStatus, DashboardWidget } from '../../types/dashboard';
import { asHealthPayload } from './payload';
import { resolveDashboardText } from './widget-i18n';

const props = defineProps<{
  widget: DashboardWidget;
}>();

const payload = computed(() => asHealthPayload(props.widget.payload));
const summaryDescription = computed(() => {
  const currentPayload = payload.value;
  if (!currentPayload) {
    return t('dashboard.health.summaryHealthy');
  }

  if (typeof currentPayload.healthy_modules === 'number' && typeof currentPayload.abnormal_services === 'number') {
    return t('dashboard.health.summaryHealthyWithCounts', {
      healthy: currentPayload.healthy_modules,
      attention: currentPayload.abnormal_services,
    });
  }

  return t('dashboard.health.summaryHealthy');
});

function healthTheme(status: DashboardHealthStatus) {
  if (status === 'healthy') return 'success';
  if (status === 'degraded') return 'warning';
  if (status === 'disabled') return 'default';
  return 'primary';
}

function healthLabel(status: DashboardHealthStatus) {
  return t(`dashboard.health.${status}`);
}

function usagePercent(item: unknown) {
  if (!item || typeof item !== 'object' || Array.isArray(item)) {
    return null;
  }

  const record = item as Record<string, unknown>;
  if (typeof record.usage_percent === 'number' && Number.isFinite(record.usage_percent)) {
    return clampPercent(record.usage_percent);
  }
  if (typeof record.used === 'number' && typeof record.total === 'number' && record.total > 0) {
    return clampPercent((record.used / record.total) * 100);
  }
  if (typeof record.active === 'number' && typeof record.capacity === 'number' && record.capacity > 0) {
    return clampPercent((record.active / record.capacity) * 100);
  }
  return null;
}

function clampPercent(value: number) {
  return Math.max(0, Math.min(100, Math.round(value)));
}

function usageStatus(status: DashboardHealthStatus) {
  if (status === 'degraded') return 'warning';
  if (status === 'disabled') return 'error';
  return undefined;
}
</script>
<style lang="less" scoped>
.dashboard-health {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-s);
}

.dashboard-health__summary {
  display: flex;
}

.dashboard-health__item {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xxs);
  min-width: 0;
}

.dashboard-health__item strong,
.dashboard-health__item p {
  overflow-wrap: anywhere;
}

.dashboard-health__item p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.dashboard-health__usage {
  margin-top: var(--td-comp-margin-xxs);
  max-width: 260px;
}

.dashboard-health__summary-description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
  overflow-wrap: anywhere;
}
</style>
