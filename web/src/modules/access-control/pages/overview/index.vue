<template>
  <div class="access-control-overview" data-page-type="overview-dashboard">
    <management-page-content>
      <management-page-header
        :title="t('accessControl.overview.title')"
        :description="t('accessControl.overview.description')"
      >
        <template #eyebrow>{{ t('accessControl.overview.navHint') }}</template>
        <template #actions>
          <t-button variant="outline" :loading="loading" @click="fetchOverview">
            {{ t('accessControl.overview.actions.refresh') }}
          </t-button>
          <t-button theme="primary" variant="outline" @click="goToUsers('create')">
            {{ t('accessControl.overview.actions.createUser') }}
          </t-button>
          <t-button theme="primary" @click="goToRoles('create')">
            {{ t('accessControl.overview.actions.createRole') }}
          </t-button>
        </template>
      </management-page-header>

      <div v-if="loadError" class="access-control-overview__feedback">
        <management-empty-state
          tone="error"
          :title="t('accessControl.overview.state.loadFailedTitle')"
          :description="loadError || t('accessControl.overview.state.loadFailedDescription')"
        >
          <template #actions>
            <t-button theme="primary" variant="outline" @click="fetchOverview">
              {{ t('accessControl.overview.state.retry') }}
            </t-button>
          </template>
        </management-empty-state>
      </div>

      <management-stats-grid :items="statItems" layout="compact" />

      <section class="access-control-overview__grid">
        <management-table-card>
          <template #head>
            <div class="section-head">
              <div>
                <h2>{{ t('accessControl.overview.quickLinks.title') }}</h2>
                <p>{{ t('accessControl.overview.quickLinks.subtitle') }}</p>
              </div>
            </div>
          </template>
          <div class="quick-link-grid">
            <button class="quick-link-card" type="button" @click="goToUsers()">
              <div class="quick-link-card__head">
                <span class="quick-link-card__title">{{ t('accessControl.overview.quickLinks.users.title') }}</span>
                <span class="quick-link-card__count">{{ displayValue(users.length) }}</span>
              </div>
              <p class="quick-link-card__description">
                {{ t('accessControl.overview.quickLinks.users.description') }}
              </p>
              <span class="quick-link-card__action">{{ t('accessControl.overview.quickLinks.users.action') }}</span>
            </button>
            <button class="quick-link-card" type="button" @click="goToRoles()">
              <div class="quick-link-card__head">
                <span class="quick-link-card__title">{{ t('accessControl.overview.quickLinks.roles.title') }}</span>
                <span class="quick-link-card__count">{{ displayValue(roles.length) }}</span>
              </div>
              <p class="quick-link-card__description">
                {{ t('accessControl.overview.quickLinks.roles.description') }}
              </p>
              <span class="quick-link-card__action">{{ t('accessControl.overview.quickLinks.roles.action') }}</span>
            </button>
          </div>
        </management-table-card>

        <management-table-card>
          <template #head>
            <div class="section-head">
              <div>
                <h2>{{ t('accessControl.overview.status.title') }}</h2>
                <p>{{ t('accessControl.overview.status.subtitle') }}</p>
              </div>
            </div>
          </template>
          <div class="status-list">
            <article v-for="item in statusItems" :key="item.label" class="status-list__item">
              <span class="status-list__label">{{ item.label }}</span>
              <strong class="status-list__value">{{ item.value }}</strong>
              <p class="status-list__description">{{ item.description }}</p>
            </article>
          </div>
        </management-table-card>
      </section>

      <management-table-card>
        <template #head>
          <div class="section-head">
            <div>
              <h2>{{ t('accessControl.overview.todo.title') }}</h2>
              <p>{{ t('accessControl.overview.todo.subtitle') }}</p>
            </div>
          </div>
        </template>
        <div class="issue-section">
          <div class="issue-section__block">
            <div class="issue-section__heading">
              <h3>{{ t('accessControl.overview.risk.groupTitle') }}</h3>
              <span class="issue-section__meta">{{ displayValue(riskItems.length) }}</span>
            </div>
            <div v-if="riskItems.length > 0" class="risk-list">
              <article v-for="item in riskItems" :key="item.label" class="risk-list__item">
                <div class="risk-list__head">
                  <span class="risk-list__label">{{ item.label }}</span>
                  <strong class="risk-list__value">{{ item.value }}</strong>
                </div>
                <p class="risk-list__description">
                  {{ item.description || t('accessControl.overview.risk.defaultDescription') }}
                </p>
              </article>
            </div>
            <management-empty-state
              v-else
              :title="t('accessControl.overview.risk.emptyTitle')"
              :description="t('accessControl.overview.risk.emptyDescription')"
            />
          </div>

          <div class="issue-section__block">
            <div class="issue-section__heading">
              <h3>{{ t('accessControl.overview.todo.title') }}</h3>
              <span class="issue-section__meta">{{ displayValue(todoItems.length) }}</span>
            </div>
            <div v-if="todoItems.length > 0" class="todo-list">
              <article v-for="item in todoItems" :key="item.label" class="todo-list__item">
                <div class="todo-list__head">
                  <span class="todo-list__label">{{ item.label }}</span>
                  <t-tag size="small" theme="default" variant="light">{{ item.state }}</t-tag>
                </div>
                <p class="todo-list__description">{{ item.description }}</p>
              </article>
            </div>
            <management-empty-state
              v-else
              :title="t('accessControl.overview.todo.emptyTitle')"
              :description="t('accessControl.overview.todo.emptyDescription')"
            />
          </div>
        </div>
      </management-table-card>
    </management-page-content>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRouter } from 'vue-router';

import { getPermissions } from '@/modules/rbac/api/rbac';
import { getRoles as getUserRoles } from '@/modules/user/api/user-roles';
import { getUsers } from '@/modules/user/api/users';
import {
  ManagementEmptyState,
  ManagementPageContent,
  ManagementPageHeader,
  type ManagementStatItem,
  ManagementStatsGrid,
  ManagementTableCard,
} from '@/shared/components/management';

import { ACCESS_CONTROL_ROUTE_PATH } from '../../contract/bootstrap';

type RoleSummary = {
  id: number;
  builtin: boolean;
  permission_count: number;
  updated_at: string;
};

type PermissionSummary = {
  id: number;
};

type UserSummary = {
  id: number;
  status: string;
};

const { t } = useI18n();
const router = useRouter();
const loading = ref(false);
const loadError = ref('');
const users = ref<UserSummary[]>([]);
const roles = ref<RoleSummary[]>([]);
const permissions = ref<PermissionSummary[]>([]);
const roleBindings = ref<Record<number, number[]>>({});

const unassignedUserCount = computed(
  () => users.value.filter((user) => (roleBindings.value[user.id] ?? []).length === 0).length,
);
const disabledUserCount = computed(() => users.value.filter((user) => user.status === 'disabled').length);
const builtinRoleCount = computed(() => roles.value.filter((role) => role.builtin).length);
const customRoleCount = computed(() => roles.value.filter((role) => !role.builtin).length);
const emptyCustomRoleCount = computed(
  () => roles.value.filter((role) => !role.builtin && role.permission_count === 0).length,
);
const staleRoleCount = computed(() => roles.value.filter((role) => !role.updated_at).length);
const totalRoleBindingCount = computed(() =>
  Object.values(roleBindings.value).reduce((sum, ids) => sum + ids.length, 0),
);

const statItems = computed<ManagementStatItem[]>(() => [
  {
    label: t('accessControl.overview.stats.totalUsers'),
    value: displayValue(users.value.length),
    description: t('accessControl.overview.stats.totalUsersDescription', {
      enabled: displayValue(users.value.length - disabledUserCount.value),
      disabled: displayValue(disabledUserCount.value),
    }),
  },
  {
    label: t('accessControl.overview.stats.totalRoles'),
    value: displayValue(roles.value.length),
    description: t('accessControl.overview.stats.totalRolesDescription', {
      builtin: displayValue(builtinRoleCount.value),
      custom: displayValue(customRoleCount.value),
    }),
  },
  {
    label: t('accessControl.overview.stats.totalPermissions'),
    value: displayValue(permissions.value.length),
    description: t('accessControl.overview.stats.totalPermissionsDescription'),
  },
  {
    label: t('accessControl.overview.stats.assignmentCount'),
    value: displayValue(totalRoleBindingCount.value),
    description: t('accessControl.overview.stats.assignmentCountDescription', {
      pending: displayValue(unassignedUserCount.value),
    }),
  },
]);

const statusItems = computed<ManagementStatItem[]>(() => [
  {
    label: t('accessControl.overview.status.assignedUsers'),
    value: displayValue(users.value.length - unassignedUserCount.value),
    description: t('accessControl.overview.status.assignedUsersDescription', {
      pending: displayValue(unassignedUserCount.value),
    }),
  },
  {
    label: t('accessControl.overview.status.customRoles'),
    value: displayValue(customRoleCount.value),
    description: t('accessControl.overview.status.customRolesDescription', {
      empty: displayValue(emptyCustomRoleCount.value),
    }),
  },
  {
    label: t('accessControl.overview.status.builtinRoles'),
    value: displayValue(builtinRoleCount.value),
    description: t('accessControl.overview.status.builtinRolesDescription'),
  },
]);

const riskItems = computed(() => {
  const items: Array<{ label: string; value: string; description?: string }> = [];

  if (unassignedUserCount.value > 0) {
    items.push({
      label: t('accessControl.overview.risk.unassignedUsers'),
      value: displayValue(unassignedUserCount.value),
    });
  }

  if (emptyCustomRoleCount.value > 0) {
    items.push({
      label: t('accessControl.overview.risk.emptyRoles'),
      value: displayValue(emptyCustomRoleCount.value),
    });
  }

  if (disabledUserCount.value > 0) {
    items.push({
      label: t('accessControl.overview.risk.disabledUsers'),
      value: displayValue(disabledUserCount.value),
    });
  }

  if (roles.value.some((role) => role.builtin)) {
    items.push({
      label: t('accessControl.overview.risk.builtinNotice'),
      value: displayValue(builtinRoleCount.value),
      description: t('accessControl.overview.risk.builtinNoticeDescription'),
    });
  }

  if (staleRoleCount.value > 0) {
    items.push({
      label: t('accessControl.overview.risk.stale'),
      value: displayValue(staleRoleCount.value),
    });
  }

  return items;
});

const todoItems = computed(() => [
  {
    label: t('accessControl.overview.todo.assignmentSyncLabel'),
    state: t('accessControl.overview.todo.assignmentSyncState'),
    description: t('accessControl.overview.todo.assignmentSyncDescription', {
      count: displayValue(unassignedUserCount.value),
    }),
  },
  {
    label: t('accessControl.overview.todo.auditLabel'),
    state: t('accessControl.overview.todo.auditState'),
    description: t('accessControl.overview.todo.auditDescription'),
  },
]);

function displayValue(value?: number | null) {
  return typeof value === 'number' && Number.isFinite(value)
    ? new Intl.NumberFormat().format(value)
    : t('accessControl.overview.state.unknown');
}

async function fetchOverview() {
  loading.value = true;
  loadError.value = '';

  try {
    const [userResult, roleResult, permissionResult] = await Promise.all([
      getUsers(),
      getUserRoles(),
      getPermissions(),
    ]);
    users.value = userResult.items;
    roles.value = roleResult.items;
    permissions.value = permissionResult.items;

    const bindings = await Promise.all(
      userResult.items.map(async (user) => {
        const { getUserRoleBindings } = await import('@/modules/user/api/user-roles');
        const response = await getUserRoleBindings(user.id);
        return [user.id, response.role_ids] as const;
      }),
    );

    roleBindings.value = Object.fromEntries(bindings);
  } catch (error) {
    users.value = [];
    roles.value = [];
    permissions.value = [];
    roleBindings.value = {};
    loadError.value = error instanceof Error ? error.message : t('accessControl.overview.state.loadFailedDescription');
  } finally {
    loading.value = false;
  }
}

function goToUsers(mode?: 'create') {
  void router.push({
    path: ACCESS_CONTROL_ROUTE_PATH.USERS,
    query: mode === 'create' ? { action: 'create' } : undefined,
  });
}

function goToRoles(mode?: 'create') {
  void router.push({
    path: ACCESS_CONTROL_ROUTE_PATH.ROLES,
    query: mode === 'create' ? { action: 'create' } : undefined,
  });
}

onMounted(() => {
  fetchOverview();
});
</script>
<style scoped lang="less">
.access-control-overview__grid {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.section-head,
.risk-list__head,
.quick-link-card__head,
.todo-list__head,
.issue-section__heading {
  align-items: flex-start;
  display: flex;
  gap: 12px;
  justify-content: space-between;
}

.section-head h2 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
  margin: 0;
}

.section-head p,
.quick-link-card__action {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
  margin: 4px 0 0;
}

.issue-section {
  display: grid;
  gap: 16px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.issue-section__block,
.status-list {
  display: grid;
  gap: 12px;
}

.issue-section__heading h3 {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-small);
  margin: 0;
}

.issue-section__meta,
.status-list__label {
  color: var(--td-text-color-secondary);
  font: var(--td-font-body-small);
}

.risk-list {
  display: grid;
  gap: 12px;
}

.risk-list__item,
.todo-list__item,
.status-list__item {
  background: color-mix(in srgb, var(--td-brand-color) 4%, var(--td-bg-color-container));
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-medium);
  display: grid;
  gap: 6px;
  padding: 12px 14px;
}

.risk-list__label,
.risk-list__description,
.quick-link-card__description,
.todo-list__description,
.status-list__description {
  color: var(--td-text-color-secondary);
}

.risk-list__value,
.quick-link-card__count,
.status-list__value {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.risk-list__description,
.todo-list__description,
.status-list__description {
  font: var(--td-font-body-small);
  margin: 0;
}

.quick-link-grid {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.quick-link-card {
  background: color-mix(in srgb, var(--td-brand-color) 3%, var(--td-bg-color-container));
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--td-radius-large);
  box-shadow: var(--td-shadow-1);
  color: inherit;
  cursor: pointer;
  display: grid;
  gap: 8px;
  padding: 14px;
  text-align: left;
  transition:
    border-color 0.2s ease,
    transform 0.2s ease;
}

.quick-link-card:hover {
  border-color: var(--td-brand-color);
  box-shadow: var(--td-shadow-2);
  transform: translateY(-1px);
}

.quick-link-card__title {
  color: var(--td-text-color-primary);
  font: var(--td-font-title-medium);
}

.quick-link-card__description {
  margin: 0;
}

@media (width <= 900px) {
  .access-control-overview__grid,
  .issue-section {
    grid-template-columns: 1fr;
  }

  .quick-link-grid {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .section-head,
  .risk-list__head,
  .quick-link-card__head,
  .todo-list__head,
  .issue-section__heading {
    flex-direction: column;
  }
}
</style>
