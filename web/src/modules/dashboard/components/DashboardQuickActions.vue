<template>
  <t-card class="dashboard-quick-actions" size="small" :bordered="true">
    <template #title>
      <div class="dashboard-quick-actions__header">
        <span>{{ t('dashboard.quickActions.title') }}</span>
        <small>{{ t('dashboard.quickActions.description') }}</small>
      </div>
    </template>

    <div v-if="visibleLinks.length" class="dashboard-quick-actions__grid">
      <t-button
        v-for="link in visibleLinks"
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
          <span class="dashboard-quick-actions__title-row">
            <strong>{{ linkTitle(link) }}</strong>
            <t-tag v-if="link.module_key" size="small" variant="light">
              {{ moduleLabel(link.module_key) }}
            </t-tag>
          </span>
          <small v-if="link.description_key || link.description">{{ linkDescription(link) }}</small>
        </span>
      </t-button>
    </div>

    <t-empty v-else size="small" :description="t('dashboard.quickActions.empty')" />

    <template v-if="hasMoreLinks" #actions>
      <t-button variant="text" theme="primary" size="small" @click="showAll = !showAll">
        {{
          showAll ? t('dashboard.quickActions.viewLess') : t('dashboard.quickActions.viewAll', { count: hiddenCount })
        }}
      </t-button>
    </template>
  </t-card>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';

import { t } from '@/locales';

import type { DashboardQuickLink } from '../types/dashboard';
import { openDashboardRoute } from './widgets/widget-actions';
import { resolveDashboardText } from './widgets/widget-i18n';

const props = defineProps<{
  links: DashboardQuickLink[];
}>();

const router = useRouter();
const showAll = ref(false);
const quickActionLimit = 8;

const sortedLinks = computed(() => [...props.links].sort((left, right) => left.order - right.order));
const visibleLinks = computed(() => (showAll.value ? sortedLinks.value : sortedLinks.value.slice(0, quickActionLimit)));
const hiddenCount = computed(() => Math.max(sortedLinks.value.length - quickActionLimit, 0));
const hasMoreLinks = computed(() => sortedLinks.value.length > quickActionLimit);

function linkTitle(link: DashboardQuickLink) {
  return resolveDashboardText(link.title_key, link.title || link.id);
}

function linkDescription(link: DashboardQuickLink) {
  return resolveDashboardText(link.description_key, link.description);
}

function moduleLabel(moduleKey: string) {
  const key = `dashboard.module.${moduleKey.replaceAll('.', '_')}`;
  return resolveDashboardText(key, moduleKey, moduleKey);
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

.dashboard-quick-actions__title-row {
  align-items: center;
  display: flex;
  gap: var(--td-comp-margin-xs);
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
