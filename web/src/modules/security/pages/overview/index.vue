<template>
  <div class="security-overview" data-page-type="overview-dashboard">
    <governance-dashboard-shell
      domain="security"
      :eyebrow="t('security.overview.navHint')"
      :title="t('security.overview.title')"
      title-key="security.overview.title"
      :description="t('security.overview.description')"
      description-key="security.overview.description"
    >
      <template #actions>
        <t-select v-model="activePreset" class="security-overview__preset" :options="timeRangeOptions" />
        <t-button variant="outline" :loading="loading" @click="fetchOverview">
          {{ t('security.overview.refresh') }}
        </t-button>
      </template>

      <template #feedback>
        <management-empty-state
          v-if="loadError"
          tone="error"
          :title="t('security.overview.errorTitle')"
          :description="loadError"
        >
          <template #actions>
            <t-button theme="primary" variant="outline" @click="fetchOverview">
              {{ t('security.overview.retry') }}
            </t-button>
          </template>
        </management-empty-state>
      </template>

      <template #summary>
        <governance-summary-card
          v-for="item in kpis"
          :key="item.key"
          kind="status"
          :title="item.title"
          :value="String(item.value)"
          :description="item.description"
        />
      </template>

      <template v-if="!loadError">
        <div class="security-overview__summary-grid">
          <governance-section
            kind="status"
            :title="t('security.overview.sections.access')"
            :description="t('security.overview.sections.accessDescription')"
          >
            <div class="security-overview__metric-grid">
              <article v-for="item in accessMetrics" :key="item.key" class="security-overview__metric">
                <span>{{ item.label }}</span>
                <strong>{{ item.value }}</strong>
                <small>{{ item.description }}</small>
              </article>
            </div>
            <template #actions>
              <t-button size="small" variant="text" @click="router.push(USER_ROUTE_PATH.LIST)">
                {{ t('security.overview.access.usersAction') }}
              </t-button>
              <t-button size="small" variant="text" @click="router.push(RBAC_BOOTSTRAP_ROUTE.ROLE_LIST.menuPath)">
                {{ t('security.overview.access.rolesAction') }}
              </t-button>
            </template>
          </governance-section>

          <governance-section
            kind="investigation"
            :title="t('security.overview.sections.audit')"
            :description="t('security.overview.sections.auditDescription')"
          >
            <div class="security-overview__metric-grid security-overview__metric-grid--audit">
              <article v-for="item in auditMetrics" :key="item.key" class="security-overview__metric">
                <span>{{ item.label }}</span>
                <strong>{{ item.value }}</strong>
                <small>{{ item.description }}</small>
              </article>
            </div>
            <template #actions>
              <t-button size="small" variant="text" @click="router.push(AUDIT_ROUTE_PATH.LOGS)">
                {{ t('security.overview.risk.viewLogs') }}
              </t-button>
            </template>
          </governance-section>
        </div>

        <div class="security-overview__detail-grid">
          <governance-section
            kind="investigation"
            :title="t('security.overview.sections.events')"
            :description="t('security.overview.sections.eventsDescription')"
          >
            <div v-if="recentEvents.length" class="security-overview__event-list">
              <button
                v-for="event in recentEvents"
                :key="event.id"
                class="security-overview__event"
                type="button"
                @click="openAuditLogs(event.request_id)"
              >
                <span class="security-overview__event-copy">
                  <strong>{{ eventLabel(event.action) }}</strong>
                  <small>{{ event.action }}</small>
                </span>
                <span class="security-overview__event-meta">
                  <t-tag size="small" :theme="riskTheme(event.risk_level)" variant="light">
                    {{ t(`audit.common.risk.${event.risk_level}`) }}
                  </t-tag>
                  <small>{{ formatLocaleDateTime(event.created_at, locale) }}</small>
                </span>
              </button>
            </div>
            <management-empty-state
              v-else
              :title="t('security.overview.events.emptyTitle')"
              :description="t('security.overview.events.emptyDescription')"
            />
          </governance-section>

          <governance-section
            kind="workflow"
            :title="t('security.overview.sections.pending')"
            :description="t('security.overview.sections.pendingDescription')"
          >
            <div v-if="pendingItems.length" class="security-overview__pending-list">
              <article v-for="item in pendingItems" :key="item.key" class="security-overview__pending-item">
                <span>{{ item.label }}</span>
                <strong>{{ item.value }}</strong>
                <small>{{ item.description }}</small>
              </article>
            </div>
            <management-empty-state
              v-else
              :title="t('security.overview.pending.emptyTitle')"
              :description="t('security.overview.pending.emptyDescription')"
            />
          </governance-section>
        </div>
      </template>
    </governance-dashboard-shell>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { AUDIT_ROUTE_PATH } from '@/modules/audit/contract/paths';
import { AUDIT_TIME_PRESET } from '@/modules/audit/contract/time-presets';
import { RBAC_BOOTSTRAP_ROUTE } from '@/modules/rbac/contract/bootstrap';
import { USER_ROUTE_PATH } from '@/modules/user/contract/paths';
import { GovernanceDashboardShell, GovernanceSection, GovernanceSummaryCard } from '@/shared/components/governance';
import { ManagementEmptyState } from '@/shared/components/management';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';

import { getSecurityOverview } from '../../api/security';
import type { SecurityOverviewResponse } from '../../types/security';

const { locale, t } = useI18n();
const router = useRouter();
const activePreset = ref<SecurityOverviewResponse['time_preset']>(AUDIT_TIME_PRESET.LAST_24H);
const loading = ref(false);
const loadError = ref('');
const overview = ref<SecurityOverviewResponse | null>(null);

const timeRangeOptions = computed(() => [
  { label: t('security.overview.timeRanges.24h'), value: AUDIT_TIME_PRESET.LAST_24H },
  { label: t('security.overview.timeRanges.7d'), value: AUDIT_TIME_PRESET.LAST_7D },
  { label: t('security.overview.timeRanges.30d'), value: AUDIT_TIME_PRESET.LAST_30D },
]);

const access = computed(() => overview.value?.access_control);
const audit = computed(() => overview.value?.audit);
const recentEvents = computed(() => audit.value?.recent_events ?? []);

const kpis = computed(() => [
  {
    key: 'users',
    title: t('security.overview.stats.users'),
    value: access.value?.total_users ?? 0,
    description: t('security.overview.stats.usersDescription'),
  },
  {
    key: 'unassigned',
    title: t('security.overview.stats.unassigned'),
    value: access.value?.unassigned_user_count ?? 0,
    description: t('security.overview.stats.unassignedDescription'),
  },
  {
    key: 'high-risk',
    title: t('security.overview.stats.highRisk'),
    value: audit.value?.high_risk_events ?? 0,
    description: t('security.overview.stats.highRiskDescription'),
  },
  {
    key: 'failed',
    title: t('security.overview.stats.denied'),
    value: audit.value?.failed_operations ?? 0,
    description: t('security.overview.stats.deniedDescription'),
  },
]);

const accessMetrics = computed(() => [
  {
    key: 'roles',
    label: t('security.overview.access.roles'),
    value: access.value?.role_count ?? 0,
    description: t('security.overview.access.rolesDescription', { custom: access.value?.custom_role_count ?? 0 }),
  },
  {
    key: 'permissions',
    label: t('security.overview.access.permissions'),
    value: access.value?.permission_count ?? 0,
    description: t('security.overview.access.permissionsDescription'),
  },
  {
    key: 'assignments',
    label: t('security.overview.access.bindings'),
    value: access.value?.role_assignment_count ?? 0,
    description: t('security.overview.access.bindingsDescription'),
  },
  {
    key: 'disabled',
    label: t('security.overview.access.disabled'),
    value: access.value?.disabled_users ?? 0,
    description: t('security.overview.access.disabledDescription'),
  },
]);

const auditMetrics = computed(() => [
  {
    key: 'events',
    label: t('security.overview.audit.total'),
    value: audit.value?.total_logs ?? 0,
    description: t('security.overview.audit.totalDescription'),
  },
  {
    key: 'sensitive',
    label: t('security.overview.audit.sensitive'),
    value: audit.value?.sensitive_operations ?? 0,
    description: t('security.overview.audit.sensitiveDescription'),
  },
  {
    key: 'risk-groups',
    label: t('security.overview.audit.riskGroups'),
    value: audit.value?.risk_groups.length ?? 0,
    description: t('security.overview.audit.riskGroupsDescription'),
  },
]);

const pendingItems = computed(() => {
  const items = [] as Array<{ key: string; label: string; value: number; description: string }>;
  const unassigned = access.value?.unassigned_user_count ?? 0;
  const emptyRoles = access.value?.empty_custom_role_count ?? 0;
  const highRisk = audit.value?.high_risk_events ?? 0;

  if (unassigned > 0) {
    items.push({
      key: 'unassigned',
      label: t('security.overview.pending.unassigned'),
      value: unassigned,
      description: t('security.overview.pending.unassignedDescription'),
    });
  }
  if (emptyRoles > 0) {
    items.push({
      key: 'empty-roles',
      label: t('security.overview.pending.emptyRoles'),
      value: emptyRoles,
      description: t('security.overview.pending.emptyRolesDescription'),
    });
  }
  if (highRisk > 0) {
    items.push({
      key: 'high-risk',
      label: t('security.overview.pending.highRisk'),
      value: highRisk,
      description: t('security.overview.pending.highRiskDescription'),
    });
  }

  return items;
});

function riskTheme(level: string) {
  if (level === 'CRITICAL') return 'danger' as const;
  if (level === 'HIGH') return 'warning' as const;
  if (level === 'MEDIUM') return 'primary' as const;
  return 'default' as const;
}

function eventLabel(action: string) {
  const key = `audit.actionLabel.${action}`;
  const translated = t(key);
  return translated === key ? action : translated;
}

function openAuditLogs(requestId: string) {
  router.push({ path: AUDIT_ROUTE_PATH.LOGS, query: requestId ? { request_id: requestId } : {} });
}

let fetchSequence = 0;

async function fetchOverview() {
  const requestSequence = ++fetchSequence;
  loading.value = true;
  loadError.value = '';
  try {
    const response = await getSecurityOverview({ preset: activePreset.value });
    if (requestSequence === fetchSequence) {
      overview.value = response;
    }
  } catch (error) {
    if (requestSequence === fetchSequence) {
      loadError.value = resolveLocalizedErrorMessage(t, error, t('security.overview.loadFailed'));
    }
  } finally {
    if (requestSequence === fetchSequence) {
      loading.value = false;
    }
  }
}

watch(activePreset, fetchOverview);
onMounted(fetchOverview);
</script>
<style scoped lang="less">
.security-overview__preset {
  width: 148px;
}

.security-overview__summary-grid,
.security-overview__detail-grid {
  display: grid;
  gap: var(--graft-density-gap-18);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.security-overview__metric-grid {
  display: grid;
  gap: var(--graft-density-gap-12);
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.security-overview__metric,
.security-overview__pending-item,
.security-overview__event {
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-border);
  min-width: 0;
}

.security-overview__metric,
.security-overview__pending-item {
  display: grid;
  gap: var(--graft-density-gap-4);
  padding: var(--graft-density-gap-12);
}

.security-overview__metric span,
.security-overview__pending-item span,
.security-overview__event-copy small,
.security-overview__event-meta small {
  color: var(--td-text-color-secondary);
}

.security-overview__metric strong,
.security-overview__pending-item strong {
  color: var(--td-text-color-primary);
  font-size: var(--td-font-size-title-extraLarge);
  line-height: var(--td-line-height-title-extraLarge);
}

.security-overview__metric small,
.security-overview__pending-item small {
  color: var(--td-text-color-placeholder);
}

.security-overview__event-list,
.security-overview__pending-list {
  display: grid;
  gap: var(--graft-density-gap-10);
}

.security-overview__event {
  align-items: center;
  color: var(--td-text-color-primary);
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  padding: var(--graft-density-gap-12);
  text-align: left;
}

.security-overview__event:hover {
  border-color: var(--td-brand-color);
}

.security-overview__event-copy,
.security-overview__event-meta {
  display: grid;
  gap: var(--graft-density-gap-4);
  min-width: 0;
}

.security-overview__event-copy strong,
.security-overview__event-copy small,
.security-overview__event-meta small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.security-overview__event-copy strong {
  color: var(--td-text-color-primary);
}

.security-overview__event-meta {
  justify-items: end;
  margin-left: var(--graft-density-gap-12);
}

@media (width <= 1199px) {
  .security-overview__summary-grid,
  .security-overview__detail-grid {
    grid-template-columns: 1fr;
  }
}

@media (width <= 767px) {
  .security-overview__preset {
    width: 100%;
  }

  .security-overview__metric-grid {
    grid-template-columns: 1fr;
  }

  .security-overview__event {
    align-items: flex-start;
    flex-direction: column;
    gap: var(--graft-density-gap-10);
  }

  .security-overview__event-meta {
    justify-items: start;
    margin-left: 0;
  }
}
</style>
