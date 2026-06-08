<template>
  <t-card class="dashboard-quick-actions" size="small" :bordered="true">
    <template #title>
      <div class="dashboard-quick-actions__header">
        <span>{{ t('dashboard.quickActions.title') }}</span>
        <small>{{ t('dashboard.quickActions.description') }}</small>
      </div>
    </template>

    <div v-if="sortedLinks.length" class="dashboard-quick-actions__grid">
      <t-button
        v-for="link in sortedLinks"
        :key="link.id"
        class="dashboard-quick-actions__item"
        variant="outline"
        theme="default"
        @click="go(link.route_location)"
      >
        <template v-if="link.icon" #icon>
          <t-icon :name="link.icon" />
        </template>
        <span class="dashboard-quick-actions__content">
          <strong>{{ linkTitle(link) }}</strong>
          <small v-if="link.description_key || link.description">{{ linkDescription(link) }}</small>
        </span>
      </t-button>
    </div>

    <t-empty v-else size="small" :description="t('dashboard.quickActions.empty')" />
  </t-card>
</template>
<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';

import { t } from '@/locales';

import type { DashboardQuickLink } from '../types/dashboard';
import { openDashboardRoute } from './widgets/widget-actions';
import { resolveDashboardText } from './widgets/widget-i18n';

const props = defineProps<{
  links: DashboardQuickLink[];
}>();

const router = useRouter();
const sortedLinks = computed(() => [...props.links].sort((left, right) => left.order - right.order));

function linkTitle(link: DashboardQuickLink) {
  return resolveDashboardText(link.title_key, link.title || link.id);
}

function linkDescription(link: DashboardQuickLink) {
  return resolveDashboardText(link.description_key, link.description);
}

function go(location: string) {
  openDashboardRoute(router, location);
}
</script>
<style lang="less" scoped>
.dashboard-quick-actions {
  min-width: 0;
}

.dashboard-quick-actions__header {
  align-items: baseline;
  display: flex;
  gap: var(--td-comp-margin-s);
  min-width: 0;
}

.dashboard-quick-actions__header span {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.dashboard-quick-actions__header small {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.dashboard-quick-actions__grid {
  display: grid;
  gap: var(--td-comp-margin-s);
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
}

.dashboard-quick-actions__item {
  height: auto;
  justify-content: flex-start;
  min-height: 56px;
  padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-m);
  text-align: left;
}

.dashboard-quick-actions__item :deep(.t-button__text) {
  min-width: 0;
}

.dashboard-quick-actions__content {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xxs);
  min-width: 0;
}

.dashboard-quick-actions__content strong,
.dashboard-quick-actions__content small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-quick-actions__content strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.dashboard-quick-actions__content small {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

@media (width <= 768px) {
  .dashboard-quick-actions__header {
    align-items: flex-start;
    flex-direction: column;
    gap: var(--td-comp-margin-xxs);
  }
}
</style>
