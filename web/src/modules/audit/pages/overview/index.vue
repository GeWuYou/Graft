<template>
  <div class="audit-overview" data-page-type="overview-dashboard">
    <governance-dashboard-shell
      domain="audit"
      :eyebrow="t('menu.audit.title')"
      :title="t('audit.overview.title')"
      :description="t('audit.overview.description')"
    >
      <template #actions>
        <t-space size="small" wrap>
          <t-radio-group v-model="activeWindow" size="small" variant="default-filled">
            <t-radio-button v-for="option in timeRangeOptions" :key="option.value" :value="option.value">
              {{ option.label }}
            </t-radio-button>
          </t-radio-group>
          <t-button theme="default" variant="outline" :loading="loading" @click="fetchOverview">
            {{ t('audit.overview.refresh') }}
          </t-button>
        </t-space>
      </template>

      <management-empty-state
        v-if="errorMessage && !loading"
        tone="error"
        :title="t('audit.overview.errorTitle')"
        :description="errorMessage"
      >
        <template #actions>
          <t-button theme="primary" variant="outline" @click="fetchOverview">
            {{ t('audit.overview.retry') }}
          </t-button>
        </template>
      </management-empty-state>

      <template #summary>
        <governance-summary-card
          v-for="item in stats"
          :key="item.key"
          kind="activity"
          :title="item.title"
          :value="item.value"
          :description="item.meta"
          :value-aside="item.unit"
        />
      </template>

      <section class="audit-overview__grid">
        <governance-section :title="t('audit.overview.sections.failedAuth')">
          <div class="audit-overview__list">
            <article v-for="item in failedAuthItems" :key="item.key" class="audit-overview__list-item">
              <div>
                <strong>{{ item.actor }}</strong>
                <p>{{ item.resource }}</p>
              </div>
              <div class="audit-overview__item-meta">
                <span>{{ item.time }}</span>
                <t-tag theme="danger" variant="light-outline" size="small">{{ item.result }}</t-tag>
              </div>
            </article>
          </div>
        </governance-section>

        <governance-section :title="t('audit.overview.sections.permissionDenied')">
          <div class="audit-overview__list">
            <article v-for="item in permissionDeniedItems" :key="item.key" class="audit-overview__list-item">
              <div>
                <strong>{{ item.actor }}</strong>
                <p>{{ item.resource }}</p>
              </div>
              <div class="audit-overview__item-meta">
                <span>{{ item.time }}</span>
                <t-tag theme="warning" variant="light-outline" size="small">{{ item.result }}</t-tag>
              </div>
            </article>
          </div>
        </governance-section>
      </section>

      <section class="audit-overview__grid audit-overview__grid--bottom">
        <governance-section :title="t('audit.overview.sections.sensitiveOps')">
          <div class="audit-overview__list">
            <article v-for="item in sensitiveItems" :key="item.key" class="audit-overview__list-item">
              <div>
                <strong>{{ item.actor }}</strong>
                <p>{{ item.resource }}</p>
              </div>
              <div class="audit-overview__item-meta">
                <span>{{ item.time }}</span>
                <t-tag theme="warning" variant="light-outline" size="small">{{ item.result }}</t-tag>
              </div>
            </article>
          </div>
        </governance-section>

        <div class="audit-overview__stack">
          <governance-section :title="t('audit.overview.sections.shortcuts')">
            <div class="audit-overview__shortcut-list">
              <button
                v-for="entry in shortcuts"
                :key="entry.key"
                class="audit-overview__shortcut"
                type="button"
                @click="openShortcut(entry.preset)"
              >
                <strong>{{ entry.title }}</strong>
                <span>{{ entry.description }}</span>
              </button>
            </div>
          </governance-section>

          <governance-section :title="t('audit.overview.sections.riskWatch')">
            <div class="audit-overview__watch-list">
              <article class="audit-overview__watch-item">
                <div>
                  <strong>{{ t('audit.overview.futureRiskWatch.title') }}</strong>
                  <p>{{ t('audit.overview.futureRiskWatch.description') }}</p>
                </div>
                <t-tag theme="default" variant="light-outline" size="small">
                  {{ t('audit.overview.futureRiskWatch.tag') }}
                </t-tag>
              </article>
            </div>
          </governance-section>
        </div>
      </section>
    </governance-dashboard-shell>
  </div>
</template>
<script setup lang="ts">
import { MessagePlugin } from 'tdesign-vue-next';
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { resolveLocalizedErrorMessage } from '@/modules/shared/localized-api-error';
import { GovernanceDashboardShell, GovernanceSection, GovernanceSummaryCard } from '@/shared/components/governance';
import { ManagementEmptyState } from '@/shared/components/management';
import { createLogger } from '@/utils/logger';

import { getAuditOverview } from '../../api/audit';
import { AUDIT_ROUTE_PATH } from '../../contract/paths';
import type { AuditOverviewItem, AuditOverviewResponse, AuditOverviewWindow } from '../../types/audit';

defineOptions({
  name: 'AuditOverviewIndex',
});

const { t } = useI18n();
const router = useRouter();
const logger = createLogger('audit.overview');
const activeWindow = ref<AuditOverviewWindow>('24h');
const loading = ref(false);
const errorMessage = ref('');
const overview = ref<AuditOverviewResponse | null>(null);

const timeRangeOptions = computed(() => [
  { label: t('audit.overview.timeRanges.24h'), value: '24h' as const },
  { label: t('audit.overview.timeRanges.7d'), value: '7d' as const },
  { label: t('audit.overview.timeRanges.30d'), value: '30d' as const },
]);

const stats = computed(() => [
  {
    key: 'total',
    title: t('audit.overview.stats.totalLogs.title'),
    value: String(overview.value?.summary.total_logs ?? 0),
    unit: t('audit.overview.stats.totalLogs.unit'),
    meta: t('audit.overview.stats.totalLogs.meta'),
  },
  {
    key: 'failed',
    title: t('audit.overview.stats.failedToday.title'),
    value: String(overview.value?.summary.failed_operations ?? 0),
    unit: t('audit.overview.stats.failedToday.unit'),
    meta: t('audit.overview.stats.failedToday.meta'),
  },
  {
    key: 'risk',
    title: t('audit.overview.stats.highRisk.title'),
    value: String(overview.value?.summary.high_risk_events ?? 0),
    unit: t('audit.overview.stats.highRisk.unit'),
    meta: t('audit.overview.stats.highRisk.meta'),
  },
  {
    key: 'sensitive',
    title: t('audit.overview.stats.sensitiveOps.title'),
    value: String(overview.value?.summary.sensitive_operations ?? 0),
    unit: t('audit.overview.stats.sensitiveOps.unit'),
    meta: t('audit.overview.stats.sensitiveOps.meta'),
  },
]);

const failedAuthItems = computed(() =>
  toOverviewCards(overview.value?.failed_auth, t('audit.overview.itemResult.failed')),
);

const permissionDeniedItems = computed(() =>
  toOverviewCards(overview.value?.permission_denied, t('audit.overview.itemResult.denied')),
);

const sensitiveItems = computed(() =>
  toOverviewCards(overview.value?.sensitive_operations, t('audit.overview.itemResult.sensitive')),
);

const shortcuts = computed(() => [
  {
    key: 'failed',
    title: t('audit.overview.shortcuts.failedAuth.title'),
    description: t('audit.overview.shortcuts.failedAuth.description'),
    preset: 'failed-auth',
  },
  {
    key: 'rbac',
    title: t('audit.overview.shortcuts.rbacChanges.title'),
    description: t('audit.overview.shortcuts.rbacChanges.description'),
    preset: 'rbac-changes',
  },
  {
    key: 'sensitive',
    title: t('audit.overview.shortcuts.sensitiveOps.title'),
    description: t('audit.overview.shortcuts.sensitiveOps.description'),
    preset: 'sensitive-ops',
  },
]);

function openShortcut(preset: string) {
  void router.push({
    path: AUDIT_ROUTE_PATH.LOGS,
    query: { preset },
  });
}

async function fetchOverview() {
  loading.value = true;
  errorMessage.value = '';

  try {
    overview.value = await getAuditOverview({ window: activeWindow.value });
  } catch (error) {
    overview.value = null;
    logger.error('failed to fetch audit overview', error);
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('audit.overview.loadFailed'));
    MessagePlugin.error(errorMessage.value);
  } finally {
    loading.value = false;
  }
}

function toOverviewCards(items: AuditOverviewItem[] | undefined, result: string) {
  return (items ?? []).map((item) => ({
    key: String(item.id),
    actor: item.actor_display_name || item.actor_username || t('audit.common.unknownActor'),
    resource:
      item.resource_name ||
      [item.resource_type, item.resource_id].filter(Boolean).join(' / ') ||
      String(item.metadata?.request_path ?? t('audit.common.unknownResource')),
    time: formatTime(item.created_at),
    result,
  }));
}

function formatTime(value?: string) {
  if (!value) {
    return '-';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
}

watch(activeWindow, () => {
  void fetchOverview();
});

onMounted(() => {
  void fetchOverview();
});
</script>
<style scoped lang="less">
.audit-overview,
.audit-overview__stack,
.audit-overview__list,
.audit-overview__watch-list {
  display: flex;
  flex-direction: column;
}

.audit-overview__grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.audit-overview__grid--bottom {
  grid-template-columns: minmax(0, 1.3fr) minmax(320px, 0.9fr);
}

.audit-overview__stack,
.audit-overview__list,
.audit-overview__watch-list {
  gap: 16px;
}

.audit-overview__list-item,
.audit-overview__watch-item,
.audit-overview__shortcut {
  align-items: center;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: flex;
  gap: 12px;
  justify-content: space-between;
  padding: 14px 16px;
}

.audit-overview__list-item p,
.audit-overview__watch-item p,
.audit-overview__shortcut span,
.audit-overview__item-meta span {
  color: var(--td-text-color-secondary);
  margin: 0;
}

.audit-overview__item-meta {
  align-items: flex-end;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.audit-overview__shortcut {
  background: var(--td-bg-color-container);
  cursor: pointer;
  text-align: left;
  width: 100%;
}

@media (width <= 1280px) {
  .audit-overview__grid,
  .audit-overview__grid--bottom {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .audit-overview__list-item,
  .audit-overview__watch-item,
  .audit-overview__shortcut {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
