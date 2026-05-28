<template>
  <div class="audit-overview" data-page-type="overview-dashboard">
    <governance-dashboard-shell
      domain="audit"
      :eyebrow="t('menu.audit.overview.title')"
      :title="t('audit.overview.title')"
      :description="t('audit.overview.description')"
    >
      <template #actions>
        <t-space size="small" wrap>
          <t-radio-group v-model="activeWindow" variant="default-filled" size="small">
            <t-radio-button v-for="option in timeRangeOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </t-radio-button>
          </t-radio-group>
          <t-tag theme="warning" variant="light-outline">{{ t('audit.overview.contractTag') }}</t-tag>
        </t-space>
      </template>

      <template #headerHint>
        <div class="audit-overview__hero-strip">
          <div v-for="signal in heroSignals" :key="signal.key" class="audit-overview__hero-signal">
            <span class="audit-overview__hero-label">{{ signal.label }}</span>
            <strong class="audit-overview__hero-value">{{ signal.value }}</strong>
            <span class="audit-overview__hero-meta">{{ signal.meta }}</span>
          </div>
        </div>
      </template>

      <template #summary>
        <governance-summary-card
          v-for="item in summaryCards"
          :key="item.key"
          kind="risk"
          :title="item.title"
          :value="item.value"
          :value-aside="item.valueAside"
          :description="item.description"
          :badge="item.badge"
        />
      </template>

      <section class="audit-overview__grid audit-overview__grid--hero">
        <governance-section
          kind="trend"
          :title="t('audit.overview.trendTitle')"
          :description="t('audit.overview.trendSubtitle')"
        >
          <div class="audit-overview__trend-list">
            <article v-for="trend in trendSignals" :key="trend.key" class="audit-overview__trend-card">
              <div class="audit-overview__trend-head">
                <div>
                  <h3>{{ trend.title }}</h3>
                  <p>{{ trend.description }}</p>
                </div>
                <t-tag :theme="trend.tone" variant="light-outline" size="small">{{ trend.changeLabel }}</t-tag>
              </div>
              <div class="audit-overview__sparkline" aria-hidden="true">
                <span
                  v-for="(point, index) in trend.points"
                  :key="`${trend.key}-${index}`"
                  class="audit-overview__sparkline-bar"
                  :style="{ height: `${point}%` }"
                />
              </div>
              <div class="audit-overview__trend-footer">
                <strong>{{ trend.currentValue }}</strong>
                <span>{{ trend.compareText }}</span>
              </div>
            </article>
          </div>
        </governance-section>

        <governance-action-panel
          kind="investigation"
          :title="t('audit.overview.anomalyTitle')"
          :description="t('audit.overview.anomalySubtitle')"
        >
          <div class="audit-overview__anomaly-list">
            <button v-for="alert in anomalySignals" :key="alert.key" type="button" class="audit-overview__anomaly-item">
              <div>
                <strong>{{ alert.title }}</strong>
                <p>{{ alert.description }}</p>
              </div>
              <t-tag :theme="alert.tone" variant="light-outline" size="small">{{ alert.value }}</t-tag>
            </button>
          </div>
        </governance-action-panel>
      </section>

      <section class="audit-overview__grid audit-overview__grid--split">
        <governance-section
          kind="investigation"
          :title="t('audit.overview.hotspotTitle')"
          :description="t('audit.overview.hotspotSubtitle')"
        >
          <div class="audit-overview__hotspot-grid">
            <article v-for="hotspot in hotspots" :key="hotspot.key" class="audit-overview__hotspot-card">
              <div class="audit-overview__hotspot-head">
                <span>{{ hotspot.group }}</span>
                <t-tag :theme="hotspot.tone" variant="light-outline" size="small">{{ hotspot.score }}</t-tag>
              </div>
              <h3>{{ hotspot.title }}</h3>
              <p>{{ hotspot.description }}</p>
              <ul class="audit-overview__hotspot-meta">
                <li v-for="item in hotspot.meta" :key="item">{{ item }}</li>
              </ul>
            </article>
          </div>
        </governance-section>

        <governance-action-panel
          kind="workflow"
          :title="t('audit.overview.entryTitle')"
          :description="t('audit.overview.entrySubtitle')"
        >
          <div class="audit-overview__entry-list">
            <button
              v-for="entry in investigationEntries"
              :key="entry.key"
              type="button"
              class="audit-overview__entry-item"
            >
              <div>
                <strong>{{ entry.title }}</strong>
                <p>{{ entry.description }}</p>
              </div>
              <span class="audit-overview__entry-query">{{ entry.query }}</span>
            </button>
          </div>
        </governance-action-panel>
      </section>

      <section class="audit-overview__grid audit-overview__grid--split">
        <governance-section
          kind="workflow"
          :title="t('audit.overview.correlationTitle')"
          :description="t('audit.overview.correlationSubtitle')"
        >
          <div class="audit-overview__correlation-list">
            <article v-for="item in correlations" :key="item.key" class="audit-overview__correlation-item">
              <div class="audit-overview__correlation-head">
                <strong>{{ item.title }}</strong>
                <t-tag :theme="item.tone" variant="light-outline" size="small">{{ item.status }}</t-tag>
              </div>
              <p>{{ item.description }}</p>
              <span>{{ item.nextStep }}</span>
            </article>
          </div>
        </governance-section>

        <governance-section
          kind="investigation"
          :title="t('audit.overview.timelineTitle')"
          :description="t('audit.overview.timelineSubtitle')"
        >
          <div class="audit-overview__timeline-list">
            <article v-for="entry in timelineEntries" :key="entry.id" class="audit-overview__timeline-item">
              <div class="audit-overview__timeline-marker">
                <span />
              </div>
              <div class="audit-overview__timeline-content">
                <div class="audit-overview__timeline-head">
                  <strong>{{ entry.title }}</strong>
                  <t-tag :theme="entry.success ? 'success' : 'danger'" variant="light-outline" size="small">
                    {{ entry.success ? t('audit.overview.statusSuccess') : t('audit.overview.statusFailed') }}
                  </t-tag>
                </div>
                <p>{{ entry.description }}</p>
                <span>{{ entry.context }}</span>
              </div>
            </article>
          </div>
        </governance-section>
      </section>
    </governance-dashboard-shell>
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  GovernanceActionPanel,
  GovernanceDashboardShell,
  GovernanceSection,
  GovernanceSummaryCard,
} from '@/shared/components/governance';

defineOptions({
  name: 'AuditOverviewIndex',
});

type AuditWindow = '24h' | '7d' | '30d';

const { t } = useI18n();
const activeWindow = ref<AuditWindow>('24h');

const timeRangeOptions = computed(() => [
  { label: t('audit.overview.timeRanges.24h'), value: '24h' as const },
  { label: t('audit.overview.timeRanges.7d'), value: '7d' as const },
  { label: t('audit.overview.timeRanges.30d'), value: '30d' as const },
]);

const heroSignals = computed(() => [
  {
    key: 'queue',
    label: t('audit.overview.heroSignals.queue.label'),
    value: t(`audit.overview.heroSignals.queue.values.${activeWindow.value}`),
    meta: t('audit.overview.heroSignals.queue.meta'),
  },
  {
    key: 'scope',
    label: t('audit.overview.heroSignals.scope.label'),
    value: t(`audit.overview.heroSignals.scope.values.${activeWindow.value}`),
    meta: t('audit.overview.heroSignals.scope.meta'),
  },
  {
    key: 'correlation',
    label: t('audit.overview.heroSignals.correlation.label'),
    value: t(`audit.overview.heroSignals.correlation.values.${activeWindow.value}`),
    meta: t('audit.overview.heroSignals.correlation.meta'),
  },
]);

const summaryCards = computed(() => [
  {
    key: 'failed-auth',
    title: t('audit.overview.cards.failedAuth.title'),
    value: t(`audit.overview.cards.failedAuth.values.${activeWindow.value}`),
    valueAside: t('audit.overview.cards.failedAuth.aside'),
    description: t('audit.overview.cards.failedAuth.description'),
    badge: t('audit.overview.cards.failedAuth.badge'),
  },
  {
    key: 'permission-denied',
    title: t('audit.overview.cards.permissionDenied.title'),
    value: t(`audit.overview.cards.permissionDenied.values.${activeWindow.value}`),
    valueAside: t('audit.overview.cards.permissionDenied.aside'),
    description: t('audit.overview.cards.permissionDenied.description'),
    badge: t('audit.overview.cards.permissionDenied.badge'),
  },
  {
    key: 'sensitive-actions',
    title: t('audit.overview.cards.sensitiveActions.title'),
    value: t(`audit.overview.cards.sensitiveActions.values.${activeWindow.value}`),
    valueAside: t('audit.overview.cards.sensitiveActions.aside'),
    description: t('audit.overview.cards.sensitiveActions.description'),
    badge: t('audit.overview.cards.sensitiveActions.badge'),
  },
  {
    key: 'escalation',
    title: t('audit.overview.cards.escalation.title'),
    value: t(`audit.overview.cards.escalation.values.${activeWindow.value}`),
    valueAside: t('audit.overview.cards.escalation.aside'),
    description: t('audit.overview.cards.escalation.description'),
    badge: t('audit.overview.cards.escalation.badge'),
  },
]);

const trendSignals = computed(() => [
  {
    key: 'failed-auth',
    title: t('audit.overview.trends.failedAuth.title'),
    description: t('audit.overview.trends.failedAuth.description'),
    changeLabel: t(`audit.overview.trends.failedAuth.changes.${activeWindow.value}`),
    currentValue: t(`audit.overview.trends.failedAuth.values.${activeWindow.value}`),
    compareText: t('audit.overview.trends.failedAuth.compareText'),
    tone: 'danger' as const,
    points:
      activeWindow.value === '24h'
        ? [22, 28, 62, 44, 80, 58, 41]
        : activeWindow.value === '7d'
          ? [16, 34, 52, 63, 71, 55, 48]
          : [18, 29, 43, 51, 70, 64, 53],
  },
  {
    key: 'permission-denied',
    title: t('audit.overview.trends.permissionDenied.title'),
    description: t('audit.overview.trends.permissionDenied.description'),
    changeLabel: t(`audit.overview.trends.permissionDenied.changes.${activeWindow.value}`),
    currentValue: t(`audit.overview.trends.permissionDenied.values.${activeWindow.value}`),
    compareText: t('audit.overview.trends.permissionDenied.compareText'),
    tone: 'warning' as const,
    points:
      activeWindow.value === '24h'
        ? [18, 22, 40, 57, 49, 61, 38]
        : activeWindow.value === '7d'
          ? [14, 20, 33, 41, 45, 51, 46]
          : [10, 16, 25, 34, 42, 39, 31],
  },
  {
    key: 'plugin-lifecycle',
    title: t('audit.overview.trends.pluginLifecycle.title'),
    description: t('audit.overview.trends.pluginLifecycle.description'),
    changeLabel: t(`audit.overview.trends.pluginLifecycle.changes.${activeWindow.value}`),
    currentValue: t(`audit.overview.trends.pluginLifecycle.values.${activeWindow.value}`),
    compareText: t('audit.overview.trends.pluginLifecycle.compareText'),
    tone: 'primary' as const,
    points:
      activeWindow.value === '24h'
        ? [12, 16, 18, 42, 36, 20, 14]
        : activeWindow.value === '7d'
          ? [8, 12, 15, 33, 37, 24, 18]
          : [6, 10, 16, 24, 31, 22, 18],
  },
]);

const anomalySignals = computed(() => [
  {
    key: 'suspicious-ip',
    title: t('audit.overview.anomalies.suspiciousIp.title'),
    description: t('audit.overview.anomalies.suspiciousIp.description'),
    value: t(`audit.overview.anomalies.suspiciousIp.values.${activeWindow.value}`),
    tone: 'danger' as const,
  },
  {
    key: 'token-anomaly',
    title: t('audit.overview.anomalies.tokenAnomaly.title'),
    description: t('audit.overview.anomalies.tokenAnomaly.description'),
    value: t(`audit.overview.anomalies.tokenAnomaly.values.${activeWindow.value}`),
    tone: 'warning' as const,
  },
  {
    key: 'privilege-escalation',
    title: t('audit.overview.anomalies.privilegeEscalation.title'),
    description: t('audit.overview.anomalies.privilegeEscalation.description'),
    value: t(`audit.overview.anomalies.privilegeEscalation.values.${activeWindow.value}`),
    tone: 'primary' as const,
  },
]);

const hotspots = computed(() => [
  {
    key: 'actor',
    group: t('audit.overview.hotspots.actor.group'),
    title: t('audit.overview.hotspots.actor.title'),
    description: t('audit.overview.hotspots.actor.description'),
    score: t(`audit.overview.hotspots.actor.scores.${activeWindow.value}`),
    tone: 'danger' as const,
    meta: [
      t('audit.overview.hotspots.actor.meta.one'),
      t('audit.overview.hotspots.actor.meta.two'),
      t('audit.overview.hotspots.actor.meta.three'),
    ],
  },
  {
    key: 'resource',
    group: t('audit.overview.hotspots.resource.group'),
    title: t('audit.overview.hotspots.resource.title'),
    description: t('audit.overview.hotspots.resource.description'),
    score: t(`audit.overview.hotspots.resource.scores.${activeWindow.value}`),
    tone: 'warning' as const,
    meta: [
      t('audit.overview.hotspots.resource.meta.one'),
      t('audit.overview.hotspots.resource.meta.two'),
      t('audit.overview.hotspots.resource.meta.three'),
    ],
  },
  {
    key: 'plugin',
    group: t('audit.overview.hotspots.plugin.group'),
    title: t('audit.overview.hotspots.plugin.title'),
    description: t('audit.overview.hotspots.plugin.description'),
    score: t(`audit.overview.hotspots.plugin.scores.${activeWindow.value}`),
    tone: 'primary' as const,
    meta: [
      t('audit.overview.hotspots.plugin.meta.one'),
      t('audit.overview.hotspots.plugin.meta.two'),
      t('audit.overview.hotspots.plugin.meta.three'),
    ],
  },
]);

const investigationEntries = computed(() => [
  {
    key: 'failed-auth',
    title: t('audit.overview.entries.failedAuth.title'),
    description: t('audit.overview.entries.failedAuth.description'),
    query: t('audit.overview.entries.failedAuth.query'),
  },
  {
    key: 'rbac-changes',
    title: t('audit.overview.entries.rbacChanges.title'),
    description: t('audit.overview.entries.rbacChanges.description'),
    query: t('audit.overview.entries.rbacChanges.query'),
  },
  {
    key: 'sensitive-ops',
    title: t('audit.overview.entries.sensitiveOps.title'),
    description: t('audit.overview.entries.sensitiveOps.description'),
    query: t('audit.overview.entries.sensitiveOps.query'),
  },
  {
    key: 'plugin-ops',
    title: t('audit.overview.entries.pluginOps.title'),
    description: t('audit.overview.entries.pluginOps.description'),
    query: t('audit.overview.entries.pluginOps.query'),
  },
]);

const correlations = computed(() => [
  {
    key: 'plugin-reload',
    title: t('audit.overview.correlations.pluginReload.title'),
    description: t('audit.overview.correlations.pluginReload.description'),
    nextStep: t('audit.overview.correlations.pluginReload.nextStep'),
    status: t('audit.overview.correlations.pluginReload.status'),
    tone: 'warning' as const,
  },
  {
    key: 'auth-spike',
    title: t('audit.overview.correlations.authSpike.title'),
    description: t('audit.overview.correlations.authSpike.description'),
    nextStep: t('audit.overview.correlations.authSpike.nextStep'),
    status: t('audit.overview.correlations.authSpike.status'),
    tone: 'danger' as const,
  },
  {
    key: 'monitor-link',
    title: t('audit.overview.correlations.monitorLink.title'),
    description: t('audit.overview.correlations.monitorLink.description'),
    nextStep: t('audit.overview.correlations.monitorLink.nextStep'),
    status: t('audit.overview.correlations.monitorLink.status'),
    tone: 'primary' as const,
  },
]);

const timelineEntries = computed(() => [
  {
    id: '1',
    title: t('audit.overview.timeline.items.failedSignin.title'),
    description: t('audit.overview.timeline.items.failedSignin.description'),
    context: t('audit.overview.timeline.items.failedSignin.context'),
    success: false,
  },
  {
    id: '2',
    title: t('audit.overview.timeline.items.roleModify.title'),
    description: t('audit.overview.timeline.items.roleModify.description'),
    context: t('audit.overview.timeline.items.roleModify.context'),
    success: true,
  },
  {
    id: '3',
    title: t('audit.overview.timeline.items.pluginReload.title'),
    description: t('audit.overview.timeline.items.pluginReload.description'),
    context: t('audit.overview.timeline.items.pluginReload.context'),
    success: true,
  },
  {
    id: '4',
    title: t('audit.overview.timeline.items.permissionDenied.title'),
    description: t('audit.overview.timeline.items.permissionDenied.description'),
    context: t('audit.overview.timeline.items.permissionDenied.context'),
    success: false,
  },
]);
</script>
<style scoped lang="less">
.audit-overview {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.audit-overview__hero-strip {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.audit-overview__hero-signal,
.audit-overview__trend-card,
.audit-overview__hotspot-card,
.audit-overview__correlation-item,
.audit-overview__timeline-item,
.audit-overview__anomaly-item,
.audit-overview__entry-item {
  background: color-mix(in srgb, var(--td-bg-color-container) 94%, var(--td-warning-color-1));
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 88%, var(--td-warning-color-4));
  border-radius: var(--td-radius-large);
}

.audit-overview__hero-signal {
  display: grid;
  gap: 4px;
  padding: 14px 16px;
}

.audit-overview__hero-label,
.audit-overview__hero-meta,
.audit-overview__trend-card p,
.audit-overview__hotspot-card p,
.audit-overview__correlation-item p,
.audit-overview__correlation-item span,
.audit-overview__timeline-content p,
.audit-overview__timeline-content span,
.audit-overview__entry-item p,
.audit-overview__entry-query {
  color: var(--td-text-color-secondary);
}

.audit-overview__hero-value {
  color: var(--td-text-color-primary);
  font-size: 20px;
  line-height: 28px;
}

.audit-overview__grid {
  display: grid;
  gap: 16px;
}

.audit-overview__grid--hero {
  grid-template-columns: minmax(0, 1.65fr) minmax(320px, 0.95fr);
}

.audit-overview__grid--split {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.audit-overview__trend-list,
.audit-overview__correlation-list,
.audit-overview__timeline-list,
.audit-overview__entry-list,
.audit-overview__anomaly-list {
  display: grid;
  gap: 12px;
}

.audit-overview__trend-list,
.audit-overview__hotspot-grid {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.audit-overview__trend-card,
.audit-overview__hotspot-card,
.audit-overview__correlation-item,
.audit-overview__timeline-item,
.audit-overview__anomaly-item,
.audit-overview__entry-item {
  padding: 16px;
}

.audit-overview__anomaly-item,
.audit-overview__entry-item {
  align-items: flex-start;
  cursor: pointer;
  display: flex;
  gap: 12px;
  justify-content: space-between;
  text-align: left;
  width: 100%;
}

.audit-overview__trend-head,
.audit-overview__correlation-head,
.audit-overview__hotspot-head,
.audit-overview__timeline-head {
  align-items: flex-start;
  display: flex;
  gap: 12px;
  justify-content: space-between;
}

.audit-overview__trend-head h3,
.audit-overview__hotspot-card h3,
.audit-overview__timeline-content strong,
.audit-overview__correlation-item strong,
.audit-overview__anomaly-item strong,
.audit-overview__entry-item strong {
  color: var(--td-text-color-primary);
  margin: 0;
}

.audit-overview__trend-head p,
.audit-overview__hotspot-card p,
.audit-overview__correlation-item p,
.audit-overview__timeline-content p,
.audit-overview__anomaly-item p,
.audit-overview__entry-item p {
  font-size: 12px;
  line-height: 18px;
  margin: 4px 0 0;
}

.audit-overview__sparkline {
  align-items: end;
  display: grid;
  gap: 6px;
  grid-template-columns: repeat(7, minmax(0, 1fr));
  height: 72px;
  margin: 18px 0 14px;
}

.audit-overview__sparkline-bar {
  background: linear-gradient(
    180deg,
    color-mix(in srgb, var(--td-warning-color-5) 32%, white),
    var(--td-warning-color-5)
  );
  border-radius: 999px 999px 6px 6px;
}

.audit-overview__trend-footer {
  align-items: baseline;
  display: flex;
  gap: 10px;
}

.audit-overview__trend-footer strong {
  color: var(--td-text-color-primary);
  font-size: 22px;
}

.audit-overview__hotspot-grid {
  display: grid;
  gap: 12px;
}

.audit-overview__hotspot-head span,
.audit-overview__hotspot-meta,
.audit-overview__entry-query {
  font-size: 12px;
  line-height: 18px;
}

.audit-overview__hotspot-meta {
  display: grid;
  gap: 4px;
  margin: 12px 0 0;
  padding-left: 18px;
}

.audit-overview__timeline-item {
  display: grid;
  gap: 12px;
  grid-template-columns: 20px minmax(0, 1fr);
}

.audit-overview__timeline-marker {
  display: flex;
  justify-content: center;
}

.audit-overview__timeline-marker span {
  background: var(--td-warning-color-5);
  border-radius: 999px;
  display: inline-flex;
  height: 10px;
  margin-top: 6px;
  width: 10px;
}

.audit-overview__timeline-content {
  display: grid;
  gap: 6px;
}

@media (width <= 1200px) {
  .audit-overview__hero-strip,
  .audit-overview__trend-list,
  .audit-overview__hotspot-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .audit-overview__grid--hero,
  .audit-overview__grid--split {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .audit-overview__hero-strip,
  .audit-overview__trend-list,
  .audit-overview__hotspot-grid {
    grid-template-columns: 1fr;
  }

  .audit-overview__anomaly-item,
  .audit-overview__entry-item,
  .audit-overview__trend-head,
  .audit-overview__correlation-head,
  .audit-overview__timeline-head,
  .audit-overview__hotspot-head {
    flex-direction: column;
  }
}
</style>
