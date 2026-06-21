<template>
  <t-list v-if="payload && payload.items.length" size="small" split>
    <t-list-item v-for="item in payload.items" :key="item.key">
      <div class="dashboard-link-list__item" :class="{ 'dashboard-link-list__item--disabled': item.disabled }">
        <strong>{{ resolveDashboardText(item.label_key, item.label) }}</strong>
        <p v-if="item.description_key || item.description">
          {{ resolveDashboardText(item.description_key, item.description) }}
        </p>
      </div>
      <template #action>
        <t-tag v-if="item.badge_key || item.badge" variant="light">
          {{ resolveDashboardText(item.badge_key, item.badge) }}
        </t-tag>
        <t-button
          v-if="hasRouteLocation(item.route_location)"
          variant="text"
          theme="primary"
          size="small"
          :disabled="item.disabled"
          @click="go(item.route_location)"
        >
          {{ t('dashboard.actions.open') }}
        </t-button>
      </template>
    </t-list-item>
  </t-list>
  <t-empty v-else-if="payload" size="small" :description="t('dashboard.widget.empty')" />
  <t-empty v-else size="small" :description="t('dashboard.widget.invalidPayload')" />
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';

import { t } from '@/locales';

import type { DashboardWidget } from '../../types/dashboard';
import { asLinkListPayload } from './payload';
import { openDashboardRoute } from './widget-actions';
import { resolveDashboardText } from './widget-i18n';

const props = defineProps<{
  widget: DashboardWidget;
}>();

const router = useRouter();
const payload = computed(() => asLinkListPayload(props.widget.payload));

function hasRouteLocation(location: string | undefined) {
  return Boolean(location?.trim());
}

function go(location: string | undefined) {
  const target = location?.trim();
  if (!target) {
    return;
  }

  openDashboardRoute(router, target);
}
</script>
<style lang="less" scoped>
.dashboard-link-list__item {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xxs);
  min-width: 0;
}

.dashboard-link-list__item strong,
.dashboard-link-list__item p {
  overflow-wrap: anywhere;
}

.dashboard-link-list__item p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 0;
}

.dashboard-link-list__item--disabled {
  color: var(--td-text-color-disabled);
}
</style>
