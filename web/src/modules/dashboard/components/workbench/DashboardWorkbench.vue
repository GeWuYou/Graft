<template>
  <section
    class="workbench-preview"
    data-page-type="overview-dashboard"
    :data-preview-scenario="props.preview ? 'fixed' : undefined"
  >
    <page-header
      :title-key="pageTitleKey"
      :title-fallback="t(pageTitleKey)"
      :description-key="pageDescriptionKey"
      :description-fallback="t(pageDescriptionKey)"
      :source="{
        labelKey: pageEyebrowKey,
        fallback: t(pageEyebrowKey),
        color: 'var(--td-brand-color-6)',
      }"
    >
      <template #actions>
        <div class="workbench-preview__header-actions">
          <span class="workbench-preview__updated-at">
            {{ t(updatedAtKey, { time: generatedAtLabel }) }}
          </span>
          <t-button variant="outline" theme="default" :loading="props.refreshing" @click="emit('refresh')">
            <template #icon><refresh-icon /></template>
            {{ t(refreshLabelKey) }}
          </t-button>
        </div>
      </template>
    </page-header>

    <t-alert v-if="props.errorMessage" theme="error" :title="t('dashboard.error.title')" :message="props.errorMessage">
      <template #operation>
        <t-button variant="text" theme="primary" size="small" @click="emit('refresh')">
          {{ t('dashboard.actions.retry') }}
        </t-button>
      </template>
    </t-alert>

    <div v-if="!props.ready && props.loading" class="workbench-preview__loading">
      <t-skeleton animation="gradient" :row-col="loadingRows" />
    </div>

    <section
      v-if="props.ready"
      class="operational-status"
      data-first-screen-region="operational-status"
      :aria-label="t('dashboard.workbench.operational.title')"
    >
      <div class="operational-status__primary">
        <span class="operational-status__eyebrow">{{ t('dashboard.workbench.operational.eyebrow') }}</span>
        <div class="operational-status__summary">
          <strong>{{ t('dashboard.workbench.operational.needsAttention') }}</strong>
          <span class="operational-status__count">
            {{ t('dashboard.workbench.operational.itemCount', { count: presentation.operational.needsReview }) }}
          </span>
          <span class="operational-status__distribution">
            {{ attentionDistribution }}
          </span>
        </div>
      </div>
      <dl class="operational-status__metrics">
        <div>
          <dt>{{ t('dashboard.workbench.operational.modules') }}</dt>
          <dd>{{ presentation.operational.enabledModules }}</dd>
        </div>
        <div>
          <dt>{{ t('dashboard.workbench.operational.failedTasks') }}</dt>
          <dd>{{ presentation.operational.failedTasks }}</dd>
        </div>
        <div>
          <dt>{{ t('dashboard.workbench.operational.highRiskEvents') }}</dt>
          <dd>{{ presentation.operational.highRiskEvents }}</dd>
        </div>
      </dl>
    </section>

    <responsive-content v-if="props.ready" class="workbench-preview__grid" layout="wide-split">
      <div class="workbench-preview__column workbench-preview__column--primary">
        <t-card
          class="workbench-surface workbench-surface--attention"
          data-first-screen-region="attention"
          :bordered="false"
          header-bordered
        >
          <template #header>
            <div class="workbench-surface__heading">
              <div>
                <h2>{{ t('dashboard.workbench.attention.title') }}</h2>
                <p>{{ t('dashboard.workbench.attention.description') }}</p>
              </div>
            </div>
          </template>
          <workbench-presentation-list
            variant="attention"
            empty-key="dashboard.workbench.attention.empty"
            expand-key="dashboard.workbench.expand.attention"
            :items="presentation.attention"
            :retrying-id="props.retryingId"
            :visible-limit="5"
            @navigate="navigate($event, 'contextual-action')"
            @retry-item="emit('retry-item', $event)"
          />
        </t-card>
      </div>

      <div class="workbench-preview__column workbench-preview__column--secondary">
        <t-card
          class="workbench-surface workbench-surface--health"
          data-first-screen-region="health"
          :bordered="false"
          header-bordered
        >
          <template #header>
            <div class="workbench-surface__heading">
              <div>
                <h2>{{ t('dashboard.workbench.health.title') }}</h2>
                <p>{{ t('dashboard.workbench.health.description') }}</p>
              </div>
            </div>
          </template>
          <workbench-presentation-list
            variant="health"
            empty-key="dashboard.workbench.health.empty"
            expand-key="dashboard.workbench.expand.health"
            :items="presentation.health"
            :visible-limit="3"
            @navigate="navigate($event, 'contextual-action')"
          />
        </t-card>

        <t-card
          class="workbench-surface workbench-surface--module-coverage"
          data-first-screen-region="module-coverage"
          :bordered="false"
          header-bordered
        >
          <template #header>
            <div class="workbench-surface__heading">
              <div>
                <h2>{{ t('dashboard.workbench.moduleCoverage.title') }}</h2>
                <p>{{ t('dashboard.workbench.moduleCoverage.description') }}</p>
              </div>
            </div>
          </template>
          <dl class="module-coverage__metrics">
            <div data-coverage-metric="registered">
              <dt>{{ t('dashboard.workbench.moduleCoverage.registered') }}</dt>
              <dd>{{ presentation.moduleCoverage.registeredModules }}</dd>
            </div>
            <div data-coverage-metric="enabled">
              <dt>{{ t('dashboard.workbench.moduleCoverage.enabled') }}</dt>
              <dd>{{ presentation.moduleCoverage.enabledModules }}</dd>
            </div>
            <div data-coverage-metric="degraded">
              <dt>{{ t('dashboard.workbench.moduleCoverage.degraded') }}</dt>
              <dd>{{ presentation.moduleCoverage.degradedModules }}</dd>
            </div>
            <div data-coverage-metric="normal-sources">
              <dt>{{ t('dashboard.workbench.moduleCoverage.normalSources') }}</dt>
              <dd>{{ presentation.moduleCoverage.normalContributionSources }}</dd>
            </div>
            <div data-coverage-metric="failed-sources">
              <dt>{{ t('dashboard.workbench.moduleCoverage.failedSources') }}</dt>
              <dd>{{ presentation.moduleCoverage.failedContributionSources }}</dd>
            </div>
          </dl>
        </t-card>
      </div>
    </responsive-content>

    <responsive-content v-if="props.ready" class="workbench-details" layout="wide-split">
      <div class="workbench-details__column workbench-details__column--primary">
        <t-card
          v-for="group in presentation.metricGroups"
          :key="group.id"
          class="workbench-surface workbench-surface--metrics"
          :data-metric-group-id="group.id"
          :bordered="false"
          header-bordered
        >
          <template #header>
            <div class="workbench-surface__heading">
              <div>
                <h2>{{ dashboardText(group.titleKey, group.titleFallback) }}</h2>
                <p v-if="group.descriptionKey || group.descriptionFallback">
                  {{ dashboardText(group.descriptionKey, group.descriptionFallback, '') }}
                </p>
              </div>
              <t-button
                v-if="group.action?.kind === 'navigate'"
                variant="text"
                theme="primary"
                size="small"
                @click="navigate(group.action.route, 'contextual-action')"
              >
                {{ workbenchActionLabel(group.action) }}
              </t-button>
            </div>
          </template>
          <div class="metric-group__items">
            <component
              :is="metric.route ? 'button' : 'div'"
              v-for="metric in group.metrics"
              :key="metric.key"
              class="metric-item"
              :class="{ 'metric-item--actionable': Boolean(metric.route) }"
              :data-metric-key="metric.key"
              :data-tone="metric.tone"
              :type="metric.route ? 'button' : undefined"
              @click="metric.route && navigate(metric.route, 'contextual-action')"
            >
              <span class="metric-item__label">{{ dashboardText(metric.labelKey, metric.labelFallback) }}</span>
              <strong class="metric-item__value">
                {{ metric.value }}
                <small v-if="metric.unitKey || metric.unitFallback">
                  {{ dashboardText(metric.unitKey, metric.unitFallback, '') }}
                </small>
              </strong>
              <span v-if="metric.descriptionKey || metric.descriptionFallback" class="metric-item__description">
                {{ dashboardText(metric.descriptionKey, metric.descriptionFallback, '') }}
              </span>
            </component>
          </div>
        </t-card>

        <t-card
          v-if="presentation.resourceSummary.state !== 'hidden'"
          class="workbench-surface workbench-surface--resources"
          :data-resource-state="presentation.resourceSummary.state"
          :bordered="false"
          header-bordered
        >
          <template #header>
            <div class="workbench-surface__heading">
              <div>
                <h2>{{ t('dashboard.workbench.resources.title') }}</h2>
                <p>{{ t('dashboard.workbench.resources.description') }}</p>
              </div>
              <t-button
                v-if="presentation.resourceSummary.route"
                variant="text"
                theme="primary"
                size="small"
                @click="navigate(presentation.resourceSummary.route, 'contextual-action')"
              >
                {{ t('dashboard.actions.details') }}
              </t-button>
            </div>
          </template>

          <template v-if="presentation.resourceSummary.state === 'loaded' && presentation.resourceSummary.overview">
            <dl class="resource-overview">
              <div>
                <dt>{{ t('dashboard.workbench.resources.overview.running') }}</dt>
                <dd>{{ presentation.resourceSummary.overview.runningContainers }}</dd>
              </div>
              <div>
                <dt>{{ t('dashboard.workbench.resources.overview.abnormal') }}</dt>
                <dd>{{ presentation.resourceSummary.overview.abnormalContainers }}</dd>
              </div>
              <div>
                <dt>{{ t('dashboard.workbench.resources.overview.cpu') }}</dt>
                <dd>{{ formatPercent(presentation.resourceSummary.overview.cpuTotalPercent) }}</dd>
              </div>
              <div>
                <dt>{{ t('dashboard.workbench.resources.overview.memory') }}</dt>
                <dd>{{ formatOverviewMemory() }}</dd>
              </div>
            </dl>

            <div class="resource-breakdown">
              <section class="resource-breakdown__group" data-resource-group="cpu">
                <h3>{{ t('dashboard.workbench.resources.topCpu') }}</h3>
                <ul v-if="presentation.resourceSummary.topCpu.length" class="resource-ranking">
                  <li v-for="item in presentation.resourceSummary.topCpu" :key="item.id">
                    <span>{{ item.name }}</span>
                    <strong>{{ formatPercent(item.cpuPercent) }}</strong>
                  </li>
                </ul>
                <p v-else class="workbench-empty workbench-empty--compact">
                  {{ t('dashboard.workbench.resources.noHotspots') }}
                </p>
              </section>
              <section class="resource-breakdown__group" data-resource-group="memory">
                <h3>{{ t('dashboard.workbench.resources.topMemory') }}</h3>
                <ul v-if="presentation.resourceSummary.topMemory.length" class="resource-ranking">
                  <li v-for="item in presentation.resourceSummary.topMemory" :key="item.id">
                    <span>{{ item.name }}</span>
                    <strong>{{ formatMemoryHotspot(item.memoryPercent, item.memoryUsageBytes) }}</strong>
                  </li>
                </ul>
                <p v-else class="workbench-empty workbench-empty--compact">
                  {{ t('dashboard.workbench.resources.noHotspots') }}
                </p>
              </section>
            </div>

            <section class="resource-anomalies" data-resource-group="anomalies">
              <h3>{{ t('dashboard.workbench.resources.anomalies') }}</h3>
              <ul v-if="presentation.resourceSummary.anomalies.length" class="resource-anomalies__list">
                <li v-for="item in presentation.resourceSummary.anomalies" :key="item.id">
                  <div>
                    <strong>{{ item.name }}</strong>
                    <span>{{ resourceAnomalyReason(item) }}</span>
                  </div>
                  <t-tag theme="warning" variant="light">
                    {{ t('dashboard.workbench.resources.restartCount', { count: item.restartCount ?? 0 }) }}
                  </t-tag>
                </li>
              </ul>
              <p v-else class="workbench-empty workbench-empty--compact">
                {{ t('dashboard.workbench.resources.noAnomalies') }}
              </p>
            </section>
          </template>

          <template v-else>
            <div
              v-for="item in presentation.resources"
              :key="item.id"
              class="resource-note"
              :data-evidence="item.evidenceState"
            >
              <info-circle-icon size="18px" />
              <div>
                <strong>{{ itemTitle(item) }}</strong>
                <p>{{ itemDescription(item) }}</p>
              </div>
            </div>
          </template>
        </t-card>

        <t-card
          v-if="presentation.activity.length"
          class="workbench-surface workbench-surface--activity"
          data-secondary-region="activity"
          :bordered="false"
          header-bordered
        >
          <template #header>
            <div class="workbench-surface__heading">
              <div>
                <h2>{{ t('dashboard.workbench.activity.title') }}</h2>
                <p>{{ t('dashboard.workbench.activity.description') }}</p>
              </div>
            </div>
          </template>
          <workbench-presentation-list
            variant="activity"
            :items="presentation.activity"
            @navigate="navigate($event, 'contextual-action')"
          />
        </t-card>
      </div>

      <div class="workbench-details__column workbench-details__column--secondary">
        <t-card
          v-for="group in presentation.contextLinkGroups"
          :key="group.id"
          class="workbench-surface workbench-surface--context-links"
          :data-context-group-id="group.id"
          :bordered="false"
          header-bordered
        >
          <template #header>
            <div class="workbench-surface__heading">
              <div>
                <h2>{{ dashboardText(group.titleKey, group.titleFallback) }}</h2>
                <p v-if="group.descriptionKey || group.descriptionFallback">
                  {{ dashboardText(group.descriptionKey, group.descriptionFallback, '') }}
                </p>
              </div>
              <t-button
                v-if="group.action?.kind === 'navigate'"
                variant="text"
                theme="primary"
                size="small"
                @click="navigate(group.action.route, 'contextual-action')"
              >
                {{ workbenchActionLabel(group.action) }}
              </t-button>
            </div>
          </template>
          <workbench-context-link-list
            :group="group"
            :visible-limit="6"
            @navigate="navigate($event, 'contextual-action')"
          />
        </t-card>
      </div>
    </responsive-content>

    <section
      v-if="props.ready && props.quickActionsEnabled && presentation.quickActions.length"
      class="quick-actions"
      :aria-label="t('dashboard.workbench.quickActions.title')"
    >
      <div class="quick-actions__heading">
        <div>
          <h2>{{ t('dashboard.workbench.quickActions.title') }}</h2>
          <p>{{ t('dashboard.workbench.quickActions.description') }}</p>
        </div>
        <t-button variant="text" theme="primary" @click="drawerVisible = true">
          {{ t('dashboard.workbench.quickActions.viewAll') }}
          <template #suffix><chevron-right-icon /></template>
        </t-button>
      </div>
      <div class="quick-actions__items" role="group" :aria-label="t('dashboard.workbench.quickActions.title')">
        <workbench-quick-action-item
          v-for="action in presentation.homeQuickActions"
          :key="action.id"
          :action="action"
          @activate="navigate(action.route, 'quick-entry')"
        />
      </div>
    </section>

    <responsive-dialog
      :visible="drawerVisible"
      :title="t('dashboard.workbench.quickActions.drawerTitle')"
      :close-label="t('dashboard.workbench.quickActions.closeDrawer')"
      purpose="workspace"
      size="compact"
      @update:visible="handleDrawerVisible"
    >
      <div class="quick-entry-drawer">
        <t-input
          v-model="entrySearch"
          class="quick-entry-drawer__search"
          type="search"
          clearable
          :placeholder="t('dashboard.workbench.quickActions.searchPlaceholder')"
        >
          <template #prefixIcon><search-icon /></template>
        </t-input>

        <div class="quick-entry-drawer__content graft-scrollbar">
          <section v-if="!normalizedEntrySearch" class="quick-entry-drawer__section">
            <h3>{{ t('dashboard.workbench.quickActions.frequent') }}</h3>
            <div
              class="quick-entry-drawer__frequent"
              role="group"
              :aria-label="t('dashboard.workbench.quickActions.frequent')"
            >
              <workbench-quick-action-item
                v-for="action in presentation.homeQuickActions"
                :key="action.id"
                :action="action"
                variant="drawer-featured"
                @activate="navigate(action.route, 'quick-entry')"
              />
            </div>
          </section>

          <nav
            v-if="filteredNavigationGroups.length"
            class="quick-entry-drawer__navigation"
            :aria-label="t('dashboard.workbench.quickActions.allEntries')"
          >
            <section v-for="group in filteredNavigationGroups" :key="group.id" class="quick-entry-drawer__section">
              <h3>{{ group.label }}</h3>
              <div class="quick-entry-drawer__navigation-list">
                <button
                  v-for="link in group.links"
                  :key="link.id"
                  class="quick-entry-drawer__navigation-item"
                  type="button"
                  @click="navigate(link.route_location, 'quick-entry')"
                >
                  <span class="quick-entry-drawer__navigation-icon">
                    <graft-menu-icon :icon-key="link.icon" />
                  </span>
                  <span>{{ navigationLinkTitle(link) }}</span>
                </button>
              </div>
            </section>
          </nav>

          <p v-else class="quick-entry-drawer__empty">
            {{ t('dashboard.workbench.quickActions.noResults') }}
          </p>
        </div>
      </div>
    </responsive-dialog>
  </section>
</template>
<script setup lang="ts">
import { ChevronRightIcon, InfoCircleIcon, RefreshIcon, SearchIcon } from 'tdesign-icons-vue-next';
import { computed, ref } from 'vue';

import { currentLocale, t } from '@/locales';
import { PageHeader } from '@/shared/components/page';
import ResponsiveContent from '@/shared/components/responsive/ResponsiveContent.vue';
import ResponsiveDialog from '@/shared/components/responsive/ResponsiveDialog.vue';
import GraftMenuIcon from '@/shared/icons/MenuIcon.vue';
import {
  formatBytes,
  formatLocaleDateTime,
  formatPercent,
  MEDIUM_DATE_TIME_FORMAT_OPTIONS,
} from '@/shared/observability';

import type { DashboardQuickActionLink } from '../../contract/quick-action-links';
import type {
  PresentationItem,
  WorkbenchAction,
  WorkbenchNavigationSource,
  WorkbenchPresentation,
} from '../../presentation/workbench';
import { hasDashboardTranslation, resolveDashboardText } from '../widgets/widget-i18n';
import WorkbenchContextLinkList from './WorkbenchContextLinkList.vue';
import WorkbenchPresentationList from './WorkbenchPresentationList.vue';
import WorkbenchQuickActionItem from './WorkbenchQuickActionItem.vue';

// 工作台只消费显式 presentation model；入口抽屉读取授权导航，不借此反推业务健康状态。
defineOptions({ name: 'DashboardWorkbench' });

const props = withDefaults(
  defineProps<{
    generatedAt: string;
    navigationLinks: DashboardQuickActionLink[];
    presentation: WorkbenchPresentation;
    preview?: boolean;
    quickActionsEnabled?: boolean;
    loading?: boolean;
    ready?: boolean;
    errorMessage?: string;
    refreshing?: boolean;
    retryingId?: string;
  }>(),
  {
    preview: false,
    quickActionsEnabled: true,
    loading: false,
    ready: true,
    errorMessage: '',
    refreshing: false,
    retryingId: '',
  },
);

const emit = defineEmits<{
  navigate: [route: string, source: WorkbenchNavigationSource];
  refresh: [];
  'retry-item': [item: PresentationItem];
}>();

const drawerVisible = ref(false);
const entrySearch = ref('');
const loadingRows = [
  { width: '34%', height: '20px' },
  { width: '100%', height: '88px' },
  { width: '100%', height: '180px' },
];
const presentation = computed(() => props.presentation);
const pageTitleKey = computed(() => (props.preview ? 'dashboard.previewWorkbench.title' : 'dashboard.page.title'));
const pageDescriptionKey = computed(() =>
  props.preview ? 'dashboard.previewWorkbench.description' : 'dashboard.page.description',
);
const pageEyebrowKey = computed(() =>
  props.preview ? 'dashboard.previewWorkbench.eyebrow' : 'dashboard.page.eyebrow',
);
const updatedAtKey = computed(() =>
  props.preview ? 'dashboard.previewWorkbench.updatedAt' : 'dashboard.page.lastUpdated',
);
const refreshLabelKey = computed(() =>
  props.preview ? 'dashboard.previewWorkbench.refresh' : 'dashboard.actions.refresh',
);
const attentionDistribution = computed(() => {
  const counts = presentation.value.operational.attentionStatusCounts;
  const key = counts.error
    ? 'dashboard.workbench.operational.distributionWithError'
    : 'dashboard.workbench.operational.distribution';
  return t(key, counts);
});
const generatedAtLabel = computed(() =>
  formatLocaleDateTime(props.generatedAt, currentLocale, MEDIUM_DATE_TIME_FORMAT_OPTIONS),
);
const normalizedEntrySearch = computed(() => entrySearch.value.trim().toLocaleLowerCase(currentLocale.value));
const filteredNavigationGroups = computed(() => {
  const groups = new Map<string, { id: string; label: string; links: DashboardQuickActionLink[]; order: number }>();
  const query = normalizedEntrySearch.value;

  props.navigationLinks.forEach((link) => {
    const label = link.section?.trim() || link.group?.trim() || link.module_key;
    const searchableText = [navigationLinkTitle(link), link.full_label, label, link.module_key]
      .filter(Boolean)
      .join(' ')
      .toLocaleLowerCase(currentLocale.value);
    if (query && !searchableText.includes(query)) {
      return;
    }

    const id = link.section_key?.trim() || link.group_key?.trim() || label;
    const group = groups.get(id) ?? { id, label, links: [], order: link.section_order ?? Number.MAX_SAFE_INTEGER };
    group.links.push(link);
    groups.set(id, group);
  });

  return [...groups.values()].sort((left, right) => left.order - right.order || left.id.localeCompare(right.id));
});

function navigate(route: string, source: WorkbenchNavigationSource) {
  drawerVisible.value = false;
  emit('navigate', route, source);
}

function navigationLinkTitle(link: DashboardQuickActionLink) {
  return link.title?.trim() || link.full_label?.trim() || link.id;
}

function handleDrawerVisible(visible: boolean) {
  drawerVisible.value = visible;
  if (!visible) {
    entrySearch.value = '';
  }
}

function dashboardText(key?: string, fallback?: string, defaultText = '-') {
  return resolveDashboardText(key, fallback, defaultText);
}

function workbenchActionLabel(action: WorkbenchAction) {
  return resolveDashboardText(action.labelKey, action.labelFallback || '');
}

function formatOverviewMemory() {
  const overview = presentation.value.resourceSummary.overview;
  if (!overview) {
    return t('dashboard.workbench.resources.notCollected');
  }
  const percent = formatPercent(overview.memoryTotalPercent, t('dashboard.workbench.resources.notCollected'));
  if (overview.memoryTotalUsageBytes === null || overview.memoryTotalLimitBytes === null) {
    return percent;
  }
  return t('dashboard.workbench.resources.memoryUsage', {
    usage: formatBytes(overview.memoryTotalUsageBytes),
    limit: formatBytes(overview.memoryTotalLimitBytes),
    percent,
  });
}

function formatMemoryHotspot(percent: number | null, usageBytes: number | null) {
  const percentLabel = formatPercent(percent, t('dashboard.workbench.resources.notCollected'));
  if (usageBytes === null) {
    return percentLabel;
  }
  return t('dashboard.workbench.resources.hotspotMemory', {
    usage: formatBytes(usageBytes),
    percent: percentLabel,
  });
}

function resourceAnomalyReason(item: WorkbenchPresentation['resourceSummary']['anomalies'][number]) {
  return (
    item.reasonLabel?.trim() ||
    item.status?.trim() ||
    item.reasonCode?.trim() ||
    item.health?.trim() ||
    item.state ||
    t('dashboard.workbench.resources.unknownAnomaly')
  );
}

function itemTitle(item: PresentationItem) {
  if (hasDashboardTranslation(item.titleKey)) {
    return t(item.titleKey, item.titleParams ?? {});
  }
  return resolveDashboardText(item.titleKey, item.titleFallback || item.id);
}

function itemDescription(item: PresentationItem) {
  if (hasDashboardTranslation(item.descriptionKey)) {
    return t(item.descriptionKey, item.descriptionParams ?? {});
  }
  return resolveDashboardText(item.descriptionKey, item.descriptionFallback || '');
}
</script>
<style scoped lang="less">
@import '@/shared/components/card-surface.less';

.workbench-preview {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-20);
  min-width: 0;
}

.workbench-preview__header-actions {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-12);
  justify-content: flex-end;
}

.workbench-preview__updated-at {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-large);
}

.workbench-preview :deep(.page-header__description) {
  font: var(--td-font-body-large);
}

.workbench-preview__loading {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-large);
  padding: var(--graft-density-gap-20);
}

.operational-status {
  align-items: center;
  background: color-mix(in srgb, var(--td-brand-color-light) 32%, var(--td-bg-color-container));
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-large);
  display: flex;
  gap: var(--graft-density-gap-24);
  justify-content: space-between;
  padding: var(--graft-density-gap-16) var(--graft-density-gap-20);
}

.operational-status__primary {
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-6);
  min-width: 0;
}

.operational-status__eyebrow {
  color: var(--td-brand-color);
  font: var(--td-font-body-medium);
  width: 100%;
}

.operational-status__summary {
  align-items: baseline;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8) var(--graft-density-gap-12);
}

.operational-status__summary strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-large);
}

.operational-status__count {
  color: var(--td-text-color-primary);
  font: var(--td-font-headline-medium);
}

.operational-status__distribution {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-large);
}

.operational-status__metrics {
  display: grid;
  flex: 0 0 auto;
  gap: var(--graft-density-gap-8);
  grid-template-columns: repeat(3, minmax(6rem, 1fr));
  margin: 0;
}

.operational-status__metrics div {
  border-left: 1px solid var(--td-border-level-1-color);
  padding-left: var(--graft-density-gap-16);
}

.operational-status__metrics dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
}

.operational-status__metrics dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: var(--graft-density-gap-4) 0 0;
}

.workbench-preview__grid {
  --graft-responsive-wide-split-template: minmax(0, 2fr) minmax(18rem, 1fr);
}

.workbench-preview__column {
  display: contents;
}

.workbench-surface--attention {
  order: 1;
}

.workbench-surface--health {
  order: 2;
}

.workbench-surface--module-coverage {
  order: 3;
}

.workbench-details {
  --graft-responsive-wide-split-template: minmax(0, 2fr) minmax(18rem, 1fr);
}

.workbench-details__column {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  min-width: 0;
}

.workbench-surface {
  .graft-card-surface();

  align-self: stretch;
  box-shadow: none;
  min-width: 0;
  width: 100%;
}

.workbench-surface__heading {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
  width: 100%;
}

.workbench-surface__heading h2,
.quick-actions__heading h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
}

.workbench-surface__heading p,
.quick-actions__heading p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-large);
  margin: var(--graft-density-gap-4) 0 0;
}

.workbench-surface__heading > span {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
  white-space: nowrap;
}

.workbench-empty {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-large);
  margin: 0;
  padding: var(--graft-density-gap-16) 0;
}

.workbench-empty--compact {
  font: var(--td-font-body-medium);
  padding: var(--graft-density-gap-8) 0;
}

.resource-note strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.resource-note p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-large);
  margin: var(--graft-density-gap-6) 0 0;
}

.module-coverage__metrics,
.resource-overview {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(auto-fit, minmax(7.5rem, 1fr));
  margin: 0;
}

.module-coverage__metrics > div,
.resource-overview > div {
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  min-width: 0;
  padding: var(--graft-density-gap-12);
}

.module-coverage__metrics dt,
.resource-overview dt {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
}

.module-coverage__metrics dd,
.resource-overview dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: var(--graft-density-gap-4) 0 0;
  overflow-wrap: anywhere;
}

.metric-group__items {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
}

.metric-item {
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-border-level-1-color);
  border-radius: var(--td-radius-medium);
  color: inherit;
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-4);
  min-width: 0;
  padding: var(--graft-density-gap-12);
  text-align: left;
}

.metric-item--actionable {
  cursor: pointer;
}

.metric-item--actionable:hover {
  background: var(--td-bg-color-container-hover);
  border-color: var(--td-border-level-2-color);
}

.metric-item--actionable:focus-visible {
  border-color: var(--td-brand-color);
  box-shadow: inset 0 0 0 1px var(--td-brand-color);
  outline: none;
}

.metric-item[data-tone='success'],
.metric-item[data-tone='warning'],
.metric-item[data-tone='error'],
.metric-item[data-tone='info'] {
  border-left-width: 3px;
}

.metric-item[data-tone='success'] {
  border-left-color: var(--td-success-color);
}

.metric-item[data-tone='warning'] {
  border-left-color: var(--td-warning-color);
}

.metric-item[data-tone='error'] {
  border-left-color: var(--td-error-color);
}

.metric-item[data-tone='info'] {
  border-left-color: var(--td-brand-color);
}

.metric-item__label,
.metric-item__description {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
}

.metric-item__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-headline-small);
}

.metric-item__value small {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  font-weight: 400;
}

.resource-breakdown {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin-top: var(--graft-density-gap-16);
}

.resource-breakdown__group,
.resource-anomalies {
  border-top: 1px solid var(--td-border-level-1-color);
  padding-top: var(--graft-density-gap-12);
}

.resource-anomalies {
  margin-top: var(--graft-density-gap-16);
}

.resource-breakdown__group h3,
.resource-anomalies h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0 0 var(--graft-density-gap-8);
}

.resource-ranking,
.resource-anomalies__list {
  display: grid;
  gap: var(--graft-density-gap-8);
  list-style: none;
  margin: 0;
  padding: 0;
}

.resource-ranking li,
.resource-anomalies__list li {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: flex;
  font: var(--td-font-body-medium);
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
  min-width: 0;
}

.resource-ranking li > span,
.resource-anomalies__list li > div {
  min-width: 0;
  overflow-wrap: anywhere;
}

.resource-ranking strong,
.resource-anomalies__list strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.resource-anomalies__list li > div {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-2);
}

.resource-note {
  align-items: flex-start;
  color: var(--td-text-color-placeholder);
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: auto minmax(0, 1fr);
  padding: var(--graft-density-gap-12) 0;
}

.quick-actions {
  border-top: 1px solid var(--td-border-level-1-color);
  display: grid;
  gap: var(--graft-density-gap-12);
  padding-top: var(--graft-density-gap-16);
}

.quick-actions__heading {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-16);
  justify-content: space-between;
}

.quick-actions__items {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(auto-fit, minmax(13.5rem, 1fr));
}

.quick-entry-drawer {
  display: flex;
  flex-direction: column;
  gap: var(--graft-density-gap-16);
  height: 100%;
  min-height: 0;
}

.quick-entry-drawer__search {
  flex: 0 0 auto;
}

.quick-entry-drawer__content {
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  gap: var(--graft-density-gap-20);
  min-height: 0;
  overflow-y: auto;
  padding-right: var(--graft-density-gap-4);
}

.quick-entry-drawer__section {
  display: grid;
  gap: var(--graft-density-gap-6);
}

.quick-entry-drawer__section h3 {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  font-weight: 500;
  margin: 0;
  padding-inline: var(--graft-density-gap-8);
}

.quick-entry-drawer__frequent,
.quick-entry-drawer__navigation,
.quick-entry-drawer__navigation-list {
  display: grid;
  gap: var(--graft-density-gap-4);
}

.quick-entry-drawer__navigation {
  gap: var(--graft-density-gap-20);
}

.quick-entry-drawer__navigation-item {
  align-items: center;
  background: transparent;
  border: 0;
  border-radius: var(--td-radius-medium);
  color: var(--td-text-color-primary);
  cursor: pointer;
  display: grid;
  font: var(--td-font-body-medium);
  gap: var(--graft-density-gap-10);
  grid-template-columns: auto minmax(0, 1fr);
  min-height: 44px;
  padding: var(--graft-density-gap-8);
  text-align: left;
  width: 100%;
}

.quick-entry-drawer__navigation-item:hover {
  background: var(--td-bg-color-container-hover);
}

.quick-entry-drawer__navigation-item:focus-visible {
  box-shadow: inset 0 0 0 1px var(--td-brand-color);
  outline: none;
}

.quick-entry-drawer__navigation-icon {
  align-items: center;
  color: var(--td-text-color-secondary);
  display: inline-flex;
  height: 18px;
  justify-content: center;
  width: 18px;
}

.quick-entry-drawer__navigation-icon :deep(svg) {
  height: 18px;
  width: 18px;
}

.quick-entry-drawer__empty {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
  margin: auto;
  padding: var(--graft-density-gap-24);
  text-align: center;
}

@container (width >= 75rem) {
  .workbench-preview__column {
    display: flex;
    flex-direction: column;
    gap: var(--graft-density-gap-16);
    min-width: 0;
  }
}

@media (width < @screen-md) {
  .operational-status {
    align-items: stretch;
    flex-direction: column;
  }

  .operational-status__metrics {
    width: 100%;
  }
}

@media (width < @screen-sm) {
  .workbench-preview__header-actions,
  .quick-actions__heading {
    align-items: stretch;
    flex-direction: column;
  }

  .workbench-preview__updated-at {
    order: 2;
  }

  .operational-status__metrics {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .resource-note {
    grid-template-columns: auto minmax(0, 1fr);
  }
}
</style>
