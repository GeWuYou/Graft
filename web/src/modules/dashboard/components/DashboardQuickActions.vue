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
        @click="go(link.route_location)"
      >
        <t-badge
          v-if="link.module_key"
          class="dashboard-quick-actions__badge"
          shape="round"
          :color="moduleColor(link.module_key)"
          :content="moduleLabel(link.module_key)"
        >
          <span class="dashboard-quick-actions__content">
            <t-icon v-if="link.icon" class="dashboard-quick-actions__icon" :name="link.icon" />
            <span>
              <strong>{{ linkTitle(link) }}</strong>
              <small v-if="link.description_key || link.description">{{ linkDescription(link) }}</small>
            </span>
          </span>
        </t-badge>
        <span v-else class="dashboard-quick-actions__content">
          <t-icon v-if="link.icon" class="dashboard-quick-actions__icon" :name="link.icon" />
          <span>
            <strong>{{ linkTitle(link) }}</strong>
            <small v-if="link.description_key || link.description">{{ linkDescription(link) }}</small>
          </span>
        </span>
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
          @click="go(link.route_location)"
        >
          <t-badge
            v-if="link.module_key"
            class="dashboard-quick-actions__badge"
            shape="round"
            :color="moduleColor(link.module_key)"
            :content="moduleLabel(link.module_key)"
          >
            <span class="dashboard-quick-actions__content">
              <t-icon v-if="link.icon" class="dashboard-quick-actions__icon" :name="link.icon" />
              <span>
                <strong>{{ linkTitle(link) }}</strong>
                <small v-if="link.description_key || link.description">{{ linkDescription(link) }}</small>
              </span>
            </span>
          </t-badge>
          <span v-else class="dashboard-quick-actions__content">
            <t-icon v-if="link.icon" class="dashboard-quick-actions__icon" :name="link.icon" />
            <span>
              <strong>{{ linkTitle(link) }}</strong>
              <small v-if="link.description_key || link.description">{{ linkDescription(link) }}</small>
            </span>
          </span>
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
import { type DashboardQuickActionConfig, DEFAULT_DASHBOARD_QUICK_ACTION_CONFIG } from '../contract/quick-actions';
import type { DashboardQuickLink } from '../types/dashboard';
import { openDashboardRoute } from './widgets/widget-actions';
import { resolveDashboardText } from './widgets/widget-i18n';

const props = defineProps<{
  links: DashboardQuickLink[];
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

const MODULE_COLOR_PREFIXES = [
  { prefixes: ['audit'], color: 'var(--td-error-color-6)' },
  { prefixes: ['monitor'], color: 'var(--td-warning-color-6)' },
  { prefixes: ['rbac', 'user'], color: 'var(--td-brand-color-6)' },
  { prefixes: ['system-config'], color: 'var(--td-success-color-6)' },
] as const;

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

function moduleColor(moduleKey: string) {
  for (const { prefixes, color } of MODULE_COLOR_PREFIXES) {
    if (prefixes.some((prefix) => moduleKey === prefix || moduleKey.startsWith(`${prefix}.`))) {
      return color;
    }
  }

  return 'var(--td-text-color-secondary)';
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
  grid-template-columns: repeat(6, minmax(0, 1fr));
}

.dashboard-quick-actions__item {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  cursor: pointer;
  min-height: 64px;
  min-width: 0;
  padding: var(--td-comp-paddingTB-s) var(--td-comp-paddingLR-m);
  text-align: left;
  transition:
    border-color 0.16s ease,
    background-color 0.16s ease;
}

.dashboard-quick-actions__item:hover {
  background: var(--td-bg-color-container-hover);
  border-color: var(--td-brand-color);
}

.dashboard-quick-actions__badge,
.dashboard-quick-actions__badge :deep(.t-badge__ribbon-outer) {
  width: 100%;
}

.dashboard-quick-actions__badge :deep(.t-badge) {
  max-width: 112px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dashboard-quick-actions__content {
  align-items: center;
  display: grid;
  gap: var(--td-comp-margin-s);
  grid-template-columns: auto minmax(0, 1fr);
  min-width: 0;
}

.dashboard-quick-actions__content > span {
  display: flex;
  flex-direction: column;
  gap: var(--td-comp-margin-xxs);
  min-width: 0;
}

.dashboard-quick-actions__icon {
  color: var(--td-brand-color);
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

@media (width >= 1600px) {
  .dashboard-quick-actions__grid {
    grid-template-columns: repeat(8, minmax(0, 1fr));
  }
}

@media (width <= 1280px) {
  .dashboard-quick-actions__grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
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
