<template>
  <section class="registry-page" data-page-type="list-form-detail">
    <management-page-content>
      <management-page-header
        title-key="menu.registries.title"
        :description="t('registry.list.description')"
        description-key="registry.list.description"
        :source="{ labelKey: 'menu.section.shared_resources', fallback: t('menu.section.shared_resources') }"
      >
        <template #actions>
          <t-button data-testid="registry-create" theme="primary" @click="openCreate">{{
            t('registry.list.add')
          }}</t-button>
        </template>
      </management-page-header>

      <resource-query-panel
        v-model="resourceQueryState"
        :config="queryConfig"
        :loading="loading"
        @reset="resetQuery"
        @search="applyQuery"
      />

      <management-paged-table
        v-model:current="pagination.current"
        v-model:page-size="pagination.pageSize"
        :columns="columns"
        :empty-description="
          t(resourceQueryState.keyword ? 'registry.list.filteredEmptyDescription' : 'registry.list.emptyDescription')
        "
        :empty-title="t(resourceQueryState.keyword ? 'registry.list.filteredEmpty' : 'registry.list.empty')"
        :footer-summary="t('registry.list.summary', { count: total })"
        :loading="loading"
        :rows="items"
        :total="total"
        row-key="connection_ref"
        :pagination-props="{ showPageNumber: true }"
        @page-change="handlePageChange"
      >
        <template #feedback><t-alert v-if="errorMessage" theme="error" :message="errorMessage" /></template>
        <template #toolbar>
          <table-view-toolbar :refresh-label="t('registry.list.refresh')" :refresh-loading="loading" @refresh="load" />
        </template>
        <template #credential="{ row }">
          <t-tag :theme="row.credential_configured ? 'success' : 'default'" variant="light">
            {{
              row.credential_configured
                ? t('registry.list.credential.configured')
                : t('registry.list.credential.anonymous')
            }}
          </t-tag>
        </template>
        <template #status="{ row }">
          <t-tag :theme="row.availability ? 'success' : 'warning'" variant="light">
            {{ registryStatusLabel(row) }}
          </t-tag>
        </template>
        <template #verified="{ row }">{{ formatLocaleDateTime(row.last_verified_at, locale) || '-' }}</template>
        <template #actions="{ row }">
          <table-action-menu
            :actions="registryRowActions(row)"
            :more-label="t('registry.list.more')"
            :more-label-fallback="t('registry.list.more')"
            @action="handleRegistryRowAction($event, row)"
          />
        </template>
        <template #empty-action>
          <t-button v-if="resourceQueryState.keyword" variant="outline" @click="resetQuery(resourceQueryState)">
            {{ t('registry.list.clearSearch') }}
          </t-button>
        </template>
      </management-paged-table>
    </management-page-content>

    <t-dialog
      v-model:visible="deleteDialogVisible"
      theme="danger"
      :header="t('registry.list.delete')"
      :body="t('registry.list.deleteConfirm')"
      :cancel-btn="t('registry.list.cancel')"
      :confirm-btn="{ content: t('registry.list.delete'), theme: 'danger', loading: deleting }"
      @confirm="confirmRemove"
    />

    <t-drawer
      v-model:visible="drawerVisible"
      :header="drawerTitle"
      size="min(720px, 92vw)"
      :confirm-btn="t('registry.list.save')"
      :confirm-loading="saving"
      @confirm="save"
    >
      <t-form :data="form" :rules="rules" label-align="top">
        <t-form-item :label="t('registry.list.form.connectionRef')" name="connection_ref">
          <t-input v-model="form.connection_ref" :disabled="Boolean(editingRef)" />
        </t-form-item>
        <t-form-item :label="t('registry.list.form.displayName')" name="display_name"
          ><t-input v-model="form.display_name"
        /></t-form-item>
        <t-form-item :label="t('registry.list.form.endpoint')" name="endpoint"
          ><t-input v-model="form.endpoint" placeholder="https://registry.example.com"
        /></t-form-item>
        <t-form-item :label="t('registry.list.form.description')"
          ><t-textarea v-model="form.description"
        /></t-form-item>
        <t-form-item :label="t('registry.list.form.enabled')"><t-switch v-model="form.enabled" /></t-form-item>
        <t-form-item :label="t('registry.list.form.insecure')"><t-switch v-model="form.insecure" /></t-form-item>
      </t-form>

      <section v-if="editingRef" class="registry-repositories">
        <div class="registry-repositories__header">
          <h2>{{ t('registry.list.repositories') }}</h2>
          <t-button size="small" variant="outline" @click="openCreateRepository">{{
            t('registry.list.addRepository')
          }}</t-button>
        </div>
        <t-table
          row-key="repository_ref"
          :data="repositories"
          :columns="repositoryColumns"
          :loading="repositoryLoading"
        >
          <template #repositoryActions="{ row }">
            <t-button theme="primary" variant="text" @click="openEditRepository(row)">
              {{ t('registry.list.edit') }}
            </t-button>
            <t-button theme="primary" variant="text" @click="openAssignments(row.repository_ref)">
              {{ t('registry.list.assignments') }}
            </t-button>
            <t-popconfirm
              :content="t('registry.list.repositoryDeleteConfirm', { repository: row.repository_ref })"
              @confirm="removeRepository(row.repository_ref)"
            >
              <t-button theme="danger" variant="text">{{ t('registry.list.delete') }}</t-button>
            </t-popconfirm>
          </template>
        </t-table>
      </section>
    </t-drawer>

    <t-drawer
      v-model:visible="assignmentDrawerVisible"
      :header="t('registry.list.assignments')"
      size="480px"
      :footer="false"
    >
      <t-form :data="assignmentForm" label-align="top" @submit="grantAssignment">
        <t-form-item :label="t('registry.list.form.userId')" name="user_id">
          <t-input-number v-model="assignmentForm.user_id" :min="1" />
        </t-form-item>
        <t-button theme="primary" type="submit" :loading="assignmentSaving">
          {{ t('registry.list.grantAssignment') }}
        </t-button>
      </t-form>
      <t-table row-key="user_id" :data="assignments" :columns="assignmentColumns" :loading="assignmentLoading">
        <template #created_at="{ row }">{{ formatLocaleDateTime(row.created_at, locale) }}</template>
        <template #assignmentActions="{ row }">
          <t-popconfirm
            :content="t('registry.list.revokeAssignmentConfirm', { userId: row.user_id })"
            @confirm="revokeAssignment(row.user_id)"
          >
            <t-button theme="danger" variant="text">{{ t('registry.list.revokeAssignment') }}</t-button>
          </t-popconfirm>
        </template>
      </t-table>
    </t-drawer>

    <t-drawer
      v-model:visible="repositoryDrawerVisible"
      :header="editingRepositoryRef ? t('registry.list.edit') : t('registry.list.addRepository')"
      size="480px"
      :confirm-btn="t('registry.list.save')"
      :confirm-loading="repositorySaving"
      @confirm="saveRepository"
    >
      <t-form :data="repositoryForm" label-align="top">
        <t-form-item :label="t('registry.list.form.repositoryRef')"
          ><t-input v-model="repositoryForm.repository_ref" :disabled="Boolean(editingRepositoryRef)"
        /></t-form-item>
        <t-form-item :label="t('registry.list.form.repositoryName')"
          ><t-input v-model="repositoryForm.display_name"
        /></t-form-item>
        <t-form-item :label="t('registry.list.form.allowPull')"
          ><t-switch v-model="repositoryForm.allow_pull"
        /></t-form-item>
        <t-form-item :label="t('registry.list.form.allowPush')"
          ><t-switch v-model="repositoryForm.allow_push"
        /></t-form-item>
      </t-form>
    </t-drawer>
  </section>
</template>
<script setup lang="ts">
// Registry 管理页协调连接、仓库路径和用户授权；Build 仅通过受限目的地 API 消费这些事实。
import type { PageInfo, TableProps } from 'tdesign-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useI18n } from 'vue-i18n';

import type { components } from '@/contracts/openapi/generated/schema';
import {
  createActionColumn,
  ManagementPageContent,
  ManagementPageHeader,
  TableActionMenu,
  TableViewToolbar,
} from '@/shared/components/management';
import ManagementPagedTable from '@/shared/components/management/ManagementPagedTable.vue';
import { type ResourceQueryConfig, ResourceQueryPanel, type ResourceQueryState } from '@/shared/components/query-list';
import { resolveLocalizedErrorMessage } from '@/shared/localized-api-error';
import { formatLocaleDateTime } from '@/shared/observability';

import {
  createRegistry,
  createRegistryRepository,
  deleteRegistry,
  deleteRegistryRepository,
  getRegistries,
  getRegistryRepositories,
  getRegistryRepositoryAssignments,
  grantRegistryRepositoryAssignment,
  revokeRegistryRepositoryAssignment,
  updateRegistry,
  updateRegistryRepository,
  verifyRegistry,
} from '../../api/registry';

type RegistryConnection = components['schemas']['registry-connection'];
type RegistryRepository = components['schemas']['registry-artifact-repository'];
type RegistryAssignment = components['schemas']['registry-artifact-repository-user-assignment'];

const { locale, t } = useI18n();
const items = ref<RegistryConnection[]>([]);
const repositories = ref<RegistryRepository[]>([]);
const assignments = ref<RegistryAssignment[]>([]);
const pagination = ref({ current: 1, pageSize: 20 });
const search = ref('');
const total = ref(0);
const loading = ref(false);
const repositoryLoading = ref(false);
const saving = ref(false);
const repositorySaving = ref(false);
const verifying = ref('');
const deleting = ref(false);
const errorMessage = ref('');
const drawerVisible = ref(false);
const repositoryDrawerVisible = ref(false);
const assignmentDrawerVisible = ref(false);
const editingRef = ref('');
const assignmentRepositoryRef = ref('');
const editingRepositoryRef = ref('');
const deleteDialogVisible = ref(false);
const deleteTargetRef = ref('');
const form = ref({
  connection_ref: '',
  display_name: '',
  provider: 'generic_oci' as const,
  endpoint: '',
  enabled: true,
  insecure: false,
  description: '',
});
const repositoryForm = ref({ repository_ref: '', display_name: '', allow_pull: true, allow_push: true });
const assignmentForm = ref({ user_id: undefined as number | undefined });
const assignmentLoading = ref(false);
const assignmentSaving = ref(false);
let registryListRequestSequence = 0;
let repositoryListRequestSequence = 0;
let assignmentListRequestSequence = 0;

const drawerTitle = computed(() => (editingRef.value ? t('registry.list.edit') : t('registry.list.add')));
const rules = computed(() => ({
  connection_ref: [{ required: true, message: t('registry.list.form.connectionRef') }],
  display_name: [{ required: true, message: t('registry.list.form.displayName') }],
  endpoint: [{ required: true, message: t('registry.list.form.endpoint') }],
}));
const queryConfig = computed<ResourceQueryConfig>(() => ({
  resource: 'registry.list',
  search: true,
  filterBuilder: { enabled: false },
  placeholder: t('registry.list.search'),
}));
const resourceQueryState = computed<ResourceQueryState>({
  get: () => ({
    keyword: search.value,
    filters: {},
    page: pagination.value.current,
    pageSize: pagination.value.pageSize,
  }),
  set: (value) => {
    search.value = value.keyword;
    pagination.value.current = value.page;
    pagination.value.pageSize = value.pageSize;
  },
});
const columns = computed<TableProps['columns']>(() => [
  { colKey: 'display_name', title: t('registry.list.columns.name'), minWidth: 160 },
  { colKey: 'provider', title: t('registry.list.columns.type'), width: 130 },
  { colKey: 'endpoint', title: t('registry.list.columns.endpoint'), minWidth: 230 },
  { colKey: 'credential', title: t('registry.list.columns.credential'), width: 120 },
  { colKey: 'status', title: t('registry.list.columns.status'), width: 130 },
  { colKey: 'verified', title: t('registry.list.columns.verified'), minWidth: 170 },
  createActionColumn(t('registry.list.columns.actions'), 160),
]);
const repositoryColumns = computed<TableProps['columns']>(() => [
  { colKey: 'repository_ref', title: t('registry.list.form.repositoryRef'), minWidth: 220 },
  { colKey: 'display_name', title: t('registry.list.form.repositoryName'), minWidth: 160 },
  { colKey: 'repositoryActions', title: t('registry.list.columns.actions'), width: 240 },
]);
const assignmentColumns = computed<TableProps['columns']>(() => [
  { colKey: 'user_id', title: t('registry.list.form.userId'), width: 120 },
  { colKey: 'created_at', title: t('registry.list.columns.assignmentCreatedAt'), minWidth: 180 },
  { colKey: 'assignmentActions', title: t('registry.list.columns.actions'), width: 120 },
]);

onMounted(() => void load());

async function load() {
  const requestSequence = ++registryListRequestSequence;
  loading.value = true;
  errorMessage.value = '';
  try {
    const response = await getRegistries({
      search: search.value || undefined,
      limit: pagination.value.pageSize,
      offset: (pagination.value.current - 1) * pagination.value.pageSize,
    });
    if (requestSequence !== registryListRequestSequence) {
      return;
    }

    items.value = response.items ?? [];
    total.value = response.total ?? 0;
  } catch (error) {
    if (requestSequence !== registryListRequestSequence) {
      return;
    }

    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    if (requestSequence === registryListRequestSequence) {
      loading.value = false;
    }
  }
}
function applyQuery(value: ResourceQueryState) {
  resourceQueryState.value = value;
  pagination.value.current = 1;
  void load();
}
function resetQuery(value: ResourceQueryState) {
  resourceQueryState.value = value;
  pagination.value.current = 1;
  void load();
}
function handlePageChange(pageInfo: PageInfo) {
  pagination.value.current = pageInfo.current;
  pagination.value.pageSize = pageInfo.pageSize;
  void load();
}
function registryStatusLabel(row: RegistryConnection) {
  if (row.availability) {
    return t('registry.list.status.available');
  }

  switch (row.verification_status) {
    case 'failed':
      return t('registry.list.status.failed');
    case 'unknown':
      return t('registry.list.status.unknown');
    case 'verified':
      return t('registry.list.status.unavailable');
  }

  return t('registry.list.status.unknown');
}
function registryRowActions(row: RegistryConnection) {
  return [
    { label: t('registry.list.edit'), value: 'edit' },
    {
      disabled: verifying.value === row.connection_ref,
      label: t('registry.list.verify'),
      value: 'verify',
    },
    { danger: true, label: t('registry.list.delete'), value: 'delete' },
  ];
}
function handleRegistryRowAction(action: string, row: RegistryConnection) {
  if (action === 'edit') {
    void openEdit(row);
    return;
  }
  if (action === 'verify') {
    void verify(row.connection_ref);
    return;
  }
  if (action === 'delete') {
    deleteTargetRef.value = row.connection_ref;
    deleteDialogVisible.value = true;
  }
}
function openCreate() {
  editingRef.value = '';
  repositories.value = [];
  form.value = {
    connection_ref: '',
    display_name: '',
    provider: 'generic_oci',
    endpoint: '',
    enabled: true,
    insecure: false,
    description: '',
  };
  drawerVisible.value = true;
}
async function openEdit(row: RegistryConnection) {
  editingRef.value = row.connection_ref;
  form.value = {
    connection_ref: row.connection_ref,
    display_name: row.display_name,
    provider: 'generic_oci',
    endpoint: row.endpoint,
    enabled: row.enabled,
    insecure: row.insecure,
    description: row.description ?? '',
  };
  drawerVisible.value = true;
  await loadRepositories();
}
async function save() {
  saving.value = true;
  try {
    if (editingRef.value)
      await updateRegistry(editingRef.value, {
        display_name: form.value.display_name,
        endpoint: form.value.endpoint,
        enabled: form.value.enabled,
        insecure: form.value.insecure,
        description: form.value.description || null,
      });
    else await createRegistry({ ...form.value, description: form.value.description || null });
    drawerVisible.value = false;
    await load();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    saving.value = false;
  }
}
async function verify(connectionRef: string) {
  verifying.value = connectionRef;
  try {
    const result = await verifyRegistry(connectionRef);
    errorMessage.value = t('registry.list.verifyResult', { status: result.status });
    await load();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    verifying.value = '';
  }
}
async function confirmRemove() {
  if (!deleteTargetRef.value) return;
  deleting.value = true;
  try {
    await deleteRegistry(deleteTargetRef.value);
    deleteDialogVisible.value = false;
    deleteTargetRef.value = '';
    await load();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    deleting.value = false;
  }
}
async function loadRepositories() {
  const connectionRef = editingRef.value;
  if (!connectionRef) return;
  const requestSequence = ++repositoryListRequestSequence;
  repositoryLoading.value = true;
  try {
    const response = await getRegistryRepositories(connectionRef);
    if (requestSequence !== repositoryListRequestSequence || editingRef.value !== connectionRef) return;
    repositories.value = response.items ?? [];
  } finally {
    if (requestSequence === repositoryListRequestSequence) repositoryLoading.value = false;
  }
}
async function saveRepository() {
  if (!editingRef.value) return;
  repositorySaving.value = true;
  try {
    if (editingRepositoryRef.value) {
      await updateRegistryRepository(editingRef.value, editingRepositoryRef.value, {
        display_name: repositoryForm.value.display_name,
        allow_pull: repositoryForm.value.allow_pull,
        allow_push: repositoryForm.value.allow_push,
      });
    } else {
      await createRegistryRepository(editingRef.value, repositoryForm.value);
    }
    repositoryDrawerVisible.value = false;
    editingRepositoryRef.value = '';
    repositoryForm.value = { repository_ref: '', display_name: '', allow_pull: true, allow_push: true };
    await loadRepositories();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    repositorySaving.value = false;
  }
}
function openCreateRepository() {
  editingRepositoryRef.value = '';
  repositoryForm.value = { repository_ref: '', display_name: '', allow_pull: true, allow_push: true };
  repositoryDrawerVisible.value = true;
}
function openEditRepository(repository: RegistryRepository) {
  editingRepositoryRef.value = repository.repository_ref;
  repositoryForm.value = {
    repository_ref: repository.repository_ref,
    display_name: repository.display_name,
    allow_pull: repository.allow_pull,
    allow_push: repository.allow_push,
  };
  repositoryDrawerVisible.value = true;
}
async function removeRepository(repositoryRef: string) {
  if (!editingRef.value) return;
  try {
    await deleteRegistryRepository(editingRef.value, repositoryRef);
    await loadRepositories();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  }
}
async function openAssignments(repositoryRef: string) {
  assignmentRepositoryRef.value = repositoryRef;
  assignmentForm.value = { user_id: undefined };
  assignmentDrawerVisible.value = true;
  await loadAssignments();
}
async function loadAssignments() {
  const connectionRef = editingRef.value;
  const repositoryRef = assignmentRepositoryRef.value;
  if (!connectionRef || !repositoryRef) return;
  const requestSequence = ++assignmentListRequestSequence;
  assignmentLoading.value = true;
  try {
    const response = await getRegistryRepositoryAssignments(connectionRef, repositoryRef);
    if (
      requestSequence !== assignmentListRequestSequence ||
      editingRef.value !== connectionRef ||
      assignmentRepositoryRef.value !== repositoryRef
    )
      return;
    assignments.value = response.items ?? [];
  } catch (error) {
    if (requestSequence === assignmentListRequestSequence) {
      errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
    }
  } finally {
    if (requestSequence === assignmentListRequestSequence) assignmentLoading.value = false;
  }
}
async function grantAssignment() {
  if (!editingRef.value || !assignmentRepositoryRef.value || !assignmentForm.value.user_id) return;
  assignmentSaving.value = true;
  try {
    await grantRegistryRepositoryAssignment(editingRef.value, assignmentRepositoryRef.value, {
      user_id: assignmentForm.value.user_id,
    });
    assignmentForm.value = { user_id: undefined };
    await loadAssignments();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  } finally {
    assignmentSaving.value = false;
  }
}
async function revokeAssignment(userId: number) {
  if (!editingRef.value || !assignmentRepositoryRef.value) return;
  try {
    await revokeRegistryRepositoryAssignment(editingRef.value, assignmentRepositoryRef.value, userId);
    await loadAssignments();
  } catch (error) {
    errorMessage.value = resolveLocalizedErrorMessage(t, error, t('registry.list.loadFailed'));
  }
}
</script>
<style scoped lang="less">
.registry-page {
  min-width: 0;
}

.registry-repositories__header {
  align-items: center;
  display: flex;
  gap: var(--graft-density-gap-12);
  justify-content: space-between;
}

.registry-repositories h2 {
  margin: 0;
}

.registry-repositories {
  display: grid;
  gap: var(--graft-density-gap-12);
  margin-top: var(--graft-density-gap-24);
}
</style>
