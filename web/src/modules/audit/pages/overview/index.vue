<template>
  <div class="audit-overview" data-page-type="overview-dashboard">
    <management-page-content>
      <management-page-header :title="t('audit.overview.title')" :description="t('audit.overview.description')">
        <template #eyebrow>{{ t('menu.audit.overview.title') }}</template>
        <template #actions>
          <t-space size="small">
            <t-tag theme="primary" variant="light-outline">{{ t('audit.overview.contractTag') }}</t-tag>
            <t-button theme="default" variant="outline" @click="refreshDashboard">
              {{ t('audit.overview.refresh') }}
            </t-button>
          </t-space>
        </template>
      </management-page-header>

      <section class="audit-overview__summary-grid">
        <t-card v-for="item in summaryCards" :key="item.key" :title="item.title" :subtitle="item.subtitle" bordered>
          <t-statistic :value="item.value" :suffix="item.unit" />
          <p class="audit-overview__card-meta">{{ item.meta }}</p>
        </t-card>
      </section>

      <section class="audit-overview__main-grid">
        <t-card :title="t('audit.overview.timelineTitle')" :subtitle="t('audit.overview.timelineSubtitle')" bordered>
          <t-list split>
            <t-list-item v-for="entry in recentEvents" :key="entry.id">
              <t-list-item-meta :title="entry.title" :description="entry.description" />
              <template #action>
                <t-tag :theme="entry.success ? 'success' : 'danger'" variant="light-outline" size="small">
                  {{ entry.success ? t('audit.overview.statusSuccess') : t('audit.overview.statusFailed') }}
                </t-tag>
              </template>
            </t-list-item>
          </t-list>
        </t-card>

        <t-card :title="t('audit.overview.surfaceTitle')" :subtitle="t('audit.overview.surfaceSubtitle')" bordered>
          <div class="audit-overview__surface-stack">
            <div v-for="surface in focusSurfaces" :key="surface.key" class="audit-overview__surface-item">
              <div>
                <h3>{{ surface.title }}</h3>
                <p>{{ surface.description }}</p>
              </div>
              <t-tag :theme="surface.tone" variant="light-outline">{{ surface.value }}</t-tag>
            </div>
          </div>
        </t-card>
      </section>

      <t-card :title="t('audit.overview.guidanceTitle')" :subtitle="t('audit.overview.guidanceSubtitle')" bordered>
        <t-space direction="vertical" size="small" class="audit-overview__guidance">
          <div v-for="item in guidanceItems" :key="item.title" class="audit-overview__guidance-item">
            <strong>{{ item.title }}</strong>
            <span>{{ item.description }}</span>
          </div>
        </t-space>
      </t-card>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue';
import { useI18n } from 'vue-i18n';

import { ManagementPageContent, ManagementPageHeader } from '@/shared/components/management';

defineOptions({
  name: 'AuditOverviewIndex',
});

const { t } = useI18n();
const refreshToken = ref(0);

const summaryCards = [
  {
    key: 'today',
    title: t('audit.overview.cards.today.title'),
    subtitle: t('audit.overview.cards.today.subtitle'),
    value: 248,
    unit: '',
    meta: t('audit.overview.cards.today.meta'),
  },
  {
    key: 'failed',
    title: t('audit.overview.cards.failed.title'),
    subtitle: t('audit.overview.cards.failed.subtitle'),
    value: 12,
    unit: '',
    meta: t('audit.overview.cards.failed.meta'),
  },
  {
    key: 'actors',
    title: t('audit.overview.cards.actors.title'),
    subtitle: t('audit.overview.cards.actors.subtitle'),
    value: 19,
    unit: '',
    meta: t('audit.overview.cards.actors.meta'),
  },
  {
    key: 'latency',
    title: t('audit.overview.cards.latency.title'),
    subtitle: t('audit.overview.cards.latency.subtitle'),
    value: 87,
    unit: 'ms',
    meta: t('audit.overview.cards.latency.meta'),
  },
];

const recentEvents = [
  {
    id: '1',
    title: t('audit.overview.timeline.items.roleExport.title'),
    description: t('audit.overview.timeline.items.roleExport.description'),
    success: true,
  },
  {
    id: '2',
    title: t('audit.overview.timeline.items.schedulerStop.title'),
    description: t('audit.overview.timeline.items.schedulerStop.description'),
    success: false,
  },
  {
    id: '3',
    title: t('audit.overview.timeline.items.permissionReplace.title'),
    description: t('audit.overview.timeline.items.permissionReplace.description'),
    success: true,
  },
];

const focusSurfaces = [
  {
    key: 'rbac',
    title: t('audit.overview.surfaces.rbac.title'),
    description: t('audit.overview.surfaces.rbac.description'),
    value: t('audit.overview.surfaces.rbac.value'),
    tone: 'warning' as const,
  },
  {
    key: 'sessions',
    title: t('audit.overview.surfaces.sessions.title'),
    description: t('audit.overview.surfaces.sessions.description'),
    value: t('audit.overview.surfaces.sessions.value'),
    tone: 'success' as const,
  },
  {
    key: 'plugins',
    title: t('audit.overview.surfaces.plugins.title'),
    description: t('audit.overview.surfaces.plugins.description'),
    value: t('audit.overview.surfaces.plugins.value'),
    tone: 'primary' as const,
  },
];

const guidanceItems = [
  {
    title: t('audit.overview.guidance.items.scope.title'),
    description: t('audit.overview.guidance.items.scope.description'),
  },
  {
    title: t('audit.overview.guidance.items.trace.title'),
    description: t('audit.overview.guidance.items.trace.description'),
  },
  {
    title: t('audit.overview.guidance.items.next.title'),
    description: t('audit.overview.guidance.items.next.description'),
  },
];

function refreshDashboard() {
  refreshToken.value += 1;
}
</script>
<style scoped lang="less">
.audit-overview {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.audit-overview__summary-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.audit-overview__card-meta {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  margin: 12px 0 0;
}

.audit-overview__main-grid {
  display: grid;
  gap: 16px;
  grid-template-columns: minmax(0, 1.5fr) minmax(320px, 1fr);
}

.audit-overview__surface-stack {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.audit-overview__surface-item {
  align-items: center;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: flex;
  gap: 16px;
  justify-content: space-between;
  padding: 16px;
}

.audit-overview__surface-item h3 {
  color: var(--td-text-color-primary);
  font-size: 14px;
  margin: 0 0 4px;
}

.audit-overview__surface-item p {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  margin: 0;
}

.audit-overview__guidance {
  width: 100%;
}

.audit-overview__guidance-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.audit-overview__guidance-item span {
  color: var(--td-text-color-secondary);
}

@media (width <= 1200px) {
  .audit-overview__summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .audit-overview__main-grid {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .audit-overview__summary-grid {
    grid-template-columns: 1fr;
  }

  .audit-overview__surface-item {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
