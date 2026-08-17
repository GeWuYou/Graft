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

    <section v-if="props.ready" class="operational-status" :aria-label="t('dashboard.workbench.operational.title')">
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
        <t-card class="workbench-surface workbench-surface--attention" :bordered="false" header-bordered>
          <template #header>
            <div class="workbench-surface__heading">
              <div>
                <h2>{{ t('dashboard.workbench.attention.title') }}</h2>
                <p>{{ t('dashboard.workbench.attention.description') }}</p>
              </div>
            </div>
          </template>
          <t-list class="workbench-list" split>
            <t-list-item
              v-for="item in presentation.attention"
              :key="item.id"
              class="attention-row"
              :class="`attention-row--${item.status}`"
              :data-attention-id="item.id"
              :data-status="item.status"
              :data-evidence="item.evidenceState"
            >
              <div class="workbench-row attention-row__content">
                <workbench-status-indicator
                  :status="item.status"
                  :label="t(`dashboard.workbench.status.${item.status}`)"
                />
                <div class="workbench-row__copy">
                  <strong>{{ itemTitle(item) }}</strong>
                  <p>{{ itemDescription(item) }}</p>
                </div>
              </div>
              <template #action>
                <t-button
                  v-if="item.action"
                  class="attention-row__action"
                  variant="outline"
                  theme="default"
                  size="small"
                  :loading="props.retryingId === item.id"
                  @click="handleAction(item)"
                >
                  {{ actionLabel(item) }}
                </t-button>
              </template>
            </t-list-item>
          </t-list>
          <p v-if="!presentation.attention.length" class="workbench-empty">
            {{ t('dashboard.workbench.attention.empty') }}
          </p>
        </t-card>

        <t-card class="workbench-surface workbench-surface--activity" :bordered="false" header-bordered>
          <template #header>
            <div class="workbench-surface__heading">
              <div>
                <h2>{{ t('dashboard.workbench.activity.title') }}</h2>
                <p>{{ t('dashboard.workbench.activity.description') }}</p>
              </div>
            </div>
          </template>
          <t-list class="workbench-list" split>
            <t-list-item v-for="item in presentation.activity" :key="item.id" :data-activity-id="item.id">
              <div class="workbench-row workbench-row--compact">
                <workbench-status-indicator
                  :status="item.status"
                  :label="t(`dashboard.workbench.status.${item.status}`)"
                  :show-label="false"
                />
                <div class="workbench-row__copy">
                  <div class="workbench-row__title-line">
                    <strong>{{ itemTitle(item) }}</strong>
                    <time v-if="item.occurredAt" :datetime="item.occurredAt">{{ formatTime(item.occurredAt) }}</time>
                  </div>
                  <p>{{ itemDescription(item) }}</p>
                </div>
              </div>
              <template #action>
                <t-button v-if="item.action" variant="text" theme="primary" size="small" @click="handleAction(item)">
                  {{ actionLabel(item) }}
                </t-button>
              </template>
            </t-list-item>
          </t-list>
          <p v-if="!presentation.activity.length" class="workbench-empty">
            {{ t('dashboard.workbench.activity.empty') }}
          </p>
        </t-card>
      </div>

      <div class="workbench-preview__column workbench-preview__column--secondary">
        <t-card class="workbench-surface workbench-surface--health" :bordered="false" header-bordered>
          <template #header>
            <div class="workbench-surface__heading">
              <div>
                <h2>{{ t('dashboard.workbench.health.title') }}</h2>
                <p>{{ t('dashboard.workbench.health.description') }}</p>
              </div>
            </div>
          </template>
          <t-list class="workbench-list workbench-list--quiet" split>
            <t-list-item
              v-for="item in presentation.health"
              :key="item.id"
              class="health-row"
              :data-health-id="item.id"
              :data-status="item.status"
            >
              <div class="workbench-row workbench-row--compact">
                <workbench-status-indicator
                  :status="item.status"
                  :label="t(`dashboard.workbench.status.${item.status}`)"
                  :show-label="false"
                />
                <div class="workbench-row__copy">
                  <strong>{{ itemTitle(item) }}</strong>
                  <p>{{ itemDescription(item) }}</p>
                </div>
              </div>
            </t-list-item>
          </t-list>
          <p v-if="!presentation.health.length" class="workbench-empty">
            {{ t('dashboard.workbench.health.empty') }}
          </p>
        </t-card>

        <t-card class="workbench-surface workbench-surface--resources" :bordered="false" header-bordered>
          <template #header>
            <div class="workbench-surface__heading">
              <div>
                <h2>{{ t('dashboard.workbench.resources.title') }}</h2>
                <p>{{ t('dashboard.workbench.resources.description') }}</p>
              </div>
            </div>
          </template>
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
            <t-button v-if="item.action" variant="text" theme="primary" size="small" @click="handleAction(item)">
              {{ actionLabel(item) }}
            </t-button>
          </div>
          <p v-if="!presentation.resources.length" class="workbench-empty">
            {{ t('dashboard.workbench.resources.empty') }}
          </p>
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
          @activate="navigate(action.route)"
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
                @activate="navigate(action.route)"
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
                  @click="navigate(link.route_location)"
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
import { formatLocaleDateTime, MEDIUM_DATE_TIME_FORMAT_OPTIONS } from '@/shared/observability';

import type { DashboardQuickActionLink } from '../../contract/quick-action-links';
import type { PresentationItem, WorkbenchPresentation } from '../../presentation/workbench';
import { hasDashboardTranslation, resolveDashboardText } from '../widgets/widget-i18n';
import WorkbenchQuickActionItem from './WorkbenchQuickActionItem.vue';
import WorkbenchStatusIndicator from './WorkbenchStatusIndicator.vue';

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
  navigate: [route: string];
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
const pageTitleKey = computed(() => (props.preview ? 'dashboard.workbench.title' : 'dashboard.page.title'));
const pageDescriptionKey = computed(() =>
  props.preview ? 'dashboard.workbench.description' : 'dashboard.page.description',
);
const pageEyebrowKey = computed(() => (props.preview ? 'dashboard.workbench.eyebrow' : 'dashboard.page.eyebrow'));
const updatedAtKey = computed(() => (props.preview ? 'dashboard.workbench.updatedAt' : 'dashboard.page.lastUpdated'));
const refreshLabelKey = computed(() => (props.preview ? 'dashboard.workbench.refresh' : 'dashboard.actions.refresh'));
const attentionDistribution = computed(() => {
  const counts = presentation.value.operational.statusCounts;
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

function formatTime(value: string) {
  return formatLocaleDateTime(value, currentLocale, MEDIUM_DATE_TIME_FORMAT_OPTIONS);
}

function navigate(route: string) {
  drawerVisible.value = false;
  emit('navigate', route);
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

function handleAction(item: PresentationItem) {
  if (!item.action) {
    return;
  }

  if (item.action.kind === 'retry') {
    emit('retry-item', item);
    return;
  }

  if (item.action.route) {
    void navigate(item.action.route);
  }
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

function actionLabel(item: PresentationItem) {
  if (!item.action) {
    return '';
  }
  return resolveDashboardText(item.action.labelKey, item.action.labelFallback || '');
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

.workbench-surface--activity {
  order: 3;
}

.workbench-surface--resources {
  order: 4;
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

.workbench-list,
.workbench-list :deep(.t-list-item) {
  background: transparent;
}

.workbench-list :deep(.t-list-item) {
  padding: var(--graft-density-gap-16) 0;
}

.workbench-list--quiet :deep(.t-list-item) {
  padding: var(--graft-density-gap-8) 0;
}

.workbench-empty {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-large);
  margin: 0;
  padding: var(--graft-density-gap-16) 0;
}

.attention-row {
  border-left: 3px solid transparent;
  padding-left: var(--graft-density-gap-16) !important;
}

.attention-row--warning {
  background: color-mix(in srgb, var(--td-warning-color-light) 12%, transparent) !important;
  border-left-color: var(--td-warning-color);
}

.attention-row--error {
  background: color-mix(in srgb, var(--td-error-color-light) 10%, transparent) !important;
  border-left-color: var(--td-error-color);
}

.attention-row--unknown {
  background: color-mix(in srgb, var(--td-bg-color-container-hover) 22%, transparent) !important;
  border-left-color: var(--td-text-color-placeholder);
}

.attention-row__content :deep(.workbench-status) {
  font: var(--td-font-body-medium);
  min-width: 4rem;
}

.attention-row__content .workbench-row__copy > strong {
  font: var(--td-font-title-small);
}

.attention-row__action {
  flex: 0 0 auto;
}

.workbench-row {
  align-items: flex-start;
  display: flex;
  gap: var(--graft-density-gap-12);
  min-width: 0;
}

.workbench-row--compact {
  gap: var(--graft-density-gap-8);
}

.workbench-row__copy {
  flex: 1 1 auto;
  min-width: 0;
}

.workbench-row__copy strong,
.resource-note strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.workbench-row__copy p,
.resource-note p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-large);
  margin: var(--graft-density-gap-6) 0 0;
}

.health-row .workbench-row__copy p {
  font: var(--td-font-body-medium);
  margin-top: var(--graft-density-gap-2);
}

.workbench-row__title-line {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
}

.workbench-row__title-line time {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-large);
}

.health-row {
  color: var(--td-text-color-secondary);
}

.resource-note {
  align-items: flex-start;
  color: var(--td-text-color-placeholder);
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: auto minmax(0, 1fr) auto;
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

  .resource-note .t-button {
    grid-column: 2;
    justify-self: start;
  }
}
</style>
