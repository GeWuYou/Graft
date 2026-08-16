<template>
  <section class="workbench-preview" data-page-type="overview-dashboard" data-preview-scenario="fixed">
    <page-header
      title-key="dashboard.workbench.title"
      :title-fallback="t('dashboard.workbench.title')"
      description-key="dashboard.workbench.description"
      :description-fallback="t('dashboard.workbench.description')"
      :source="{
        labelKey: 'dashboard.workbench.eyebrow',
        fallback: t('dashboard.workbench.eyebrow'),
        color: 'var(--td-brand-color-6)',
      }"
    >
      <template #actions>
        <div class="workbench-preview__header-actions">
          <span class="workbench-preview__updated-at">
            {{ t('dashboard.workbench.updatedAt', { time: generatedAtLabel }) }}
          </span>
          <t-button variant="outline" theme="default" :loading="refreshing" @click="refreshPreview">
            <template #icon><refresh-icon /></template>
            {{ t('dashboard.workbench.refresh') }}
          </t-button>
        </div>
      </template>
    </page-header>

    <section class="operational-status" :aria-label="t('dashboard.workbench.operational.title')">
      <div class="operational-status__primary">
        <span class="operational-status__eyebrow">{{ t('dashboard.workbench.operational.eyebrow') }}</span>
        <strong>{{
          t('dashboard.workbench.operational.needsReview', { count: presentation.operational.needsReview })
        }}</strong>
        <span class="operational-status__distribution">
          {{
            t('dashboard.workbench.operational.distribution', {
              warning: presentation.operational.statusCounts.warning,
              unknown: presentation.operational.statusCounts.unknown,
            })
          }}
        </span>
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

    <responsive-content class="workbench-preview__grid" layout="wide-split">
      <t-card class="workbench-surface workbench-surface--attention" :bordered="false" header-bordered>
        <template #header>
          <div class="workbench-surface__heading">
            <div>
              <h2>{{ t('dashboard.workbench.attention.title') }}</h2>
              <p>{{ t('dashboard.workbench.attention.description') }}</p>
            </div>
            <span>{{ t('dashboard.workbench.attention.count', { count: presentation.attention.length }) }}</span>
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
            <div class="workbench-row">
              <div class="workbench-row__copy">
                <div class="workbench-row__title-line">
                  <strong>{{ t(item.titleKey) }}</strong>
                  <workbench-status-indicator
                    :status="item.status"
                    :label="t(`dashboard.workbench.status.${item.status}`)"
                  />
                </div>
                <p>{{ t(item.descriptionKey) }}</p>
              </div>
            </div>
            <template #action>
              <t-button
                v-if="item.action"
                variant="text"
                theme="primary"
                size="small"
                :loading="retryingId === item.id"
                @click="handleAction(item)"
              >
                {{ t(item.action.labelKey) }}
              </t-button>
            </template>
          </t-list-item>
        </t-list>
      </t-card>

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
                <strong>{{ t(item.titleKey) }}</strong>
                <p>{{ t(item.descriptionKey) }}</p>
              </div>
            </div>
          </t-list-item>
        </t-list>
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
                  <strong>{{ t(item.titleKey) }}</strong>
                  <time v-if="item.occurredAt" :datetime="item.occurredAt">{{ formatTime(item.occurredAt) }}</time>
                </div>
                <p>{{ t(item.descriptionKey) }}</p>
              </div>
            </div>
            <template #action>
              <t-button v-if="item.action" variant="text" theme="primary" size="small" @click="handleAction(item)">
                {{ t(item.action.labelKey) }}
              </t-button>
            </template>
          </t-list-item>
        </t-list>
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
            <strong>{{ t(item.titleKey) }}</strong>
            <p>{{ t(item.descriptionKey) }}</p>
          </div>
          <t-button v-if="item.action" variant="text" theme="primary" size="small" @click="handleAction(item)">
            {{ t(item.action.labelKey) }}
          </t-button>
        </div>
      </t-card>
    </responsive-content>

    <section class="quick-actions" :aria-label="t('dashboard.workbench.quickActions.title')">
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
      <div class="quick-actions__items">
        <t-button
          v-for="action in primaryQuickActions"
          :key="action.id"
          class="quick-action"
          variant="text"
          theme="default"
          @click="navigate(action.route)"
        >
          <template #icon><graft-menu-icon :icon-key="action.iconKey" /></template>
          {{ t(action.titleKey) }}
        </t-button>
      </div>
    </section>

    <t-drawer
      v-model:visible="drawerVisible"
      :header="t('dashboard.workbench.quickActions.drawerTitle')"
      :footer="false"
      placement="right"
      size="420px"
    >
      <t-list class="quick-action-drawer" split>
        <t-list-item v-for="action in presentation.quickActions" :key="action.id" @click="navigate(action.route)">
          <div class="quick-action-drawer__item">
            <graft-menu-icon :icon-key="action.iconKey" />
            <div>
              <strong>{{ t(action.titleKey) }}</strong>
              <p>{{ t(action.descriptionKey) }}</p>
            </div>
            <chevron-right-icon />
          </div>
        </t-list-item>
      </t-list>
    </t-drawer>
  </section>
</template>
<script setup lang="ts">
import { ChevronRightIcon, InfoCircleIcon, RefreshIcon } from 'tdesign-icons-vue-next';
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';

import { currentLocale, t } from '@/locales';
import { PageHeader } from '@/shared/components/page';
import ResponsiveContent from '@/shared/components/responsive/ResponsiveContent.vue';
import GraftMenuIcon from '@/shared/icons/MenuIcon.vue';
import { formatLocaleDateTime, MEDIUM_DATE_TIME_FORMAT_OPTIONS } from '@/shared/observability';

import WorkbenchStatusIndicator from '../../components/workbench/WorkbenchStatusIndicator.vue';
import { DASHBOARD_PREVIEW_PRESENTATION, type PresentationItem } from '../../presentation/workbench';

defineOptions({ name: 'DashboardWorkbenchPreviewPage' });

const router = useRouter();
const presentation = DASHBOARD_PREVIEW_PRESENTATION;
const drawerVisible = ref(false);
const refreshing = ref(false);
const retryingId = ref('');
const generatedAt = ref(presentation.generatedAt);
const primaryQuickActions = computed(() => presentation.quickActions.slice(0, 4));
const generatedAtLabel = computed(() =>
  formatLocaleDateTime(generatedAt.value, currentLocale, MEDIUM_DATE_TIME_FORMAT_OPTIONS),
);

function formatTime(value: string) {
  return formatLocaleDateTime(value, currentLocale, MEDIUM_DATE_TIME_FORMAT_OPTIONS);
}

async function navigate(route: string) {
  drawerVisible.value = false;
  await router.push(route);
}

function refreshPreview() {
  refreshing.value = true;
  generatedAt.value = new Date().toISOString();
  window.setTimeout(() => {
    refreshing.value = false;
  }, 450);
}

function handleAction(item: PresentationItem) {
  if (!item.action) {
    return;
  }

  if (item.action.kind === 'retry') {
    retryingId.value = item.id;
    window.setTimeout(() => {
      retryingId.value = '';
    }, 450);
    return;
  }

  if (item.action.route) {
    void navigate(item.action.route);
  }
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
  font: var(--td-font-body-small);
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
  align-items: baseline;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8) var(--graft-density-gap-12);
  min-width: 0;
}

.operational-status__eyebrow {
  color: var(--td-brand-color);
  font: var(--td-font-body-small);
  width: 100%;
}

.operational-status__primary strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-headline-medium);
}

.operational-status__distribution {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-medium);
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
  font: var(--td-font-body-small);
}

.operational-status__metrics dd {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: var(--graft-density-gap-4) 0 0;
}

.workbench-preview__grid {
  --graft-responsive-wide-split-template: minmax(0, 2fr) minmax(18rem, 1fr);
}

.workbench-surface {
  .graft-card-surface();

  align-self: start;
  box-shadow: none;
  min-width: 0;
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
  font: var(--td-font-body-small);
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

.attention-row {
  border-left: 2px solid transparent;
  padding-left: var(--graft-density-gap-12) !important;
}

.attention-row--warning {
  border-left-color: var(--td-warning-color);
}

.attention-row--unknown {
  border-left-color: var(--td-text-color-placeholder);
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
.resource-note strong,
.quick-action-drawer__item strong {
  color: var(--td-text-color-primary);
  font: var(--td-font-body-medium);
}

.workbench-row__copy p,
.resource-note p,
.quick-action-drawer__item p {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: var(--graft-density-gap-4) 0 0;
}

.workbench-row__title-line {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: var(--graft-density-gap-8);
  justify-content: space-between;
}

.workbench-row__title-line time {
  color: var(--td-text-color-placeholder);
  font: var(--td-font-body-small);
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
  gap: var(--graft-density-gap-8);
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.quick-action {
  justify-content: flex-start;
  min-width: 0;
}

.quick-action-drawer :deep(.t-list-item) {
  cursor: pointer;
}

.quick-action-drawer__item {
  align-items: center;
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: auto minmax(0, 1fr) auto;
  width: 100%;
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

  .quick-actions__items {
    grid-template-columns: repeat(2, minmax(0, 1fr));
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
