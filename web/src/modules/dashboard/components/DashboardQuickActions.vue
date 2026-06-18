<!--
  Copyright (c) 2025-2026 GeWuYou
  SPDX-License-Identifier: Apache-2.0
-->

<template>
  <t-card v-if="config.enabled" class="dashboard-quick-actions" size="small" :bordered="true">
    <template #title>
      <div class="dashboard-quick-actions__header">
        <span>{{ t('dashboard.quickActions.title') }}</span>
        <small>{{ t('dashboard.quickActions.description') }}</small>
      </div>
    </template>

    <div v-if="visibleLinks.length" class="dashboard-quick-actions__grid">
      <button
        v-for="link in visibleLinks"
        :key="link.id"
        class="dashboard-quick-actions__item"
        type="button"
        :title="linkFullLabel(link)"
        @click="go(link.route_location)"
      >
        <t-tooltip :content="linkFullLabel(link)" placement="top" theme="default">
          <span class="dashboard-quick-actions__inner">
            <t-icon v-if="link.icon" class="dashboard-quick-actions__icon" :name="link.icon" />
            <span class="dashboard-quick-actions__content">
              <strong>{{ linkTitle(link) }}</strong>
              <small>{{ linkGroup(link) }}</small>
            </span>
          </span>
        </t-tooltip>
      </button>
    </div>

    <t-empty v-else size="small" :description="t('dashboard.quickActions.empty')" />

    <template v-if="hasMoreLinks" #actions>
      <t-button variant="text" theme="primary" size="small" @click="drawerVisible = true">
        {{ t('dashboard.quickActions.viewAll', { count: sortedLinks.length }) }}
      </t-button>
    </template>

    <t-drawer
      v-model:visible="drawerVisible"
      :header="t('dashboard.quickActions.drawerTitle')"
      size="560px"
      :footer="false"
      destroy-on-close
    >
      <div class="dashboard-quick-actions__drawer-grid">
        <button
          v-for="link in sortedLinks"
          :key="link.id"
          type="button"
          class="dashboard-quick-actions__item dashboard-quick-actions__item--drawer"
          :title="linkFullLabel(link)"
          @click="go(link.route_location)"
        >
          <t-tooltip :content="linkFullLabel(link)" placement="top" theme="default">
            <span class="dashboard-quick-actions__inner">
              <t-icon v-if="link.icon" class="dashboard-quick-actions__icon" :name="link.icon" />
              <span class="dashboard-quick-actions__content">
                <strong>{{ linkTitle(link) }}</strong>
                <small>{{ linkGroup(link) }}</small>
              </span>
            </span>
          </t-tooltip>
        </button>
      </div>
    </t-drawer>
  </t-card>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';

import { t } from '@/locales';

import { useDashboardQuickActions } from '../composables/use-dashboard-quick-actions';
import type { DashboardQuickActionLink } from '../contract/quick-action-links';
import { type DashboardQuickActionConfig, DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG } from '../contract/quick-actions';
import { openDashboardRoute } from './widgets/widget-actions';
import { resolveDashboardText } from './widgets/widget-i18n';

const props = defineProps<{
  links: DashboardQuickActionLink[];
  config?: DashboardQuickActionConfig;
}>();

const router = useRouter();
const drawerVisible = ref(false);

const config = computed(() => props.config ?? DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG);
const { rankedLinks: sortedLinks, recordAccess } = useDashboardQuickActions(
  () => props.links,
  () => config.value,
);
const visibleLinks = computed(() => sortedLinks.value.slice(0, config.value.maxItems));
const hasMoreLinks = computed(() => sortedLinks.value.length > config.value.maxItems);

function linkTitle(link: DashboardQuickActionLink) {
  return resolveDashboardText(link.title_key, link.title || link.id);
}

function linkGroup(link: DashboardQuickActionLink) {
  return resolveDashboardText(link.group_key, link.group, moduleLabel(link.module_key));
}

function linkFullLabel(link: DashboardQuickActionLink) {
  return link.full_label?.trim() || linkTitle(link);
}

function moduleLabel(moduleKey: string) {
  const key = `dashboard.module.${moduleKey.replaceAll('.', '_')}`;
  return resolveDashboardText(key, moduleKey, moduleKey);
}

function go(location: string) {
  recordAccess(location);
  drawerVisible.value = false;
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
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.dashboard-quick-actions__item {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  cursor: pointer;
  min-height: 72px;
  min-width: 0;
  padding: var(--td-comp-paddingTB-m) var(--td-comp-paddingLR-m);
  text-align: left;
  transition:
    border-color 0.16s ease,
    background-color 0.16s ease,
    box-shadow 0.16s ease;
}

.dashboard-quick-actions__item:hover,
.dashboard-quick-actions__item:focus-visible {
  background: var(--td-bg-color-container-hover);
  border-color: var(--td-brand-color);
  box-shadow: var(--td-shadow-1);
  outline: none;
}

.dashboard-quick-actions__inner {
  align-items: center;
  display: grid;
  gap: var(--td-comp-margin-s);
  grid-template-columns: auto minmax(0, 1fr);
  min-width: 0;
}

.dashboard-quick-actions__content {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xxs);
  min-width: 0;
}

.dashboard-quick-actions__icon {
  color: var(--td-brand-color);
  flex-shrink: 0;
  font-size: var(--td-font-size-title-medium);
  margin-bottom: var(--td-comp-margin-xs);
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

.dashboard-quick-actions__drawer-grid {
  display: grid;
  gap: var(--td-comp-margin-s);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.dashboard-quick-actions__item--drawer {
  min-height: 72px;
}

.dashboard-quick-actions__item :deep(.t-tooltip),
.dashboard-quick-actions__item :deep(.t-popup__reference) {
  display: block;
  min-width: 0;
  width: 100%;
}

@media (width >= 1600px) {
  .dashboard-quick-actions__grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

@media (width <= 1280px) {
  .dashboard-quick-actions__grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .dashboard-quick-actions__drawer-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}

@media (width <= 768px) {
  .dashboard-quick-actions__header {
    align-items: flex-start;
    flex-direction: column;
    gap: var(--td-comp-margin-xxs);
  }

  .dashboard-quick-actions__grid,
  .dashboard-quick-actions__drawer-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (width <= 480px) {
  .dashboard-quick-actions__grid,
  .dashboard-quick-actions__drawer-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
