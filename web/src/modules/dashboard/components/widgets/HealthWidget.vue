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
        </div>
        <template #action>
          <t-tag :theme="healthTheme(item.status)" variant="light">{{ healthLabel(item.status) }}</t-tag>
        </template>
      </t-list-item>
    </t-list>
    <t-empty v-else size="small" :description="t('dashboard.widget.empty')" />
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

function healthTheme(status: DashboardHealthStatus) {
  if (status === 'healthy') return 'success';
  if (status === 'degraded') return 'warning';
  if (status === 'disabled') return 'default';
  return 'primary';
}

function healthLabel(status: DashboardHealthStatus) {
  return t(`dashboard.health.${status}`);
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
</style>
